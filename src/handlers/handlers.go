package handlers

import (
	"fmt"
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
		if data == "" || !strings.HasPrefix(data, "Bearer ") {
			c.Header("WWW-Authenticate", `Bearer realm="http://192.168.64.1:5050/api/auth",service="Docker Registry"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})

		}
		payload := strings.TrimPrefix(data, "Bearer ")
		fmt.Println(payload)
		valid := secure.ValidateJWT(payload)
		if !valid {
			c.Header("WWW-Authenticate", `Bearer realm="http://192.168.64.1:5050/api/auth",service="Docker Registry"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		}

		// data := c.GetHeader("Authorization")
		// if data == "" || !strings.HasPrefix(data, "Basic ") {
		// 	c.Header("WWW-Authenticate", `Basic realm="Docker Registry"`)
		// 	c.JSON(http.StatusUnauthorized, gin.H{"err": "invalid credentials"})
		// 	return
		// }
		// payload := strings.TrimPrefix(data, "Basic ")
		// decoded, err := base64.StdEncoding.DecodeString(payload)
		// if err != nil {
		// 	logrus.Debug(err)
		// 	return
		// }
		// fmt.Println(string(decoded))
		// username := strings.Split(string(decoded), ":")[0]
		// password := strings.Split(string(decoded), ":")[1]
		// user := db.User{Name: username, Password: password}
		// if err := user.Login(sql); err != nil {
		// 	c.Header("WWW-Authenticate", `Basic realm="Docker Registry"`)
		// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		// 	return
		// }

		c.Next()
	}
}

func (h *Handler) InitRouters() *gin.Engine {

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.LoadHTMLGlob("templates/*")
	router.Static("/static/", "./static")

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
	}
	return router
}
