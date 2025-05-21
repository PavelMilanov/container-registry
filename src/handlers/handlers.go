// Package handlers реализует REST API приложения.
// Интеграция спецификации https://distribution.github.io/distribution/spec/api/
// с кастомным API.
package handlers

import (
	"net/http"

	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Handler основная сущность взаимодействия с API.
type Handler struct {
	DB      *db.SQLite
	STORAGE *storage.Storage
	ENV     *config.Env
}

func NewHandler(storage *storage.Storage, db *db.SQLite, env *config.Env) *Handler {
	return &Handler{STORAGE: storage, DB: db, ENV: env}
}

func (h *Handler) InitRouters() *gin.Engine {

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{h.ENV.Server.Realm},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))
	router.LoadHTMLGlob("./index.html")
	router.Static("/assets/", "./assets")

	router.POST("/login", h.login)
	router.GET("/check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.POST("/registration", h.registration)
	router.GET("/v2/auth", h.authHandler)

	v2 := router.Group("/v2/", loginRegistryMiddleware(h.ENV))
	{
		// Пинг для проверки
		v2.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Docker Registry API"})
		})
		// docker pull
		// получение manifest
		v2.HEAD("/:repository/:name/manifests/:reference", h.getManifest)
		v2.GET("/:repository/:name/manifests/:reference", h.getManifest)
		// скачивание blobs
		v2.GET("/:repository/:name/blobs/:digest", h.getBlob)

		// docker push
		// загрузка blobs
		v2.HEAD("/:repository/:name/blobs/:uuid", h.checkBlob, baseRegistryMiddleware(h.DB.Sql))
		v2.POST("/:repository/:name/blobs/uploads/", h.startBlobUpload)
		v2.PATCH("/:repository/:name/blobs/uploads/:uuid", h.uploadBlobPart)
		v2.PUT("/:repository/:name/blobs/uploads/:uuid", h.finalizeBlobUpload)
		// получение манифеста
		v2.PUT("/:repository/:name/manifests/:reference", h.uploadManifest)

	}

	api := router.Group("/api/", baseApiMiddleware([]byte(h.ENV.Server.Jwt)))
	{
		api.GET("/registry", h.getRegistry)
		api.GET("/registry/:name", h.getRegistry)
		api.POST("/registry/:name", h.addRegistry)
		api.DELETE("/registry/:name", h.deleteRegistry)
		api.GET("/registry/:name/:image", h.getImage)
		api.DELETE("/registry/:name/:image", h.deleteImage)
		api.POST("/settings", h.settings)
		api.GET("/settings", h.settings)
	}

	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{"URL": h.ENV.Server.Realm})
	})
	return router
}
