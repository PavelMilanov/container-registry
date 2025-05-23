package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/system"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) authHandler(c *gin.Context) {
	username, password, ok := c.Request.BasicAuth()
	c.Header("Content-Type", "application/json")
	if !ok {
		c.Header("WWW-Authenticate", `Basic realm="registry"`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}
	user := db.User{Name: username, Password: password}
	if err := user.Login(h.DB.Sql, c.Query("service"), h.ENV); err != nil {
		c.Header("WWW-Authenticate", `Basic realm="registry"`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	// Генерируем JWT-токен (срок действия 24 часа)
	tokenString, err := system.GenerateJWT(username, c.Query("service"), h.ENV)
	if err != nil {
		logrus.Errorf("Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	scope := c.Query("scope")
	var access []map[string]interface{}
	if scope != "" {
		parts := strings.Split(scope, ":")
		if len(parts) == 3 {
			access = []map[string]interface{}{
				{
					"type":    parts[0],                     // repository
					"name":    parts[1],                     // dev/registry
					"actions": strings.Split(parts[2], ","), // push,pull
				},
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"token":      tokenString,
		"access":     access,
		"expires_in": 86400,
		"issued_at":  time.Now().UTC().Format(time.RFC3339),
	})
}
