package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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
func RequireNamespace(checker NamespaceChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checker == nil {
			addRequestError(c, errors.New("namespace checker is not configured"))
			writeNamespaceError(
				c,
				http.StatusInternalServerError,
				"UNKNOWN",
				"registry lookup failed",
			)
			return
		}

		name := c.Param("repository")
		exists, err := checker.NamespaceExists(c.Request.Context(), name)
		if err != nil {
			addRequestError(c, err)
			writeNamespaceError(
				c,
				http.StatusInternalServerError,
				"UNKNOWN",
				"registry lookup failed",
			)
			return
		}
		if !exists {
			err := fmt.Errorf("%w: %s", errNamespaceNotFound, name)
			addRequestError(c, err)
			writeNamespaceError(
				c,
				http.StatusNotFound,
				"NAME_UNKNOWN",
				errNamespaceNotFound.Error(),
			)
			return
		}

		c.Next()
	}
}

/*
writeNamespaceError возвращает ошибку Docker Registry API.

	status - HTTP-статус ответа.
	code - код ошибки Docker Registry.
	message - описание ошибки.
*/
func writeNamespaceError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.AbortWithStatusJSON(status, gin.H{
		"errors": []gin.H{
			{
				"code":    code,
				"message": message,
			},
		},
	})
}
