package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

type S3Storage struct {
	S3 *minio.Client
}

func newS3Storage(env *config.Env) (*S3Storage, error) {
	s3Client, err := minio.New(env.Storage.Credentials.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(env.Storage.Credentials.AccessKey, env.Storage.Credentials.SecretKey, ""),
		Secure: env.Storage.Credentials.SSL,
	})
	if err != nil {
		return &S3Storage{}, nil
	}
	_, errBucketExists := s3Client.BucketExists(context.Background(), config.BACKET_NAME)
	if errBucketExists != nil {
		return &S3Storage{}, nil
	}
	return &S3Storage{
		S3: s3Client,
	}, nil
}

// CheckBlob
func (s *S3Storage) CheckBlob(uuid string) error {
	path := filepath.Join(config.BLOBS_PATH, strings.Replace(uuid, "sha256:", "", 1))

	if _, err := s.S3.StatObject(context.Background(), config.BACKET_NAME, path, minio.GetObjectOptions{}); err != nil {
		return errors.New("Blob not found")
	}

	return nil
}

// SaveBlob
func (s *S3Storage) SaveBlob(tmpPath string, digest string) error {
	finalPath := filepath.Join(config.BLOBS_PATH, strings.Replace(digest, "sha256:", "", 1))

	file, _ := os.Open(tmpPath)
	defer file.Close()
	fileStat, _ := file.Stat()
	_, err := s.S3.PutObject(context.Background(), config.BACKET_NAME, finalPath, file, fileStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return err
	}
	os.Remove(tmpPath)
	return nil
}

// GetBlob
// Return:
//
//	config.Blob
func (s *S3Storage) GetBlob(digest string) (config.Blob, error) {
	var data config.Blob
	digest = strings.Replace(digest, "sha256:", "", 1)
	blobPath := filepath.Join(config.BLOBS_PATH, digest)

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
	fileInfo, err := reader.Stat()
	if err != nil {
		return data, errors.New("Failed to stat blob file")
	}
	data.Digest = path
	data.Size = fileInfo.Size
	return data, nil
}

// SaveManifest
func (s *S3Storage) SaveManifest(body []byte, repository, image, reference, calculatedDigest string) (string, error) {
	manifestPath := filepath.Join(config.MANIFEST_PATH, repository, image, calculatedDigest)
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, image, "tags", reference)
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

	return manifestPath, nil
}

// GetManifest
// Return:
//
//	[]byte - docker manifest
func (s *S3Storage) GetManifest(repository string, image string, reference string) ([]byte, error) {
	var manifest []byte
	manifestPath := ""
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, image, "tags", reference)
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
	return manifest, nil
}

func (s *S3Storage) AddRegistry(registry string) error {

	return nil
}

// DeleteRegistry
func (s *S3Storage) DeleteRegistry(registry string) error {
	path := filepath.Join(config.MANIFEST_PATH, registry)
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
	return nil
}

// DeleteImage
func (s *S3Storage) DeleteImage(repository string, imageName string, imageTag string, imageHash string) error {
	path := filepath.Join(config.MANIFEST_PATH, repository, imageName, imageHash)
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, imageName, "tags", imageTag)
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
	return nil
}

// DeleteRepository
func (s *S3Storage) DeleteRepository(name string, image string) error {
	path := filepath.Join(config.MANIFEST_PATH, name, image)
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
	return nil
}

func (s *S3Storage) GarbageCollection() {
}
