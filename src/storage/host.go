package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/minio/minio-go/v7"
)

// CheckBlob
func (s *Storage) CheckBlob(uuid string) error {
	path := filepath.Join(s.BlobPath, strings.Replace(uuid, "sha256:", "", 1))
	switch s.Type {
	case "local":
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.New("Blob not found")
		}
	case "s3":
		if _, err := s.S3.Client.StatObject(context.Background(), config.BACKET_NAME, path, minio.GetObjectOptions{}); err != nil {
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
	case "s3":

	}
	return info, nil
}

// SaveManifest
func (s *Storage) SaveManifest(body []byte, repository string, image string, reference string, calculatedDigest string) error {
	switch s.Type {
	case "local":
		manifestPath := filepath.Join(s.ManifestPath, repository, image, calculatedDigest)
		err := os.MkdirAll(filepath.Dir(manifestPath), 0755)
		if err != nil {
			return errors.New("Failed to create manifest directory")
		}
		err = os.WriteFile(manifestPath, body, 0644)
		if err != nil {
			return errors.New("Failed to save manifest")
		}
		// Если это тег (а не digest), создаём символическую ссылку
		if !strings.HasPrefix(reference, "sha256:") {
			tagPath := filepath.Join(s.ManifestPath, repository, image, "tags", reference)
			err = os.MkdirAll(filepath.Dir(tagPath), 0755)
			if err != nil {
				return errors.New("Failed to create tag directory")
			}
			err = os.WriteFile(tagPath, []byte(calculatedDigest), 0644)
			if err != nil {
				return errors.New("Failed to save tag reference")
			}
		}
	}
	return nil
}

// GetManifest
// Return:
//
//	[]byte - docker manifest
func (s *Storage) GetManifest(repository string, image string, reference string) ([]byte, error) {
	var manifest []byte
	switch s.Type {
	case "local":
		// Определяем путь к файлу манифеста
		manifestPath := ""
		if strings.HasPrefix(reference, "sha256:") {
			// Если reference — это digest
			manifestPath = filepath.Join(s.ManifestPath, repository, image, reference)
		} else {
			// Если reference — это тег
			tagPath := filepath.Join(s.ManifestPath, repository, image, "tags", reference)
			tagData, err := os.ReadFile(tagPath)
			if err != nil {
				return manifest, errors.New("Tag not found")
			}
			manifestDigest := string(tagData)
			manifestPath = filepath.Join(s.ManifestPath, repository, image, manifestDigest)
		}
		// Читаем содержимое манифеста
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return manifest, errors.New("Manifest not found")
		}
		manifest = data
	}
	return manifest, nil
}

// DeleteRegistry
func (s *Storage) DeleteRegistry(registry string) error {
	switch s.Type {
	case "local":
		if err := os.RemoveAll(filepath.Join(s.ManifestPath, registry)); err != nil {
			return err
		}
	}
	return nil
}

// DeleteImage
func (s *Storage) DeleteImage(repository string, imageName string, imageTag string, imageHash string) error {
	switch s.Type {
	case "local":
		err := os.Remove(filepath.Join(s.ManifestPath, repository, imageName, "tags", imageTag))
		err = os.Remove(filepath.Join(s.ManifestPath, repository, imageName, imageHash))
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteRepository
func (s *Storage) DeleteRepository(name string, image string) error {
	switch s.Type {
	case "local":
		if err := os.RemoveAll(filepath.Join(s.ManifestPath, name, image)); err != nil {
			return err
		}
	}
	return nil
}
