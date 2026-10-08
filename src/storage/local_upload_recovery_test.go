package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
)

func TestLocalUploadRecoversUncommittedTail(t *testing.T) {
	withTempStoragePaths(t)
	s := &LocalStorage{}
	ctx := context.Background()
	id := uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendBlobUpload(ctx, id, 0, strings.NewReader("abc")); err != nil {
		t.Fatal(err)
	}
	path, _ := localUploadPath(id)
	// Имитируем остановку после записи данных, но до замены JSON.
	if err := os.WriteFile(path, []byte("abc-uncommitted"), 0600); err != nil {
		t.Fatal(err)
	}
	restarted := &LocalStorage{}
	assertUploadStatus(t, restarted, id, "uploading", 3)
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "abc" {
		t.Fatalf("data=%q err=%v", data, err)
	}
	if _, err := restarted.CompleteBlobUpload(ctx, id, testBlobDigest(data), strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(config.TMP_PATH)
	if err != nil || len(entries) != 0 {
		t.Fatalf("remaining files=%d err=%v", len(entries), err)
	}
}

func TestLocalUploadMigratesLegacySession(t *testing.T) {
	withTempStoragePaths(t)
	id := uuid.NewV4().String()
	path, _ := localUploadPath(id)
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &LocalStorage{}
	assertUploadStatus(t, s, id, "uploading", 3)
	if _, err := os.Stat(path + ".json"); err != nil {
		t.Fatalf("legacy metadata: %v", err)
	}
	if _, err := s.CompleteBlobUpload(context.Background(), id, testBlobDigest([]byte("abcdef")), strings.NewReader("def")); err != nil {
		t.Fatal(err)
	}
}

func TestLocalCleanupUsesFreshestSessionFile(t *testing.T) {
	withTempStoragePaths(t)
	s := &LocalStorage{}
	ctx := context.Background()
	id := uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	path, _ := localUploadPath(id)
	old := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if count, err := s.CleanupUploads(ctx, 24*time.Hour); err != nil || count != 0 {
		t.Fatalf("fresh metadata: deleted=%d err=%v", count, err)
	}
	if err := os.Chtimes(path+".json", old, old); err != nil {
		t.Fatal(err)
	}
	if count, err := s.CleanupUploads(ctx, 24*time.Hour); err != nil || count != 1 {
		t.Fatalf("old session: deleted=%d err=%v", count, err)
	}
	// Остаток atomic state write также принадлежит сессии, даже без файла данных.
	orphan := filepath.Join(config.TMP_PATH, uuid.NewV4().String()+".json-orphan")
	if err := os.WriteFile(orphan, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(orphan, old, old); err != nil {
		t.Fatal(err)
	}
	if count, err := s.CleanupUploads(ctx, 24*time.Hour); err != nil || count != 1 {
		t.Fatalf("orphan metadata: deleted=%d err=%v", count, err)
	}
}

func TestUploadChunkLimitAllowsEmptyFinalBody(t *testing.T) {
	state := blobUploadState{Size: 1, Chunks: maxUploadChunks}
	if err := validateNextChunk(state, 1); err != ErrInvalidLength {
		t.Fatal(err)
	}
	if err := validateNextChunk(state, 0); err != nil {
		t.Fatal(err)
	}
}
