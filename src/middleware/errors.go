package middleware

import (
	"errors"

	"github.com/labstack/echo/v5"
)

const requestErrorsKey = "request_errors"

var (
	errInvalidToken          = errors.New("token is not valid")
	errAuthorizationRequired = errors.New("authorization required")
	errNamespaceNotFound     = errors.New("registry does not exist")
)

/*
AddRequestError добавляет ошибку в контекст HTTP-запроса.

	err - ошибка обработки запроса.
*/
func AddRequestError(c *echo.Context, err error) {
	if err == nil {
		return
	}

	errors, _ := c.Get(requestErrorsKey).([]error)
	c.Set(requestErrorsKey, append(errors, err))
}

/*
requestErrors возвращает ошибки, зарегистрированные при обработке HTTP-запроса.
*/
func requestErrors(c *echo.Context) []error {
	errors, _ := c.Get(requestErrorsKey).([]error)
	return errors
}
