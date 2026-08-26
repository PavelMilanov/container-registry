package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
)

// NamespaceChecker проверяет наличие пространства имён registry.
type NamespaceChecker interface {
	NamespaceExists(
		ctx context.Context,
		name string,
	) (bool, error)
}

/*
RequireNamespace проверяет существование пространства из параметра repository.

	checker - реализация проверки пространства в текущем хранилище.
*/
func RequireNamespace(checker NamespaceChecker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if checker == nil {
				AddRequestError(c, errors.New("namespace checker is not configured"))
				return writeNamespaceError(
					c,
					http.StatusInternalServerError,
					"UNKNOWN",
					"registry lookup failed",
				)
			}

			name := c.Param("repository")
			exists, err := checker.NamespaceExists(c.Request().Context(), name)
			if err != nil {
				AddRequestError(c, err)
				return writeNamespaceError(
					c,
					http.StatusInternalServerError,
					"UNKNOWN",
					"registry lookup failed",
				)
			}
			if !exists {
				err := fmt.Errorf("%w: %s", errNamespaceNotFound, name)
				AddRequestError(c, err)
				return writeNamespaceError(
					c,
					http.StatusNotFound,
					"NAME_UNKNOWN",
					errNamespaceNotFound.Error(),
				)
			}

			return next(c)
		}
	}
}

/*
writeNamespaceError возвращает ошибку Docker Registry API.

	status - HTTP-статус ответа.
	code - код ошибки Docker Registry.
	message - описание ошибки.
*/
func writeNamespaceError(
	c *echo.Context,
	status int,
	code string,
	message string,
) error {
	return c.JSON(status, map[string]any{
		"errors": []map[string]any{
			{
				"code":    code,
				"message": message,
			},
		},
	})
}
