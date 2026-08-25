package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

/*
RequireAPIAuth проверяет Bearer-токен REST API.

	validate - функция проверки токена.
*/
func RequireAPIAuth(validate TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok || validate == nil {
			addRequestError(c, errInvalidToken)
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": errInvalidToken.Error()},
			)
			return
		}
		identity, err := validate(token)
		if err != nil {
			addRequestError(c, err)
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": errInvalidToken.Error()},
			)
			return
		}
		c.Set(AuthenticatedSubjectKey, identity.Subject)

		c.Next()
	}
}
