package handlers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/gin-gonic/gin"
)

/*
checkBlob реализация.

	https://distribution.github.io/distribution/spec/api/#existing-layers
*/
func (h *Handler) checkBlob(c *gin.Context) {
	uuid := c.Param("uuid")
	// Проверяем, существует ли слой
	if err := h.STORAGE.CheckBlob(uuid); err != nil {
		addRequestError(c, err)
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
	uuid := uuid.NewV4().String()
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
	tempPath := filepath.Join(config.TMP_PATH, uuid)
	f, err := os.OpenFile(tempPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}
	defer f.Close()

	if _, err := io.Copy(f, c.Request.Body); err != nil {
		addRequestError(c, err)
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

	info, err := f.Stat()
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}
	lastByte := info.Size() - 1
	if lastByte < 0 {
		lastByte = 0
	}
	c.Header("Docker-Upload-UUID", uuid)
	c.Header("Range", fmt.Sprintf("%d-%d", 0, lastByte))
	c.Status(http.StatusNoContent)
}

/*
finalizeBlobUpload реализация.

	https://distribution.github.io/distribution/spec/api/#completed-upload - при загрузке чанками.
	https://distribution.github.io/distribution/spec/api/#monolithic-upload - при монолитной загрузке.
*/
func (h *Handler) finalizeBlobUpload(c *gin.Context) {
	uuid := c.Param("uuid")
	digest := c.Query("digest")
	if digest == "" {
		addRequestErrorMessage(c, "digest not specified")
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

	f, err := os.OpenFile(tempPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}
	if _, err := io.Copy(f, c.Request.Body); err != nil {
		_ = f.Close()
		addRequestError(c, err)
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
	if err := f.Close(); err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}

	file, err := os.Open(tempPath)
	if err != nil {
		if os.IsNotExist(err) {
			addRequestError(c, err)
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
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}

	calculatedDigest := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
	if calculatedDigest != digest {
		addRequestErrorMessage(c, "digest mismatch")
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
		_ = c.Error(err)
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
			addRequestError(c, err)
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
			_ = c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
	}
	// Возвращаем блоб клиенту
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Header("Docker-Content-Digest", info.Digest)
	c.File(info.Path)
}
