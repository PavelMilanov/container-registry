package system

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelMilanov/container-registry/storage"
	"github.com/sirupsen/logrus"
)

func GarbageCollection() {
	blobs := getBlobDigest()
	manifests := getManifestDigest()
	var cache []string
	for _, v := range blobs {
		if !contains(manifests, v) {
			cache = append(cache, v)
		}
	}

	for _, i := range cache {
		// if err := os.Remove(filepath.Join(storage.NewStorage().BlobPath, i)); err != nil {
		// 	logrus.Error(err)
		logrus.Infof("Тест удаления файла %s", i)
		// }
	}
	logrus.Infof("Удален кеш blobs %+v", cache)
}

func getManifestDigest() []string {
	var digests []string
	dir := storage.NewStorage().ManifestPath
	registies, _ := os.ReadDir(dir)
	for _, d := range registies {
		repoDir := filepath.Join(dir, d.Name())
		repositories, _ := os.ReadDir(repoDir)
		for _, file := range repositories {
			tagDir := filepath.Join(repoDir, file.Name())
			manifests, _ := os.ReadDir(tagDir)
			for _, manifest := range manifests {
				if !manifest.IsDir() { // читаем файлы sha256:...
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
					manifestDigest := manifestDigestString[1]
					digests = append(digests, manifestDigest)
				}
			}
		}
	}
	return digests
}

func getBlobDigest() []string {
	var blobs []string
	dir := storage.NewStorage().BlobPath
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
