package handlers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	defer c.Request.Body.Close()
	// Вычисление хеша от содержимого файла
	hasher := sha256.New()
	hasher.Write(body)
	calculatedDigest := fmt.Sprintf("sha256:%x", hasher.Sum(nil))

	// Проверяем, что клиент передал digest как reference, если это digest (а не тег)
	if strings.HasPrefix(reference, "sha256:") && reference != calculatedDigest {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Manifest digest mismatch",
			"calculatedDigest":  calculatedDigest,
			"providedReference": reference,
		})
		return
	}

	// Сохраняем манифест в хранилище
	manifestPath := filepath.Join(config.STORAGEPATH, config.MANIFESTSPATH, imageName, calculatedDigest)
	err = os.MkdirAll(filepath.Dir(manifestPath), 0755)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create manifest directory"})
		return
	}

	err = os.WriteFile(manifestPath, body, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save manifest"})
		return
	}

	// Если это тег (а не digest), создаём символическую ссылку
	if !strings.HasPrefix(reference, "sha256:") {
		tagPath := filepath.Join(config.STORAGEPATH, config.MANIFESTSPATH, imageName, "tags", reference)
		err = os.MkdirAll(filepath.Dir(tagPath), 0755)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag directory"})
			return
		}

		err = os.WriteFile(tagPath, []byte(calculatedDigest), 0644)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save tag reference"})
			return
		}
	}

	// Возвращаем успешный ответ
	c.Header("Docker-Content-Digest", calculatedDigest)
	c.JSON(http.StatusCreated, gin.H{"message": "Manifest uploaded", "digest": calculatedDigest})

}

func (h *Handler) getManifest(c *gin.Context) {
	imageName := c.Param("name") // Имя репозитория
	reference := c.Param("reference")

	// Путь к манифесту
	manifestPath := filepath.Join(config.STORAGEPATH, "manifests", imageName, reference+".json")

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
