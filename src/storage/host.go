package storage

import (
	"path/filepath"

	"github.com/PavelMilanov/container-registry/config"
)

type Storage struct {
	ManifestPath string
	BlobPath     string
}

func NewStorage() *Storage {
	return &Storage{
		ManifestPath: filepath.Join(config.STORAGE_PATH, config.MANIFEST_PATH),
		BlobPath:     filepath.Join(config.STORAGE_PATH, config.BLOBS_PATH),
	}
}
