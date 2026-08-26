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
	"github.com/labstack/echo/v5"
)

/*
uploadManifest реализация.

	https://distribution.github.io/distribution/spec/api/#pulling-an-image-manifest
*/
func (h *Handler) uploadManifest(c *echo.Context) error {
	repository := c.Param("repository")
	imageName := c.Param("name")      // название образа
	reference := c.Param("reference") // Тег или SHA-256 хэш манифеста
	body, err := io.ReadAll(c.Request().Body)
	mediaType := c.Request().Header.Get("Content-Type")
	if err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Failed to read request body"})
	}
	defer c.Request().Body.Close()
	// Вычисление хеша от содержимого файла
	hasher := sha256.New()
	hasher.Write(body)
	calculatedDigest := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
	// Проверяем, что клиент передал digest как reference, если это digest (а не тег)
	if strings.HasPrefix(reference, "sha256:") && reference != calculatedDigest {
		addRequestErrorMessage(c, "digest mismatch")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": []map[string]any{
				{
					"code":    "MANIFEST_UNVERIFIED",
					"message": "digest mismatch",
					"detail":  "The provided digest does not match the calculated digest.",
				},
			},
		})
	}
	meta := config.Meta{
		Repository: repository,
		Image:      imageName,
		Tag:        reference,
		MediaType:  mediaType,
		Digest:     calculatedDigest,
	}
	if err := services.SaveManifest(h.MANIFESTS, meta, body); err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusInternalServerError, map[string]any{})
	}
	c.Response().Header().Set("Docker-Content-Digest", calculatedDigest)
	return c.JSON(http.StatusCreated, map[string]any{})
}

/*
getManifest реализация.

https://distribution.github.io/distribution/spec/api/#existing-manifests
*/
func (h *Handler) getManifest(c *echo.Context) error {
	repository := c.Param("repository")
	imageName := c.Param("name")
	reference := c.Param("reference")
	data, err := h.MANIFESTS.GetManifest(repository, imageName, reference)
	if err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusNotFound, map[string]any{
			"errors": []map[string]any{
				{
					"code":    "MANIFEST_UNKNOWN",
					"message": "manifest unknown",
					"detail":  "The requested manifest was not found.",
				},
			},
		})
	}
	hasher := sha256.New()
	hasher.Write(data)
	calculatedDigest := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
	var manifest config.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusNotFound, map[string]any{
			"errors": []map[string]any{
				{
					"code":    "MANIFEST_UNKNOWN",
					"message": "manifest unknown",
					"detail":  "The requested manifest was not found.",
				},
			},
		})
	}
	c.Response().Header().Set("Docker-Content-Digest", calculatedDigest)
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	return c.Blob(http.StatusOK, manifest.MediaType, data)
}
