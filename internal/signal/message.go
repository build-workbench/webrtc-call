package signal

import "encoding/json"

// 消息类型常量
// 定义所有支持的信令消息类型，避免字符串字面量散落各处。
const (
	// 客户端发送的消息类型
	MsgTypeJoin      = "join"
	MsgTypeLeave     = "leave"
	MsgTypePing      = "ping"
	MsgTypeOffer     = "offer"
	MsgTypeAnswer    = "answer"
	MsgTypeCandidate = "candidate"
	MsgTypeHangup    = "hangup"

	// 服务器发送的消息类型
	MsgTypeJoined      = "joined"
	MsgTypePong        = "pong"
	MsgTypeRoomMembers = "room_members"
	MsgTypeError       = "error"
)

// Message 表示信令消息结构。
// 字段含义取决于 Type：
//   - join: Room, From 必填
//   - offer/answer: To, SDP 必填，From 由服务器填充
//   - candidate: To, Candidate 必填，From 由服务器填充
//   - hangup: To 可选（空则广播），From 由服务器填充
type Message struct {
	Type      string          `json:"type"`
	Room      string          `json:"room"`
	From      string          `json:"from"`
	To        string          `json:"to,omitempty"`
	SDP       json.RawMessage `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
	Members   []string        `json:"members,omitempty"`
	Code      string          `json:"code,omitempty"`
	Error     string          `json:"error,omitempty"`
}


