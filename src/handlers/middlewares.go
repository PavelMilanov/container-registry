package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/system"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func addRequestError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	_ = c.Error(err)
}

func addRequestErrorMessage(c *gin.Context, msg string) {
	addRequestError(c, errors.New(msg))
}

/*
requestLoggerMiddleware логирует HTTP-запросы в едином structured logrus формате.
*/
func requestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		fields := logrus.Fields{
			"method":    c.Request.Method,
			"uri":       c.Request.RequestURI,
			"path":      c.Request.URL.Path,
			"status":    status,
			"latency":   time.Since(start).String(),
			"remote_ip": c.ClientIP(),
			"host":      c.Request.Host,
		}

		if len(c.Errors) > 0 {
			fields["error"] = c.Errors.String()
		} else if status >= http.StatusInternalServerError {
			fields["error"] = http.StatusText(status)
		}

		entry := logrus.WithFields(fields)
		if len(c.Errors) > 0 {
			entry.Error("HTTP request")
			return
		}
		if status >= http.StatusInternalServerError {
			entry.Error("HTTP request")
			return
		}
		if status >= http.StatusBadRequest {
			entry.Warn("HTTP request")
			return
		}
		entry.Info("HTTP request")
	}
}

/*
baseApiMiddleware для авторизации на уровне REST-API.

	проверяет валидность токена для REST-API.
*/
func baseApiMiddleware(jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.GetHeader("Authorization")
		payload := strings.TrimPrefix(data, "Bearer ")
		if !system.ValidateJWT(payload, jwtKey) {
			addRequestErrorMessage(c, "token is not valid")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token is not valid"})
			c.Abort()
			return
		}
		c.Next()
	}
}

/*
baseRegistryMiddleware для проверки корректности указанного репозитория.

	проверяет соответствие указанного образа репозиторию.
*/
func baseRegistryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		repo := c.Param("repository")
		if _, err := os.Stat(filepath.Join(config.MANIFEST_PATH, repo)); err != nil {
			if os.IsNotExist(err) {
				addRequestError(c, err)
				c.JSON(http.StatusNotFound, gin.H{
					"errors": []gin.H{
						{
							"code":    "NAME_UNKNOWN",
							"message": "registry does not exist",
						},
					},
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

/*
urlChallenge перенапрваление на url авторизации для docker client.
*/
func urlChallenge(c *gin.Context, realm string) {
	challenge := fmt.Sprintf(`Bearer realm="%s/v2/auth"`, realm)
	if service := c.Query("service"); service != "" {
		challenge += fmt.Sprintf(`,service="%s"`, service)
	}
	if scope := c.Query("scope"); scope != "" {
		challenge += fmt.Sprintf(`,scope="%s"`, scope)
	}
	c.Header("WWW-Authenticate", challenge)
	addRequestErrorMessage(c, "authorization required")
	c.AbortWithStatus(http.StatusUnauthorized)
}

/*
loginRegistryMiddleware мидлварь для авторизации на уровне docker client.

	https://distribution.github.io/distribution/spec/auth/token/
*/
func loginRegistryMiddleware(cred *config.Env) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		payload := strings.TrimPrefix(authHeader, "Bearer ")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// Формируем challenge согласно спецификации
			urlChallenge(c, cred.Server.Realm)
		}
		if !system.ValidateJWT(payload, []byte(cred.Server.Jwt)) {
			// Формируем challenge согласно спецификации
			urlChallenge(c, cred.Server.Realm)
		}
		c.Next()
	}
}
