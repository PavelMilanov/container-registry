// Package handlers реализует REST API приложения.
// Интеграция спецификации https://distribution.github.io/distribution/spec/api/
// с кастомным API.
package handlers

import (
	"net/http"
	"strings"

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
	router := gin.New()
	router.Use(requestLoggerMiddleware(), gin.Recovery())
	// router.Use(cors.New(cors.Config{
	// 	AllowOrigins:     []string{"http://localhost:5050"},
	// 	AllowMethods:     []string{"GET", "POST", "DELETE"},
	// 	AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
	// 	ExposeHeaders:    []string{"Content-Length"},
	// 	AllowCredentials: true,
	// 	MaxAge:           24 * time.Hour,
	// }))
	router.POST("/login", h.login)
	router.GET("/check", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.POST("/registration", h.registration)
	router.GET("/v2/auth", h.authHandler)
	router.POST("/v2/auth", h.authHandler)

	v2 := router.Group("/v2/", loginRegistryMiddleware(h.ENV), baseRegistryMiddleware())
	{
		// Пинг для проверки
		v2.GET("/", func(c *gin.Context) {
			c.Header("Docker-Distribution-Api-Version", "registry/2.0")
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
		cloud := api.Group("/cloud/")
		{
			cloud.GET("/:cloud/:repository", h.getImagesList)
			cloud.GET("/:cloud", h.getRepoList)
			cloud.GET("/list", h.getCloudList)
			cloud.POST("/create", h.addCloud)
			cloud.DELETE("/delete", h.deleteCloud)
			cloud.DELETE("/:cloud/:repository", h.deleteRepositoryOrImage)
		}
		garbage := api.Group("/garbage/")
		{
			garbage.POST("/collection", h.garbageCollection)
			garbage.POST("/tags", h.deleteOlderTags)

		}
		api.GET("/settings", h.settings)
		api.POST("/settings", h.settings)
	}
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/v2/") {
			addRequestErrorMessage(c, "route not found")
			c.Status(http.StatusNotFound)
			return
		}
		c.String(http.StatusOK, "is OK.")
	})
	return router
}
