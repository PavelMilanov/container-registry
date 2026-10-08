package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/PavelMilanov/container-registry/internal/s3test"
	"github.com/PavelMilanov/container-registry/storage"
)

func TestS3BlobUploadHTTPEndToEnd(t *testing.T) {
	client, bucket := s3test.New(t)
	store := &storage.S3Storage{S3: client, Bucket: bucket}
	router := newUploadRouter(store)
	handler := &Handler{BLOBS: store, MANIFESTS: store}
	router.GET("/v2/:repository/:name/blobs/:uuid", handler.getBlob)
	router.HEAD("/v2/:repository/:name/blobs/:uuid", handler.checkBlob)
	router.PUT("/v2/:repository/:name/manifests/:reference", handler.uploadManifest)
	router.GET("/v2/:repository/:name/manifests/:reference", handler.getManifest)
	data := []byte("s3 streamed blob content")
	id := startTestUpload(t, router)
	location := "/v2/dev/image/blobs/uploads/" + id
	// Неполное тело PATCH не должно изменять offset.
	bad := uploadRequest(router, http.MethodPatch, location, []byte("four"), map[string]string{"Content-Range": "0-4"})
	if bad.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("invalid PATCH: %d %s", bad.Code, bad.Body.String())
	}
	for _, offset := range []int{0, 3} {
		end := 3
		if offset == 3 {
			end = len(data)
		}
		patch := uploadRequest(router, http.MethodPatch, location, data[offset:end], map[string]string{"Content-Range": fmt.Sprintf("%d-%d", offset, end-1)})
		if patch.Code != http.StatusNoContent {
			t.Fatalf("PATCH: %d %s", patch.Code, patch.Body.String())
		}
	}
	digest := testDigest(data)
	put := uploadRequest(router, http.MethodPut, location+"?digest="+digest, nil, nil)
	if put.Code != http.StatusCreated {
		t.Fatalf("PUT: %d %s", put.Code, put.Body.String())
	}
	blobURL := put.Header().Get("Location")
	get := uploadRequest(router, http.MethodGet, blobURL, nil, nil)
	if get.Code != http.StatusOK || get.Body.String() != string(data) || get.Header().Get("Docker-Content-Digest") != digest {
		t.Fatalf("GET: %d %s", get.Code, get.Body.String())
	}
	partial := uploadRequest(router, http.MethodGet, blobURL, nil, map[string]string{"Range": "bytes=3-6"})
	if partial.Code != http.StatusPartialContent || partial.Body.String() != string(data[3:7]) || partial.Header().Get("Content-Length") != "4" {
		t.Fatalf("Range: %d %s headers=%v", partial.Code, partial.Body.String(), partial.Header())
	}
	head := uploadRequest(router, http.MethodHead, blobURL, nil, nil)
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD: %d", head.Code)
	}
	missing := uploadRequest(router, http.MethodGet, "/v2/dev/image/blobs/"+testDigest([]byte("missing")), nil, nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing GET: %d", missing.Code)
	}
	manifest := []byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"digest":%q},"layers":[{"digest":%q}]}`, digest, digest))
	manifestURL := "/v2/dev/image/manifests/latest"
	manifestPut := uploadRequest(router, http.MethodPut, manifestURL, manifest, map[string]string{"Content-Type": "application/vnd.oci.image.manifest.v1+json"})
	if manifestPut.Code != http.StatusCreated {
		t.Fatalf("manifest PUT: %d %s", manifestPut.Code, manifestPut.Body.String())
	}
	manifestGet := uploadRequest(router, http.MethodGet, manifestURL, nil, nil)
	if manifestGet.Code != http.StatusOK || manifestGet.Body.String() != string(manifest) {
		t.Fatalf("manifest GET: %d %s", manifestGet.Code, manifestGet.Body.String())
	}
	id = startTestUpload(t, router)
	abort := uploadRequest(router, http.MethodDelete, "/v2/dev/image/blobs/uploads/"+id, nil, nil)
	if abort.Code != http.StatusNoContent {
		t.Fatalf("DELETE: %d", abort.Code)
	}
}
