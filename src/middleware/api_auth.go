package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

/*
RequireAPIAuth проверяет Bearer-токен REST API.

	validate - функция проверки токена.
*/
func RequireAPIAuth(validate TokenValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			token, ok := bearerToken(c.Request().Header.Get("Authorization"))
			if !ok || validate == nil {
				AddRequestError(c, errInvalidToken)
				return c.JSON(
					http.StatusUnauthorized,
					map[string]any{"error": errInvalidToken.Error()},
				)
			}
			identity, err := validate(token)
			if err != nil {
				AddRequestError(c, err)
				return c.JSON(
					http.StatusUnauthorized,
					map[string]any{"error": errInvalidToken.Error()},
				)
			}
			c.Set(AuthenticatedSubjectKey, identity.Subject)

			return next(c)
		}
	}
}
