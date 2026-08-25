package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/gin-gonic/gin"
)

type fakeUploadStore struct {
	uploads map[string][]byte
	blobs   map[string][]byte
}

func newFakeUploadStore() *fakeUploadStore {
	return &fakeUploadStore{
		uploads: make(map[string][]byte),
		blobs:   make(map[string][]byte),
	}
}

func (f *fakeUploadStore) StartBlobUpload(
	ctx context.Context,
	uploadID string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.uploads[uploadID] = nil
	return nil
}

func (f *fakeUploadStore) AppendBlobUpload(
	ctx context.Context,
	uploadID string,
	expectedOffset int64,
	body io.Reader,
) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	current, ok := f.uploads[uploadID]
	if !ok {
		return 0, storage.ErrUploadNotFound
	}
	if int64(len(current)) != expectedOffset {
		return int64(len(current)), storage.ErrInvalidOffset
	}

	part, err := io.ReadAll(body)
	if err != nil {
		return int64(len(current)), err
	}
	f.uploads[uploadID] = append(current, part...)
	return int64(len(f.uploads[uploadID])), nil
}

func (f *fakeUploadStore) CompleteBlobUpload(
	ctx context.Context,
	uploadID string,
	expectedDigest string,
	finalBody io.Reader,
) (config.Blob, error) {
	var result config.Blob

	current, ok := f.uploads[uploadID]
	if !ok {
		return result, storage.ErrUploadNotFound
	}
	finalPart, err := io.ReadAll(finalBody)
	if err != nil {
		return result, err
	}
	body := append(append([]byte(nil), current...), finalPart...)
	if testDigest(body) != expectedDigest {
		return result, storage.ErrDigestMismatch
	}

	delete(f.uploads, uploadID)
	f.blobs[expectedDigest] = body
	return config.Blob{
		Digest: expectedDigest,
		Size:   int64(len(body)),
	}, nil
}

func (f *fakeUploadStore) AbortBlobUpload(
	ctx context.Context,
	uploadID string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(f.uploads, uploadID)
	return nil
}

func testDigest(body []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(body))
}

func newUploadRouter(uploadStore storage.BlobUploadStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := &Handler{UPLOADS: uploadStore}
	router := gin.New()
	router.POST("/v2/:repository/:name/blobs/uploads/", handler.startBlobUpload)
	router.PATCH("/v2/:repository/:name/blobs/uploads/:uuid", handler.uploadBlobPart)
	router.PUT("/v2/:repository/:name/blobs/uploads/:uuid", handler.finalizeBlobUpload)
	router.DELETE("/v2/:repository/:name/blobs/uploads/:uuid", handler.abortBlobUpload)
	return router
}

func uploadRequest(
	router http.Handler,
	method string,
	path string,
	body []byte,
	headers map[string]string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func startTestUpload(t *testing.T, router http.Handler) string {
	t.Helper()

	recorder := uploadRequest(
		router,
		http.MethodPost,
		"/v2/dev/image/blobs/uploads/",
		nil,
		nil,
	)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	uploadID := recorder.Header().Get("Docker-Upload-UUID")
	if uploadID == "" {
		t.Fatal("Docker-Upload-UUID is empty")
	}
	return uploadID
}

func TestChunkedBlobUploadHTTPPipeline(t *testing.T) {
	store := newFakeUploadStore()
	router := newUploadRouter(store)
	uploadID := startTestUpload(t, router)

	firstPart := []byte("first-")
	secondPart := []byte("second")
	wantBody := append(append([]byte(nil), firstPart...), secondPart...)

	firstPatch := uploadRequest(
		router,
		http.MethodPatch,
		"/v2/dev/image/blobs/uploads/"+uploadID,
		firstPart,
		map[string]string{"Content-Range": "0-5"},
	)
	if firstPatch.Code != http.StatusNoContent {
		t.Fatalf("first PATCH status = %d, want %d", firstPatch.Code, http.StatusNoContent)
	}

	secondPatch := uploadRequest(
		router,
		http.MethodPatch,
		"/v2/dev/image/blobs/uploads/"+uploadID,
		secondPart,
		map[string]string{"Content-Range": "bytes 6-11"},
	)
	if secondPatch.Code != http.StatusNoContent {
		t.Fatalf("second PATCH status = %d, want %d", secondPatch.Code, http.StatusNoContent)
	}
	if got := secondPatch.Header().Get("Range"); got != "0-11" {
		t.Fatalf("Range = %q, want %q", got, "0-11")
	}

	digest := testDigest(wantBody)
	put := uploadRequest(
		router,
		http.MethodPut,
		"/v2/dev/image/blobs/uploads/"+uploadID+"?digest="+digest,
		nil,
		nil,
	)
	if put.Code != http.StatusCreated {
		t.Fatalf("PUT status = %d, want %d; body: %s", put.Code, http.StatusCreated, put.Body.String())
	}
	if got := put.Header().Get("Docker-Content-Digest"); got != digest {
		t.Fatalf("Docker-Content-Digest = %q, want %q", got, digest)
	}
	if got := string(store.blobs[digest]); got != string(wantBody) {
		t.Fatalf("stored body = %q, want %q", got, wantBody)
	}
}

func TestMonolithicBlobUploadHTTPPipeline(t *testing.T) {
	store := newFakeUploadStore()
	router := newUploadRouter(store)
	uploadID := startTestUpload(t, router)
	body := []byte("monolithic")
	digest := testDigest(body)

	put := uploadRequest(
		router,
		http.MethodPut,
		"/v2/dev/image/blobs/uploads/"+uploadID+"?digest="+digest,
		body,
		nil,
	)
	if put.Code != http.StatusCreated {
		t.Fatalf("PUT status = %d, want %d; body: %s", put.Code, http.StatusCreated, put.Body.String())
	}
	if got := string(store.blobs[digest]); got != string(body) {
		t.Fatalf("stored body = %q, want %q", got, body)
	}
}

func TestBlobUploadPatchRejectsInvalidBodyLength(t *testing.T) {
	tests := []struct {
		name        string
		rangeHeader string
		body        []byte
	}{
		{name: "short", rangeHeader: "0-4", body: []byte("four")},
		{name: "long", rangeHeader: "0-2", body: []byte("four")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newFakeUploadStore()
			router := newUploadRouter(store)
			uploadID := startTestUpload(t, router)

			patch := uploadRequest(
				router,
				http.MethodPatch,
				"/v2/dev/image/blobs/uploads/"+uploadID,
				test.body,
				map[string]string{"Content-Range": test.rangeHeader},
			)
			if patch.Code != http.StatusRequestedRangeNotSatisfiable {
				t.Fatalf("PATCH status = %d, want %d", patch.Code, http.StatusRequestedRangeNotSatisfiable)
			}
			if len(store.uploads[uploadID]) != 0 {
				t.Fatal("invalid PATCH changed upload")
			}
		})
	}
}

func TestAbortBlobUploadHTTPPipeline(t *testing.T) {
	store := newFakeUploadStore()
	router := newUploadRouter(store)
	uploadID := startTestUpload(t, router)

	response := uploadRequest(
		router,
		http.MethodDelete,
		"/v2/dev/image/blobs/uploads/"+uploadID,
		nil,
		nil,
	)
	if response.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if _, ok := store.uploads[uploadID]; ok {
		t.Fatal("upload still exists after DELETE")
	}
}

func TestLocalStorageBlobUploadHTTPEndToEnd(t *testing.T) {
	oldDataPath := config.DATA_PATH
	oldManifestPath := config.MANIFEST_PATH
	oldBlobsPath := config.BLOBS_PATH
	oldTmpPath := config.TMP_PATH

	config.DATA_PATH = t.TempDir()
	config.MANIFEST_PATH = filepath.Join(config.DATA_PATH, "manifests")
	config.BLOBS_PATH = filepath.Join(config.DATA_PATH, "blobs")
	config.TMP_PATH = filepath.Join(config.DATA_PATH, "tmp")
	for _, path := range []string{
		config.MANIFEST_PATH,
		config.BLOBS_PATH,
		config.TMP_PATH,
	} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		config.DATA_PATH = oldDataPath
		config.MANIFEST_PATH = oldManifestPath
		config.BLOBS_PATH = oldBlobsPath
		config.TMP_PATH = oldTmpPath
	})

	localStore := &storage.LocalStorage{}
	router := newUploadRouter(localStore)
	uploadID := startTestUpload(t, router)
	body := []byte("real local storage blob")

	patch := uploadRequest(
		router,
		http.MethodPatch,
		"/v2/dev/image/blobs/uploads/"+uploadID,
		body,
		map[string]string{
			"Content-Range": fmt.Sprintf("0-%d", len(body)-1),
		},
	)
	if patch.Code != http.StatusNoContent {
		t.Fatalf("PATCH status = %d, want %d; body: %s", patch.Code, http.StatusNoContent, patch.Body.String())
	}

	digest := testDigest(body)
	put := uploadRequest(
		router,
		http.MethodPut,
		"/v2/dev/image/blobs/uploads/"+uploadID+"?digest="+digest,
		nil,
		nil,
	)
	if put.Code != http.StatusCreated {
		t.Fatalf("PUT status = %d, want %d; body: %s", put.Code, http.StatusCreated, put.Body.String())
	}

	storedBody, err := os.ReadFile(filepath.Join(
		config.BLOBS_PATH,
		strings.TrimPrefix(digest, "sha256:"),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(storedBody, body) {
		t.Fatalf("stored body = %q, want %q", storedBody, body)
	}
	if _, err := os.Stat(filepath.Join(config.TMP_PATH, uploadID)); !os.IsNotExist(err) {
		t.Fatalf("staging upload still exists: %v", err)
	}

	invalidUploadID := startTestUpload(t, router)
	invalidPatch := uploadRequest(
		router,
		http.MethodPatch,
		"/v2/dev/image/blobs/uploads/"+invalidUploadID,
		[]byte("four"),
		map[string]string{"Content-Range": "0-4"},
	)
	if invalidPatch.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf(
			"invalid PATCH status = %d, want %d",
			invalidPatch.Code,
			http.StatusRequestedRangeNotSatisfiable,
		)
	}
	info, err := os.Stat(filepath.Join(config.TMP_PATH, invalidUploadID))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Fatalf("invalid PATCH changed upload size to %d", info.Size())
	}
}
