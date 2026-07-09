// Package storage реализовывает логику работы с разными видами хранилищами данных.
package storage

import (
	"errors"

	"github.com/PavelMilanov/container-registry/config"
)

/*
Storage абстракция хранилища.

	local - файловая система.
	s3 - S3-хранилище.
*/
type Storage interface {
	CheckBlob(uuid string) error
	SaveBlob(tmpPath, digest string) error
	GetBlob(digest string) (config.Blob, error)
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
