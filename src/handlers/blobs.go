package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/labstack/echo/v5"
)

/*
checkBlob реализация.

	https://distribution.github.io/distribution/spec/api/#existing-layers
*/
func (h *Handler) checkBlob(c *echo.Context) error {
	digest := c.Param("uuid")
	// Проверяем, существует ли слой
	if err := h.BLOBS.CheckBlob(digest); err != nil {
		addRequestError(c, err)
		if errors.Is(err, storage.ErrBlobNotFound) ||
			errors.Is(err, storage.ErrInvalidDigest) {
			return c.JSON(http.StatusNotFound, map[string]any{
				"errors": []map[string]any{
					{
						"code":    "BLOB_UNKNOWN",
						"message": "blob not found",
					},
				},
			})
		}

		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, map[string]any{})
}

/*
startBlobUpload реализация.

	https://distribution.github.io/distribution/spec/api/#starting-an-upload
*/
func (h *Handler) startBlobUpload(c *echo.Context) error {

	uploadID := uuid.NewV4().String()

	if err := h.UPLOADS.StartBlobUpload(
		c.Request().Context(),
		uploadID,
	); err != nil {
		addRequestError(c, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	repository := c.Param("repository")
	imageName := c.Param("name")

	c.Response().Header().Set("Location", fmt.Sprintf(
		"/v2/%s/%s/blobs/uploads/%s",
		repository,
		imageName,
		uploadID,
	))
	c.Response().Header().Set("Docker-Upload-UUID", uploadID)
	c.Response().Header().Set("Range", "0-0")
	c.Response().Header().Set("X-Upload-Offset", "0")
	c.Response().Header().Set("X-Upload-State", "uploading")
	return c.NoContent(http.StatusAccepted)
}

/*
getBlobUpload возвращает подтверждённый offset после ошибок и перезапуска.

Range содержит последний записанный байт; X-Upload-Offset — точный размер,
включая нулевой, X-Upload-State — uploading, finalizing или ready.
*/
func (h *Handler) getBlobUpload(c *echo.Context) error {
	id := c.Param("uuid")
	status, err := h.UPLOADS.GetBlobUpload(c.Request().Context(), id)
	if err != nil {
		return respondBlobUploadError(c, err)
	}
	last := status.Offset - 1
	if last < 0 {
		last = 0
	}
	c.Response().Header().Set("Docker-Upload-UUID", id)
	c.Response().Header().Set("Location", c.Request().URL.Path)
	c.Response().Header().Set("Range", fmt.Sprintf("0-%d", last))
	c.Response().Header().Set("X-Upload-Offset", fmt.Sprint(status.Offset))
	c.Response().Header().Set("X-Upload-State", status.Phase)
	if status.Digest != "" {
		c.Response().Header().Set("X-Upload-Digest", status.Digest)
	}
	return c.NoContent(http.StatusNoContent)
}

/*
uploadBlobPart реализация.

	https://distribution.github.io/distribution/spec/api/#chunked-upload
*/
func (h *Handler) uploadBlobPart(c *echo.Context) error {
	uploadID := c.Param("uuid")

	uploadRange, err := parseContentRange(
		c.Request().Header.Get("Content-Range"),
	)
	if err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusRequestedRangeNotSatisfiable, map[string]any{
			"errors": []map[string]any{
				{
					"code":    "RANGE_INVALID",
					"message": "invalid Content-Range",
				},
			},
		})
	}

	newOffset, err := h.UPLOADS.AppendBlobUpload(
		c.Request().Context(),
		uploadID,
		uploadRange.start,
		newExactLengthReader(c.Request().Body, uploadRange.size()),
	)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidOffset) {
			last := newOffset - 1
			if last < 0 {
				last = 0
			}
			c.Response().Header().Set("Range", fmt.Sprintf("0-%d", last))
			c.Response().Header().Set("X-Upload-Offset", fmt.Sprint(newOffset))
		}
		return respondBlobUploadError(c, err)
	}

	lastByte := newOffset - 1
	if lastByte < 0 {
		lastByte = 0
	}

	c.Response().Header().Set("Docker-Upload-UUID", uploadID)
	c.Response().Header().Set("Range", fmt.Sprintf("0-%d", lastByte))
	c.Response().Header().Set("Location", c.Request().URL.Path)
	c.Response().Header().Set("X-Upload-Offset", fmt.Sprint(newOffset))
	return c.NoContent(http.StatusNoContent)
}

/*
finalizeBlobUpload реализация.

	https://distribution.github.io/distribution/spec/api/#completed-upload - при загрузке чанками.
	https://distribution.github.io/distribution/spec/api/#monolithic-upload - при монолитной загрузке.
*/
func (h *Handler) finalizeBlobUpload(c *echo.Context) error {
	uploadID := c.Param("uuid")
	expectedDigest := c.QueryParam("digest")

	if expectedDigest == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": []map[string]any{
				{
					"code":    "DIGEST_INVALID",
					"message": "digest not specified",
				},
			},
		})
	}

	blob, err := h.UPLOADS.CompleteBlobUpload(
		c.Request().Context(),
		uploadID,
		expectedDigest,
		c.Request().Body,
	)
	if err != nil {
		return respondBlobUploadError(c, err)
	}

	c.Response().Header().Set("Docker-Content-Digest", blob.Digest)
	c.Response().Header().Set("Location", fmt.Sprintf(
		"/v2/%s/%s/blobs/%s",
		c.Param("repository"),
		c.Param("name"),
		blob.Digest,
	))
	return c.NoContent(http.StatusCreated)
}

/*
getBlob реализация.

	https://distribution.github.io/distribution/spec/api/#pulling-a-layer
*/
func (h *Handler) getBlob(c *echo.Context) error {
	digest := c.Param("uuid")
	var stream io.ReadSeekCloser
	var info config.Blob
	var err error
	if reader, ok := h.BLOBS.(storage.BlobReader); ok {
		stream, info, err = reader.OpenBlob(c.Request().Context(), digest)
	} else {
		info, err = h.BLOBS.GetBlob(digest)
	}
	if err != nil {
		addRequestError(c, err)
		if errors.Is(err, storage.ErrBlobNotFound) ||
			errors.Is(err, storage.ErrInvalidDigest) {
			return c.JSON(http.StatusNotFound, map[string]any{
				"errors": []map[string]any{
					{
						"code":    "BLOB_UNKNOWN",
						"message": "blob not found",
					},
				},
			})
		}

		return c.NoContent(http.StatusInternalServerError)
	}
	// Возвращаем блоб клиенту
	c.Response().Header().Set("Content-Type", "application/octet-stream")
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Response().Header().Set("Docker-Content-Digest", info.Digest)
	if stream != nil {
		defer stream.Close()
		http.ServeContent(c.Response(), c.Request(), info.Digest, time.Time{}, stream)
		return nil
	}
	http.ServeFile(c.Response(), c.Request(), info.Path)
	return nil
}

/*
abortBlobUpload реализация.

	https://distribution.github.io/distribution/spec/api/#canceling-an-upload
*/
func (h *Handler) abortBlobUpload(c *echo.Context) error {
	uploadID := c.Param("uuid")

	if err := h.UPLOADS.AbortBlobUpload(
		c.Request().Context(),
		uploadID,
	); err != nil {
		return respondBlobUploadError(c, err)
	}

	c.Response().Header().Set("Docker-Upload-UUID", uploadID)
	return c.NoContent(http.StatusNoContent)
}
