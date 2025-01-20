package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) webView(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{"WEB_API_URL": "http://localhost"})
}
