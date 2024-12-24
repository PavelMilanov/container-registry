package handlers

import (
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

func (h *Handler) InitRouters() *gin.Engine {
	router := gin.Default()

	v2 := router.Group("/v2/")
	{
		// Пинг для проверки
		v2.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})
		// manifests
		// получение
		v2.GET("/:name/manifests/:reference", h.getManifestHandler)
		// Загрузка
		v2.PUT("/:name/manifests/:reference", h.UploadManifestHandler)

		// загрузка blobs
		v2.HEAD("/:name/blobs/:uuid", h.getBlobHandler)
		v2.POST("/:name/blobs/uploads/", h.startBlobUpload)
		v2.PATCH("/:name/blobs/uploads/:uuid", h.uploadBlobPart)
		v2.PUT("/:name/blobs/uploads/:uuid", h.finalizeBlobUpload)
	}
	return router
}
