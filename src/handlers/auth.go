package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) authHandler(c *gin.Context) {
	data := c.GetHeader("Authorization")
	if data == "" || !strings.HasPrefix(data, "Basic ") {
		c.Header("WWW-Authenticate", `Basic realm="Docker Registry"`)
		c.JSON(http.StatusUnauthorized, gin.H{"err": "invalid credentials"})
		return
	}
	payload := strings.TrimPrefix(data, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		logrus.Debug(err)
		return
	}
	fmt.Println(string(decoded))
	username := strings.Split(string(decoded), ":")[0]
	password := strings.Split(string(decoded), ":")[1]
	user := db.User{Name: username, Password: password}
	if err := user.Login(h.DB.Sql); err != nil {
		c.Header("WWW-Authenticate", `Basic realm="Docker Registry"`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	fmt.Println(user.Token)
	c.JSON(http.StatusOK, gin.H{"token": user.Token})
}
