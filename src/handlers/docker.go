package handlers

import (
	"net/http"
	"time"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

func (h *Handler) authHandler(c *gin.Context) {
	username, password, ok := c.Request.BasicAuth()
	if !ok {
		c.Header("WWW-Authenticate", `Basic realm="registry"`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}
	user := db.User{Name: username, Password: password}
	if err := user.Login(h.DB.Sql, []byte(h.ENV.Server.Jwt)); err != nil {
		c.Header("WWW-Authenticate", `Basic realm="registry"`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	// Генерируем JWT-токен (срок действия 24 часа)
	tokenString, err := generateJWT(username, h.ENV.Server.Jwt)
	if err != nil {
		logrus.Errorf("Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
	// c.JSON(http.StatusOK, gin.H{"token": user.Token})
}

// generateJWT генерирует JWT-токен с использованием алгоритма HS256.
func generateJWT(username, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub": username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
