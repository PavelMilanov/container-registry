package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
)

var (
	errInvalidToken          = errors.New("token is not valid")
	errAuthorizationRequired = errors.New("authorization required")
	errNamespaceNotFound     = errors.New("registry does not exist")
)

/*
addRequestError добавляет ошибку в контекст HTTP-запроса.

	err - ошибка обработки запроса.
*/
func addRequestError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	_ = c.Error(err)
}
