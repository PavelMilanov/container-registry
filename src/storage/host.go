package storage

import (
	"os"
	"path/filepath"

	"github.com/PavelMilanov/container-registry/config"
)

type Storage struct {
	ManifestPath string
	BlobPath     string
}

func NewStorage() *Storage {
	blobPath := filepath.Join(config.STORAGE_PATH, config.BLOBS_PATH)
	manifestPath := filepath.Join(config.STORAGE_PATH, config.MANIFEST_PATH)
	os.MkdirAll(blobPath, 0755)
	os.MkdirAll(manifestPath, 0755)
	os.Mkdir(config.DATA_PATH, 0755)
	return &Storage{
		ManifestPath: manifestPath,
		BlobPath:     blobPath,
	}
}
