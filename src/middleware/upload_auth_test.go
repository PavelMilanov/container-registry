package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestUploadStatusRequiresPushAccess(t *testing.T) {
	for _, action := range []string{"pull", "push"} {
		t.Run(action, func(t *testing.T) {
			router := echo.New()
			router.Use(RequireRegistryAuth("https://registry.example", "container-registry", func(string) (Identity, error) {
				return Identity{Subject: "user", Access: []ResourceAccess{{Type: "repository", Name: "dev/image", Actions: []string{action}}}}, nil
			}))
			router.GET("/v2/:repository/:name/blobs/uploads/:uuid", func(c *echo.Context) error { return c.NoContent(204) })
			request := httptest.NewRequest(http.MethodGet, "/v2/dev/image/blobs/uploads/test-id", nil)
			request.Header.Set("Authorization", "Bearer valid-token")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			want := 204
			if action == "pull" {
				want = 401
			}
			if response.Code != want {
				t.Fatalf("status=%d want=%d", response.Code, want)
			}
			if action == "pull" && !strings.Contains(response.Header().Get("WWW-Authenticate"), "repository:dev/image:pull,push") {
				t.Fatal(response.Header())
			}
		})
	}
}
