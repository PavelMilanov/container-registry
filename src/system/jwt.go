package system

import (
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(username, aud string, cred *config.Env) (string, error) {
	payload := jwt.MapClaims{
		"sub": username,
		"aud": aud,
		"iss": cred.Server.Issuer,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString([]byte(cred.Server.Jwt))
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
