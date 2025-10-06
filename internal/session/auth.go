package session

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
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
		return "secretKey", nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid jwt claims")
}
