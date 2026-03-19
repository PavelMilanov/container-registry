package client

import (
	"net/http"
	"net/http/httptest"
	"os"
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

func setupAuthToken(t *testing.T) {
	t.Helper()
	original, err := os.ReadFile("/tmp/.auth")
	hasOriginal := err == nil
	if err := os.WriteFile("/tmp/.auth", []byte("test-token"), 0600); err != nil {
		t.Fatalf("failed to prepare auth token: %v", err)
	}
	t.Cleanup(func() {
		if hasOriginal {
			_ = os.WriteFile("/tmp/.auth", original, 0600)
			return
		}
		_ = os.Remove("/tmp/.auth")
	})
}

func TestGetCloudList(t *testing.T) {
	setupAuthToken(t)
	server := setupCloudTestServer(t)
	defer server.Close()

	cr := NewClient()
	cr.ServerURL = server.URL

	data, err := cr.GetCloudList()
	if err != nil {
		t.Fatalf("GetCloudList() error = %v", err)
	}
	if len(data) != 2 || data[0] != "dev" || data[1] != "stage" {
		t.Fatalf("unexpected cloud list: %+v", data)
	}
}

func TestGetImagesList(t *testing.T) {
	setupAuthToken(t)
	server := setupCloudTestServer(t)
	defer server.Close()

	cr := NewClient()
	cr.ServerURL = server.URL

	data, err := cr.GetImagesList("dev", "registry")
	if err != nil {
		t.Fatalf("GetImagesList() error = %v", err)
	}
	if len(data) != 2 || data[0] != "latest" || data[1] != "1.0.0" {
		t.Fatalf("unexpected images list: %+v", data)
	}
}

func TestDelImage(t *testing.T) {
	setupAuthToken(t)
	server := setupCloudTestServer(t)
	defer server.Close()

	cr := NewClient()
	cr.ServerURL = server.URL

	if err := cr.DelImage("dev", "registry", "latest"); err != nil {
		t.Fatalf("DelImage() error = %v", err)
	}
}
