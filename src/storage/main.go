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
	SaveManifest(body []byte, repository, image, reference, calculatedDigest string) (string, error)
	GetManifest(repository string, image string, reference string) ([]byte, error)
	DeleteRegistry(registry string) error
	DeleteImage(repository string, imageName string, imageTag string, imageHash string) error
	DeleteRepository(name string, image string) error
	GarbageCollection()
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
