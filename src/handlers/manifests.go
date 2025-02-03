package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) uploadManifest(c *gin.Context) {
	repository := c.Param("repository")
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
	manifestPath := filepath.Join(h.STORAGE.ManifestPath, repository, imageName, calculatedDigest)
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
		tagPath := filepath.Join(h.STORAGE.ManifestPath, repository, imageName, "tags", reference)
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
	go func() {
		// Считаем размер
		type layers struct {
			Size int `json:"size"`
		}
		type manifest struct {
			Layers []layers `json:"layers"`
		}

		data := manifest{}
		json.Unmarshal(body, &data)
		var size int
		for _, layer := range data.Layers {
			size += layer.Size
		}
		registy := db.Registry{}
		registy.Get(repository, h.DB.Sql)

		repo := db.Repository{
			Name:       imageName,
			RegistryID: registy.ID,
		}
		repo.Add(h.DB.Sql)
		image := db.Image{
			Name:         imageName,
			Hash:         calculatedDigest,
			Tag:          reference,
			Size:         size,
			RepositoryID: repo.ID,
		}
		image.Add(h.DB.Sql)
	}()
}

func (h *Handler) getManifest(c *gin.Context) {
	repository := c.Param("repository")
	imageName := c.Param("name")
	reference := c.Param("reference")

	// Определяем путь к файлу манифеста
	manifestPath := ""
	if strings.HasPrefix(reference, "sha256:") {
		// Если reference — это digest
		manifestPath = filepath.Join(h.STORAGE.ManifestPath, repository, imageName, reference)
	} else {
		// Если reference — это тег
		tagPath := filepath.Join(h.STORAGE.ManifestPath, repository, imageName, "tags", reference)
		tagData, err := os.ReadFile(tagPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
			return
		}
		manifestDigest := string(tagData)
		manifestPath = filepath.Join(h.STORAGE.ManifestPath, repository, imageName, manifestDigest)
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
	c.Header("Content-Length", fmt.Sprintf("%d", len(manifest)))
	c.Data(http.StatusOK, "application/vnd.docker.distribution.manifest.v2+json", manifest)
}
