// Package middleware содержит переиспользуемые HTTP-middleware приложения.
package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

/*
RequestLogger логирует HTTP-запросы в едином structured logrus формате.

	logger - экземпляр logger, в который записываются HTTP-запросы.
*/
func RequestLogger(logger *logrus.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)
		fields := logrus.Fields{
			"method":        c.Request.Method,
			"uri":           c.Request.RequestURI,
			"path":          c.Request.URL.Path,
			"status":        status,
			"latency":       latency.String(),
			"latency_ms":    latency.Milliseconds(),
			"remote_ip":     c.ClientIP(),
			"host":          c.Request.Host,
			"response_size": c.Writer.Size(),
			"user_agent":    c.Request.UserAgent(),
		}

		if len(c.Errors) > 0 {
			fields["error"] = c.Errors.String()
		} else if status >= http.StatusInternalServerError {
			fields["error"] = http.StatusText(status)
		}

		entry := logger.WithFields(fields)
		switch {
		case status >= http.StatusInternalServerError:
			entry.Error("HTTP request")
		case status >= http.StatusBadRequest:
			entry.Warn("HTTP request")
		default:
			entry.Info("HTTP request")
		}
	}
}
