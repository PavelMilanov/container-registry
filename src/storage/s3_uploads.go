package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/minio/minio-go/v7"
)

const s3PartSize = 16 * 1024 * 1024

/*
uploadPrefix возвращает отдельный префикс временных объектов загрузки.

	uploadID - UUID загрузки; пустое значение возвращает общий префикс.
*/
func (s *S3Storage) uploadPrefix(uploadID string) string {
	return path.Join(s.objectPrefix("tmp"), "uploads", uploadID) + "/"
}

/*
filepathToS3 преобразует разделители пути в разделители ключа S3.
*/
func filepathToS3(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}

/*
StartBlobUpload создаёт состояние загрузки в S3 без локального файла.
*/
func (s *S3Storage) StartBlobUpload(ctx context.Context, uploadID string) error {
	unlock, err := lockUpload(ctx, uploadID, &s.uploadLocks)
	if err != nil {
		return err
	}
	defer unlock()
	_, err = s.S3.StatObject(ctx, s.bucketName(), s.uploadPrefix(uploadID)+"state.json", minio.StatObjectOptions{})
	if err == nil {
		return ErrUploadExists
	}
	if !isS3NotFound(err) {
		return err
	}
	return s.writeUploadState(ctx, uploadID, blobUploadState{})
}

/*
AppendBlobUpload сохраняет очередной Registry-чанк как объект S3.

	expectedOffset - ожидаемый размер загрузки до записи.
	body - поток чанка, включая проверку длины на уровне HTTP-обработчика.

Состояние обновляется только после успешного чтения и сохранения всей части.
*/
func (s *S3Storage) AppendBlobUpload(ctx context.Context, id string, offset int64, body io.Reader) (int64, error) {
	return appendBlobPart(ctx, id, offset, body, &s.uploadLocks, s)
}

/*
appendUpload записывает часть и фиксирует новый offset под mutex загрузки.
*/
func (s *S3Storage) appendUpload(ctx context.Context, uploadID string, state *blobUploadState, body io.Reader) (int64, error) {
	key := s.uploadPrefix(uploadID) + "parts/" + uuid.NewV4().String()
	info, err := s.S3.PutObject(ctx, s.bucketName(), key, body, -1, minio.PutObjectOptions{
		ContentType: "application/octet-stream", PartSize: s3PartSize, NumThreads: 1,
	})
	if err != nil {
		s.discardUploadObject(key)
		return state.Size, err
	}
	if info.Size == 0 {
		s.discardUploadObject(key)
		return state.Size, s.writeUploadState(ctx, uploadID, *state)
	}
	if err := validateNextChunk(*state, info.Size); err != nil {
		s.discardUploadObject(key)
		return state.Size, err
	}
	oldSize := state.Size
	state.Parts = append(state.Parts, uploadPart{Key: key, Size: info.Size})
	state.Size += info.Size
	state.Chunks++
	if err := s.writeUploadState(ctx, uploadID, *state); err != nil {
		// Ответ PUT состояния мог потеряться после его фиксации. Чанк сохраняется:
		// удаление здесь могло бы повредить уже зафиксированную загрузку.
		return oldSize, err
	}
	return state.Size, nil
}

/*
CompleteBlobUpload собирает Blob потоково, проверяет SHA-256 и публикует в S3.

Проверяемый объект находится в uploads до успешной проверки digest.
Состояние ReadyDigest позволяет повторить публикацию после сетевой ошибки.
*/
func (s *S3Storage) CompleteBlobUpload(ctx context.Context, id, digest string, body io.Reader) (config.Blob, error) {
	return completeBlob(ctx, id, digest, body, &s.uploadLocks, s)
}

/*
assembleUpload переносит части в проверяемый объект с ограниченной памятью.

Каждый S3-поток закрывается до открытия следующего. Ошибки чтения передаются
через pipe; при ошибке PUT контекст отменяется и goroutine завершается.
*/
func (s *S3Storage) assembleUpload(ctx context.Context, key string, state blobUploadState) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	hasher := sha256.New()
	done := make(chan error, 1)
	go func() {
		var resultErr error
		for _, part := range state.Parts {
			object, err := s.S3.GetObject(ctx, s.bucketName(), part.Key, minio.GetObjectOptions{})
			if err != nil {
				resultErr = err
				break
			}
			info, err := object.Stat()
			if err != nil || info.Size != part.Size {
				_ = object.Close()
				resultErr = errors.Join(ErrBlobCorrupted, err)
				break
			}
			written, err := copyWithContext(ctx, io.MultiWriter(writer, hasher), object)
			closeErr := object.Close()
			if err == nil && written != part.Size {
				err = ErrBlobCorrupted
			}
			resultErr = errors.Join(err, closeErr)
			if resultErr != nil {
				break
			}
		}
		_ = writer.CloseWithError(resultErr)
		done <- resultErr
	}()
	partSize := uint64(s3PartSize)
	// S3 допускает не более 10 000 multipart parts. Размер собранного объекта
	// уже известен, поэтому для больших Blob увеличиваем размер SDK-буфера.
	if state.Size > int64(s3PartSize)*10000 {
		partSize = uint64((state.Size-1)/10000 + 1)
	}
	_, err := s.S3.PutObject(ctx, s.bucketName(), key, reader, state.Size, minio.PutObjectOptions{
		ContentType: "application/octet-stream", PartSize: partSize, NumThreads: 1,
	})
	if err != nil {
		cancel()
		_ = reader.CloseWithError(err)
	}
	readErr := <-done
	if err := errors.Join(err, readErr); err != nil {
		s.discardUploadObject(key)
		return "", err
	}
	return fmt.Sprintf("sha256:%x", hasher.Sum(nil)), nil
}

/*
hashObject вычисляет SHA-256 объекта потоково, не создавая локальный файл.
*/
func (s *S3Storage) hashObject(ctx context.Context, key string) (string, error) {
	object, err := s.S3.GetObject(ctx, s.bucketName(), key, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}
	defer object.Close()
	hasher := sha256.New()
	if _, err := copyWithContext(ctx, hasher, object); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", hasher.Sum(nil)), nil
}

/*
readUploadState читает и проверяет сохранённое состояние загрузки.
*/
func (s *S3Storage) readUploadState(ctx context.Context, uploadID string) (blobUploadState, error) {
	var state blobUploadState
	object, err := s.S3.GetObject(ctx, s.bucketName(), s.uploadPrefix(uploadID)+"state.json", minio.GetObjectOptions{})
	if err != nil {
		return state, err
	}
	defer object.Close()
	if err := json.NewDecoder(io.LimitReader(object, 8*1024*1024)).Decode(&state); err != nil {
		if isS3NotFound(err) {
			return state, ErrUploadNotFound
		}
		return state, errors.Join(ErrBlobCorrupted, err)
	}
	var size int64
	if len(state.Parts) > maxUploadChunks {
		return state, ErrBlobCorrupted
	}
	for _, part := range state.Parts {
		if !strings.HasPrefix(part.Key, s.uploadPrefix(uploadID)+"parts/") || part.Size <= 0 || part.Size > (1<<63-1)-size {
			return state, ErrBlobCorrupted
		}
		size += part.Size
	}
	if size != state.Size {
		return state, ErrBlobCorrupted
	}
	// Совместимость с upload state, записанным до общего счётчика чанков.
	if state.Chunks == 0 {
		state.Chunks = len(state.Parts)
	}
	if state.Chunks != len(state.Parts) || state.Chunks > maxUploadChunks {
		return state, ErrBlobCorrupted
	}
	if state.ReadyDigest != "" {
		if _, err := parseSHA256Digest(state.ReadyDigest); err != nil {
			return state, ErrBlobCorrupted
		}
		if state.ExpectedDigest == "" {
			state.ExpectedDigest = state.ReadyDigest
		}
	}
	if state.ExpectedDigest != "" {
		if _, err := parseSHA256Digest(state.ExpectedDigest); err != nil {
			return state, ErrBlobCorrupted
		}
	}
	if err := validateUploadState(state); err != nil {
		return state, err
	}
	return state, nil
}

/*
GetBlobUpload возвращает подтверждённый offset и фазу сессии S3.
*/
func (s *S3Storage) GetBlobUpload(ctx context.Context, id string) (BlobUploadStatus, error) {
	return uploadStatus(ctx, id, &s.uploadLocks, s)
}

/*
digestUpload собирает приватный S3-объект и вычисляет SHA-256 его содержимого.
*/
func (s *S3Storage) digestUpload(ctx context.Context, id string, state blobUploadState) (string, error) {
	return s.assembleUpload(ctx, s.uploadPrefix(id)+"blob", state)
}

/*
publishUpload публикует проверенный Blob либо проверяет существующий объект.
*/
func (s *S3Storage) publishUpload(ctx context.Context, id string, state blobUploadState) (config.Blob, error) {
	var empty config.Blob
	unlock, err := s.maintenance.lock(ctx, false)
	if err != nil {
		return empty, err
	}
	defer unlock()
	key, err := s.blobKeyFromDigest(state.ReadyDigest)
	if err != nil {
		return empty, err
	}
	existing, err := s.statBlob(ctx, state.ReadyDigest)
	if err == nil {
		if existing.Size != state.Size {
			return empty, ErrBlobCorrupted
		}
		digest, err := s.hashObject(ctx, key)
		if err != nil {
			return empty, err
		}
		if digest != state.ReadyDigest {
			return empty, ErrBlobCorrupted
		}
	} else if errors.Is(err, ErrBlobNotFound) {
		_, err = s.S3.ComposeObject(ctx, minio.CopyDestOptions{Bucket: s.bucketName(), Object: key}, minio.CopySrcOptions{Bucket: s.bucketName(), Object: s.uploadPrefix(id) + "blob"})
		if err != nil {
			return empty, err
		}
	} else {
		return empty, err
	}
	return config.Blob{Digest: state.ReadyDigest, Size: state.Size}, nil
}

/*
writeUploadState атомарно заменяет объект состояния загрузки в S3.
*/
func (s *S3Storage) writeUploadState(ctx context.Context, uploadID string, state blobUploadState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.S3.PutObject(ctx, s.bucketName(), s.uploadPrefix(uploadID)+"state.json", bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: "application/json"})
	return err
}

/*
AbortBlobUpload удаляет состояние, части и незавершённые S3 multipart uploads.
*/
func (s *S3Storage) AbortBlobUpload(ctx context.Context, id string) error {
	return abortBlob(ctx, id, &s.uploadLocks, s)
}

/*
removeUpload удаляет временные объекты только указанного uploadID.
*/
func (s *S3Storage) removeUpload(ctx context.Context, uploadID string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	prefix := s.uploadPrefix(uploadID)
	stateKey := prefix + "state.json"
	var resultErr error
	for object := range s.S3.ListObjects(ctx, s.bucketName(), minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if object.Err != nil {
			return errors.Join(resultErr, object.Err)
		}
		if object.Key == stateKey {
			continue
		}
		resultErr = errors.Join(resultErr, s.S3.RemoveObject(ctx, s.bucketName(), object.Key, minio.RemoveObjectOptions{}))
	}
	for upload := range s.S3.ListIncompleteUploads(ctx, s.bucketName(), prefix, true) {
		if upload.Err != nil {
			return errors.Join(resultErr, upload.Err)
		}
		core := minio.Core{Client: s.S3}
		resultErr = errors.Join(resultErr, core.AbortMultipartUpload(ctx, s.bucketName(), upload.Key, upload.UploadID))
	}
	if err := errors.Join(resultErr, ctx.Err()); err != nil {
		return err
	}
	return s.S3.RemoveObject(ctx, s.bucketName(), stateKey, minio.RemoveObjectOptions{})
}

/*
discardUploadObject удаляет незафиксированный объект даже после отмены запроса.
*/
func (s *S3Storage) discardUploadObject(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = s.S3.RemoveObject(ctx, s.bucketName(), key, minio.RemoveObjectOptions{})
	_ = s.S3.RemoveIncompleteUpload(ctx, s.bucketName(), key)
}

/*
CleanupUploads удаляет устаревшие сессии и оставшиеся S3 multipart uploads.

Активные загрузки пропускаются; это позволяет соблюдать timeout cron-задачи.
*/
func (s *S3Storage) CleanupUploads(ctx context.Context, olderThan time.Duration) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if olderThan <= 0 {
		return 0, errors.New("upload retention must be greater than zero")
	}
	ids := make(map[string]struct{})
	prefix := s.uploadPrefix("")
	for object := range s.S3.ListObjects(ctx, s.bucketName(), minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if object.Err != nil {
			return 0, object.Err
		}
		ids[firstKeyPart(strings.TrimPrefix(object.Key, prefix))] = struct{}{}
	}
	for upload := range s.S3.ListIncompleteUploads(ctx, s.bucketName(), prefix, true) {
		if upload.Err != nil {
			return 0, upload.Err
		}
		ids[firstKeyPart(strings.TrimPrefix(upload.Key, prefix))] = struct{}{}
	}
	deleted := 0
	deadline := time.Now().Add(-olderThan)
	for id := range ids {
		if _, err := localUploadPath(id); err != nil {
			continue
		}
		unlock, ok := s.uploadLocks.TryLock(id)
		if !ok {
			continue
		}
		removed, err := s.cleanupS3Upload(ctx, id, deadline)
		unlock()
		if err != nil {
			return deleted, err
		}
		if removed {
			deleted++
		}
	}
	return deleted, ctx.Err()
}

/*
cleanupS3Upload повторно проверяет возраст всех объектов под mutex сессии.
*/
func (s *S3Storage) cleanupS3Upload(ctx context.Context, id string, deadline time.Time) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	prefix := s.uploadPrefix(id)
	found := false
	for object := range s.S3.ListObjects(ctx, s.bucketName(), minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if object.Err != nil {
			return false, object.Err
		}
		found = true
		if !object.LastModified.Before(deadline) {
			return false, nil
		}
	}
	for upload := range s.S3.ListIncompleteUploads(ctx, s.bucketName(), prefix, true) {
		if upload.Err != nil {
			return false, upload.Err
		}
		found = true
		if !upload.Initiated.Before(deadline) {
			return false, nil
		}
	}
	if !found {
		return false, ctx.Err()
	}
	return true, s.removeUpload(ctx, id)
}
