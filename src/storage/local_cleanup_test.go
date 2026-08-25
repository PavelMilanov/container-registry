package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
)

func TestCleanupUploadsDeletesOnlyExpiredUUIDFiles(t *testing.T) {
	withTempStoragePaths(t)

	oldUpload := filepath.Join(config.TMP_PATH, uuid.NewV4().String())
	newUpload := filepath.Join(config.TMP_PATH, uuid.NewV4().String())
	nonUpload := filepath.Join(config.TMP_PATH, "keep-me")
	uploadDirectory := filepath.Join(config.TMP_PATH, uuid.NewV4().String())

	writeFile(t, oldUpload, "old")
	writeFile(t, newUpload, "new")
	writeFile(t, nonUpload, "not an upload")
	if err := os.Mkdir(uploadDirectory, 0700); err != nil {
		t.Fatal(err)
	}

	oldTime := time.Now().Add(-25 * time.Hour)
	for _, path := range []string{oldUpload, nonUpload, uploadDirectory} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
	}

	store := &LocalStorage{}
	deleted, err := store.CleanupUploads(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}

	if fileExists(oldUpload) {
		t.Fatal("expired upload was not deleted")
	}
	for _, path := range []string{newUpload, nonUpload, uploadDirectory} {
		if !fileExists(path) {
			t.Fatalf("cleanup removed %s", path)
		}
	}
}

func TestCleanupUploadsHonorsCanceledContext(t *testing.T) {
	withTempStoragePaths(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := &LocalStorage{}
	deleted, err := store.CleanupUploads(ctx, 24*time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want %v", err, context.Canceled)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}

func TestCleanupUploadsRejectsNonPositiveRetention(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	if _, err := store.CleanupUploads(context.Background(), 0); err == nil {
		t.Fatal("expected error for non-positive retention")
	}
}
