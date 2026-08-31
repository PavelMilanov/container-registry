package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
)

type blockingReader struct {
	started chan struct{}
	release <-chan struct{}
	body    []byte
	once    sync.Once
}

/*
Read сообщает о начале чтения и ожидает разрешения продолжить запись.
*/
func (r *blockingReader) Read(buffer []byte) (int, error) {
	r.once.Do(func() {
		close(r.started)
		<-r.release
	})
	if len(r.body) == 0 {
		return 0, io.EOF
	}

	read := copy(buffer, r.body)
	r.body = r.body[read:]
	return read, nil
}

type notifyingReader struct {
	started chan struct{}
	reader  io.Reader
	once    sync.Once
}

/*
Read сообщает о первом обращении и передаёт чтение исходному потоку.
*/
func (r *notifyingReader) Read(buffer []byte) (int, error) {
	r.once.Do(func() {
		close(r.started)
	})
	return r.reader.Read(buffer)
}

type appendResult struct {
	offset int64
	err    error
}

/*
waitSignal ожидает тестовый сигнал с ограничением времени.
*/
func waitSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for %s", name)
	}
}

/*
waitAppendResult ожидает результат AppendBlobUpload.
*/
func waitAppendResult(
	t *testing.T,
	result <-chan appendResult,
	name string,
) appendResult {
	t.Helper()

	select {
	case value := <-result:
		return value
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for %s", name)
		return appendResult{}
	}
}

func TestLocalStorageSerializesAppendForSameUpload(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	uploadID := uuid.NewV4().String()
	if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
		t.Fatal(err)
	}

	releaseFirst := make(chan struct{})
	firstReadStarted := make(chan struct{})
	firstResult := make(chan appendResult, 1)
	go func() {
		offset, err := store.AppendBlobUpload(
			context.Background(),
			uploadID,
			0,
			&blockingReader{
				started: firstReadStarted,
				release: releaseFirst,
				body:    []byte("first"),
			},
		)
		firstResult <- appendResult{offset: offset, err: err}
	}()
	waitSignal(t, firstReadStarted, "first append")

	secondReadStarted := make(chan struct{})
	secondResult := make(chan appendResult, 1)
	go func() {
		offset, err := store.AppendBlobUpload(
			context.Background(),
			uploadID,
			0,
			&notifyingReader{
				started: secondReadStarted,
				reader:  strings.NewReader("second"),
			},
		)
		secondResult <- appendResult{offset: offset, err: err}
	}()

	select {
	case <-secondReadStarted:
		close(releaseFirst)
		_ = waitAppendResult(t, firstResult, "first append result")
		t.Fatal("second append started reading before first append completed")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseFirst)
	first := waitAppendResult(t, firstResult, "first append result")
	if first.err != nil || first.offset != int64(len("first")) {
		t.Fatalf("first append = (%d, %v)", first.offset, first.err)
	}
	second := waitAppendResult(t, secondResult, "second append result")
	if !errors.Is(second.err, ErrInvalidOffset) {
		t.Fatalf("second error = %v, want %v", second.err, ErrInvalidOffset)
	}
	if second.offset != int64(len("first")) {
		t.Fatalf("second offset = %d, want %d", second.offset, len("first"))
	}
}

func TestLocalStorageAllowsParallelDifferentUploads(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	firstUploadID := uuid.NewV4().String()
	secondUploadID := uuid.NewV4().String()
	for _, uploadID := range []string{firstUploadID, secondUploadID} {
		if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
			t.Fatal(err)
		}
	}

	releaseFirst := make(chan struct{})
	firstReadStarted := make(chan struct{})
	firstResult := make(chan appendResult, 1)
	go func() {
		offset, err := store.AppendBlobUpload(
			context.Background(),
			firstUploadID,
			0,
			&blockingReader{
				started: firstReadStarted,
				release: releaseFirst,
				body:    []byte("first"),
			},
		)
		firstResult <- appendResult{offset: offset, err: err}
	}()
	waitSignal(t, firstReadStarted, "first upload")

	secondResult := make(chan appendResult, 1)
	go func() {
		offset, err := store.AppendBlobUpload(
			context.Background(),
			secondUploadID,
			0,
			strings.NewReader("second"),
		)
		secondResult <- appendResult{offset: offset, err: err}
	}()

	second := waitAppendResult(t, secondResult, "second upload")
	if second.err != nil || second.offset != int64(len("second")) {
		close(releaseFirst)
		_ = waitAppendResult(t, firstResult, "first upload result")
		t.Fatalf("second append = (%d, %v)", second.offset, second.err)
	}

	close(releaseFirst)
	first := waitAppendResult(t, firstResult, "first upload result")
	if first.err != nil {
		t.Fatal(first.err)
	}
}

func TestLocalStorageCompleteWaitsForAppend(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	uploadID := uuid.NewV4().String()
	if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
		t.Fatal(err)
	}

	releaseAppend := make(chan struct{})
	appendStarted := make(chan struct{})
	appendDone := make(chan appendResult, 1)
	go func() {
		offset, err := store.AppendBlobUpload(
			context.Background(),
			uploadID,
			0,
			&blockingReader{
				started: appendStarted,
				release: releaseAppend,
				body:    []byte("body"),
			},
		)
		appendDone <- appendResult{offset: offset, err: err}
	}()
	waitSignal(t, appendStarted, "append")

	type completeResult struct {
		err error
	}
	completeDone := make(chan completeResult, 1)
	go func() {
		_, err := store.CompleteBlobUpload(
			context.Background(),
			uploadID,
			testBlobDigest([]byte("body")),
			strings.NewReader(""),
		)
		completeDone <- completeResult{err: err}
	}()

	select {
	case result := <-completeDone:
		close(releaseAppend)
		_ = waitAppendResult(t, appendDone, "append result")
		t.Fatalf("complete finished during append: %v", result.err)
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseAppend)
	if result := waitAppendResult(t, appendDone, "append result"); result.err != nil {
		t.Fatal(result.err)
	}
	select {
	case result := <-completeDone:
		if result.err != nil {
			t.Fatal(result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for complete")
	}
}

func TestLocalStorageAbortWaitsForAppend(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	uploadID := uuid.NewV4().String()
	if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
		t.Fatal(err)
	}

	releaseAppend := make(chan struct{})
	appendStarted := make(chan struct{})
	appendDone := make(chan appendResult, 1)
	go func() {
		offset, err := store.AppendBlobUpload(
			context.Background(),
			uploadID,
			0,
			&blockingReader{
				started: appendStarted,
				release: releaseAppend,
				body:    []byte("body"),
			},
		)
		appendDone <- appendResult{offset: offset, err: err}
	}()
	waitSignal(t, appendStarted, "append")

	abortDone := make(chan error, 1)
	go func() {
		abortDone <- store.AbortBlobUpload(context.Background(), uploadID)
	}()

	select {
	case err := <-abortDone:
		close(releaseAppend)
		_ = waitAppendResult(t, appendDone, "append result")
		t.Fatalf("abort finished during append: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseAppend)
	if result := waitAppendResult(t, appendDone, "append result"); result.err != nil {
		t.Fatal(result.err)
	}
	select {
	case err := <-abortDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for abort")
	}
	if fileExists(filepath.Join(config.TMP_PATH, uploadID)) {
		t.Fatal("upload still exists after abort")
	}
}

func TestCleanupUploadsWaitsForActiveAppend(t *testing.T) {
	withTempStoragePaths(t)

	store := &LocalStorage{}
	uploadID := uuid.NewV4().String()
	if err := store.StartBlobUpload(context.Background(), uploadID); err != nil {
		t.Fatal(err)
	}
	uploadPath := filepath.Join(config.TMP_PATH, uploadID)
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(uploadPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	releaseAppend := make(chan struct{})
	appendStarted := make(chan struct{})
	appendDone := make(chan appendResult, 1)
	go func() {
		offset, err := store.AppendBlobUpload(
			context.Background(),
			uploadID,
			0,
			&blockingReader{
				started: appendStarted,
				release: releaseAppend,
				body:    []byte("body"),
			},
		)
		appendDone <- appendResult{offset: offset, err: err}
	}()
	waitSignal(t, appendStarted, "append")

	type cleanupResult struct {
		deleted int
		err     error
	}
	cleanupDone := make(chan cleanupResult, 1)
	go func() {
		deleted, err := store.CleanupUploads(
			context.Background(),
			24*time.Hour,
		)
		cleanupDone <- cleanupResult{deleted: deleted, err: err}
	}()

	select {
	case result := <-cleanupDone:
		close(releaseAppend)
		_ = waitAppendResult(t, appendDone, "append result")
		t.Fatalf("cleanup finished during append: (%d, %v)", result.deleted, result.err)
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseAppend)
	if result := waitAppendResult(t, appendDone, "append result"); result.err != nil {
		t.Fatal(result.err)
	}
	select {
	case result := <-cleanupDone:
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.deleted != 0 {
			t.Fatalf("deleted = %d, want 0", result.deleted)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for cleanup")
	}
	if !fileExists(uploadPath) {
		t.Fatal("cleanup removed active upload")
	}
}

func TestUploadLockManagerRemovesUnusedLocks(t *testing.T) {
	var manager uploadLockManager

	unlock := manager.Lock("upload")
	manager.mu.Lock()
	locksWhileUsed := len(manager.locks)
	manager.mu.Unlock()
	if locksWhileUsed != 1 {
		t.Fatalf("locks while used = %d, want 1", locksWhileUsed)
	}

	unlock()
	manager.mu.Lock()
	locksAfterUnlock := len(manager.locks)
	manager.mu.Unlock()
	if locksAfterUnlock != 0 {
		t.Fatalf("locks after unlock = %d, want 0", locksAfterUnlock)
	}
}
