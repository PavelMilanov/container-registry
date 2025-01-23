package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/PavelMilanov/container-registry/secure"
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
	username := strings.Split(string(decoded), ":")[0]
	//password := strings.Split(string(decoded), ":")[1]
	token, err := secure.GenerateJWT(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// decodeBase64 decodes a Base64 string
func decodeBase64(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func validateCredentials(decoded string) bool {
	parts := strings.Split(decoded, ":")
	if len(parts) != 2 {
		return false
	}
	fmt.Println(parts)
	return false
	// password, ok := users[parts[0]]
	// return ok && password == parts[1]
}
