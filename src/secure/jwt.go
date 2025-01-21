package secure

import (
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

func GenerateJWT(username string) (string, error) {
	payload := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(1 * time.Hour).Unix(),
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
	// fmt.Println(claims["username"])
	now := time.Now()                                   // 2025-01-21 15:19:21 +0300 MSK
	exp := time.Unix(int64(claims["exp"].(float64)), 0) // 2025-01-21 15:19:21 +0300 MSK
	difference := exp.Sub(now)                          // вычисляем срок действия токена
	if difference < 0 {
		return false
	}
	return true
}
