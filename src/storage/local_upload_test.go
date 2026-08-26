package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
)

func testBlobDigest(body []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(body))
}

func TestLocalStorageChunkedBlobUpload(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	uploadID := uuid.NewV4().String()
	firstPart := []byte("first-")
	secondPart := []byte("second")
	wantBody := append(append([]byte(nil), firstPart...), secondPart...)

	if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
		t.Fatal(err)
	}
	offset, err := store.AppendBlobUpload(
		context.Background(),
		uploadID,
		0,
		strings.NewReader(string(firstPart)),
	)
	if err != nil {
		t.Fatal(err)
	}
	offset, err = store.AppendBlobUpload(
		context.Background(),
		uploadID,
		offset,
		strings.NewReader(string(secondPart)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if offset != int64(len(wantBody)) {
		t.Fatalf("offset = %d, want %d", offset, len(wantBody))
	}

	digest := testBlobDigest(wantBody)
	blob, err := store.CompleteBlobUpload(
		context.Background(),
		uploadID,
		digest,
		strings.NewReader(""),
	)
	if err != nil {
		t.Fatal(err)
	}

	assertStoredBlob(t, uploadID, blob, digest, wantBody)
}

func TestLocalStorageMonolithicBlobUpload(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	uploadID := uuid.NewV4().String()
	body := []byte("monolithic blob")
	digest := testBlobDigest(body)

	if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
		t.Fatal(err)
	}
	blob, err := store.CompleteBlobUpload(
		context.Background(),
		uploadID,
		digest,
		strings.NewReader(string(body)),
	)
	if err != nil {
		t.Fatal(err)
	}

	assertStoredBlob(t, uploadID, blob, digest, body)
}

func TestLocalStorageAppendRejectsWrongOffsetWithoutChangingUpload(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	uploadID := uuid.NewV4().String()
	if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendBlobUpload(
		context.Background(),
		uploadID,
		1,
		strings.NewReader("data"),
	); !errors.Is(err, ErrInvalidOffset) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidOffset)
	}

	info, err := os.Stat(filepath.Join(config.TMP_PATH, uploadID))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Fatalf("upload size = %d, want 0", info.Size())
	}
}

func TestLocalStorageDigestMismatchRemovesUpload(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	uploadID := uuid.NewV4().String()
	if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
		t.Fatal(err)
	}
	_, err := store.CompleteBlobUpload(
		context.Background(),
		uploadID,
		"sha256:"+strings.Repeat("a", 64),
		strings.NewReader("body"),
	)
	if !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("error = %v, want %v", err, ErrDigestMismatch)
	}
	if fileExists(filepath.Join(config.TMP_PATH, uploadID)) {
		t.Fatal("upload was not removed after digest mismatch")
	}
}

func TestLocalStorageDeduplicatesValidBlob(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	body := []byte("same blob")
	digest := testBlobDigest(body)

	for i := 0; i < 2; i++ {
		uploadID := uuid.NewV4().String()
		if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
			t.Fatal(err)
		}
		blob, err := store.CompleteBlobUpload(
			context.Background(),
			uploadID,
			digest,
			strings.NewReader(string(body)),
		)
		if err != nil {
			t.Fatal(err)
		}
		assertStoredBlob(t, uploadID, blob, digest, body)
	}
}

func TestLocalStorageRejectsCorruptedDeduplicatedBlob(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	body := []byte("expected")
	digest := testBlobDigest(body)
	encodedDigest, err := parseSHA256Digest(digest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(config.BLOBS_PATH, encodedDigest),
		[]byte("corrupt!"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	uploadID := uuid.NewV4().String()
	if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
		t.Fatal(err)
	}
	_, err = store.CompleteBlobUpload(
		context.Background(),
		uploadID,
		digest,
		strings.NewReader(string(body)),
	)
	if !errors.Is(err, ErrBlobCorrupted) {
		t.Fatalf("error = %v, want %v", err, ErrBlobCorrupted)
	}
}

func TestParseSHA256DigestRejectsUppercase(t *testing.T) {
	_, err := parseSHA256Digest("sha256:" + strings.Repeat("A", 64))
	if !errors.Is(err, ErrInvalidDigest) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidDigest)
	}
}

func assertStoredBlob(
	t *testing.T,
	uploadID string,
	blob config.Blob,
	digest string,
	body []byte,
) {
	t.Helper()

	if blob.Digest != digest {
		t.Fatalf("digest = %q, want %q", blob.Digest, digest)
	}
	if blob.Size != int64(len(body)) {
		t.Fatalf("size = %d, want %d", blob.Size, len(body))
	}
	if fileExists(filepath.Join(config.TMP_PATH, uploadID)) {
		t.Fatal("staging upload still exists")
	}

	storedBody, err := os.ReadFile(blob.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(storedBody) != string(body) {
		t.Fatalf("stored body = %q, want %q", storedBody, body)
	}
}
