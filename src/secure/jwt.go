package secure

import (
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(username string, password string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"password": password,
		"exp":      time.Now().Add(1 * time.Hour).Unix(),
	})
	return token.SignedString(config.JWT_SECRET)
}
