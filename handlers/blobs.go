package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// // Реализация загрузки слоя
// func (h *Handler) uploadBlobHandler(c *gin.Context) {
// 	imageName := c.Param("name") // Название образа
// 	uuid := uuid.New().String()  // Уникальный идентификатор загрузки

// 	// Создаём временный путь для блоба
// 	tempPath := filepath.Join("data", "blobs", imageName, uuid)
// 	err := os.MkdirAll(filepath.Dir(tempPath), 0755)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
// 		return
// 	}

// 	// Возвращаем URL для продолжения загрузки
// 	c.Header("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", imageName, uuid))
// 	c.JSON(http.StatusAccepted, gin.H{
// 		"uuid":     uuid,
// 		"location": fmt.Sprintf("/v2/%s/blobs/uploads/%s", imageName, uuid),
// 	})
// }

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

func (h *Handler) startBlobUpload(c *gin.Context) {
	imageName := c.Param("name") // Название образа

	// Генерируем уникальный UUID для загрузки
	uuid := uuid.New().String()

	// Создаём временный путь для блоба
	tempPath := filepath.Join("data", "blobs", imageName, uuid)
	err := os.MkdirAll(filepath.Dir(tempPath), 0755)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Возвращаем URL для продолжения загрузки
	c.Header("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", imageName, uuid))
	c.JSON(http.StatusAccepted, gin.H{
		"uuid":     uuid,
		"location": fmt.Sprintf("/v2/%s/blobs/uploads/%s", imageName, uuid),
	})
}

func (h *Handler) uploadBlobPart(c *gin.Context) {
	imageName := c.Param("name") // Название образа
	uuid := c.Param("uuid")      // Уникальный идентификатор загрузки

	// Читаем тело запроса (часть блоба в бинарном формате)
	file, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read blob part"})
		return
	}

	// Путь к временному файлу
	tempPath := filepath.Join("data", "blobs", imageName, uuid)
	f, err := os.OpenFile(tempPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open temporary file"})
		return
	}
	defer f.Close()

	// Записываем данные во временный файл
	_, err = f.Write(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write to temporary file"})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{"message": "Blob part uploaded"})
}

func (h *Handler) finalizeBlobUpload(c *gin.Context) {
	imageName := c.Param("name")
	uuid := c.Param("uuid")

	// Получаем digest из query параметров
	digest := c.Query("digest")
	if digest == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing digest query parameter"})
		return
	}

	// Путь к временному и конечному файлам
	tempPath := filepath.Join("data", "blobs", imageName, uuid)
	finalPath := filepath.Join("data", "blobs", imageName, strings.Replace(digest, "sha256:", "", 1))

	// Перемещаем файл
	err := os.Rename(tempPath, finalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize blob upload"})
		return
	}

	c.Header("Location", fmt.Sprintf("/v2/%s/blobs/%s", imageName, digest))
	c.Header("Docker-Content-Digest", digest)
	c.JSON(http.StatusCreated, gin.H{"message": "Blob finalized", "digest": digest})

}
