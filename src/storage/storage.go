package storage

import (
	"os"
	"path/filepath"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	ManifestPath string
	BlobPath     string
	TmpPath      string
	Type         string
	Store        *S3
}

func NewStorage(env *config.Env) *Storage {
	switch env.Storage.Type {
	case "local":
		blobPath := filepath.Join(config.DATA_PATH, config.STORAGE_PATH, config.BLOBS_PATH)
		manifestPath := filepath.Join(config.DATA_PATH, config.STORAGE_PATH, config.MANIFEST_PATH)
		os.MkdirAll(blobPath, 0755)
		os.MkdirAll(manifestPath, 0755)
		os.MkdirAll(config.TMP_PATH, 0755)
		os.Mkdir(config.DATA_PATH, 0755)
		return &Storage{
			ManifestPath: manifestPath,
			BlobPath:     blobPath,
			TmpPath:      config.TMP_PATH,
			Type:         env.Storage.Type,
		}
	case "s3":
		s3 := newS3(env.Storage.Endpoint, env.Storage.AccessKey, env.Storage.SecretKey)
		return &Storage{
			ManifestPath: filepath.Join(config.BACKET_NAME, config.MANIFEST_PATH),
			BlobPath:     filepath.Join(config.BACKET_NAME, config.BLOBS_PATH),
			TmpPath:      config.TMP_PATH,
			Type:         env.Storage.Type,
			Store:        s3,
		}
	}
	logrus.Fatal("неудалось инициализировать хранилище")
	return &Storage{}
}
