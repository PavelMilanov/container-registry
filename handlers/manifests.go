package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/gin-gonic/gin"
)

func (h *Handler) uploadManifest(c *gin.Context) {
	imageName := c.Param("name")      // название образа
	reference := c.Param("reference") // Тег или SHA-256 хэш манифеста
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	manifestPath := filepath.Join("data", "manifests", imageName, reference+".json")

	// Создаём директорию, если её нет
	err = os.MkdirAll(filepath.Dir(manifestPath), 0755)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	// Сохраняем манифест
	err = os.WriteFile(manifestPath, body, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save manifest"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Manifest uploaded", "image": imageName, "reference": reference})
}

// func (h *Handler) getManifestHandler(c *gin.Context) {
// 	imageName := c.Param("name")
// 	reference := c.Param("reference")

// 	// Путь к манифесту
// 	manifestPath := filepath.Join("data", "manifests", imageName, reference+".json")
// 	fmt.Println(manifestPath)
// 	// Проверяем, существует ли манифест
// 	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Manifest not found"})
// 		return
// 	}

// 	// Читаем содержимое манифеста
// 	manifest, err := os.ReadFile(manifestPath)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read manifest"})
// 		return
// 	}

// 	// Возвращаем манифест
// 	c.Data(http.StatusOK, "application/json", manifest)
// }

func (h *Handler) getManifest(c *gin.Context) {
	imageName := c.Param("name") // Имя репозитория
	reference := c.Param("reference")

	// Путь к манифесту
	manifestPath := filepath.Join(config.STORAGEPATH, "manifests", imageName, reference+".json")
	fmt.Println(manifestPath)
	// Проверяем, существует ли файл манифеста
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Manifest not found"})
		return
	}

	// Для HEAD-запроса просто возвращаем статус 200
	if c.Request.Method == "HEAD" {
		c.Status(http.StatusOK)
		return
	}

	// Открываем и возвращаем содержимое манифеста
	file, err := os.Open(manifestPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open manifest file"})
		return
	}
	defer file.Close()

	// Устанавливаем заголовки
	c.Header("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")

	// Передаём содержимое файла в ответ
	_, err = io.Copy(c.Writer, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send manifest data"})
		return
	}
}
