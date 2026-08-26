package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func TestInitRoutersRegistersPublicAndRegistryRoutes(t *testing.T) {
	env := &config.Env{}
	env.Server.Realm = "https://registry.example.com"
	handler := &Handler{ENV: env}
	router := handler.InitRouters()

	routes := make(map[string]struct{})
	for _, route := range router.Router().Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"POST /login",
		"GET /check",
		"GET /v2/auth",
		"GET /v2/",
		"PUT /v2/:repository/:name/manifests/:reference",
		"PATCH /v2/:repository/:name/blobs/uploads/:uuid",
		"GET /api/cloud/list",
		"POST /api/garbage/collection",
	} {
		if _, ok := routes[route]; !ok {
			t.Errorf("route %q is not registered", route)
		}
	}
}

func TestInitRoutersFallbackResponses(t *testing.T) {
	env := &config.Env{}
	env.Server.Realm = "https://registry.example.com"
	handler := &Handler{ENV: env}
	router := handler.InitRouters()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "health fallback",
			path:       "/unknown",
			wantStatus: http.StatusOK,
			wantBody:   "is OK.",
		},
		{
			name:       "registry route not found",
			path:       "/v2/unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantBody != "" && response.Body.String() != test.wantBody {
				t.Fatalf("body = %q, want %q", response.Body.String(), test.wantBody)
			}
		})
	}
}
