package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) authHandler(c *gin.Context) {
	// body, _ := io.ReadAll(c.Request.Body)
	// defer c.Request.Body.Close()
	data := c.GetHeader("Authorization")
	if data == "" || !strings.HasPrefix(data, "Basic ") {
		c.Header("WWW-Authenticate", `Basic realm="Docker Registry"`)
		c.JSON(http.StatusUnauthorized, gin.H{"err": "invalid credentials"})
		return
	}
	fmt.Println(data)
	c.JSON(http.StatusOK, gin.H{})
}
