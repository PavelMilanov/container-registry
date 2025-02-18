package system

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelMilanov/container-registry/storage"
	"github.com/sirupsen/logrus"
)

func GarbageCollection(store *storage.Storage) {
	blobs := getBlobDigest(store.BlobPath)
	manifests := getManifestDigest(store.ManifestPath)
	var cache []string
	for _, v := range blobs {
		if !contains(manifests, v) {
			cache = append(cache, v)
		}
	}

	for _, i := range cache {
		if err := os.Remove(filepath.Join(store.BlobPath, i)); err != nil {
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
