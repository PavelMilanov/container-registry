// Package storage реализовывает логику работы с разными видами хранилищами данных:
// local - локальный диск на хосте;
// S3 - удаленное хранилище;
package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

// Storage абстракция над моделью подключаемого хранилища.
type Storage struct {
	ManifestPath string
	BlobPath     string
	Type         string
	S3           *minio.Client
}

func NewStorage(env *config.Env) *Storage {
	os.MkdirAll(config.TMP_PATH, 0755)
	switch env.Storage.Type {
	case "local":
		blobPath := filepath.Join(config.DATA_PATH, config.STORAGE_PATH, config.BLOBS_PATH)
		manifestPath := filepath.Join(config.DATA_PATH, config.STORAGE_PATH, config.MANIFEST_PATH)
		os.MkdirAll(blobPath, 0755)
		os.MkdirAll(manifestPath, 0755)
		os.Mkdir(config.DATA_PATH, 0755)
		return &Storage{
			ManifestPath: manifestPath,
			BlobPath:     blobPath,
			Type:         env.Storage.Type,
		}
	case "s3":
		s3Client, err := minio.New(env.Storage.Credentials.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(env.Storage.Credentials.AccessKey, env.Storage.Credentials.SecretKey, ""),
			Secure: env.Storage.Credentials.SSL,
		})
		if err != nil {
			logrus.Fatal(err)
		}
		_, errBucketExists := s3Client.BucketExists(context.Background(), config.BACKET_NAME)
		if errBucketExists != nil {
			logrus.Fatal(err)
		}
		return &Storage{
			ManifestPath: filepath.Join(config.BACKET_NAME, config.MANIFEST_PATH),
			BlobPath:     filepath.Join(config.BACKET_NAME, config.BLOBS_PATH),
			Type:         env.Storage.Type,
			S3:           s3Client,
		}
	}
	logrus.Fatal("неудалось инициализировать хранилище")
	return &Storage{}
}

// CheckBlob
func (s *Storage) CheckBlob(uuid string) error {
	path := filepath.Join(s.BlobPath, strings.Replace(uuid, "sha256:", "", 1))
	switch s.Type {
	case "local":
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.New("Blob not found")
		}
	case "s3":
		if _, err := s.S3.StatObject(context.Background(), config.BACKET_NAME, path, minio.GetObjectOptions{}); err != nil {
			return errors.New("Blob not found")
		}
	}
	return nil
}

// SaveBlob
func (s *Storage) SaveBlob(tmpPath string, digest string) error {
	finalPath := filepath.Join(s.BlobPath, strings.Replace(digest, "sha256:", "", 1))
	switch s.Type {
	case "local":
		if err := os.Rename(tmpPath, finalPath); err != nil {
			return errors.New("Failed to finalize blob upload")
		}
	case "s3":
		file, _ := os.Open(tmpPath)
		defer file.Close()
		fileStat, _ := file.Stat()
		_, err := s.S3.PutObject(context.Background(), config.BACKET_NAME, finalPath, file, fileStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			return err
		}
	}
	defer os.Remove(tmpPath)
	return nil
}

// GetBlob
// Return:
//
//	map["path"] - путь к файлу.
//	map["size"] - размер файла.
func (s *Storage) GetBlob(digest string) (map[string]string, error) {
	info := make(map[string]string)
	blobPath := filepath.Join(s.BlobPath, strings.Replace(digest, "sha256:", "", 1))
	switch s.Type {
	case "local":
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
		reader, err := s.S3.GetObject(context.Background(), config.BACKET_NAME, blobPath, minio.GetObjectOptions{})
		if err != nil {
			return info, err
		}
		defer reader.Close()
		//создание временного файла и отложенное удаление
		data, _ := io.ReadAll(reader)
		path := filepath.Join(config.TMP_PATH, digest)
		err = os.WriteFile(path, data, 0644)
		timer := time.NewTimer(5 * time.Second)
		go func() {
			<-timer.C
			os.Remove(path)
		}()
		//
		fileInfo, err := reader.Stat()
		if err != nil {
			return info, errors.New("Failed to stat blob file")
		}
		info["path"] = path
		info["size"] = fmt.Sprintf("%d", fileInfo.Size)
	}
	return info, nil
}

// SaveManifest
func (s *Storage) SaveManifest(body []byte, repository string, image string, reference string, calculatedDigest string) error {
	manifestPath := filepath.Join(s.ManifestPath, repository, image, calculatedDigest)
	tagPath := filepath.Join(s.ManifestPath, repository, image, "tags", reference)
	switch s.Type {
	case "local":
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
			err = os.MkdirAll(filepath.Dir(tagPath), 0755)
			if err != nil {
				return errors.New("Failed to create tag directory")
			}
			err = os.WriteFile(tagPath, []byte(calculatedDigest), 0644)
			if err != nil {
				return errors.New("Failed to save tag reference")
			}
		}
	case "s3":
		reader := bytes.NewReader(body)
		size := reader.Size()
		_, err := s.S3.PutObject(context.Background(), config.BACKET_NAME, manifestPath, reader, size, minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			logrus.Error(err)
		}
		if !strings.HasPrefix(reference, "sha256:") {
			reader := bytes.NewReader([]byte(calculatedDigest))
			size := reader.Size()
			_, err = s.S3.PutObject(context.Background(), config.BACKET_NAME, tagPath, reader, size, minio.PutObjectOptions{ContentType: "application/octet-stream"})
			if err != nil {
				logrus.Error(err)
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
	manifestPath := ""
	tagPath := filepath.Join(s.ManifestPath, repository, image, "tags", reference)
	switch s.Type {
	case "local":
		// Определяем путь к файлу манифеста
		if strings.HasPrefix(reference, "sha256:") {
			// Если reference — это digest
			manifestPath = filepath.Join(s.ManifestPath, repository, image, reference)
		} else {
			// Если reference — это тег
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
	case "s3":
		if strings.HasPrefix(reference, "sha256:") {
			// Если reference — это digest
			manifestPath = filepath.Join(s.ManifestPath, repository, image, reference)
		} else {
			// Если reference — это тег
			reader, err := s.S3.GetObject(context.Background(), config.BACKET_NAME, tagPath, minio.GetObjectOptions{})
			if err != nil {
				return manifest, errors.New("Tag not found")
			}
			defer reader.Close()
			tagData, err := io.ReadAll(reader)
			if err != nil {
				return manifest, err
			}
			manifestDigest := string(tagData)
			manifestPath = filepath.Join(s.ManifestPath, repository, image, manifestDigest)
		}
		reader, err := s.S3.GetObject(context.Background(), config.BACKET_NAME, manifestPath, minio.GetObjectOptions{})
		if err != nil {
			return manifest, err
		}
		defer reader.Close()
		data, err := io.ReadAll(reader)
		manifest = data
	}
	return manifest, nil
}

// DeleteRegistry
func (s *Storage) DeleteRegistry(registry string) error {
	path := filepath.Join(s.ManifestPath, registry)
	switch s.Type {
	case "local":
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	case "s3":
		objectsCh := make(chan minio.ObjectInfo)
		go func() {
			defer close(objectsCh)
			opts := minio.ListObjectsOptions{Prefix: path, Recursive: true}
			for object := range s.S3.ListObjects(context.Background(), config.BACKET_NAME, opts) {
				if object.Err != nil {
					logrus.Error(object.Err)
				}
				objectsCh <- object
			}
		}()
		err := s.S3.RemoveObjects(context.Background(), config.BACKET_NAME, objectsCh, minio.RemoveObjectsOptions{})
		for e := range err {
			return e.Err
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
	path := filepath.Join(s.ManifestPath, name, image)
	switch s.Type {
	case "local":
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	case "s3":
		objectsCh := make(chan minio.ObjectInfo)
		go func() {
			defer close(objectsCh)
			opts := minio.ListObjectsOptions{Prefix: path, Recursive: true}
			for object := range s.S3.ListObjects(context.Background(), config.BACKET_NAME, opts) {
				if object.Err != nil {
					logrus.Error(object.Err)
				}
				objectsCh <- object
			}
		}()
		err := s.S3.RemoveObjects(context.Background(), config.BACKET_NAME, objectsCh, minio.RemoveObjectsOptions{})
		for e := range err {
			return e.Err
		}
	}
	return nil
}

func (s *Storage) GarbageCollection() {
	blobs := getBlobDigest(s.BlobPath)
	manifests := getManifestDigest(s.ManifestPath)
	var cache []string
	for _, v := range blobs {
		if !slices.Contains(manifests, v) {
			cache = append(cache, v)
		}
	}

	for _, i := range cache {
		if err := os.Remove(filepath.Join(s.BlobPath, i)); err != nil {
			logrus.Error(err)
		}
	}
	logrus.Infof("Удален кеш blobs %+v", cache)
}

func getManifestDigest(dir string) []string {
	var digests []string
	registies, _ := os.ReadDir(dir)
	for _, d := range registies {
		repoDir := filepath.Join(dir, d.Name())
		repositories, _ := os.ReadDir(repoDir)
		for _, file := range repositories {
			tagDir := filepath.Join(repoDir, file.Name())
			manifests, _ := os.ReadDir(tagDir)
			tagsDir := filepath.Join(tagDir, "tags")
			tags, _ := os.ReadDir(tagsDir)
			var buffer []string
			for _, tag := range tags { // ищем ссылки на манифесты в тегах, добавляем в буффер
				data, _ := os.ReadFile(filepath.Join(tagsDir, tag.Name()))
				buffer = append(buffer, string(data))
			}
			for _, manifest := range manifests {
				if !manifest.IsDir() { // читаем файлы sha256:...
					if !slices.Contains(buffer, manifest.Name()) { // ищем манифесты без ссылок на теги, удаляем
						os.Remove(filepath.Join(tagDir, manifest.Name()))
					}
					data, _ := os.ReadFile(filepath.Join(tagDir, manifest.Name()))
					type config struct {
						Digest string `json:"digest"`
					}
					type layers struct {
						Digest string `json:"digest"`
					}
					type manifest struct {
						Config config   `json:"config"`
						Layers []layers `json:"layers"`
					}
					jsonData := manifest{}
					json.Unmarshal(data, &jsonData)
					for _, layer := range jsonData.Layers {
						layerDigestString := strings.Split(layer.Digest, ":")
						layerDigest := layerDigestString[1]
						digests = append(digests, layerDigest)
					}
					manifestDigestString := strings.Split(jsonData.Config.Digest, ":")
					if len(manifestDigestString) > 1 {
						manifestDigest := manifestDigestString[1]
						digests = append(digests, manifestDigest)
					}
				}
			}
		}
	}
	return digests
}

func getBlobDigest(dir string) []string {
	var blobs []string
	digests, _ := os.ReadDir(dir)
	for _, blob := range digests {
		blobs = append(blobs, blob.Name())
	}
	return blobs
}
