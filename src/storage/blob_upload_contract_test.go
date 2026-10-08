package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
)

type uploadFixture struct {
	store   BlobUploadStore
	backend uploadBackend
	locks   *uploadLockManager
	restart func() BlobUploadStore
}

type faultUploadBackend struct {
	uploadBackend
	digestErr  error
	publishErr error
	removeErr  error
	appendErr  error
}

func (b faultUploadBackend) appendUpload(ctx context.Context, id string, state *blobUploadState, body io.Reader) (int64, error) {
	oldSize := state.Size
	offset, err := b.uploadBackend.appendUpload(ctx, id, state, body)
	if err == nil && b.appendErr != nil {
		return oldSize, b.appendErr
	} // Запись успешна, подтверждение потеряно.
	return offset, err
}

func (b faultUploadBackend) digestUpload(ctx context.Context, id string, state blobUploadState) (string, error) {
	if b.digestErr != nil {
		return "", b.digestErr
	}
	return b.uploadBackend.digestUpload(ctx, id, state)
}

func (b faultUploadBackend) publishUpload(ctx context.Context, id string, state blobUploadState) (config.Blob, error) {
	if b.publishErr != nil {
		return config.Blob{}, b.publishErr
	}
	return b.uploadBackend.publishUpload(ctx, id, state)
}

func (b faultUploadBackend) removeUpload(ctx context.Context, id string) error {
	if b.removeErr != nil {
		return b.removeErr
	}
	return b.uploadBackend.removeUpload(ctx, id)
}

/*
TestBlobUploadContract запускает одинаковые сценарии для LocalStorage и S3.
*/
func TestBlobUploadContract(t *testing.T) {
	factories := map[string]func(*testing.T) uploadFixture{
		"local": func(t *testing.T) uploadFixture {
			withTempStoragePaths(t)
			s := &LocalStorage{}
			return uploadFixture{s, s, &s.uploadLocks, func() BlobUploadStore { return &LocalStorage{} }}
		},
		"s3": func(t *testing.T) uploadFixture {
			s := newTestS3Storage(t)
			return uploadFixture{s, s, &s.uploadLocks, func() BlobUploadStore { return &S3Storage{S3: s.S3, Bucket: s.Bucket, Prefix: s.Prefix} }}
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			for _, test := range []struct {
				name string
				run  func(*testing.T, uploadFixture, string)
			}{
				{"append-and-restart", contractAppend},
				{"lost-append-response", contractLostResponse},
				{"final-body-error", contractFinalBodyError},
				{"verify-retry", contractVerifyRetry},
				{"publish-retry", contractPublishRetry},
				{"cleanup-after-publication", contractCleanupError},
				{"cancel-lock-wait", contractCancelWait},
				{"digest-mismatch-and-abort", contractMismatch},
			} {
				t.Run(test.name, func(t *testing.T) {
					f := factory(t)
					id := uuid.NewV4().String()
					if err := f.store.StartBlobUpload(context.Background(), id); err != nil {
						t.Fatal(err)
					}
					if err := f.store.StartBlobUpload(context.Background(), id); !errors.Is(err, ErrUploadExists) {
						t.Fatalf("duplicate start: %v", err)
					}
					test.run(t, f, id)
				})
			}
		})
	}
}

/*
assertUploadStatus проверяет фазу и подтверждённый offset общего контракта.
*/
func assertUploadStatus(t *testing.T, store BlobUploadStore, id, phase string, offset int64) {
	t.Helper()
	status, err := store.GetBlobUpload(context.Background(), id)
	if err != nil || status.Phase != phase || status.Offset != offset {
		t.Fatalf("status=%+v err=%v; want %s/%d", status, err, phase, offset)
	}
}

/*
contractAppend проверяет offset, откат неполной части и продолжение после перезапуска.
*/
func contractAppend(t *testing.T, f uploadFixture, id string) {
	ctx := context.Background()
	assertUploadStatus(t, f.store, id, "uploading", 0)
	if _, err := f.store.AppendBlobUpload(ctx, id, 2, strings.NewReader("abc")); !errors.Is(err, ErrInvalidOffset) {
		t.Fatal(err)
	}
	injected := errors.New("read failed")
	if _, err := f.store.AppendBlobUpload(ctx, id, 0, io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(injected))); !errors.Is(err, injected) {
		t.Fatal(err)
	}
	assertUploadStatus(t, f.store, id, "uploading", 0)
	if offset, err := f.store.AppendBlobUpload(ctx, id, 0, strings.NewReader("abc")); err != nil || offset != 3 {
		t.Fatalf("offset=%d err=%v", offset, err)
	}
	restarted := f.restart()
	assertUploadStatus(t, restarted, id, "uploading", 3)
	if _, err := restarted.CompleteBlobUpload(ctx, id, testBlobDigest([]byte("abcdef")), strings.NewReader("def")); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.GetBlobUpload(ctx, id); !errors.Is(err, ErrUploadNotFound) {
		t.Fatal(err)
	}
}

/*
contractLostResponse восстанавливает offset из state, а не из ошибочного ответа PATCH.
*/
func contractLostResponse(t *testing.T, f uploadFixture, id string) {
	ctx := context.Background()
	injected := errors.New("response lost after commit")
	backend := faultUploadBackend{uploadBackend: f.backend, appendErr: injected}
	if offset, err := appendBlobPart(ctx, id, 0, strings.NewReader("abc"), f.locks, backend); !errors.Is(err, injected) || offset != 0 {
		t.Fatalf("offset=%d err=%v", offset, err)
	}
	restarted := f.restart()
	assertUploadStatus(t, restarted, id, "uploading", 3)
	if _, err := restarted.AppendBlobUpload(ctx, id, 0, strings.NewReader("abc")); !errors.Is(err, ErrInvalidOffset) {
		t.Fatal(err)
	}
	if offset, err := restarted.AppendBlobUpload(ctx, id, 3, strings.NewReader("def")); err != nil || offset != 6 {
		t.Fatalf("offset=%d err=%v", offset, err)
	}
	if blob, err := restarted.CompleteBlobUpload(ctx, id, testBlobDigest([]byte("abcdef")), strings.NewReader("")); err != nil || blob.Size != 6 {
		t.Fatalf("blob=%+v err=%v", blob, err)
	}
}

/*
contractFinalBodyError не допускает заморозки сессии при неполном финальном теле.
*/
func contractFinalBodyError(t *testing.T, f uploadFixture, id string) {
	ctx := context.Background()
	injected := errors.New("final body failed")
	digest := testBlobDigest([]byte("abc"))
	if _, err := f.store.CompleteBlobUpload(ctx, id, digest, io.MultiReader(strings.NewReader("ab"), iotest.ErrReader(injected))); !errors.Is(err, injected) {
		t.Fatal(err)
	}
	assertUploadStatus(t, f.restart(), id, "uploading", 0)
	if _, err := f.store.CompleteBlobUpload(ctx, id, digest, strings.NewReader("abc")); err != nil {
		t.Fatal(err)
	}
}

/*
contractVerifyRetry проверяет заморозку тела до проверки SHA-256.
*/
func contractVerifyRetry(t *testing.T, f uploadFixture, id string) {
	contractFrozenRetry(t, f, id, false)
}

/*
contractPublishRetry проверяет восстановление проверенной сессии после ошибки публикации.
*/
func contractPublishRetry(t *testing.T, f uploadFixture, id string) {
	contractFrozenRetry(t, f, id, true)
}

/*
contractFrozenRetry запрещает PATCH и повторное финальное тело для обеих фаз.
*/
func contractFrozenRetry(t *testing.T, f uploadFixture, id string, ready bool) {
	ctx := context.Background()
	injected := errors.New("temporary backend failure")
	backend := faultUploadBackend{uploadBackend: f.backend, digestErr: injected}
	phase := "finalizing"
	if ready {
		backend.digestErr = nil
		backend.publishErr = injected
		phase = "ready"
	}
	digest := testBlobDigest([]byte("abc"))
	if _, err := completeBlob(ctx, id, digest, strings.NewReader("abc"), f.locks, backend); !errors.Is(err, injected) {
		t.Fatal(err)
	}
	restarted := f.restart()
	assertUploadStatus(t, restarted, id, phase, 3)
	if _, err := restarted.AppendBlobUpload(ctx, id, 3, strings.NewReader("extra")); !errors.Is(err, ErrUploadFinalizing) {
		t.Fatal(err)
	}
	if _, err := restarted.CompleteBlobUpload(ctx, id, digest, strings.NewReader("abc")); !errors.Is(err, ErrUploadFinalizing) {
		t.Fatal(err)
	}
	if _, err := restarted.CompleteBlobUpload(ctx, id, testBlobDigest([]byte("other")), strings.NewReader("")); !errors.Is(err, ErrDigestMismatch) {
		t.Fatal(err)
	}
	assertUploadStatus(t, restarted, id, phase, 3)
	if blob, err := restarted.CompleteBlobUpload(ctx, id, digest, strings.NewReader("")); err != nil || blob.Size != 3 {
		t.Fatalf("blob=%+v err=%v", blob, err)
	}
}

/*
contractCleanupError подтверждает Blob независимо от ошибки удаления временных данных.
*/
func contractCleanupError(t *testing.T, f uploadFixture, id string) {
	ctx := context.Background()
	backend := faultUploadBackend{uploadBackend: f.backend, removeErr: errors.New("cleanup failed")}
	digest := testBlobDigest([]byte("abc"))
	if blob, err := completeBlob(ctx, id, digest, strings.NewReader("abc"), f.locks, backend); err != nil || blob.Size != 3 {
		t.Fatalf("blob=%+v err=%v", blob, err)
	}
	restarted := f.restart()
	assertUploadStatus(t, restarted, id, "ready", 3)
	if _, err := restarted.CompleteBlobUpload(ctx, id, digest, strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
}

/*
contractCancelWait проверяет отмену операции до освобождения занятого mutex.
*/
func contractCancelWait(t *testing.T, f uploadFixture, id string) {
	unlock := f.locks.Lock(id)
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := f.store.GetBlobUpload(ctx, id); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

/*
contractMismatch проверяет удаление неверной загрузки и идемпотентную отмену.
*/
func contractMismatch(t *testing.T, f uploadFixture, id string) {
	ctx := context.Background()
	if _, err := f.store.CompleteBlobUpload(ctx, id, testBlobDigest([]byte("wrong")), strings.NewReader("abc")); !errors.Is(err, ErrDigestMismatch) {
		t.Fatal(err)
	}
	if _, err := f.store.GetBlobUpload(ctx, id); !errors.Is(err, ErrUploadNotFound) {
		t.Fatal(err)
	}
	for range 2 {
		if err := f.store.AbortBlobUpload(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
}
