// Package storage реализовывает логику работы с разными видами хранилищами данных:
// local - локальный диск на хосте;
// S3 - удаленное хранилище;
package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/sirupsen/logrus"
)

// Storage абстракция над моделью подключаемого хранилища.
type Storage struct {
	ManifestPath string
	BlobPath     string
	Type         string
	S3           *S3
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
			Type:         env.Storage.Type,
		}
	case "s3":
		s3 := newS3(env.Storage.Endpoint, env.Storage.AccessKey, env.Storage.SecretKey, env.Storage.SSL)
		return &Storage{
			ManifestPath: filepath.Join(config.BACKET_NAME, config.MANIFEST_PATH),
			BlobPath:     filepath.Join(config.BACKET_NAME, config.BLOBS_PATH),
			Type:         env.Storage.Type,
			S3:           s3,
		}
	}
	logrus.Fatal("неудалось инициализировать хранилище")
	return &Storage{}
}

func (s *Storage) GarbageCollection() {
	blobs := getBlobDigest(s.BlobPath)
	manifests := getManifestDigest(s.ManifestPath)
	var cache []string
	for _, v := range blobs {
		if !contains(manifests, v) {
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
					if !contains(buffer, manifest.Name()) { // ищем манифесты без ссылок на теги, удаляем
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

func contains(list []string, str string) bool {
	for _, i := range list {
		if i == str {
			return true
		}
	}
	return false
}
