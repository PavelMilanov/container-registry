package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) uploadManifest(c *gin.Context) {
	repository := c.Param("repository")
	imageName := c.Param("name")      // название образа
	reference := c.Param("reference") // Тег или SHA-256 хэш манифеста
	body, err := io.ReadAll(c.Request.Body)
	mediaType := c.GetHeader("Content-Type")
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Manifest digest mismatch"})
		return
	}
	link, err := h.STORAGE.SaveManifest(body, repository, imageName, reference, calculatedDigest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	c.Header("Docker-Content-Digest", calculatedDigest)
	c.JSON(http.StatusCreated, gin.H{})
	go services.SaveManifestToDB(mediaType, link, reference, h.DB.Sql)
}

func (h *Handler) getManifest(c *gin.Context) {
	repository := c.Param("repository")
	imageName := c.Param("name")
	reference := c.Param("reference")
	data, err := h.STORAGE.GetManifest(repository, imageName, reference)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err})
		return
	}
	hasher := sha256.New()
	hasher.Write(data)
	calculatedDigest := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
	var manifest config.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err})
	}
	c.Header("Docker-Content-Digest", calculatedDigest)
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Data(http.StatusOK, manifest.MediaType, data)
}
