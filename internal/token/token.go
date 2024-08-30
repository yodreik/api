package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secret []byte
}

func New(secret string) *Manager {
	return &Manager{
		secret: []byte(secret),
	}
}

func (m *Manager) GenerateToken(id string) (token string, err error) {
	jsonwebtoken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
		"id":  id,
	})

	token, err = jsonwebtoken.SignedString([]byte(m.secret))
	if err != nil {
		return "", nil
	}
	return token, err
}

func (m *Manager) ParseToID(token string) (id string, err error) {
	jsonwebtoken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(m.secret), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := jsonwebtoken.Claims.(jwt.MapClaims)
	if !ok || !jsonwebtoken.Valid {
		return "", fmt.Errorf("token.ParseToID: can't parse invalid jsonwebtoken")
	}
	userID, exists := claims["id"].(string)
	if !exists {
		return "", fmt.Errorf("token.ParseToID: no `id` field found in token's claims")
	}
	return userID, nil

}
