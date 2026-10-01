package signal

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// joinClaims 与 webrtc-signaling 的 JoinClaims 结构保持一致，
// 使 webrtc-call 能校验 signaling 签发的 join-token。
type joinClaims struct {
	Rid         string `json:"rid,omitempty"`
	Role        string `json:"role,omitempty"`
	DisplayName string `json:"name,omitempty"`
	jwt.RegisteredClaims
}

// parseJoinToken 校验并解析 join-token,返回其中绑定的房间 ID。
// 校验算法、claim 结构与 webrtc-signaling 完全一致(HMAC-SHA256)。
func parseJoinToken(tokenStr string, secret []byte) (roomID string, err error) {
	if tokenStr == "" {
		return "", errors.New("empty token")
	}
	claims := &joinClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", err
	}
	if !parsed.Valid {
		return "", errors.New("invalid token")
	}
	if claims.Rid == "" {
		return "", errors.New("token missing room claim")
	}
	if time.Now().After(claims.ExpiresAt.Time) {
		return "", errors.New("token expired")
	}
	return claims.Rid, nil
}
