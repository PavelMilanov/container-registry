package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type namespaceCheckerFunc func(
	ctx context.Context,
	name string,
) (bool, error)

func (f namespaceCheckerFunc) NamespaceExists(
	ctx context.Context,
	name string,
) (bool, error) {
	return f(ctx, name)
}

func TestRequireNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		check       namespaceCheckerFunc
		wantStatus  int
		wantHandled bool
	}{
		{
			name: "namespace exists",
			check: func(ctx context.Context, name string) (bool, error) {
				return name == "dev", nil
			},
			wantStatus:  http.StatusNoContent,
			wantHandled: true,
		},
		{
			name: "namespace not found",
			check: func(ctx context.Context, name string) (bool, error) {
				return false, nil
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "storage failure",
			check: func(ctx context.Context, name string) (bool, error) {
				return false, errors.New("storage unavailable")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handled := false
			router := gin.New()
			router.Use(RequireNamespace(tt.check))
			router.GET("/v2/:repository/:name", func(c *gin.Context) {
				handled = true
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/v2/dev/image", nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if handled != tt.wantHandled {
				t.Fatalf("handled = %t, want %t", handled, tt.wantHandled)
			}
		})
	}
}

func TestRequireNamespaceWithoutChecker(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handled := false
	router := gin.New()
	router.Use(RequireNamespace(nil))
	router.GET("/v2/:repository/:name", func(c *gin.Context) {
		handled = true
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/v2/dev/image", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, want %d",
			response.Code,
			http.StatusInternalServerError,
		)
	}
	if handled {
		t.Fatal("protected handler was called")
	}
}
