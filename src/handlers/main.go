// Package handlers реализует REST API приложения.
// Интеграция спецификации https://distribution.github.io/distribution/spec/api/
// с кастомным API.
package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/gin-gonic/gin"
)

// Handler основная сущность взаимодействия с API.
type Handler struct {
	DB      *db.SQLite
	STORAGE storage.Storage
	ENV     *config.Env
}

func NewHandler(storage storage.Storage, db *db.SQLite, env *config.Env) *Handler {
	return &Handler{STORAGE: storage, DB: db, ENV: env}
}

func (h *Handler) InitRouters() *gin.Engine {

	router := gin.Default()
	setupCORS(router, h)

	router.Static("/assets/", "./assets")

	router.POST("/login", h.login)
	router.GET("/check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.POST("/registration", h.registration)
	router.GET("/v2/auth", h.authHandler)
	router.POST("/v2/auth", h.authHandler)

	v2 := router.Group("/v2/", loginRegistryMiddleware(h.ENV), baseRegistryMiddleware())
	{
		// Пинг для проверки
		v2.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Docker Registry API"})
		})
		// manifests
		v2.HEAD("/:repository/:name/manifests/:reference", h.getManifest)
		v2.GET("/:repository/:name/manifests/:reference", h.getManifest)
		v2.PUT("/:repository/:name/manifests/:reference", h.uploadManifest)
		// blobs
		v2.GET("/:repository/:name/blobs/:uuid", h.getBlob)
		v2.HEAD("/:repository/:name/blobs/:uuid", h.checkBlob)
		v2.POST("/:repository/:name/blobs/uploads/", h.startBlobUpload)
		v2.PATCH("/:repository/:name/blobs/uploads/:uuid", h.uploadBlobPart)
		v2.PUT("/:repository/:name/blobs/uploads/:uuid", h.finalizeBlobUpload)
	}

	api := router.Group("/api/", baseApiMiddleware([]byte(h.ENV.Server.Jwt)))
	{
		api.GET("/", h.getRegistry)
		api.GET("/:name", h.getRegistry)
		api.POST("/:name", h.addRegistry)
		api.DELETE("/:name", h.deleteRegistry)
		api.GET("/:name/:image", h.getImages)
		api.DELETE("/:name/:image", h.deleteImage)
		api.POST("/settings", h.settings)
		api.GET("/settings", h.settings)
	}
	noRouter(router, h)
	return router
}
