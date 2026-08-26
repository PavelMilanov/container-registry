// Package middleware содержит переиспользуемые HTTP-middleware приложения.
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

/*
RequestLogger логирует HTTP-запросы в едином structured logrus формате.

	logger - экземпляр logger, в который записываются HTTP-запросы.
*/
func RequestLogger(logger *logrus.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()
			err := next(c)

			_, status := echo.ResolveResponseStatus(c.Response(), err)
			latency := time.Since(start)
			fields := logrus.Fields{
				"method":     c.Request().Method,
				"uri":        c.Request().RequestURI,
				"path":       c.Request().URL.Path,
				"status":     status,
				"latency":    latency.String(),
				"latency_ms": latency.Milliseconds(),
				"remote_ip":  c.RealIP(),
			}

			errors := requestErrors(c)
			if err != nil {
				errors = append(errors, err)
			}
			if len(errors) > 0 {
				fields["error"] = errorsString(errors)
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

			return err
		}
	}
}

/*
errorsString объединяет ошибки запроса в строку для structured log.

	errors - ошибки, зарегистрированные во время обработки запроса.
*/
func errorsString(errors []error) string {
	messages := make([]string, 0, len(errors))
	for _, err := range errors {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}
	return strings.Join(messages, "; ")
}
