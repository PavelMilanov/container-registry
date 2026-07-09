package handlers

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/gin-gonic/gin"
)

type fakeStorage struct {
	savedBlobDigest string
	savedBlobBody   []byte

	savedManifestMeta config.Meta
	savedManifestBody []byte
	savedManifestPath string
}

func (f *fakeStorage) CheckBlob(uuid string) error { return nil }

func (f *fakeStorage) SaveBlob(tmpPath, digest string) error {
	body, err := os.ReadFile(tmpPath)
	if err != nil {
		return err
	}
	f.savedBlobDigest = digest
	f.savedBlobBody = body
	return nil
}

func (f *fakeStorage) GetBlob(digest string) (config.Blob, error) {
	return config.Blob{}, errors.New("not implemented")
}

func (f *fakeStorage) SaveManifest(meta config.Meta, body []byte, link string) error {
	f.savedManifestMeta = meta
	f.savedManifestBody = append([]byte(nil), body...)
	f.savedManifestPath = link
	return nil
}

func (f *fakeStorage) GetManifest(repository, image, reference string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeStorage) GetManifestList(cloud, repository string) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeStorage) DeleteManifest(cloud, repository, tag string) error { return nil }
func (f *fakeStorage) AddCloud(cloud string) error                        { return nil }
func (f *fakeStorage) DeleteCloud(cloud string) error                     { return nil }
func (f *fakeStorage) GetCloudList() ([]string, error)                    { return nil, nil }
func (f *fakeStorage) GetRepositoriesList(cloud string) ([]string, error) { return nil, nil }
func (f *fakeStorage) DeleteRepository(cloud, repository string) error    { return nil }
func (f *fakeStorage) GarbageCollection() error                           { return nil }
func (f *fakeStorage) DeleteOlderTags(count int) error                    { return nil }

func testHandler(storage *fakeStorage) *Handler {
	return &Handler{STORAGE: storage}
}

func request(router http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func digestOf(data []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

func withTempRegistryPaths(t *testing.T) {
	t.Helper()

	oldDataPath := config.DATA_PATH
	oldManifestPath := config.MANIFEST_PATH
	oldBlobsPath := config.BLOBS_PATH
	oldTmpPath := config.TMP_PATH

	config.DATA_PATH = t.TempDir()
	config.MANIFEST_PATH = filepath.Join(config.DATA_PATH, "manifests")
	config.BLOBS_PATH = filepath.Join(config.DATA_PATH, "blobs")
	config.TMP_PATH = filepath.Join(config.DATA_PATH, "tmp")

	if err := os.MkdirAll(config.TMP_PATH, 0755); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		config.DATA_PATH = oldDataPath
		config.MANIFEST_PATH = oldManifestPath
		config.BLOBS_PATH = oldBlobsPath
		config.TMP_PATH = oldTmpPath
	})
}

func TestUploadManifestSavesManifest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withTempRegistryPaths(t)

	storage := &fakeStorage{}
	handler := testHandler(storage)
	router := gin.New()
	router.PUT("/v2/:repository/:name/manifests/:reference", handler.uploadManifest)

	body := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1},"layers":[]}`)
	req := httptest.NewRequest(http.MethodPut, "/v2/dev/postgres/manifests/latest", bytes.NewReader(body))
	req.Header.Set("Content-Type", config.MANIFEST_TYPE["manifest"])
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if storage.savedManifestMeta.Repository != "dev" {
		t.Fatalf("repository = %q, want dev", storage.savedManifestMeta.Repository)
	}
	if storage.savedManifestMeta.Image != "postgres" {
		t.Fatalf("image = %q, want postgres", storage.savedManifestMeta.Image)
	}
	if storage.savedManifestMeta.Tag != "latest" {
		t.Fatalf("tag = %q, want latest", storage.savedManifestMeta.Tag)
	}
	if storage.savedManifestMeta.MediaType != config.MANIFEST_TYPE["manifest"] {
		t.Fatalf("media type = %q, want %q", storage.savedManifestMeta.MediaType, config.MANIFEST_TYPE["manifest"])
	}
	if storage.savedManifestMeta.Digest != digestOf(body) {
		t.Fatalf("digest = %q, want %q", storage.savedManifestMeta.Digest, digestOf(body))
	}
	if !bytes.Equal(storage.savedManifestBody, body) {
		t.Fatal("manifest body was not saved unchanged")
	}
}

func TestUploadManifestRejectsDigestMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withTempRegistryPaths(t)

	storage := &fakeStorage{}
	handler := testHandler(storage)
	router := gin.New()
	router.PUT("/v2/:repository/:name/manifests/:reference", handler.uploadManifest)

	body := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)
	rec := request(router, http.MethodPut, "/v2/dev/postgres/manifests/sha256:bad", body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if storage.savedManifestBody != nil {
		t.Fatal("manifest was saved despite digest mismatch")
	}
}

func TestChunkedBlobUploadFinalizesSavedBlob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withTempRegistryPaths(t)

	storage := &fakeStorage{}
	handler := testHandler(storage)
	router := gin.New()
	router.PATCH("/v2/:repository/:name/blobs/uploads/:uuid", handler.uploadBlobPart)
	router.PUT("/v2/:repository/:name/blobs/uploads/:uuid", handler.finalizeBlobUpload)

	body := []byte("chunked blob body")
	patchRec := request(router, http.MethodPatch, "/v2/dev/postgres/blobs/uploads/upload-1", body)
	if patchRec.Code != http.StatusNoContent {
		t.Fatalf("PATCH status = %d, want %d; body: %s", patchRec.Code, http.StatusNoContent, patchRec.Body.String())
	}
	if patchRec.Header().Get("Range") != "0-16" {
		t.Fatalf("range = %q, want 0-16", patchRec.Header().Get("Range"))
	}

	digest := digestOf(body)
	putRec := request(router, http.MethodPut, "/v2/dev/postgres/blobs/uploads/upload-1?digest="+digest, nil)
	if putRec.Code != http.StatusCreated {
		t.Fatalf("PUT status = %d, want %d; body: %s", putRec.Code, http.StatusCreated, putRec.Body.String())
	}
	if storage.savedBlobDigest != digest {
		t.Fatalf("saved digest = %q, want %q", storage.savedBlobDigest, digest)
	}
	if !bytes.Equal(storage.savedBlobBody, body) {
		t.Fatalf("saved body = %q, want %q", storage.savedBlobBody, body)
	}
}

func TestMonolithicBlobUploadFinalizesSavedBlob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withTempRegistryPaths(t)

	storage := &fakeStorage{}
	handler := testHandler(storage)
	router := gin.New()
	router.PUT("/v2/:repository/:name/blobs/uploads/:uuid", handler.finalizeBlobUpload)

	body := []byte("monolithic blob body")
	digest := digestOf(body)
	rec := request(router, http.MethodPut, "/v2/dev/postgres/blobs/uploads/upload-2?digest="+digest, body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if storage.savedBlobDigest != digest {
		t.Fatalf("saved digest = %q, want %q", storage.savedBlobDigest, digest)
	}
	if !bytes.Equal(storage.savedBlobBody, body) {
		t.Fatalf("saved body = %q, want %q", storage.savedBlobBody, body)
	}
}

func TestFinalizeBlobUploadRejectsDigestMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withTempRegistryPaths(t)

	storage := &fakeStorage{}
	handler := testHandler(storage)
	router := gin.New()
	router.PUT("/v2/:repository/:name/blobs/uploads/:uuid", handler.finalizeBlobUpload)

	rec := request(router, http.MethodPut, "/v2/dev/postgres/blobs/uploads/upload-3?digest=sha256:bad", []byte("blob body"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if storage.savedBlobBody != nil {
		t.Fatal("blob was saved despite digest mismatch")
	}
}
