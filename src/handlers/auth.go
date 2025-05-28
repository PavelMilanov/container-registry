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

/*
authHandler аутентификация на уровне docker client и api.

	/v2/auth
*/
func (h *Handler) authHandler(c *gin.Context) {
	username, password, _ := c.Request.BasicAuth()
	c.Header("Content-Type", "application/json")
	user := db.User{Name: username, Password: password}
	if err := user.Login(h.DB.Sql, h.ENV); err != nil {
		c.Header("WWW-Authenticate", `Basic realm="registry"`)
		c.JSON(http.StatusUnauthorized, gin.H{
			"errors": []gin.H{
				{
					"code":    "UNAUTHORIZED",
					"message": "invalid username or password",
				},
			},
		})
		return
	}
	// Генерируем JWT-токен (срок действия 24 часа)
	tokenString, err := system.GenerateJWT(username, h.ENV)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
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
		"access_token": tokenString,
		"scope":        access,
		"expires_in":   86400,
		"issued_at":    time.Now().UTC().Format(time.RFC3339),
	})
}
