package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireAPIAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		authorization string
		wantStatus    int
		wantHandled   bool
	}{
		{
			name:       "missing authorization",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "missing bearer scheme",
			authorization: "valid-token",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "invalid token",
			authorization: "Bearer invalid-token",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "valid token",
			authorization: "Bearer valid-token",
			wantStatus:    http.StatusNoContent,
			wantHandled:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handled := false
			router := gin.New()
			router.Use(RequireAPIAuth(func(token string) (Identity, error) {
				if token != "valid-token" {
					return Identity{}, errors.New("invalid token")
				}
				return Identity{Subject: "pavel"}, nil
			}))
			router.GET("/api/check", func(c *gin.Context) {
				handled = true
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/api/check", nil)
			if tt.authorization != "" {
				request.Header.Set("Authorization", tt.authorization)
			}
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

func TestRequireRegistryAuthChallenge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validationCalls := 0
	handled := false
	router := gin.New()
	router.Use(RequireRegistryAuth(
		"https://registry.example.com/",
		"container-registry",
		func(token string) (Identity, error) {
			validationCalls++
			return Identity{}, errors.New("invalid token")
		},
	))
	router.GET("/v2/", func(c *gin.Context) {
		handled = true
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/?scope=repository:dev/image:pull",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if handled {
		t.Fatal("protected handler was called")
	}
	if validationCalls != 0 {
		t.Fatalf("validation calls = %d, want 0", validationCalls)
	}
	wantChallenge := `Bearer realm="https://registry.example.com/v2/auth",service="container-registry",scope="repository:dev/image:pull"`
	if got := response.Header().Get("WWW-Authenticate"); got != wantChallenge {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, wantChallenge)
	}
}

func TestRequireRegistryAuthAllowsValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handled := false
	router := gin.New()
	router.Use(RequireRegistryAuth("https://registry.example.com", "", func(token string) (Identity, error) {
		if token != "valid-token" {
			return Identity{}, errors.New("invalid token")
		}
		return Identity{Subject: "pavel"}, nil
	}))
	router.GET("/v2/", func(c *gin.Context) {
		handled = true
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/v2/", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !handled {
		t.Fatal("protected handler was not called")
	}
}

func TestRequireRegistryAuthEnforcesRepositoryAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		method     string
		access     []ResourceAccess
		wantStatus int
		wantScope  string
	}{
		{
			name:   "pull is allowed",
			method: http.MethodGet,
			access: []ResourceAccess{
				{Type: "repository", Name: "dev/image", Actions: []string{"pull"}},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:   "push is allowed",
			method: http.MethodPut,
			access: []ResourceAccess{
				{Type: "repository", Name: "dev/image", Actions: []string{"push"}},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:   "pull token cannot push",
			method: http.MethodPut,
			access: []ResourceAccess{
				{Type: "repository", Name: "dev/image", Actions: []string{"pull"}},
			},
			wantStatus: http.StatusUnauthorized,
			wantScope:  "repository:dev/image:pull,push",
		},
		{
			name:       "missing repository access",
			method:     http.MethodGet,
			wantStatus: http.StatusUnauthorized,
			wantScope:  "repository:dev/image:pull",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequireRegistryAuth(
				"https://registry.example.com",
				"registry.example.com",
				func(token string) (Identity, error) {
					return Identity{
						Subject: "pavel",
						Access:  tt.access,
					}, nil
				},
			))
			router.Handle(
				tt.method,
				"/v2/:repository/:name/manifests/:reference",
				func(c *gin.Context) {
					c.Status(http.StatusNoContent)
				},
			)

			request := httptest.NewRequest(
				tt.method,
				"/v2/dev/image/manifests/latest",
				nil,
			)
			request.Header.Set("Authorization", "Bearer valid-token")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusUnauthorized {
				wantScope := `scope="` + tt.wantScope + `"`
				if !strings.Contains(
					response.Header().Get("WWW-Authenticate"),
					wantScope,
				) {
					t.Fatalf(
						"WWW-Authenticate = %q, want %q",
						response.Header().Get("WWW-Authenticate"),
						wantScope,
					)
				}
			}
		})
	}
}
