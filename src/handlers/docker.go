package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/system"
	"github.com/gin-gonic/gin"
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
	tokenString, err := system.GenerateJWT(username, []byte(h.ENV.Server.Jwt))
	if err != nil {
		logrus.Errorf("Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
