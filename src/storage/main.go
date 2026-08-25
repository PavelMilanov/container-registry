// Package storage реализовывает логику работы с разными видами хранилищами данных.
package storage

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/PavelMilanov/container-registry/config"
)

/*
Storage абстракция хранилища.

	local - файловая система.
	s3 - S3-хранилище.
*/
type Storage interface {
	BlobStore
	NamespaceStore
	SaveManifest(meta config.Meta, body []byte, link string) error
	GetManifest(repository, image, reference string) ([]byte, error)
	GetManifestList(cloud, repository string) ([]string, error)
	DeleteManifest(cloud, repository, tag string) error
	AddCloud(cloud string) error
	DeleteCloud(cloud string) error
	GetCloudList() ([]string, error)
	GetRepositoriesList(cloud string) ([]string, error)
	DeleteRepository(cloud, repository string) error
	GarbageCollection() error
	DeleteOlderTags(count int) error
}

/*
NamespaceStore контракт для проверки пространств имён registry.
*/
type NamespaceStore interface {
	NamespaceExists(
		ctx context.Context,
		name string,
	) (bool, error)
}

type BlobStore interface {
	CheckBlob(digest string) error
	GetBlob(digest string) (config.Blob, error)
}

/*
BlobUploadStore контракт для загрузки blob.
*/
type BlobUploadStore interface {
	StartBlobUpload(
		ctx context.Context,
		uuid string,
	) error

	AppendBlobUpload(
		ctx context.Context,
		uuid string,
		expectedOffset int64,
		body io.Reader,
	) (newOffset int64, err error)

	CompleteBlobUpload(
		ctx context.Context,
		uuid string,
		expectedDigest string,
		finalBody io.Reader,
	) (config.Blob, error)

	AbortBlobUpload(
		ctx context.Context,
		uuid string,
	) error
}

/*
UploadCleaner контракт для очистки незавершённых загрузок Blob.
*/
type UploadCleaner interface {
	CleanupUploads(
		ctx context.Context,
		olderThan time.Duration,
	) (deleted int, err error)
}

/*
NewStorage инициализирует хранилище на основе конфигурации.
*/
func NewStorage(env *config.Env) (Storage, error) {
	switch env.Storage.Type {
	case "local":
		storage, err := newLocalStorage()
		if err != nil {
			return nil, err
		}
		return storage, nil
	case "s3":
		storage, err := newS3Storage(env)
		if err != nil {
			return nil, err
		}
		return storage, nil
	default:
		return nil, errors.New("Не удалось инициализировать хранилище")
	}
}
