package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/storage"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	// DB   *db.SQLite
	// CRON *cron.Cron
	STORAGE *storage.Storage
}

func NewHandler(storage *storage.Storage) *Handler {
	return &Handler{STORAGE: storage}
}

func (h *Handler) InitRouters() *gin.Engine {

	router := gin.Default()

	v2 := router.Group("/v2/")
	{
		// Пинг для проверки
		v2.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})
		// docker pull
		// получение manifest
		v2.HEAD("/:name/manifests/:reference", h.getManifest)
		v2.GET("/:name/manifests/:reference", h.getManifest)
		// скачивание blobs
		v2.GET("/:name/blobs/:digest", h.getBlob)

		// docker push
		// загрузка blobs
		v2.HEAD("/:name/blobs/:uuid", h.checkBlob)
		v2.POST("/:name/blobs/uploads/", h.startBlobUpload)
		v2.PATCH("/:name/blobs/uploads/:uuid", h.uploadBlobPart)
		v2.PUT("/:name/blobs/uploads/:uuid", h.finalizeBlobUpload)
		// получение манифеста
		v2.PUT("/:name/manifests/:reference", h.uploadManifest)

	}
	return router
}
