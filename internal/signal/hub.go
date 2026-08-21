package signal

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// 资源消耗上限,防止服务被拖垮
const (
	MaxRooms          = 1000
	MaxClientsPerRoom = 50
	MaxRoomIDLength   = 64
	MaxClientIDLength = 64
	SendBufferSize    = 64
	SendTimeout       = 2 * time.Second
	WriteWait         = 10 * time.Second
	PongWait          = 20 * time.Second
	PingPeriod        = 15 * time.Second
	MaxMessageSize    = 1 << 20
	// 速率限制:每客户端每秒最大消息数
	MaxMessagesPerSecond = 30
	RateLimitBurst       = 50
)

var errClientClosed = errors.New("client closed")

// Options 配置 Hub 的行为。
type Options struct {
	AllowedOrigins  []string
	AllowAllOrigins bool
}

// Hub 管理房间,并在客户端之间路由信令消息。
type Hub struct {
	mu      sync.RWMutex
	rooms   map[string]map[string]*Client
	clients map[*Client]struct{}
	upg     websocket.Upgrader

	allowedOrigins  []string
	allowAllOrigins bool
	closed          bool
	nextConnID      atomic.Uint64
}

// NewHubWithOptions 使用自定义选项创建新的 Hub。
func NewHubWithOptions(opts Options) *Hub {
	h := &Hub{
		rooms:           make(map[string]map[string]*Client),
		clients:         make(map[*Client]struct{}),
		allowedOrigins:  append([]string(nil), opts.AllowedOrigins...),
		allowAllOrigins: opts.AllowAllOrigins,
	}
	h.upg = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return h.isOriginAllowed(r)
		},
	}
	return h
}

// registerClient 将客户端加入 Hub 的客户端注册表。
func (h *Hub) registerClient(c *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return false
	}
	h.clients[c] = struct{}{}
	return true
}

// unregisterClient 从 Hub 的客户端注册表中移除客户端。
func (h *Hub) unregisterClient(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// addClient 将客户端加入房间。
func (h *Hub) addClient(c *Client) *ProtocolError {
	id, room := c.identity()
	h.mu.Lock()
	defer h.mu.Unlock()
	if room == "" || id == "" {
		return ErrInvalidJoin
	}
	m, ok := h.rooms[room]
	if !ok {
		if len(h.rooms) >= MaxRooms {
			log.Printf("signal: room limit reached (%d), rejecting join room=%s id=%s", MaxRooms, room, id)
			return ErrRoomLimitReached
		}
		m = make(map[string]*Client)
		h.rooms[room] = m
	}
	if existing, exists := m[id]; exists {
		if existing == c {
			return nil
		}
		log.Printf("signal: duplicate id rejected room=%s id=%s", room, id)
		return ErrDuplicateID
	}
	if len(m) >= MaxClientsPerRoom {
		log.Printf("signal: room %s full (%d clients), rejecting id=%s", room, MaxClientsPerRoom, id)
		return ErrRoomFull
	}
	m[id] = c
	log.Printf("signal: join room=%s id=%s conn=%d (room size: %d, total rooms: %d)", room, id, c.connID, len(m), len(h.rooms))
	return nil
}

// removeClient 将客户端从其所在房间移除。
func (h *Hub) removeClient(c *Client) {
	id, room := c.identity()
	if room == "" || id == "" {
		return
	}

	var shouldBroadcast bool

	h.mu.Lock()
	if m, ok := h.rooms[room]; ok {
		current, ok2 := m[id]
		if !ok2 || current != c {
			h.mu.Unlock()
			c.setRoom("")
			return
		}
		delete(m, id)
		log.Printf("signal: leave room=%s id=%s conn=%d", room, id, c.connID)
		if len(m) == 0 {
			delete(h.rooms, room)
			log.Printf("signal: room %s closed", room)
		} else {
			shouldBroadcast = true
		}
	}
	h.mu.Unlock()
	c.setRoom("")
	if shouldBroadcast {
		h.broadcastMembers(room)
	}
}

// broadcastMembers 向房间内所有客户端广播成员列表。
func (h *Hub) broadcastMembers(room string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	m, ok := h.rooms[room]
	if !ok {
		return
	}

	// 单个临界区快照:同时捕获成员 ID 与客户端指针
	members := make([]string, 0, len(m))
	recipients := make([]*Client, 0, len(m))
	for id, cli := range m {
		members = append(members, id)
		recipients = append(recipients, cli)
	}

	sort.Strings(members)
	msg := Message{
		Type:    MsgTypeRoomMembers,
		Room:    room,
		Members: members,
	}

	// 持锁期间向所有接收方发送(enqueue 为非阻塞)
	for _, cli := range recipients {
		if err := cli.enqueue(msg); err != nil {
			log.Printf("signal: members broadcast failed room=%s conn=%d: %v", room, cli.connID, err)
			// 异步移除客户端以避免死锁
			go h.removeClient(cli)
			go cli.close()
		}
	}
}

// Close 关闭 Hub 并断开所有客户端连接。
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.Unlock()

	for _, client := range clients {
		h.removeClient(client)
		client.close()
		h.unregisterClient(client)
	}
}

// IsClosed 返回 Hub 是否已关闭。
func (h *Hub) IsClosed() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.closed
}

// isOriginAllowed 检查请求的 Origin 是否被允许。
func (h *Hub) isOriginAllowed(r *http.Request) bool {
	if h.allowAllOrigins {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		host := r.Host
		// 检查是否为 localhost(正确处理 host:port)
		if isLocalhostHost(host) {
			return true
		}
		return false
	}
	if len(h.allowedOrigins) == 0 {
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return isLocalhostHost(u.Hostname())
	}
	for _, o := range h.allowedOrigins {
		if o == origin {
			return true
		}
	}
	return false
}

// isLocalhostHost 判断主机字符串(带或不带 :port)是否指向 localhost。
func isLocalhostHost(host string) bool {
	// 去掉端口(若存在)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
