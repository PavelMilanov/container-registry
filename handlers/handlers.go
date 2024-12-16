package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	// DB   *db.SQLite
	// CRON *cron.Cron
}

func NewHandler() *Handler {
	return &Handler{}
}

func logMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Printf("Received request: %s %s\n", c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}

func (h *Handler) InitRouters() *gin.Engine {
	router := gin.Default()
	// router.Use(logMiddleware())

	v2 := router.Group("/v2/")
	{
		// Пинг для проверки
		v2.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})

		// Получение манифеста
		v2.GET("/:name/manifests/:reference", h.getManifestHandler)
		// Загрузка манифеста образа
		v2.PUT("/:name/manifests/:reference", h.UploadManifestHandler)

		// Получение слоя образа
		v2.GET("/:name/blobs/:uuid", h.getBlobHandler)
		// Загрузка слоев образа
		v2.POST("/:name/blobs/uploads/", h.startBlobUpload)
		v2.PATCH("/:name/blobs/uploads/:uuid", h.uploadBlobPart)
		v2.PUT("/:name/blobs/uploads/:uuid", h.finalizeBlobUpload)

	}
	return router
}
