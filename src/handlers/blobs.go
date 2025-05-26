package handlers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/gin-gonic/gin"
	uid "github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// checkBlob реализация
func (h *Handler) checkBlob(c *gin.Context) {
	uuid := c.Param("uuid")
	// Проверяем, существует ли слой
	if err := h.STORAGE.CheckBlob(uuid); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *Handler) startBlobUpload(c *gin.Context) {
	repository := c.Param("repository")
	imageName := c.Param("name")
	uuid := uid.New().String()
	c.Header("Location", fmt.Sprintf("/v2/%s/%s/blobs/uploads/%s", repository, imageName, uuid))
	c.Header("Docker-Upload-UUID", uuid)
	c.Header("Range", fmt.Sprintf("%d-%d", 0, 0))
	c.JSON(http.StatusAccepted, gin.H{})
}

func (h *Handler) uploadBlobPart(c *gin.Context) {
	uuid := c.Param("uuid")
	file, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read blob part"})
		return
	}
	// Путь к временному файлу
	tempPath := filepath.Join(config.TMP_PATH, uuid)
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
	c.Header("Docker-Upload-UUID", uuid)
	c.Header("Range", fmt.Sprintf("%d-%d", 0, len(file)-1))
	c.JSON(http.StatusNoContent, gin.H{"message": "Blob part uploaded"})
}

func (h *Handler) finalizeBlobUpload(c *gin.Context) {
	uuid := c.Param("uuid")
	status := c.Request.Header.Get("Content-Type")
	// если есть заголовок Content-Type, то это не первый запрос и образ грузится частями
	digest := c.Query("digest")
	if digest == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing digest query parameter"})
		return
	}
	tempPath := filepath.Join(config.TMP_PATH, uuid)
	hasher := sha256.New()
	if status == "" {
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
		if _, err := io.Copy(hasher, file); err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logrus.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read blob part"})
			return
		}
		f, err := os.OpenFile(tempPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot resume upload"})
			return
		}
		defer f.Close()
		if _, err = f.Write(body); err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if _, err := f.Seek(0, 0); err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hasher.Write(body)
	}
	calculatedDigest := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
	if calculatedDigest != digest {
		logrus.Error("Digest mismatch")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Digest mismatch"})
		return
	}

	// переименование временного файла в итоговый файл
	if err := h.STORAGE.SaveBlob(tempPath, digest); err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	c.Header("Docker-Content-Digest", digest)
	c.JSON(http.StatusCreated, gin.H{"message": "Blob finalized", "digest": digest})
}

func (h *Handler) getBlob(c *gin.Context) {
	uuid := c.Param("uuid")
	// Определяем путь к блобу
	info, err := h.STORAGE.GetBlob(uuid)
	if err != nil {
		if err.Error() == "Blob not found" {
			logrus.Error(err)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// Возвращаем блоб клиенту
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Header("Docker-Content-Digest", info.Digest)
	c.File(filepath.Join(config.BLOBS_PATH, info.Digest))
}
