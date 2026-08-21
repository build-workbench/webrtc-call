package signal

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client 表示一个已连接的 WebSocket 客户端。
type Client struct {
	mu        sync.RWMutex
	id        string
	room      string
	connID    uint64
	conn      *websocket.Conn
	send      chan Message
	closed    chan struct{}
	closeOnce sync.Once
	// 速率限制
	msgCount    int
	msgWindow   time.Time
	rateLimited bool
}

// identity 返回客户端的 ID 与房间。
func (c *Client) identity() (userID, userRoom string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id, c.room
}

// setIdentity 设置客户端的 ID 与房间。
func (c *Client) setIdentity(id, room string) {
	c.mu.Lock()
	c.id = id
	c.room = room
	c.mu.Unlock()
}

// setRoom 设置客户端的房间。
func (c *Client) setRoom(room string) {
	c.mu.Lock()
	c.room = room
	c.mu.Unlock()
}

// checkRateLimit 实现令牌桶速率限制。
// 若消息应被放行返回 true,被限速则返回 false。
// 允许最多 RateLimitBurst(50)条突发,之后强制 MaxMessagesPerSecond(每秒 30 条)。
func (c *Client) checkRateLimit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	// 若超过 1 秒则重置统计窗口
	if now.Sub(c.msgWindow) >= time.Second {
		c.msgWindow = now
		c.msgCount = 0
		c.rateLimited = false
	}

	c.msgCount++

	// 第一重检查:绝对突发上限(硬上限 50)
	if c.msgCount > RateLimitBurst {
		if !c.rateLimited {
			c.rateLimited = true
			log.Printf("signal: rate limiting client conn=%d (burst exceeded)", c.connID)
		}
		return false
	}

	// 第二重检查:首秒突发窗口之后的每秒速率限制
	// 仅当首秒内已超过突发配额(30)时强制执行
	if c.msgCount > MaxMessagesPerSecond && now.Sub(c.msgWindow) < time.Second {
		if !c.rateLimited {
			c.rateLimited = true
			log.Printf("signal: rate limiting client conn=%d (rate exceeded)", c.connID)
		}
		return false
	}

	return true
}

// sendError 向客户端发送错误消息。
func (c *Client) sendError(err *ProtocolError) error {
	_, room := c.identity()
	return c.enqueue(Message{Type: MsgTypeError, Room: room, Code: err.Code, Error: err.Message})
}

// sendErrorAndLog 向客户端发送错误消息;若发送失败则记录日志。
func (c *Client) sendErrorAndLog(err *ProtocolError) error {
	sendErr := c.sendError(err)
	if sendErr != nil {
		log.Printf("signal: failed to send %s error to conn=%d: %v", err.Code, c.connID, sendErr)
	}
	return sendErr
}

// enqueue 将消息排队,待发送给客户端。
func (c *Client) enqueue(msg Message) error {
	select {
	case <-c.closed:
		return errClientClosed
	default:
	}

	timer := time.NewTimer(SendTimeout)

	select {
	case <-c.closed:
		timer.Stop()
		return errClientClosed
	case c.send <- msg:
		// 排空定时器,防止资源泄漏
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		return nil
	case <-timer.C:
		return errors.New("send timeout")
	}
}

// close 关闭客户端的 WebSocket 连接。
func (c *Client) close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		if c.conn == nil {
			return
		}
		defer func() {
			if r := recover(); r != nil {
				log.Printf("signal: conn close panic conn=%d: %v", c.connID, r)
			}
		}()
		_ = c.conn.Close()
	})
}

// writePump 负责向 WebSocket 连接发送消息。
func (c *Client) writePump() {
	ticker := time.NewTicker(PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.closed:
			return
		case msg := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(WriteWait)); err != nil {
				log.Printf("signal: set write deadline failed conn=%d: %v", c.connID, err)
				c.close()
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				id, room := c.identity()
				log.Printf("signal: write message error room=%s id=%s conn=%d: %v", room, id, c.connID, err)
				c.close()
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(WriteWait)); err != nil {
				log.Printf("signal: set ping deadline failed conn=%d: %v", c.connID, err)
				c.close()
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				id, room := c.identity()
				log.Printf("signal: ping failed room=%s id=%s conn=%d: %v", room, id, c.connID, err)
				c.close()
				return
			}
		}
	}
}
