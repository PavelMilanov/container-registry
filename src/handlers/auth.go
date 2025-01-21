package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) authHandler(c *gin.Context) {
	// data := c.GetHeader("Authorization")
	// if data == "" || !strings.HasPrefix(data, "Basic ") {
	// 	c.Header("WWW-Authenticate", `Basic realm="Docker Registry"`)
	// 	c.JSON(http.StatusUnauthorized, gin.H{"err": "invalid credentials"})
	// 	return
	// }
	// // Decode Basic Auth
	// payload := strings.TrimPrefix(data, "Basic ")
	// decoded, err := decodeBase64(payload)
	// token := strings.Split(decoded, ":")[1]
	// if err != nil || !secure.ValidateJWT(token) {
	// 	c.Header("WWW-Authenticate", `Basic realm="Docker Registry"`)
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
	// 	return
	// }

	// fmt.Println(strings.Split(decoded, ":"))
	// Generate JWT Token
	// credentials := strings.Split(decoded, ":") //["username", "password"]
	// token, err := secure.GenerateJWT(credentials[0], credentials[1])
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
	// 	return
	// }
	c.JSON(http.StatusOK, gin.H{"token": "token"})
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
