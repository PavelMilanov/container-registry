package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Пинг для проверки
	r.GET("/v2/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Загрузка манифеста образа
	r.PUT("/v2/:name/manifests/:reference", func(c *gin.Context) {
		imageName := c.Param("name")
		reference := c.Param("reference")
		// Обработка загрузки манифеста
		c.JSON(http.StatusCreated, gin.H{"message": "Manifest uploaded", "image": imageName, "reference": reference})
	})

	// Получение манифеста
	r.GET("/v2/:name/manifests/:reference", func(c *gin.Context) {
		imageName := c.Param("name")
		reference := c.Param("reference")
		// Возврат загруженного манифеста
		c.JSON(http.StatusOK, gin.H{"message": "Manifest retrieved", "image": imageName, "reference": reference})
	})

	// Запуск сервера
	r.Run(":5000") // Запуск на порту 5000
}
