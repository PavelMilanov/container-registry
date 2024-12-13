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
			c.Status(http.StatusOK)
		})

		// Загрузка манифеста образа
		v2.PUT("/:name/manifests/:reference", func(c *gin.Context) {
			imageName := c.Param("name")
			reference := c.Param("reference")
			// Обработка загрузки манифеста
			c.JSON(http.StatusCreated, gin.H{"message": "Manifest uploaded", "image": imageName, "reference": reference})
		})

		// Получение манифеста
		v2.GET("/:name/manifests/:reference", func(c *gin.Context) {
			imageName := c.Param("name")
			reference := c.Param("reference")
			// Возврат загруженного манифеста
			c.JSON(http.StatusOK, gin.H{"message": "Manifest retrieved", "image": imageName, "reference": reference})
		})

	}
	return router
}
