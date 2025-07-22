//go:build prod

package handlers

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func setupCORS(router *gin.Engine, h *Handler) {
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{h.ENV.Server.Realm},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))
}

func noRouter(router *gin.Engine, h *Handler) {
	router.LoadHTMLGlob("./index.html")
	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{"URL": h.ENV.Server.Realm, "Title": h.ENV.Server.Service})
	})
}
