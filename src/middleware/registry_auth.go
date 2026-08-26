package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

/*
RequireRegistryAuth проверяет Bearer-токен Docker Registry API.

	realm - публичный адрес сервера авторизации.
	service - имя сервиса Docker Registry.
	validate - функция проверки токена.
*/
func RequireRegistryAuth(
	realm string,
	service string,
	validate TokenValidator,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			token, ok := bearerToken(c.Request().Header.Get("Authorization"))
			if !ok || validate == nil {
				return writeRegistryChallenge(c, realm, service)
			}
			identity, err := validate(token)
			if err != nil {
				AddRequestError(c, err)
				return writeRegistryChallenge(c, realm, service)
			}
			if !registryAccessAllowed(c, identity.Access) {
				return writeRegistryChallenge(c, realm, service)
			}
			c.Set(AuthenticatedSubjectKey, identity.Subject)

			return next(c)
		}
	}
}

/*
registryAccessAllowed проверяет доступ JWT к ресурсу текущего Registry-запроса.

	access - разрешения из проверенного JWT.
*/
func registryAccessAllowed(
	c *echo.Context,
	access []ResourceAccess,
) bool {
	repository := c.Param("repository")
	name := c.Param("name")
	if repository == "" || name == "" {
		return true
	}

	requiredAction := "push"
	if c.Request().Method == http.MethodGet ||
		c.Request().Method == http.MethodHead {
		requiredAction = "pull"
	}
	resourceName := repository + "/" + name
	for _, resource := range access {
		if resource.Type != "repository" || resource.Name != resourceName {
			continue
		}
		for _, action := range resource.Actions {
			if action == requiredAction || action == "*" {
				return true
			}
		}
	}
	return false
}

/*
writeRegistryChallenge возвращает Docker Registry Bearer challenge.

	realm - публичный адрес сервера авторизации.
	service - имя сервиса Docker Registry.
*/
func writeRegistryChallenge(
	c *echo.Context,
	realm string,
	service string,
) error {
	challenge := fmt.Sprintf(
		"Bearer realm=%q",
		strings.TrimRight(realm, "/")+"/v2/auth",
	)
	if service != "" {
		challenge += fmt.Sprintf(",service=%q", service)
	}
	scope := c.QueryParam("scope")
	if scope == "" {
		scope = requiredRegistryScope(c)
	}
	if scope != "" {
		challenge += fmt.Sprintf(",scope=%q", scope)
	}

	c.Response().Header().Set("WWW-Authenticate", challenge)
	AddRequestError(c, errAuthorizationRequired)
	return c.NoContent(http.StatusUnauthorized)
}

/*
requiredRegistryScope формирует scope для текущего Registry-запроса.
*/
func requiredRegistryScope(c *echo.Context) string {
	repository := c.Param("repository")
	name := c.Param("name")
	if repository == "" || name == "" {
		return ""
	}

	actions := "pull,push"
	if c.Request().Method == http.MethodGet ||
		c.Request().Method == http.MethodHead {
		actions = "pull"
	}
	return fmt.Sprintf(
		"repository:%s/%s:%s",
		repository,
		name,
		actions,
	)
}
