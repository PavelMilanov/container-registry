package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/PavelMilanov/container-registry/config"
)

/*
StartBlobUpload создаёт файл данных и устойчивое состояние локальной сессии.
*/
func (s *LocalStorage) StartBlobUpload(ctx context.Context, id string) error {
	unlock, err := lockUpload(ctx, id, &s.uploadLocks)
	if err != nil {
		return err
	}
	defer unlock()
	path, _ := localUploadPath(id)
	if _, err := os.Stat(path + ".json"); err == nil {
		return ErrUploadExists
	} else if !os.IsNotExist(err) {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return ErrUploadExists
	}
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	if err := s.writeUploadState(ctx, id, blobUploadState{}); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

/*
GetBlobUpload возвращает подтверждённый offset и фазу локальной сессии.
*/
func (s *LocalStorage) GetBlobUpload(ctx context.Context, id string) (BlobUploadStatus, error) {
	return uploadStatus(ctx, id, &s.uploadLocks, s)
}

/*
AppendBlobUpload дописывает часть с общей проверкой offset и фазы загрузки.
*/
func (s *LocalStorage) AppendBlobUpload(ctx context.Context, id string, offset int64, body io.Reader) (int64, error) {
	return appendBlobPart(ctx, id, offset, body, &s.uploadLocks, s)
}

/*
CompleteBlobUpload завершает локальную загрузку по общему жизненному циклу.
*/
func (s *LocalStorage) CompleteBlobUpload(ctx context.Context, id, digest string, body io.Reader) (config.Blob, error) {
	return completeBlob(ctx, id, digest, body, &s.uploadLocks, s)
}

/*
AbortBlobUpload удаляет локальную сессию под отменяемой блокировкой.
*/
func (s *LocalStorage) AbortBlobUpload(ctx context.Context, id string) error {
	return abortBlob(ctx, id, &s.uploadLocks, s)
}

/*
readUploadState читает JSON-состояние и восстанавливает подтверждённый размер.

Старые uploads без JSON поддерживаются. Если процесс остановился между записью
данных и фиксацией состояния, лишний хвост откатывается до сохранённого offset.
*/
func (s *LocalStorage) readUploadState(ctx context.Context, id string) (blobUploadState, error) {
	var state blobUploadState
	if err := ctx.Err(); err != nil {
		return state, err
	}
	path, err := localUploadPath(id)
	if err != nil {
		return state, err
	}
	data, metaErr := os.ReadFile(path + ".json")
	info, fileErr := os.Stat(path)
	if os.IsNotExist(metaErr) {
		if os.IsNotExist(fileErr) {
			return state, ErrUploadNotFound
		}
		if fileErr != nil {
			return state, fileErr
		}
		state.Size = info.Size()
		if state.Size > 0 {
			state.Chunks = 1
		}
		// Миграция старой сессии до первой новой записи: незавершённый append
		// после перезапуска не должен ошибочно стать подтверждённым offset.
		return state, s.writeUploadState(ctx, id, state)
	}
	if metaErr != nil {
		return state, metaErr
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, errors.Join(ErrBlobCorrupted, err)
	}
	if err := validateUploadState(state); err != nil {
		return state, err
	}
	if os.IsNotExist(fileErr) && state.ReadyDigest != "" {
		return state, nil
	} // Rename уже опубликовал Blob.
	if fileErr != nil {
		return state, errors.Join(ErrBlobCorrupted, fileErr)
	}
	if !info.Mode().IsRegular() || info.Size() < state.Size {
		return state, ErrBlobCorrupted
	}
	if info.Size() > state.Size {
		if err := os.Truncate(path, state.Size); err != nil {
			return state, err
		}
	}
	return state, nil
}

/*
writeUploadState атомарно заменяет локальный JSON через файл в том же каталоге.

Sync данных и состояния выполняется до подтверждения offset клиенту.
*/
func (s *LocalStorage) writeUploadState(ctx context.Context, id string, state blobUploadState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := localUploadPath(id)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(config.TMP_PATH, id+".json-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	writeErr := json.NewEncoder(file).Encode(state)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path+".json")
}

/*
appendUpload фиксирует данные, размер, число чанков и фазу одним обновлением state.

При ошибке до фиксации JSON файл откатывается до старого подтверждённого размера.
*/
func (s *LocalStorage) appendUpload(ctx context.Context, id string, state *blobUploadState, body io.Reader) (int64, error) {
	oldSize := state.Size
	path, _ := localUploadPath(id)
	file, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		return oldSize, err
	}
	if _, err := file.Seek(oldSize, io.SeekStart); err != nil {
		file.Close()
		return oldSize, err
	}
	written, copyErr := copyWithContext(ctx, file, body)
	if copyErr == nil {
		copyErr = validateNextChunk(*state, written)
	}
	if copyErr == nil {
		copyErr = file.Sync()
	}
	closeErr := file.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return oldSize, errors.Join(err, os.Truncate(path, oldSize))
	}
	state.Size += written
	if written > 0 {
		state.Chunks++
	}
	if err := s.writeUploadState(ctx, id, *state); err != nil {
		return oldSize, errors.Join(err, os.Truncate(path, oldSize))
	}
	return state.Size, nil
}

/*
digestUpload вычисляет SHA-256 локального файла без копирования Blob.
*/
func (s *LocalStorage) digestUpload(ctx context.Context, id string, state blobUploadState) (string, error) {
	path, _ := localUploadPath(id)
	return calculateFileDigest(ctx, path)
}

/*
publishUpload проверяет существующий Blob или атомарно переносит файл через Rename.
*/
func (s *LocalStorage) publishUpload(ctx context.Context, id string, state blobUploadState) (config.Blob, error) {
	var empty config.Blob
	unlock, err := s.maintenance.lock(ctx, false)
	if err != nil {
		return empty, err
	}
	defer unlock()
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	encoded, err := parseSHA256Digest(state.ReadyDigest)
	if err != nil {
		return empty, err
	}
	finalPath := filepath.Join(config.BLOBS_PATH, encoded)
	path, _ := localUploadPath(id)
	if info, err := os.Stat(finalPath); err == nil {
		if !info.Mode().IsRegular() || info.Size() != state.Size {
			return empty, ErrBlobCorrupted
		}
		digest, err := calculateFileDigest(ctx, finalPath)
		if err != nil {
			return empty, err
		}
		if digest != state.ReadyDigest {
			return empty, ErrBlobCorrupted
		}
	} else if !os.IsNotExist(err) {
		return empty, err
	} else if err := os.Rename(path, finalPath); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return empty, fmt.Errorf("%w: TMP_PATH=%s, BLOBS_PATH=%s", ErrCrossDevice, config.TMP_PATH, config.BLOBS_PATH)
		}
		return empty, err
	}
	return config.Blob{Digest: state.ReadyDigest, Path: finalPath, Size: state.Size}, nil
}

/*
removeUpload удаляет данные, JSON и оставшиеся файлы атомарной записи состояния.
*/
func (s *LocalStorage) removeUpload(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := localUploadPath(id)
	if err != nil {
		return err
	}
	files, err := filepath.Glob(path + ".json-*")
	if err != nil {
		return err
	}
	files = append([]string{path, path + ".json"}, files...)
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
