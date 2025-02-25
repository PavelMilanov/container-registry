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
	uid "github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func (h *Handler) checkBlob(c *gin.Context) {
	// imageName := c.Param("name")
	uuid := c.Param("uuid")

	blobPath := filepath.Join(config.STORAGE_PATH, config.BLOBS_PATH, strings.Replace(uuid, "sha256:", "", 1))

	// Проверяем, существует ли слой
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blob not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *Handler) startBlobUpload(c *gin.Context) {
	repository := c.Param("repository")
	imageName := c.Param("name")

	// Генерируем уникальный UUID для загрузки
	uuid := uid.New().String()

	// Возвращаем URL для продолжения загрузки
	c.Header("Location", fmt.Sprintf("/v2/%s/%s/blobs/uploads/%s", repository, imageName, uuid))
	c.Header("Docker-Upload-UUID", uuid)
	c.JSON(http.StatusAccepted, gin.H{
		// "location": fmt.Sprintf("/v2/%s/%s/blobs/uploads/%s", repository, imageName, uuid),
	})
}

func (h *Handler) uploadBlobPart(c *gin.Context) {
	// imageName := c.Param("name") // Название образа
	uuid := c.Param("uuid") // Уникальный идентификатор загрузки

	// Читаем тело запроса (часть блоба в бинарном формате)
	file, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read blob part"})
		return
	}

	// Путь к временному файлу
	tempPath := filepath.Join(h.STORAGE.BlobPath, uuid)
	f, err := os.OpenFile(tempPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open temporary file"})
		return
	}
	defer f.Close()

	// Записываем данные во временный файл
	_, err = f.Write(file)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write to temporary file"})
		return
	}
	// c.Header("Content-Type", "application/octet-stream")
	c.Header("Docker-Upload-UUID", uuid)
	c.Header("Range", fmt.Sprintf("%d-%d", 0, len(file)-1))
	c.JSON(http.StatusNoContent, gin.H{"message": "Blob part uploaded"})
}

func (h *Handler) finalizeBlobUpload(c *gin.Context) {
	// imageName := c.Param("name")
	uuid := c.Param("uuid")

	// Получаем digest из query параметров
	digest := c.Query("digest")
	if digest == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing digest query parameter"})
		return
	}

	// Путь к временному и конечному файлам
	tempPath := filepath.Join(h.STORAGE.BlobPath, uuid)

	// Открытие временного файла
	file, err := os.Open(tempPath)
	if err != nil {
		logrus.Error(err)
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Temporary file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	defer file.Close()

	// Вычисление хеша от содержимого файла
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	calculatedDigest := fmt.Sprintf("sha256:%x", hasher.Sum(nil))

	// сравнение хешей
	if calculatedDigest != digest {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Digest mismatch", "digest": digest, "calculatedDigest": calculatedDigest})
		return
	}

	// переименование временного файла в итоговый файл
	finalPath := filepath.Join(h.STORAGE.BlobPath, strings.Replace(digest, "sha256:", "", 1))

	err = os.Rename(tempPath, finalPath)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize blob upload"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Blob finalized", "digest": digest})
}

func (h *Handler) getBlob(c *gin.Context) {
	// imageName := c.Param("name")
	digest := c.Param("digest")

	// Определяем путь к блобу
	blobPath := filepath.Join(h.STORAGE.BlobPath, strings.Replace(digest, "sha256:", "", 1))

	// Открываем файл блоба
	file, err := os.Open(blobPath)
	if err != nil {
		logrus.Error(err)
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Blob not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	// Получаем информацию о файле
	fileInfo, err := file.Stat()
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stat blob file"})
		return
	}

	// Возвращаем блоб клиенту
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	c.Header("Docker-Content-Digest", digest)
	c.File(blobPath)
}
