package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"uuid"

	"github.com/PavelMilanov/container-registry/storage"
	"github.com/gin-gonic/gin"
)

/*
checkBlob реализация.

	https://distribution.github.io/distribution/spec/api/#existing-layers
*/
func (h *Handler) checkBlob(c *gin.Context) {
	digest := c.Param("uuid")
	// Проверяем, существует ли слой
	if err := h.STORAGE.CheckBlob(digest); err != nil {
		addRequestError(c, err)
		if errors.Is(err, storage.ErrBlobNotFound) ||
			errors.Is(err, storage.ErrInvalidDigest) {
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

		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

/*
startBlobUpload реализация.

	https://distribution.github.io/distribution/spec/api/#starting-an-upload
*/
func (h *Handler) startBlobUpload(c *gin.Context) {

	uploadID := uuid.NewV4().String()

	if err := h.UPLOADS.StartBlobUpload(
		c.Request.Context(),
		uploadID,
	); err != nil {
		_ = c.Error(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	repository := c.Param("repository")
	imageName := c.Param("name")

	c.Header("Location", fmt.Sprintf(
		"/v2/%s/%s/blobs/uploads/%s",
		repository,
		imageName,
		uploadID,
	))
	c.Header("Docker-Upload-UUID", uploadID)
	c.Header("Range", "0-0")
	c.Status(http.StatusAccepted)
}

/*
uploadBlobPart реализация.

	https://distribution.github.io/distribution/spec/api/#chunked-upload
*/
func (h *Handler) uploadBlobPart(c *gin.Context) {
	uploadID := c.Param("uuid")

	uploadRange, err := parseContentRange(
		c.GetHeader("Content-Range"),
	)
	if err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusRequestedRangeNotSatisfiable, gin.H{
			"errors": []gin.H{
				{
					"code":    "RANGE_INVALID",
					"message": "invalid Content-Range",
				},
			},
		})
		return
	}

	newOffset, err := h.UPLOADS.AppendBlobUpload(
		c.Request.Context(),
		uploadID,
		uploadRange.start,
		newExactLengthReader(c.Request.Body, uploadRange.size()),
	)
	if err != nil {
		respondBlobUploadError(c, err)
		return
	}

	lastByte := newOffset - 1
	if lastByte < 0 {
		lastByte = 0
	}

	c.Header("Docker-Upload-UUID", uploadID)
	c.Header("Range", fmt.Sprintf("0-%d", lastByte))
	c.Status(http.StatusNoContent)
}

/*
finalizeBlobUpload реализация.

	https://distribution.github.io/distribution/spec/api/#completed-upload - при загрузке чанками.
	https://distribution.github.io/distribution/spec/api/#monolithic-upload - при монолитной загрузке.
*/
func (h *Handler) finalizeBlobUpload(c *gin.Context) {
	uploadID := c.Param("uuid")
	expectedDigest := c.Query("digest")

	if expectedDigest == "" {
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

	blob, err := h.UPLOADS.CompleteBlobUpload(
		c.Request.Context(),
		uploadID,
		expectedDigest,
		c.Request.Body,
	)
	if err != nil {
		respondBlobUploadError(c, err)
		return
	}

	c.Header("Docker-Content-Digest", blob.Digest)
	c.Header("Location", fmt.Sprintf(
		"/v2/%s/%s/blobs/%s",
		c.Param("repository"),
		c.Param("name"),
		blob.Digest,
	))
	c.Status(http.StatusCreated)
}

/*
getBlob реализация.

	https://distribution.github.io/distribution/spec/api/#pulling-a-layer
*/
func (h *Handler) getBlob(c *gin.Context) {
	digest := c.Param("uuid")
	// Определяем путь к блобу
	info, err := h.STORAGE.GetBlob(digest)
	if err != nil {
		addRequestError(c, err)
		if errors.Is(err, storage.ErrBlobNotFound) ||
			errors.Is(err, storage.ErrInvalidDigest) {
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

		c.Status(http.StatusInternalServerError)
		return
	}
	// Возвращаем блоб клиенту
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Header("Docker-Content-Digest", info.Digest)
	c.File(info.Path)
}

/*
abortBlobUpload реализация.

	https://distribution.github.io/distribution/spec/api/#canceling-an-upload
*/
func (h *Handler) abortBlobUpload(c *gin.Context) {
	uploadID := c.Param("uuid")

	if err := h.UPLOADS.AbortBlobUpload(
		c.Request.Context(),
		uploadID,
	); err != nil {
		respondBlobUploadError(c, err)
		return
	}

	c.Header("Docker-Upload-UUID", uploadID)
	c.Status(http.StatusNoContent)
}
