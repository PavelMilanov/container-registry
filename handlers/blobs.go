package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Реализация загрузки слоя
func (h *Handler) uploadBlobHandler(c *gin.Context) {
	imageName := c.Param("name") // Название образа
	uuid := c.Param("uuid")      // Уникальный идентификатор загрузки

	// Читаем тело запроса (слой образа в бинарном формате)
	file, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read blob"})
		return
	}

	// Определяем путь для сохранения слоя
	blobPath := filepath.Join("data", "blobs", imageName, uuid)

	// Создаём директорию, если её нет
	err = os.MkdirAll(filepath.Dir(blobPath), 0755)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	// Сохраняем слой на диск
	err = os.WriteFile(blobPath, file, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save blob"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Blob uploaded", "image": imageName, "blob": uuid})
}

func (h *Handler) getBlobHandler(c *gin.Context) {
	imageName := c.Param("name")
	uuid := c.Param("uuid")

	// Путь к слою
	blobPath := filepath.Join("data", "blobs", imageName, uuid)

	// Проверяем, существует ли слой
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blob not found"})
		return
	}

	// Читаем слой
	blob, err := os.ReadFile(blobPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read blob"})
		return
	}

	// Возвращаем слой
	c.Data(http.StatusOK, "application/octet-stream", blob)
}
