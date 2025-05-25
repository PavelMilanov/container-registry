// Package storage реализовывает логику работы с разными видами хранилищами данных:
// local - локальный диск на хосте;
// S3 - удаленное хранилище;
package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	Type string
	S3   *minio.Client
}

func NewStorage(env *config.Env) *Storage {
	os.Mkdir(config.DATA_PATH, 0755)
	os.MkdirAll(config.TMP_PATH, 0755)
	os.MkdirAll(config.BLOBS_PATH, 0755)
	os.MkdirAll(config.MANIFEST_PATH, 0755)
	switch env.Storage.Type {
	case "local":
		return &Storage{
			Type: env.Storage.Type,
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
			Type: env.Storage.Type,
			S3:   s3Client,
		}
	}
	logrus.Fatal("неудалось инициализировать хранилище")
	return &Storage{}
}

// CheckBlob
func (s *Storage) CheckBlob(uuid string) error {
	path := filepath.Join(config.BLOBS_PATH, strings.Replace(uuid, "sha256:", "", 1))
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
	finalPath := filepath.Join(config.BLOBS_PATH, strings.Replace(digest, "sha256:", "", 1))
	switch s.Type {
	case "local":
		if err := os.Rename(tmpPath, finalPath); err != nil {
			return err
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
	os.Remove(tmpPath)
	return nil
}

// GetBlob
// Return:
//
//	config.Blob
func (s *Storage) GetBlob(digest string) (config.Blob, error) {
	var data config.Blob
	digest = strings.Replace(digest, "sha256:", "", 1)
	blobPath := filepath.Join(config.BLOBS_PATH, digest)
	switch s.Type {
	case "local":
		file, err := os.Open(blobPath)
		if err != nil {
			if os.IsNotExist(err) {
				return data, errors.New("Blob not found")
			}
			return data, err
		}
		defer file.Close()
		fileInfo, err := file.Stat()
		if err != nil {
			return data, errors.New("Failed to stat blob file")
		}
		data.Digest = digest
		data.Size = fileInfo.Size()
	case "s3":
		reader, err := s.S3.GetObject(context.Background(), config.BACKET_NAME, blobPath, minio.GetObjectOptions{})
		if err != nil {
			return data, err
		}
		defer reader.Close()
		//создание временного файла и отложенное удаление
		body, _ := io.ReadAll(reader)
		path := filepath.Join(config.TMP_PATH, digest)
		err = os.WriteFile(path, body, 0644)
		timer := time.NewTimer(5 * time.Second)
		go func() {
			<-timer.C
			os.Remove(path)
		}()
		//
		fileInfo, err := reader.Stat()
		if err != nil {
			return data, errors.New("Failed to stat blob file")
		}
		data.Digest = path
		data.Size = fileInfo.Size
	}
	return data, nil
}

// SaveManifest
func (s *Storage) SaveManifest(body []byte, repository , image , reference , calculatedDigest string) (string, error) {
	manifestPath := filepath.Join(config.MANIFEST_PATH, repository, image, calculatedDigest)
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, image, "tags", reference)
	switch s.Type {
	case "local":
		err := os.MkdirAll(filepath.Dir(manifestPath), 0755)
		if err != nil {
			return "", errors.New("Failed to create manifest directory")
		}
		err = os.WriteFile(manifestPath, body, 0644)
		if err != nil {
			return "", errors.New("Failed to save manifest")
		}
		// Если это тег (а не digest), создаём символическую ссылку
		if !strings.HasPrefix(reference, "sha256:") {
			err = os.MkdirAll(filepath.Dir(tagPath), 0755)
			if err != nil {
				return "", errors.New("Failed to create tag directory")
			}
			err = os.WriteFile(tagPath, []byte(calculatedDigest), 0644)
			if err != nil {
				return "", errors.New("Failed to save tag reference")
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
	return manifestPath, nil
}

// GetManifest
// Return:
//
//	[]byte - docker manifest
func (s *Storage) GetManifest(repository string, image string, reference string) ([]byte, error) {
	var manifest []byte
	manifestPath := ""
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, image, "tags", reference)
	switch s.Type {
	case "local":
		// Определяем путь к файлу манифеста
		if strings.HasPrefix(reference, "sha256:") {
			// Если reference — это digest
			manifestPath = filepath.Join(config.MANIFEST_PATH, repository, image, reference)
		} else {
			// Если reference — это тег
			tagData, err := os.ReadFile(tagPath)
			if err != nil {
				return manifest, errors.New("Tag not found")
			}
			manifestDigest := string(tagData)
			manifestPath = filepath.Join(config.MANIFEST_PATH, repository, image, manifestDigest)
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
			manifestPath = filepath.Join(config.MANIFEST_PATH, repository, image, reference)
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
			manifestPath = filepath.Join(config.MANIFEST_PATH, repository, image, manifestDigest)
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
	path := filepath.Join(config.MANIFEST_PATH, registry)
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
	path := filepath.Join(config.MANIFEST_PATH, repository, imageName, imageHash)
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, imageName, "tags", imageTag)
	switch s.Type {
	case "local":
		err := os.Remove(tagPath)
		err = os.Remove(path)
		if err != nil {
			return err
		}
	case "s3":
		opts := minio.RemoveObjectOptions{
			GovernanceBypass: true,
		}

		err := s.S3.RemoveObject(context.Background(), config.DATA_PATH, tagPath, opts)
		if err != nil {
			logrus.Error(err)
			return err
		}
		err = s.S3.RemoveObject(context.Background(), config.DATA_PATH, path, opts)
		if err != nil {
			logrus.Error(err)
			return err
		}
	}
	return nil
}

// DeleteRepository
func (s *Storage) DeleteRepository(name string, image string) error {
	path := filepath.Join(config.MANIFEST_PATH, name, image)
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
	blobs := getBlobDigest(config.BLOBS_PATH)
	manifests := getManifestDigest(config.MANIFEST_PATH)
	var cache []string
	for _, v := range blobs {
		if !slices.Contains(manifests, v) {
			cache = append(cache, v)
		}
	}

	for _, i := range cache {
		if err := os.Remove(filepath.Join(config.BLOBS_PATH, i)); err != nil {
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
