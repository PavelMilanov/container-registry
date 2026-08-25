package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
	"github.com/PavelMilanov/container-registry/services"
	"github.com/gin-gonic/gin"
)

type fakeAuthenticator struct {
	loginCalls int
	loginErr   error
	access     []registryauth.ResourceAction
	issued     registryauth.IssuedToken
}

func (f *fakeAuthenticator) Register(
	ctx context.Context,
	username string,
	password string,
) error {
	return nil
}

func (f *fakeAuthenticator) Login(
	ctx context.Context,
	username string,
	password string,
	access []registryauth.ResourceAction,
) (registryauth.IssuedToken, error) {
	f.loginCalls++
	f.access = access
	return f.issued, f.loginErr
}

func (f *fakeAuthenticator) ValidateToken(
	rawToken string,
) (registryauth.Claims, error) {
	return registryauth.Claims{}, nil
}

func TestRequestedRegistryAccess(t *testing.T) {
	access := requestedRegistryAccess([]string{
		"repository:dev/image:pull,push",
		"invalid",
		"repository:team/backend:pull",
	})

	if len(access) != 2 {
		t.Fatalf("access count = %d, want 2", len(access))
	}
	if access[0].Type != "repository" || access[0].Name != "dev/image" {
		t.Fatalf("first access = %#v", access[0])
	}
	if len(access[0].Actions) != 2 || access[0].Actions[1] != "push" {
		t.Fatalf("first actions = %#v", access[0].Actions)
	}
	if access[1].Name != "team/backend" {
		t.Fatalf("second access = %#v", access[1])
	}
}

func TestRegistryAuthHandlerIssuesOneTokenWithActualTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	issuedAt := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	authenticator := &fakeAuthenticator{
		issued: registryauth.IssuedToken{
			Value:     "signed-token",
			IssuedAt:  issuedAt,
			ExpiresAt: issuedAt.Add(2 * time.Hour),
		},
	}
	handler := &Handler{AUTH: authenticator}
	router := gin.New()
	router.GET("/v2/auth", handler.authHandler)

	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/auth?scope=repository:dev/image:pull,push",
		nil,
	)
	request.SetBasicAuth("pavel", "password")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if authenticator.loginCalls != 1 {
		t.Fatalf("login calls = %d, want 1", authenticator.loginCalls)
	}
	if len(authenticator.access) != 1 ||
		authenticator.access[0].Name != "dev/image" {
		t.Fatalf("access = %#v", authenticator.access)
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		IssuedAt    string `json:"issued_at"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.AccessToken != "signed-token" {
		t.Fatalf("access token = %q", body.AccessToken)
	}
	if body.ExpiresIn != 7200 {
		t.Fatalf("expires_in = %d, want 7200", body.ExpiresIn)
	}
	if body.IssuedAt != issuedAt.Format(time.RFC3339) {
		t.Fatalf("issued_at = %q", body.IssuedAt)
	}
}

func TestLoginHandlerReturnsUnauthorizedForInvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authenticator := &fakeAuthenticator{loginErr: services.ErrInvalidCredentials}
	handler := &Handler{AUTH: authenticator}
	router := gin.New()
	router.POST("/login", handler.login)

	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"pavel","password":"wrong"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status = %d, want %d",
			response.Code,
			http.StatusUnauthorized,
		)
	}
}

func TestRegistryAuthHandlerReturnsInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authenticator := &fakeAuthenticator{loginErr: errors.New("database unavailable")}
	handler := &Handler{AUTH: authenticator}
	router := gin.New()
	router.GET("/v2/auth", handler.authHandler)

	request := httptest.NewRequest(http.MethodGet, "/v2/auth", nil)
	request.SetBasicAuth("pavel", "password")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, want %d",
			response.Code,
			http.StatusInternalServerError,
		)
	}
}
