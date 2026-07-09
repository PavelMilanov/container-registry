package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func setupCloudTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/cloud/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"clouds":["dev","stage"]}`))
	})

	mux.HandleFunc("/api/cloud/dev/registry", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"images":["latest","1.0.0"]}`))
		case http.MethodDelete:
			if r.URL.Query().Get("tag") != "latest" {
				http.Error(w, `{"error":"tag is required"}`, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func setupAuthToken(t *testing.T, cr *Client) {
	t.Helper()
	cr.tokenPath = filepath.Join(t.TempDir(), "auth")
	if err := cr.SetToken("test-token"); err != nil {
		t.Fatalf("failed to prepare auth token: %v", err)
	}
}

func TestGetCloudList(t *testing.T) {
	server := setupCloudTestServer(t)
	defer server.Close()

	cr := NewClient()
	cr.ServerURL = server.URL
	setupAuthToken(t, cr)

	data, err := cr.GetCloudList(context.Background())
	if err != nil {
		t.Fatalf("GetCloudList() error = %v", err)
	}
	if len(data) != 2 || data[0] != "dev" || data[1] != "stage" {
		t.Fatalf("unexpected cloud list: %+v", data)
	}
}

func TestGetImagesList(t *testing.T) {
	server := setupCloudTestServer(t)
	defer server.Close()

	cr := NewClient()
	cr.ServerURL = server.URL
	setupAuthToken(t, cr)

	data, err := cr.GetImagesList(context.Background(), "dev", "registry")
	if err != nil {
		t.Fatalf("GetImagesList() error = %v", err)
	}
	if len(data) != 2 || data[0] != "latest" || data[1] != "1.0.0" {
		t.Fatalf("unexpected images list: %+v", data)
	}
}

func TestDelImage(t *testing.T) {
	server := setupCloudTestServer(t)
	defer server.Close()

	cr := NewClient()
	cr.ServerURL = server.URL
	setupAuthToken(t, cr)

	if err := cr.DelImage(context.Background(), "dev", "registry", "latest"); err != nil {
		t.Fatalf("DelImage() error = %v", err)
	}
}
