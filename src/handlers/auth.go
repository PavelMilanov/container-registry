package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
	"github.com/PavelMilanov/container-registry/services"
	"github.com/labstack/echo/v5"
)

/*
authHandler аутентифицирует Docker client и выдаёт Registry JWT.

	/v2/auth
*/
func (h *Handler) authHandler(c *echo.Context) error {
	username, password, ok := c.Request().BasicAuth()
	if !ok {
		return writeInvalidRegistryCredentials(c)
	}

	access := requestedRegistryAccess(c.QueryParams()["scope"])
	token, err := h.AUTH.Login(
		c.Request().Context(),
		username,
		password,
		access,
	)
	if err != nil {
		addRequestError(c, err)
		if errors.Is(err, services.ErrInvalidCredentials) {
			return writeInvalidRegistryCredentials(c)
		}
		return writeRegistryAuthError(c)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"access_token": token.Value,
		"scope":        access,
		"expires_in": int64(
			token.ExpiresAt.Sub(token.IssuedAt).Seconds(),
		),
		"issued_at": token.IssuedAt.Format(time.RFC3339),
	})
}

/*
requestedRegistryAccess преобразует scope-запросы Docker Registry в JWT access.

	scopes - значения query-параметра scope.
*/
func requestedRegistryAccess(
	scopes []string,
) []registryauth.ResourceAction {
	access := make([]registryauth.ResourceAction, 0, len(scopes))
	for _, scope := range scopes {
		parts := strings.SplitN(scope, ":", 3)
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
			continue
		}
		actions := strings.Split(parts[2], ",")
		filteredActions := actions[:0]
		for _, action := range actions {
			if action != "" {
				filteredActions = append(filteredActions, action)
			}
		}
		access = append(access, registryauth.ResourceAction{
			Type:    parts[0],
			Name:    parts[1],
			Actions: filteredActions,
		})
	}
	return access
}

/*
writeInvalidRegistryCredentials возвращает ошибку аутентификации Registry API.
*/
func writeInvalidRegistryCredentials(c *echo.Context) error {
	c.Response().Header().Set("WWW-Authenticate", `Basic realm="registry"`)
	return c.JSON(http.StatusUnauthorized, map[string]any{
		"errors": []map[string]any{
			{
				"code":    "UNAUTHORIZED",
				"message": "invalid username or password",
			},
		},
	})
}

/*
writeRegistryAuthError возвращает внутреннюю ошибку Registry token service.
*/
func writeRegistryAuthError(c *echo.Context) error {
	return c.JSON(http.StatusInternalServerError, map[string]any{
		"errors": []map[string]any{
			{
				"code":    "UNKNOWN",
				"message": "token service failed",
			},
		},
	})
}
