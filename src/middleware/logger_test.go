package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func TestRequestLoggerUsesStatusLevel(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		withError bool
		wantLevel logrus.Level
	}{
		{
			name:      "success",
			status:    http.StatusOK,
			wantLevel: logrus.InfoLevel,
		},
		{
			name:      "client error with context error",
			status:    http.StatusUnauthorized,
			withError: true,
			wantLevel: logrus.WarnLevel,
		},
		{
			name:      "server error",
			status:    http.StatusInternalServerError,
			wantLevel: logrus.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			router := echo.New()
			router.Use(RequestLogger(logger))
			router.GET("/test", func(c *echo.Context) error {
				if tt.withError {
					AddRequestError(c, errors.New("request failed"))
				}
				return c.NoContent(tt.status)
			})

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			entry := hook.LastEntry()
			if entry == nil {
				t.Fatal("request log entry was not created")
			}
			if entry.Level != tt.wantLevel {
				t.Fatalf("level = %s, want %s", entry.Level, tt.wantLevel)
			}
			if got := entry.Data["status"]; got != tt.status {
				t.Fatalf("status field = %v, want %d", got, tt.status)
			}
		})
	}
}
