package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateFileDigest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blob")
	if err := os.WriteFile(path, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	digest, err := calculateFileDigest(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}

	const expected = "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if digest != expected {
		t.Fatalf("digest = %q, want %q", digest, expected)
	}
}

func TestCalculateFileDigestHonorsCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blob")
	if err := os.WriteFile(path, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := calculateFileDigest(ctx, path)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want %v", err, context.Canceled)
	}
}
