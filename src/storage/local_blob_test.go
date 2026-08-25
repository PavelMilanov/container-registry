package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func TestLocalStorageBlobLookup(t *testing.T) {
	withTempStoragePaths(t)

	encodedDigest := strings.Repeat("a", 64)
	digest := "sha256:" + encodedDigest
	blobPath := filepath.Join(config.BLOBS_PATH, encodedDigest)
	if err := os.WriteFile(blobPath, []byte("blob"), 0600); err != nil {
		t.Fatal(err)
	}

	store := &LocalStorage{}
	if err := store.CheckBlob(digest); err != nil {
		t.Fatal(err)
	}

	blob, err := store.GetBlob(digest)
	if err != nil {
		t.Fatal(err)
	}
	if blob.Digest != digest {
		t.Fatalf("digest = %q, want %q", blob.Digest, digest)
	}
	if blob.Path != blobPath {
		t.Fatalf("path = %q, want %q", blob.Path, blobPath)
	}
	if blob.Size != 4 {
		t.Fatalf("size = %d, want 4", blob.Size)
	}
}

func TestLocalStorageBlobLookupRejectsInvalidDigest(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	if err := store.CheckBlob("invalid"); !errors.Is(err, ErrInvalidDigest) {
		t.Fatalf("CheckBlob error = %v, want %v", err, ErrInvalidDigest)
	}
	if _, err := store.GetBlob("invalid"); !errors.Is(err, ErrInvalidDigest) {
		t.Fatalf("GetBlob error = %v, want %v", err, ErrInvalidDigest)
	}
}

func TestLocalStorageBlobLookupReturnsNotFound(t *testing.T) {
	withTempStoragePaths(t)

	digest := "sha256:" + strings.Repeat("b", 64)
	store := &LocalStorage{}

	if err := store.CheckBlob(digest); !errors.Is(err, ErrBlobNotFound) {
		t.Fatalf("CheckBlob error = %v, want %v", err, ErrBlobNotFound)
	}
	if _, err := store.GetBlob(digest); !errors.Is(err, ErrBlobNotFound) {
		t.Fatalf("GetBlob error = %v, want %v", err, ErrBlobNotFound)
	}
}
