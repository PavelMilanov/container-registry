package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func (s *Storage) CheckBlob(uuid string) error {
	switch s.Type {
	case "local":
		path := filepath.Join(s.BlobPath, strings.Replace(uuid, "sha256:", "", 1))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.New("Blob not found")
		}
	}
	return nil
}

func (s *Storage) SaveBlob(tmpPath string, digest string) error {
	switch s.Type {
	case "local":
		finalPath := filepath.Join(s.BlobPath, strings.Replace(digest, "sha256:", "", 1))
		if err := os.Rename(tmpPath, finalPath); err != nil {
			return errors.New("Failed to finalize blob upload")
		}
	}
	return nil
}
