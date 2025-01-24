package secure

import (
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

func GenerateJWT() (string, error) {
	payload := jwt.MapClaims{
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString(config.JWT_SECRET)
}

// Валидирует токен аутентификации.
func ValidateJWT(tokenString string) bool {
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return config.JWT_SECRET, nil
	})
	if err != nil {
		logrus.Error(err)
		return false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		logrus.Debug("Токен не валиден")
		return false
	}
	iat := time.Unix(int64(claims["iat"].(float64)), 0) // 2025-01-21 15:19:21 +0300 MSK
	exp := time.Unix(int64(claims["exp"].(float64)), 0) // 2025-01-21 15:19:21 +0300 MSK
	// difference := exp.Sub(iat)                          // вычисляем срок действия токена
	if exp.Sub(iat) < 0 {
		logrus.Debug("Срок жизни токена истек")
		return false
	}
	return true
}
