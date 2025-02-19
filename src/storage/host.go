package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckBlob
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

// SaveBlob
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

// GetBlob
// Return:
//
//	map["path"] - путь к файлу.
//	map["size"] - размер файла.
func (s *Storage) GetBlob(digest string) (map[string]string, error) {
	info := make(map[string]string)
	switch s.Type {
	case "local":
		blobPath := filepath.Join(s.BlobPath, strings.Replace(digest, "sha256:", "", 1))

		// Открываем файл блоба
		file, err := os.Open(blobPath)
		if err != nil {
			if os.IsNotExist(err) {
				return info, errors.New("Blob not found")
			}
			return info, err
		}
		defer file.Close()
		fileInfo, err := file.Stat()
		if err != nil {
			return info, errors.New("Failed to stat blob file")
		}
		info["path"] = blobPath
		info["size"] = fmt.Sprintf("%d", fileInfo.Size())
	}
	return info, nil
}
