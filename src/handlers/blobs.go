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

/*
checkBlob реализация.

	https://distribution.github.io/distribution/spec/api/#existing-layers
*/
func (h *Handler) checkBlob(c *gin.Context) {
	uuid := c.Param("uuid")
	// Проверяем, существует ли слой
	if err := h.STORAGE.CheckBlob(uuid); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"errors": []gin.H{
				{
					"code":    "BLOB_UNKNOWN",
					"message": "blob not found",
				},
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

/*
startBlobUpload реализация.

	https://distribution.github.io/distribution/spec/api/#starting-an-upload
*/
func (h *Handler) startBlobUpload(c *gin.Context) {
	repository := c.Param("repository")
	imageName := c.Param("name")
	uuid := uid.New().String()
	c.Header("Location", fmt.Sprintf("/v2/%s/%s/blobs/uploads/%s", repository, imageName, uuid))
	c.Header("Docker-Upload-UUID", uuid)
	c.Header("Range", fmt.Sprintf("%d-%d", 0, 0))
	c.JSON(http.StatusAccepted, gin.H{})
}

/*
uploadBlobPart реализация.

	https://distribution.github.io/distribution/spec/api/#chunked-upload
*/
func (h *Handler) uploadBlobPart(c *gin.Context) {
	uuid := c.Param("uuid")
	file, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logrus.WithError(err).Error(uuid)
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []gin.H{
				{
					"code":    "BLOB_UPLOAD_INVALID",
					"message": "failed to read blob part",
				},
			},
		})
		return
	}
	// Путь к временному файлу
	tempPath := filepath.Join(config.TMP_PATH, uuid)
	f, err := os.OpenFile(tempPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}
	defer f.Close()
	// Записываем данные во временный файл
	_, err = f.Write(file)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}
	c.Header("Docker-Upload-UUID", uuid)
	c.Header("Range", fmt.Sprintf("%d-%d", 0, len(file)-1))
	c.JSON(http.StatusNoContent, gin.H{})
}

/*
finalizeBlobUpload реализация.

	https://distribution.github.io/distribution/spec/api/#completed-upload - при загрузке чанками.
	https://distribution.github.io/distribution/spec/api/#monolithic-upload - при монолитной загрузке.
*/
func (h *Handler) finalizeBlobUpload(c *gin.Context) {
	uuid := c.Param("uuid")
	status := c.Request.Header.Get("Content-Type")
	// если есть заголовок Content-Type, то это не первый запрос и образ грузится частями
	digest := c.Query("digest")
	if digest == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []gin.H{
				{
					"code":    "DIGEST_INVALID",
					"message": "digest not specified",
				},
			},
		})
		return
	}
	tempPath := filepath.Join(config.TMP_PATH, uuid)
	hasher := sha256.New()
	if status == "" {
		file, err := os.Open(tempPath)
		if err != nil {
			logrus.Error(err)
			if os.IsNotExist(err) {
				logrus.WithError(err).Error(uuid)
				c.JSON(http.StatusNotFound, gin.H{
					"errors": []gin.H{
						{
							"code":    "BLOB_UPLOAD_INVALID",
							"message": "failed to read blob part",
						},
					},
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{})
		}
		defer file.Close()
		if _, err := io.Copy(hasher, file); err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
	} else {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logrus.WithError(err).Error(uuid)
			c.JSON(http.StatusBadRequest, gin.H{
				"errors": []gin.H{
					{
						"code":    "BLOB_UPLOAD_INVALID",
						"message": "failed to read blob part",
					},
				},
			})
			return
		}
		f, err := os.OpenFile(tempPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
		defer f.Close()
		if _, err = f.Write(body); err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
		if _, err := f.Seek(0, 0); err != nil {
			logrus.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
		hasher.Write(body)
	}
	calculatedDigest := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
	if calculatedDigest != digest {
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []gin.H{
				{
					"code":    "MANIFEST_UNVERIFIED",
					"message": "digest mismatch",
					"detail":  "The provided digest does not match the calculated digest.",
				},
			},
		})
		return
	}

	// переименование временного файла в итоговый файл
	if err := h.STORAGE.SaveBlob(tempPath, digest); err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}
	c.Header("Docker-Content-Digest", digest)
	c.JSON(http.StatusCreated, gin.H{"message": "Blob finalized", "digest": digest})
}

/*
getBlob реализация.

	https://distribution.github.io/distribution/spec/api/#pulling-a-layer
*/
func (h *Handler) getBlob(c *gin.Context) {
	uuid := c.Param("uuid")
	// Определяем путь к блобу
	info, err := h.STORAGE.GetBlob(uuid)
	if err != nil {
		if err.Error() == "Blob not found" {
			logrus.WithError(err).Error(uuid)
			c.JSON(http.StatusNotFound, gin.H{
				"errors": []gin.H{
					{
						"code":    "BLOB_UNKNOWN",
						"message": "blob not found",
					},
				},
			})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
	}
	// Возвращаем блоб клиенту
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Header("Docker-Content-Digest", info.Digest)
	c.File(filepath.Join(config.BLOBS_PATH, info.Digest))
}
