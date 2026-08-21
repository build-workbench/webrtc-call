package signal

import (
	"errors"
	"strings"
	"unicode"
)

// handleJoin 处理加入房间的请求。
func (h *Hub) handleJoin(c *Client, msg Message) error {
	id := normalizeClientID(msg.From, MaxClientIDLength)
	if id == "" {
		_ = c.sendErrorAndLog(ErrInvalidID)
		return errors.New("invalid client id")
	}
	room := normalizeRoomName(msg.Room, MaxRoomIDLength)
	if room == "" {
		_ = c.sendErrorAndLog(ErrInvalidRoom)
		return errors.New("invalid room")
	}

	boundID, currentRoom := c.identity()
	if boundID != "" && boundID != id {
		_ = c.sendErrorAndLog(ErrIdentityLocked)
		return errors.New("identity mismatch")
	}
	if currentRoom != "" && currentRoom != room {
		_ = c.sendErrorAndLog(ErrAlreadyJoined)
		return errors.New("already joined")
	}

	c.setIdentity(id, room)
	if err := h.addClient(c); err != nil {
		c.setIdentity("", "") // 失败时清空 ID 与房间
		_ = c.sendErrorAndLog(err)
		return err
	}
	if err := c.enqueue(Message{Type: MsgTypeJoined, Room: room, From: id}); err != nil {
		return err
	}
	h.broadcastMembers(room)
	return nil
}

// normalizeClientID 校验并规范化客户端 ID。
// 非法时返回空字符串。
func normalizeClientID(raw string, maxLen int) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > maxLen {
		return ""
	}
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return ""
		}
	}
	return trimmed
}

// normalizeRoomName 校验并规范化房间名。
// 非法时返回空字符串。
func normalizeRoomName(raw string, maxLen int) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > maxLen {
		return ""
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return ""
		}
	}
	return trimmed
}
