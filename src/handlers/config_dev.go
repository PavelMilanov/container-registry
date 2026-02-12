//go:build dev

package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func setupCORS(router *gin.Engine, h *Handler) {
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{h.ENV.Server.Realm, "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))
}

func noRouter(router *gin.Engine, h *Handler) {
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/v2/") {
			c.Status(http.StatusNotFound)
			return
		}
		c.String(http.StatusOK, "📦 Dev Mode: index.html is not found, but everything is OK.")
	})
}
