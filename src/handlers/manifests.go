package handlers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) uploadManifest(c *gin.Context) {
	imageName := c.Param("name")      // название образа
	reference := c.Param("reference") // Тег или SHA-256 хэш манифеста
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logrus.Error(err)
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
	manifestPath := filepath.Join(h.STORAGE.ManifestPath, imageName, calculatedDigest)
	err = os.MkdirAll(filepath.Dir(manifestPath), 0755)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create manifest directory"})
		return
	}

	err = os.WriteFile(manifestPath, body, 0644)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save manifest"})
		return
	}

	// Если это тег (а не digest), создаём символическую ссылку
	if !strings.HasPrefix(reference, "sha256:") {
		tagPath := filepath.Join(h.STORAGE.ManifestPath, imageName, "tags", reference)
		err = os.MkdirAll(filepath.Dir(tagPath), 0755)
		if err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag directory"})
			return
		}

		err = os.WriteFile(tagPath, []byte(calculatedDigest), 0644)
		if err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save tag reference"})
			return
		}
	}
	c.Header("Docker-Content-Digest", calculatedDigest)
	c.JSON(http.StatusCreated, gin.H{"message": "Manifest uploaded", "digest": calculatedDigest})
	logrus.Infof("Загружен образ %s:%s | %s", imageName, reference, calculatedDigest)
}

func (h *Handler) getManifest(c *gin.Context) {
	imageName := c.Param("name")
	reference := c.Param("reference")

	// Определяем путь к файлу манифеста
	manifestPath := ""
	if strings.HasPrefix(reference, "sha256:") {
		// Если reference — это digest
		manifestPath = filepath.Join(h.STORAGE.ManifestPath, imageName, reference)
	} else {
		// Если reference — это тег
		tagPath := filepath.Join(h.STORAGE.ManifestPath, imageName, "tags", reference)
		tagData, err := os.ReadFile(tagPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
			return
		}
		manifestDigest := string(tagData)
		manifestPath = filepath.Join(h.STORAGE.ManifestPath, imageName, manifestDigest)
	}
	// Читаем содержимое манифеста
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Manifest not found"})
		return
	}
	hasher := sha256.New()
	hasher.Write(manifest)
	calculatedDigest := fmt.Sprintf("sha256:%x", hasher.Sum(nil))

	// Возвращаем манифест клиенту
	c.Header("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	c.Header("Docker-Content-Digest", calculatedDigest)
	c.Data(http.StatusOK, "application/vnd.docker.distribution.manifest.v2+json", manifest)
	logrus.Infof("Скачан образ %s:%s | %s", imageName, reference, calculatedDigest)
}