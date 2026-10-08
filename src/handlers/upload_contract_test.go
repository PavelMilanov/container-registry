package handlers

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/internal/s3test"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/minio/minio-go/v7"
)

type httpUploadFixture struct {
	store   storage.BlobUploadStore
	setBlob func(string, []byte) error
}

func TestBlobUploadHTTPContract(t *testing.T) {
	for name, factory := range map[string]func(*testing.T) httpUploadFixture{
		"local": func(t *testing.T) httpUploadFixture {
			oldTmp, oldBlobs := config.TMP_PATH, config.BLOBS_PATH
			config.TMP_PATH, config.BLOBS_PATH = t.TempDir(), t.TempDir()
			t.Cleanup(func() { config.TMP_PATH, config.BLOBS_PATH = oldTmp, oldBlobs })
			return httpUploadFixture{&storage.LocalStorage{}, func(digest string, data []byte) error {
				return os.WriteFile(filepath.Join(config.BLOBS_PATH, strings.TrimPrefix(digest, "sha256:")), data, 0600)
			}}
		},
		"s3": func(t *testing.T) httpUploadFixture {
			client, bucket := s3test.New(t)
			return httpUploadFixture{&storage.S3Storage{S3: client, Bucket: bucket}, func(digest string, data []byte) error {
				_, err := client.PutObject(context.Background(), bucket, "var/blobs/"+strings.TrimPrefix(digest, "sha256:"), bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
				return err
			}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := factory(t)
			router := newUploadRouter(f.store)
			id := startTestUpload(t, router)
			url := "/v2/dev/image/blobs/uploads/" + id
			status := uploadRequest(router, http.MethodGet, url, nil, nil)
			if status.Code != 204 || status.Header().Get("X-Upload-Offset") != "0" {
				t.Fatalf("initial status: %d %v", status.Code, status.Header())
			}
			bad := uploadRequest(router, http.MethodPatch, url, []byte("abc"), map[string]string{"Content-Range": "0-3"})
			if bad.Code != 416 {
				t.Fatalf("partial PATCH=%d", bad.Code)
			}
			patch := uploadRequest(router, http.MethodPatch, url, []byte("abc"), map[string]string{"Content-Range": "0-2"})
			if patch.Code != 204 || patch.Header().Get("X-Upload-Offset") != "3" {
				t.Fatalf("PATCH=%d %s", patch.Code, patch.Body.String())
			}
			badOffset := uploadRequest(router, http.MethodPatch, url, []byte("abc"), map[string]string{"Content-Range": "0-2"})
			if badOffset.Code != 416 || badOffset.Header().Get("X-Upload-Offset") != "3" || badOffset.Header().Get("Range") != "0-2" {
				t.Fatalf("offset error=%d %v", badOffset.Code, badOffset.Header())
			}
			digest := testDigest([]byte("abc"))
			if err := f.setBlob(digest, []byte("bad")); err != nil {
				t.Fatal(err)
			}
			putURL := url + "?digest=" + digest
			failed := uploadRequest(router, http.MethodPut, putURL, nil, nil)
			if failed.Code != 500 {
				t.Fatalf("corrupted destination=%d %s", failed.Code, failed.Body.String())
			}
			status = uploadRequest(router, http.MethodGet, url, nil, nil)
			if status.Code != 204 || status.Header().Get("X-Upload-State") != "ready" || status.Header().Get("X-Upload-Offset") != "3" {
				t.Fatalf("ready status=%d %v", status.Code, status.Header())
			}
			for _, method := range []string{http.MethodPatch, http.MethodPut} {
				target := putURL
				if method == http.MethodPatch {
					target = url
				}
				response := uploadRequest(router, method, target, []byte("abc"), map[string]string{"Content-Range": "3-5"})
				if response.Code != 409 {
					t.Fatalf("frozen %s=%d %s", method, response.Code, response.Body.String())
				}
			}
			if err := f.setBlob(digest, []byte("abc")); err != nil {
				t.Fatal(err)
			}
			completed := uploadRequest(router, http.MethodPut, putURL, nil, nil)
			if completed.Code != 201 {
				t.Fatalf("retry PUT=%d %s", completed.Code, completed.Body.String())
			}
			status = uploadRequest(router, http.MethodGet, url, nil, nil)
			if status.Code != 404 {
				t.Fatalf("completed status=%d", status.Code)
			}
		})
	}
}
