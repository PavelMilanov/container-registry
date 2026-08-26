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
ManifestStore контракт для чтения и сохранения манифестов.
*/
type ManifestStore interface {
	SaveManifest(meta config.Meta, body []byte, link string) error
	GetManifest(repository, image, reference string) ([]byte, error)
}

/*
CloudStore контракт для управления пространствами registry.
*/
type CloudStore interface {
	AddCloud(cloud string) error
	DeleteCloud(cloud string) error
	GetCloudList() ([]string, error)
}

/*
RepositoryStore контракт для управления репозиториями.
*/
type RepositoryStore interface {
	GetRepositoriesList(cloud string) ([]string, error)
	DeleteRepository(cloud, repository string) error
}

/*
TagStore контракт для просмотра и удаления тегов.
*/
type TagStore interface {
	GetManifestList(cloud, repository string) ([]string, error)
	DeleteManifest(cloud, repository, tag string) error
}

/*
GarbageCollector контракт для сборки мусора хранилища.
*/
type GarbageCollector interface {
	GarbageCollection() error
}

/*
TagPruner контракт для удаления старых тегов.
*/
type TagPruner interface {
	DeleteOlderTags(count int) error
}

/*
Backend содержит отдельные возможности выбранного storage backend.

Необязательные возможности имеют nil-значение.
*/
type Backend struct {
	Blobs        BlobStore
	Manifests    ManifestStore
	Namespaces   NamespaceStore
	Clouds       CloudStore
	Repositories RepositoryStore
	Tags         TagStore

	GarbageCollector GarbageCollector
	TagPruner        TagPruner

	Uploads       BlobUploadStore
	UploadCleaner UploadCleaner
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

/*
BlobStore контракт для чтения готовых Blob.
*/
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
func NewStorage(env *config.Env) (Backend, error) {
	switch env.Storage.Type {
	case "local":
		local, err := newLocalStorage()
		if err != nil {
			return Backend{}, err
		}
		return backendFromLocal(local), nil
	case "s3":
		s3, err := newS3Storage(env)
		if err != nil {
			return Backend{}, err
		}
		return backendFromS3(s3), nil
	default:
		return Backend{}, errors.New("Не удалось инициализировать хранилище")
	}
}

/*
backendFromLocal собирает возможности локального хранилища.

	local - локальная реализация storage.
*/
func backendFromLocal(local *LocalStorage) Backend {
	return Backend{
		Blobs:            local,
		Manifests:        local,
		Namespaces:       local,
		Clouds:           local,
		Repositories:     local,
		Tags:             local,
		GarbageCollector: local,
		TagPruner:        local,
		Uploads:          local,
		UploadCleaner:    local,
	}
}

/*
backendFromS3 собирает возможности S3-хранилища.

	s3 - реализация storage на основе S3.
*/
func backendFromS3(s3 *S3Storage) Backend {
	return Backend{
		Blobs:            s3,
		Manifests:        s3,
		Namespaces:       s3,
		Clouds:           s3,
		Repositories:     s3,
		Tags:             s3,
		GarbageCollector: s3,
		TagPruner:        s3,
	}
}
