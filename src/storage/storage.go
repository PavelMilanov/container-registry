package storage

import (
	"os"
	"path/filepath"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/minio/minio-go"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	ManifestPath string
	BlobPath     string
}

func NewStorage(env *config.Env) *Storage {
	switch env.Storage.Type {
	case "local":
		blobPath := filepath.Join(config.STORAGE_PATH, config.BLOBS_PATH)
		manifestPath := filepath.Join(config.STORAGE_PATH, config.MANIFEST_PATH)
		os.MkdirAll(blobPath, 0755)
		os.MkdirAll(manifestPath, 0755)
		os.Mkdir(config.DATA_PATH, 0755)
		return &Storage{
			ManifestPath: manifestPath,
			BlobPath:     blobPath,
		}
	case "s3":
		minioClient, err := minio.New(env.Storage.Endpoint, env.Storage.AccessKey, env.Storage.AccessKey, false)
		if err != nil {
			logrus.Fatal(err)
		}
		err = minioClient.MakeBucket(config.BACKET_NAME, os.Getenv("TZ"))
		if err != nil {
			exists, errBucketExists := minioClient.BucketExists(config.BACKET_NAME)
			if errBucketExists == nil && exists {
				logrus.Debugf("Подключен bucket %s", config.BACKET_NAME)
			} else {
				logrus.Error(err)
			}
		} else {
			logrus.Infof("Успешно инициализирован bucket %s", config.BACKET_NAME)
		}
		return &Storage{
			ManifestPath: filepath.Join(config.BACKET_NAME, config.MANIFEST_PATH),
			BlobPath:     filepath.Join(config.BACKET_NAME, config.BLOBS_PATH),
		}
	}
	logrus.Fatal("неудалось инициализировать хранилище")
	return &Storage{}
}
