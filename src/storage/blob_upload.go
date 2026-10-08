package storage

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/sirupsen/logrus"
)

const maxUploadChunks = 10000
const uploadCleanupGrace = 30 * time.Second

type uploadPart struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

// blobUploadState хранится рядом с данными: локально в JSON, в S3 в state.json.
type blobUploadState struct {
	Parts          []uploadPart `json:"parts,omitempty"`
	Size           int64        `json:"size"`
	Chunks         int          `json:"chunks"`
	ExpectedDigest string       `json:"expected_digest,omitempty"`
	ReadyDigest    string       `json:"ready_digest,omitempty"`
}

// uploadBackend выполняет только операции ввода-вывода под блокировкой сессии.
type uploadBackend interface {
	readUploadState(context.Context, string) (blobUploadState, error)
	writeUploadState(context.Context, string, blobUploadState) error
	appendUpload(context.Context, string, *blobUploadState, io.Reader) (int64, error)
	digestUpload(context.Context, string, blobUploadState) (string, error)
	publishUpload(context.Context, string, blobUploadState) (config.Blob, error)
	removeUpload(context.Context, string) error
}

/*
uploadStatus читает подтверждённое состояние под отменяемой блокировкой сессии.
*/
func uploadStatus(ctx context.Context, id string, locks *uploadLockManager, backend uploadBackend) (BlobUploadStatus, error) {
	unlock, err := lockUpload(ctx, id, locks)
	if err != nil {
		return BlobUploadStatus{}, err
	}
	defer unlock()
	state, err := backend.readUploadState(ctx, id)
	if err != nil {
		return BlobUploadStatus{}, err
	}
	phase := "uploading"
	if state.ExpectedDigest != "" {
		phase = "finalizing"
	}
	if state.ReadyDigest != "" {
		phase = "ready"
	}
	return BlobUploadStatus{Offset: state.Size, Phase: phase, Digest: state.ExpectedDigest}, nil
}

/*
lockUpload проверяет UUID и context до ожидания блокировки загрузки.
*/
func lockUpload(ctx context.Context, id string, locks *uploadLockManager) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := localUploadPath(id); err != nil {
		return nil, err
	}
	return locks.LockContext(ctx, id)
}

/*
appendBlobPart применяет общие правила PATCH для всех backend.
*/
func appendBlobPart(ctx context.Context, id string, offset int64, body io.Reader, locks *uploadLockManager, backend uploadBackend) (int64, error) {
	unlock, err := lockUpload(ctx, id, locks)
	if err != nil {
		return 0, err
	}
	defer unlock()
	state, err := backend.readUploadState(ctx, id)
	if err != nil {
		return 0, err
	}
	if state.ExpectedDigest != "" || state.ReadyDigest != "" {
		return state.Size, ErrUploadFinalizing
	}
	if state.Size != offset {
		return state.Size, ErrInvalidOffset
	}
	return backend.appendUpload(ctx, id, &state, body)
}

/*
completeBlob применяет единый жизненный цикл финализации LocalStorage и S3.

Финальное тело и заморозка offset фиксируются вместе. После этого разрешён
только пустой PUT с прежним digest. Ошибка очистки не отменяет публикацию Blob.
*/
func completeBlob(ctx context.Context, id, digest string, body io.Reader, locks *uploadLockManager, backend uploadBackend) (config.Blob, error) {
	var empty config.Blob
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if _, err := parseSHA256Digest(digest); err != nil {
		return empty, err
	}
	unlock, err := lockUpload(ctx, id, locks)
	if err != nil {
		return empty, err
	}
	defer unlock()
	state, err := backend.readUploadState(ctx, id)
	if err != nil {
		return empty, err
	}
	if state.ExpectedDigest != "" || state.ReadyDigest != "" {
		if state.ExpectedDigest != "" && state.ExpectedDigest != digest || state.ReadyDigest != "" && state.ReadyDigest != digest {
			return empty, ErrDigestMismatch
		}
		var probe [1]byte
		if n, err := io.ReadFull(body, probe[:]); n != 0 {
			return empty, ErrUploadFinalizing
		} else if err != nil && !errors.Is(err, io.EOF) {
			return empty, err
		}
	} else {
		state.ExpectedDigest = digest
		if _, err := backend.appendUpload(ctx, id, &state, body); err != nil {
			return empty, err
		}
	}
	if state.ReadyDigest == "" {
		actual, err := backend.digestUpload(ctx, id, state)
		if err != nil {
			return empty, err
		}
		if actual != digest {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), uploadCleanupGrace)
			defer cancel()
			return empty, errors.Join(ErrDigestMismatch, backend.removeUpload(cleanupCtx, id))
		}
		state.ReadyDigest = digest
		if err := backend.writeUploadState(ctx, id, state); err != nil {
			return empty, err
		}
	}
	blob, err := backend.publishUpload(ctx, id, state)
	if err != nil {
		return empty, err
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), uploadCleanupGrace)
	defer cancel()
	if err := backend.removeUpload(cleanupCtx, id); err != nil {
		logrus.WithField("upload_id", id).WithError(err).Warn("Blob опубликован; очистка upload будет повторена планировщиком")
	}
	return blob, nil
}

/*
abortBlob применяет одинаковую проверку UUID, context и блокировку при отмене.
*/
func abortBlob(ctx context.Context, id string, locks *uploadLockManager, backend uploadBackend) error {
	unlock, err := lockUpload(ctx, id, locks)
	if err != nil {
		return err
	}
	defer unlock()
	return backend.removeUpload(ctx, id)
}

/*
validateNextChunk проверяет общий лимит непустых чанков и переполнение offset.
*/
func validateNextChunk(state blobUploadState, size int64) error {
	if size < 0 || size > (1<<63-1)-state.Size || size > 0 && state.Chunks >= maxUploadChunks {
		return ErrInvalidLength
	}
	return nil
}

/*
validateUploadState проверяет общие инварианты сохранённой сессии.
*/
func validateUploadState(state blobUploadState) error {
	if state.Size < 0 || state.Chunks < 0 || state.Chunks > maxUploadChunks {
		return ErrBlobCorrupted
	}
	for _, digest := range []string{state.ExpectedDigest, state.ReadyDigest} {
		if digest != "" {
			if _, err := parseSHA256Digest(digest); err != nil {
				return ErrBlobCorrupted
			}
		}
	}
	if state.ReadyDigest != "" && state.ReadyDigest != state.ExpectedDigest {
		return ErrBlobCorrupted
	}
	return nil
}
