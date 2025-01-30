package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/secure"
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

func loginRegistryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.GetHeader("Authorization")
		// if data == "" || !strings.HasPrefix(data, "Bearer ") {
		// 	c.Header("WWW-Authenticate", `Bearer realm="http://192.168.64.1:5050/api/auth",service="Docker Registry"`)
		// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})

		// }
		payload := strings.TrimPrefix(data, "Bearer ")
		valid := secure.ValidateJWT(payload)
		if !valid {
			c.Header("WWW-Authenticate", `Bearer realm="http://0.0.0.0:5050/api/auth",service="Docker Registry"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		}
		c.Next()
	}
}

func (h *Handler) InitRouters() *gin.Engine {

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.LoadHTMLGlob("./index.html")
	router.Static("/assets/", "./assets")

	router.GET("/", h.webView)

	v2 := router.Group("/v2/", loginRegistryMiddleware())
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

	api := router.Group("/api/")
	{
		api.POST("/login", h.login)
		api.GET("/auth", h.authHandler)
		api.POST("/registration", h.registration)
		api.GET("/registry", h.getRegistry)
		api.GET("/registry/:name/:image", h.getImage)
		api.DELETE("/registry/:name/:image", h.deleteRepository)
		api.GET("/registry/:name", h.getRepository)
		api.POST("/registry/:name", h.addRegistry)
		api.DELETE("/registry/:name", h.deleteRegistry)
		api.GET("/settings", h.settingsView)
		api.POST("/settings", h.settingsView)
		api.GET("/check", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
	return router
}
