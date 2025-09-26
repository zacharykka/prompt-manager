package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 定义标准化的访问令牌载荷。
type Claims struct {
	UserID    string            `json:"user_id"`
	Role      string            `json:"role"`
	TokenType string            `json:"token_type"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT 字符串。
func GenerateToken(secret string, ttl time.Duration, claims Claims) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret missing")
	}
	now := time.Now()
	claims.RegisteredClaims.IssuedAt = jwt.NewNumericDate(now)
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 验证并解析 JWT。
func ParseToken(tokenStr string, secret string) (*Claims, error) {
	if tokenStr == "" {
		return nil, errors.New("token empty")
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token invalid")
	}
	return claims, nil
}
