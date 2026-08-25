package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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
) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok || validate == nil {
			writeRegistryChallenge(c, realm, service)
			return
		}
		identity, err := validate(token)
		if err != nil {
			writeRegistryChallenge(c, realm, service)
			return
		}
		if !registryAccessAllowed(c, identity.Access) {
			writeRegistryChallenge(c, realm, service)
			return
		}
		c.Set(AuthenticatedSubjectKey, identity.Subject)

		c.Next()
	}
}

/*
registryAccessAllowed проверяет доступ JWT к ресурсу текущего Registry-запроса.

	access - разрешения из проверенного JWT.
*/
func registryAccessAllowed(
	c *gin.Context,
	access []ResourceAccess,
) bool {
	repository := c.Param("repository")
	name := c.Param("name")
	if repository == "" || name == "" {
		return true
	}

	requiredAction := "push"
	if c.Request.Method == http.MethodGet ||
		c.Request.Method == http.MethodHead {
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
	c *gin.Context,
	realm string,
	service string,
) {
	challenge := fmt.Sprintf(
		"Bearer realm=%q",
		strings.TrimRight(realm, "/")+"/v2/auth",
	)
	if service != "" {
		challenge += fmt.Sprintf(",service=%q", service)
	}
	scope := c.Query("scope")
	if scope == "" {
		scope = requiredRegistryScope(c)
	}
	if scope != "" {
		challenge += fmt.Sprintf(",scope=%q", scope)
	}

	c.Header("WWW-Authenticate", challenge)
	addRequestError(c, errAuthorizationRequired)
	c.AbortWithStatus(http.StatusUnauthorized)
}

/*
requiredRegistryScope формирует scope для текущего Registry-запроса.
*/
func requiredRegistryScope(c *gin.Context) string {
	repository := c.Param("repository")
	name := c.Param("name")
	if repository == "" || name == "" {
		return ""
	}

	actions := "pull,push"
	if c.Request.Method == http.MethodGet ||
		c.Request.Method == http.MethodHead {
		actions = "pull"
	}
	return fmt.Sprintf(
		"repository:%s/%s:%s",
		repository,
		name,
		actions,
	)
}
