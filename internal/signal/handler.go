package signal

import (
	"errors"
	"log"
	"net/http"
	"time"
)

// HandleWS 处理 WebSocket 连接升级与消息处理。
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	// 配置了共享密钥时强制校验 join-token：连接必须携带有效的 ?token=,
	// 且 token 绑定的房间与后续 join 消息一致(在 handleJoin 中复核)。
	allowedRoom := ""
	if len(h.authSecret) > 0 {
		tok := r.URL.Query().Get("token")
		room, err := parseJoinToken(tok, h.authSecret)
		if err != nil {
			log.Printf("signal: ws auth rejected from %s path=%s: %v", r.RemoteAddr, r.URL.Path, err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		allowedRoom = room
	}

	conn, err := h.upg.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("signal: ws upgrade failed from %s path=%s: %v", r.RemoteAddr, r.URL.Path, err)
		return
	}

	client := &Client{
		connID: h.nextConnID.Add(1),
		conn:   conn,
		send:   make(chan Message, SendBufferSize),
		closed: make(chan struct{}),
	}
	if allowedRoom != "" {
		client.setAllowedRoom(allowedRoom)
	}
	if !h.registerClient(client) {
		_ = conn.Close()
		return
	}

	log.Printf("signal: ws connected from %s conn=%d", r.RemoteAddr, client.connID)
	defer func() {
		h.removeClient(client)
		h.unregisterClient(client)
		client.close()
	}()

	go client.writePump()

	conn.SetReadLimit(MaxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
		log.Printf("signal: set initial read deadline failed conn=%d: %v", client.connID, err)
		client.close()
		return
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
			log.Printf("signal: set pong read deadline failed conn=%d: %v", client.connID, err)
			return err
		}
		return nil
	})

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			id, room := client.identity()
			log.Printf("signal: read message error room=%s id=%s conn=%d: %v", room, id, client.connID, err)
			return
		}
		if err := h.handleMessage(client, msg); err != nil {
			log.Printf("signal: handle message error conn=%d type=%s: %v", client.connID, msg.Type, err)
		}
	}
}

// handleMessage 将收到的消息路由到对应的处理函数。
func (h *Hub) handleMessage(client *Client, msg Message) error {
	// 速率限制检查
	if !client.checkRateLimit() {
		_ = client.sendErrorAndLog(ErrRateLimited)
		return errors.New("rate limited")
	}

	switch msg.Type {
	case MsgTypeJoin:
		return h.handleJoin(client, msg)
	case MsgTypeLeave:
		h.removeClient(client)
		return nil
	case MsgTypePing:
		return client.enqueue(Message{Type: MsgTypePong})
	case MsgTypeOffer, MsgTypeAnswer, MsgTypeCandidate, MsgTypeHangup:
		return h.forward(client, msg)
	default:
		_ = client.sendErrorAndLog(ErrUnknownType)
		return errors.New("unknown message type")
	}
}
