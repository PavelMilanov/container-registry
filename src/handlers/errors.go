package handlers

import (
	"errors"

	"github.com/PavelMilanov/container-registry/middleware"
	"github.com/labstack/echo/v5"
)

/*
addRequestError добавляет ошибку в контекст HTTP-запроса.

	err - ошибка обработки запроса.
*/
func addRequestError(c *echo.Context, err error) {
	middleware.AddRequestError(c, err)
}

/*
addRequestErrorMessage добавляет текстовую ошибку в контекст HTTP-запроса.

	msg - текст ошибки.
*/
func addRequestErrorMessage(c *echo.Context, msg string) {
	addRequestError(c, errors.New(msg))
}
