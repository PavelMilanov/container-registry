package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
	"github.com/PavelMilanov/container-registry/services"
	"github.com/gin-gonic/gin"
)

/*
authHandler аутентифицирует Docker client и выдаёт Registry JWT.

	/v2/auth
*/
func (h *Handler) authHandler(c *gin.Context) {
	username, password, ok := c.Request.BasicAuth()
	if !ok {
		writeInvalidRegistryCredentials(c)
		return
	}

	access := requestedRegistryAccess(c.QueryArray("scope"))
	token, err := h.AUTH.Login(
		c.Request.Context(),
		username,
		password,
		access,
	)
	if err != nil {
		addRequestError(c, err)
		if errors.Is(err, services.ErrInvalidCredentials) {
			writeInvalidRegistryCredentials(c)
			return
		}
		writeRegistryAuthError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
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
func writeInvalidRegistryCredentials(c *gin.Context) {
	c.Header("WWW-Authenticate", `Basic realm="registry"`)
	c.JSON(http.StatusUnauthorized, gin.H{
		"errors": []gin.H{
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
func writeRegistryAuthError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"errors": []gin.H{
			{
				"code":    "UNKNOWN",
				"message": "token service failed",
			},
		},
	})
}
