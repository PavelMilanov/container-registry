package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/labstack/echo/v5"
)

type fakeBlobStore struct {
	blob config.Blob
	err  error
}

func (f fakeBlobStore) CheckBlob(digest string) error {
	return f.err
}

func (f fakeBlobStore) GetBlob(digest string) (config.Blob, error) {
	return f.blob, f.err
}

func TestGetBlobServesAbsoluteStoragePath(t *testing.T) {
	body := []byte("blob body")
	path := filepath.Join(t.TempDir(), "blob")
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}

	handler := &Handler{
		BLOBS: fakeBlobStore{
			blob: config.Blob{
				Digest: "sha256:test",
				Path:   path,
				Size:   int64(len(body)),
			},
		},
	}
	router := echo.New()
	router.GET("/v2/:repository/:name/blobs/:uuid", handler.getBlob)

	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/dev/image/blobs/sha256:test",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != string(body) {
		t.Fatalf("body = %q, want %q", response.Body.String(), body)
	}
	if got := response.Header().Get("Docker-Content-Digest"); got != "sha256:test" {
		t.Fatalf("Docker-Content-Digest = %q", got)
	}
}
