package storage

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/minio/minio-go/v7"
)

/*
splitGCManifestKey отделяет repository от digest в пути или ключе manifest.
*/
func splitGCManifestKey(key string) gcRoot {
	return gcRoot{Repository: filepath.Dir(key), Digest: filepath.Base(key)}
}

/*
gcInventory полностью сканирует LocalStorage; symlinks запрещены для безопасного GC.
*/
func (s *LocalStorage) gcInventory(ctx context.Context) (gcSnapshot, error) {
	snapshot := gcSnapshot{Manifests: make(map[string]gcObject), Blobs: make(map[string]gcObject)}
	info, err := os.Lstat(config.MANIFEST_PATH)
	if err != nil {
		return snapshot, err
	}
	if !info.IsDir() {
		return snapshot, fmt.Errorf("MANIFEST_PATH не является каталогом")
	}
	info, err = os.Lstat(config.BLOBS_PATH)
	if err != nil {
		return snapshot, err
	}
	if !info.IsDir() {
		return snapshot, fmt.Errorf("BLOBS_PATH не является каталогом")
	}
	err = filepath.WalkDir(config.MANIFEST_PATH, func(key string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink в manifests: %s", key)
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(config.MANIFEST_PATH, key)
		if err != nil {
			return err
		}
		parts := strings.Split(relative, string(filepath.Separator))
		if len(parts) == 4 && parts[2] == "tags" {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("необычный файл тега: %s", key)
			}
			data, err := s.gcRead(ctx, key)
			if err != nil {
				return err
			}
			snapshot.Roots = append(snapshot.Roots, gcRoot{Repository: filepath.Dir(filepath.Dir(key)), Digest: string(data)})
		} else if len(parts) == 3 {
			if _, err := parseSHA256Digest(entry.Name()); err != nil {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("необычный файл manifest: %s", key)
			}
			snapshot.Manifests[key] = gcObject{key, info.ModTime()}
		}
		return nil
	})
	if err != nil {
		return snapshot, err
	}
	entries, err := os.ReadDir(config.BLOBS_PATH)
	if err != nil {
		return snapshot, err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return snapshot, err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return snapshot, fmt.Errorf("symlink в blobs: %s", entry.Name())
		}
		if entry.IsDir() {
			continue
		}
		if _, err := parseSHA256Digest("sha256:" + entry.Name()); err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return snapshot, err
		}
		if !info.Mode().IsRegular() {
			return snapshot, fmt.Errorf("необычный файл Blob: %s", entry.Name())
		}
		key := filepath.Join(config.BLOBS_PATH, entry.Name())
		snapshot.Blobs[key] = gcObject{key, info.ModTime()}
	}
	return snapshot, nil
}

/*
gcRead читает локальные metadata с проверкой context.
*/
func (s *LocalStorage) gcRead(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(key)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var buffer bytes.Buffer
	_, err = copyWithContext(ctx, &buffer, file)
	return buffer.Bytes(), err
}

/*
gcDelete удаляет один локальный объект из проверенного плана GC.
*/
func (s *LocalStorage) gcDelete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := os.Remove(key)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

/*
gcManifestKey формирует путь локального manifest без выхода из repository.
*/
func (s *LocalStorage) gcManifestKey(repository, digest string) string {
	return filepath.Join(repository, digest)
}

/*
gcBlobKey преобразует валидный digest в путь Blob LocalStorage.
*/
func (s *LocalStorage) gcBlobKey(digest string) (string, error) {
	encoded, err := parseSHA256Digest(digest)
	if err != nil {
		return "", err
	}
	return filepath.Join(config.BLOBS_PATH, encoded), nil
}

/*
gcInventory получает полный inventory S3 без локальных файлов и удаления объектов.
*/
func (s *S3Storage) gcInventory(ctx context.Context) (gcSnapshot, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	snapshot := gcSnapshot{Manifests: make(map[string]gcObject), Blobs: make(map[string]gcObject)}
	for _, directory := range []string{"manifests", "blobs"} {
		prefix := s.objectPrefix(directory) + "/"
		for object := range s.S3.ListObjects(ctx, s.bucketName(), minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
			if object.Err != nil {
				return snapshot, object.Err
			}
			parts := strings.Split(strings.TrimPrefix(object.Key, prefix), "/")
			if directory == "blobs" {
				if len(parts) != 1 {
					continue
				}
				if _, err := parseSHA256Digest("sha256:" + parts[0]); err != nil {
					continue
				}
				snapshot.Blobs[object.Key] = gcObject{object.Key, object.LastModified}
			} else if len(parts) == 4 && parts[2] == "tags" {
				data, err := s.gcRead(ctx, object.Key)
				if err != nil {
					return snapshot, err
				}
				snapshot.Roots = append(snapshot.Roots, gcRoot{Repository: filepath.Dir(filepath.Dir(object.Key)), Digest: string(data)})
			} else if len(parts) == 3 {
				if _, err := parseSHA256Digest(parts[2]); err != nil {
					continue
				}
				snapshot.Manifests[object.Key] = gcObject{object.Key, object.LastModified}
			}
		}
	}
	return snapshot, ctx.Err()
}

/*
gcRead читает metadata S3 с context текущего запуска GC.
*/
func (s *S3Storage) gcRead(ctx context.Context, key string) ([]byte, error) {
	object, err := s.S3.GetObject(ctx, s.bucketName(), key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()
	var buffer strings.Builder
	_, err = copyWithContext(ctx, &buffer, object)
	return []byte(buffer.String()), err
}

/*
gcDelete удаляет один S3-объект из проверенного плана GC.
*/
func (s *S3Storage) gcDelete(ctx context.Context, key string) error {
	return s.S3.RemoveObject(ctx, s.bucketName(), key, minio.RemoveObjectOptions{})
}

/*
gcManifestKey формирует ключ manifest в текущем repository S3.
*/
func (s *S3Storage) gcManifestKey(repository, digest string) string {
	return filepath.Join(repository, digest)
}

/*
gcBlobKey использует настроенный prefix S3 вместо локального DATA_PATH.
*/
func (s *S3Storage) gcBlobKey(digest string) (string, error) { return s.blobKeyFromDigest(digest) }

/*
runGC ограничивает запуск по времени и исключает мутации inventory во время обхода.
*/
func runGC(ctx context.Context, gate *maintenanceGate, backend gcBackend) error {
	ctx, cancel := context.WithTimeout(ctx, gcTimeout)
	defer cancel()
	unlock, err := gate.lock(ctx, true)
	if err != nil {
		return err
	}
	defer unlock()
	return collectGarbage(ctx, backend, time.Now())
}
