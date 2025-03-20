// Package handlers реализует REST API приложения.
// Интеграция спецификации https://distribution.github.io/distribution/spec/api/
// с кастомным API.
package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/PavelMilanov/container-registry/system"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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

// baseApiMiddleware мидлварь для авторизации на уровне REST-API.
func baseApiMiddleware(jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.GetHeader("Authorization")
		payload := strings.TrimPrefix(data, "Bearer ")
		if !system.ValidateJWT(payload, jwtKey) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token is not valid"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// baseRegistryMiddleware мидлварь для авторизации на уровне docker client.
func baseRegistryMiddleware(sql *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		repository := c.Param("repository")
		var registry db.Registry
		if err := registry.Get(sql, repository); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get registry"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// loginRegistryMiddleware мидлварь для авторизации на уровне docker client.
// см. https://distribution.github.io/distribution/spec/auth/token/
func loginRegistryMiddleware(url string, jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.GetHeader("Authorization")
		logrus.Debug(data)
		payload := strings.TrimPrefix(data, "Bearer ")
		valid := system.ValidateJWT(payload, jwtKey)
		if !valid {
			realm := fmt.Sprintf(`Bearer realm="%s/v2/auth"`, url)
			c.Header("WWW-Authenticate", realm)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handler) InitRouters() *gin.Engine {

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{h.ENV.Server.Url},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))
	router.LoadHTMLGlob("./index.html")
	router.Static("/assets/", "./assets")

	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{"URL": h.ENV.Server.Url})
	})

	router.POST("/login", h.login)
	router.GET("/check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.POST("/registration", h.registration)
	router.GET("/v2/auth", h.authHandler)

	v2 := router.Group("/v2/", loginRegistryMiddleware(h.ENV.Server.Url, []byte(h.ENV.Server.Jwt)))
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
	return router
}
