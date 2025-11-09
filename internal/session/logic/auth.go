package logic

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/wsx864321/kim/internal/session/pkg/config"
	"time"
)

type Claims struct {
	UserID     string
	ExpireTime int64
	jwt.RegisteredClaims
}

// ParseJWT 解析 JWT token
func ParseJWT(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(config.GetJWTSecretKey()), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid jwt claims")
}

// GenerateJWT 生成 JWT token
func GenerateJWT(userID string, expireTime int64) (string, error) {
	claims := Claims{
		UserID:     userID,
		ExpireTime: expireTime,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(jwt.TimeFunc().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(jwt.TimeFunc()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.GetJWTSecretKey()))
}
