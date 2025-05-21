package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/system"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// baseApiMiddleware мидлварь для авторизации на уровне REST-API.
func baseApiMiddleware(jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.GetHeader("Authorization")
		payload := strings.TrimPrefix(data, "Bearer ")
		if !system.ValidateJWT(payload, jwtKey) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token is not valid"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// baseRegistryMiddleware мидлварь для авторизации на уровне docker client.
func baseRegistryMiddleware(sql *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		repository := c.Param("repository")
		_, err := db.GetRegistry(sql, "name = ?", repository)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get registry"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// loginRegistryMiddleware мидлварь для авторизации на уровне docker client.
// см. https://distribution.github.io/distribution/spec/auth/token/
func loginRegistryMiddleware(cred *config.Env) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		payload := strings.TrimPrefix(authHeader, "Bearer ")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// Формируем challenge согласно спецификации
			challenge := fmt.Sprintf(`Bearer realm="%s/v2/auth"`, cred.Server.Realm)
			if service := c.Query("service"); service != "" {
				challenge += fmt.Sprintf(`,service="%s"`, service)
			}
			if scope := c.Query("scope"); scope != "" {
				challenge += fmt.Sprintf(`,scope="%s"`, scope)
			}
			c.Header("WWW-Authenticate", challenge)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !system.ValidateJWT(payload, []byte(cred.Server.Jwt)) {
			challenge := fmt.Sprintf(`Bearer realm="%s/v2/auth"`, cred.Server.Realm)
			if service := c.Query("service"); service != "" {
				challenge += fmt.Sprintf(`,service="%s"`, service)
			}
			if scope := c.Query("scope"); scope != "" {
				challenge += fmt.Sprintf(`,scope="%s"`, scope)
			}
			c.Header("WWW-Authenticate", challenge)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
