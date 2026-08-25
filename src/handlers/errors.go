package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
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

/*
addRequestErrorMessage добавляет текстовую ошибку в контекст HTTP-запроса.

	msg - текст ошибки.
*/
func addRequestErrorMessage(c *gin.Context, msg string) {
	addRequestError(c, errors.New(msg))
}
