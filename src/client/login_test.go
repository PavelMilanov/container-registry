package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogin(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if req.Username == "admin" && req.Password == "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"token":"test-token"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid username or password"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	cr := NewClient()
	cr.ServerURL = server.URL

	t.Run("Failure", func(t *testing.T) {
		_, err := cr.Login(context.Background(), "test", "test")
		if err == nil {
			t.Fatal("expected error for invalid credentials, got nil")
		}
		if !strings.Contains(err.Error(), "invalid username or password") {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	})
	t.Run("Success", func(t *testing.T) {
		token, err := cr.Login(context.Background(), "admin", "admin")
		if err != nil {
			t.Fatal(err)
		}
		if token != "test-token" {
			t.Fatalf("unexpected token: %s", token)
		}
	})
}
