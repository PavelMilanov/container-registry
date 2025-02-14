package system

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(key []byte) (string, error) {
	payload := jwt.MapClaims{
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString(key)
}

// Валидирует токен аутентификации.
func ValidateJWT(tokenString string, key []byte) bool {
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return false
	}
	_, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return false
	}
	return true
}
