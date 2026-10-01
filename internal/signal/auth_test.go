package signal

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// signTestToken 使用与 webrtc-signaling 相同的结构和算法签发测试 token。
func signTestToken(t *testing.T, secret []byte, roomID, userID string, ttl time.Duration) string {
	t.Helper()
	claims := joinClaims{
		Rid:  roomID,
		Role: "speaker",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-5 * time.Second)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func TestParseJoinTokenValid(t *testing.T) {
	secret := []byte("shared-secret")
	tok := signTestToken(t, secret, "room-7", "u1", time.Minute)

	room, err := parseJoinToken(tok, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room != "room-7" {
		t.Fatalf("expected room room-7, got %q", room)
	}
}

func TestParseJoinTokenRejects(t *testing.T) {
	secret := []byte("shared-secret")
	other := []byte("different-secret")

	cases := []struct {
		name  string
		token string
		sec   []byte
	}{
		{"empty token", "", secret},
		{"wrong secret", signTestToken(t, other, "r", "u", time.Minute), secret},
		{"expired", signTestToken(t, secret, "r", "u", -time.Minute), secret},
		{"garbage", "not-a-jwt", secret},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseJoinToken(tc.token, tc.sec); err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

func TestParseJoinTokenMissingRoomClaim(t *testing.T) {
	secret := []byte("shared-secret")
	claims := joinClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if _, err := parseJoinToken(s, secret); err == nil {
		t.Fatal("expected error for missing rid claim")
	}
}

func TestParseJoinTokenRejectsNonHS256(t *testing.T) {
	secret := []byte("shared-secret")
	// 用 RS256 需要 RSA 密钥,这里用算法混淆的 HS256 token 绕不过白名单;
	// 直接验证不支持的签名方法名也会被拒。
	tok := signTestToken(t, secret, "r", "u", time.Minute)
	// 篡改算法头为 none,应被 WithValidMethods 拒绝
	parts := splitTokenParts(t, tok)
	parts[0] = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0" // {"alg":"none","typ":"JWT"}
	altered := parts[0] + "." + parts[1] + "." + parts[2]
	if _, err := parseJoinToken(altered, secret); err == nil {
		t.Fatal("expected error for alg=none token")
	}
}

func splitTokenParts(t *testing.T, tok string) []string {
	t.Helper()
	// JWT 由三段组成,这里简单按 "." 切分。
	parts := []string{}
	start := 0
	for i := 0; i < len(tok); i++ {
		if tok[i] == '.' {
			parts = append(parts, tok[start:i])
			start = i + 1
		}
	}
	parts = append(parts, tok[start:])
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	return parts
}

// TestHandleWSAuth 验证启用共享密钥后:无 token 拒绝、有效 token 可连、
// join 房间与 token 绑定一致才允许。
func TestHandleWSAuth(t *testing.T) {
	const secret = "shared-secret"
	h := NewHubWithOptions(Options{AuthSecret: []byte(secret)})
	ts := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer ts.Close()

	wsURL := "ws" + ts.URL[4:] // http:// -> ws://

	t.Run("rejects without token", func(t *testing.T) {
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL+"/ws", nil)
		if err == nil {
			conn.Close()
			t.Fatal("expected dial to fail without token")
		}
		if resp != nil && resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL+"/ws?token=bad-token", nil)
		if err == nil {
			conn.Close()
			t.Fatal("expected dial to fail with invalid token")
		}
		if resp != nil && resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("accepts valid token and enforces room binding", func(t *testing.T) {
		tok := signTestToken(t, []byte(secret), "room-auth", "u1", time.Minute)
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL+"/ws?token="+tok, nil)
		if err != nil {
			t.Fatalf("dial with valid token: %v", err)
		}
		defer conn.Close()
		if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
			t.Fatalf("expected 101, got %d", resp.StatusCode)
		}

		// join 与 token 绑定一致 -> 成功
		if err := conn.WriteJSON(Message{Type: MsgTypeJoin, Room: "room-auth", From: "u1"}); err != nil {
			t.Fatalf("write join: %v", err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var joined Message
		if err := conn.ReadJSON(&joined); err != nil {
			t.Fatalf("read joined: %v", err)
		}
		if joined.Type != MsgTypeJoined {
			t.Fatalf("expected joined, got %s", joined.Type)
		}
	})

	t.Run("rejects join to room not in token", func(t *testing.T) {
		tok := signTestToken(t, []byte(secret), "room-bound", "u2", time.Minute)
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+"/ws?token="+tok, nil)
		if err != nil {
			t.Fatalf("dial with valid token: %v", err)
		}
		defer conn.Close()

		if err := conn.WriteJSON(Message{Type: MsgTypeJoin, Room: "other-room", From: "u2"}); err != nil {
			t.Fatalf("write join: %v", err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var respMsg Message
		if err := conn.ReadJSON(&respMsg); err != nil {
			t.Fatalf("read response: %v", err)
		}
		if respMsg.Type != MsgTypeError {
			t.Fatalf("expected error for mismatched room, got %s", respMsg.Type)
		}
	})
}
