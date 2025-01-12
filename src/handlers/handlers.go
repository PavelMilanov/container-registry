package handlers

import (
	"net/http"
	"time"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Handler struct {
	DB      *db.SQLite
	STORAGE *storage.Storage
}

func NewHandler(storage *storage.Storage, db *db.SQLite) *Handler {
	return &Handler{STORAGE: storage, DB: db}
}

func baseRegistryMiddleware(sql *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		repository := c.Param("repository")
		var registry db.Registry
		if err := registry.Get(repository, sql); err != nil {
			logrus.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get registry"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handler) InitRouters() *gin.Engine {

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.LoadHTMLGlob("templates/*")
	router.Static("/static/", "./static")

	router.GET("/", h.webView)
	router.GET("/registration", h.registrationView)
	router.POST("/registration", h.registrationView)
	router.GET("/login", h.loginView)
	router.POST("/login", h.loginView)

	v2 := router.Group("/v2/")
	{
		// Пинг для проверки
		v2.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
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

	api := router.Group("/api/")
	{
		api.GET("/logout", h.logoutView)
		api.POST("/logout", h.logoutView)
		// web.GET("/", h.repositoryView)
		api.GET("/registry", h.getRegistry)
		api.GET("/registry/:name/:image", h.getImage)
		api.GET("/registry/:name", h.getRepository)
		api.POST("/registry/add/:name", h.addRegistry)
		api.GET("/settings", h.settingsView)
		api.POST("/settings", h.settingsView)
	}
	return router
}
