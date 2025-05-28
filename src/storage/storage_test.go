package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func initConfig() *config.Env {
	env, _ := config.NewEnv("../", "test.config")
	return env
}

// func TestGarbageCollection(t *testing.T) {
// 	env := initConfig()
// 	s := NewStorage(env)
// 	s.GarbageCollection()
// }

func TestInventoryBlobs(t *testing.T) {
	path := "../var/manifests/"
	func(path string) {
		var blobsBuffer []string
		var buffer []string
		registies, _ := os.ReadDir(path)
		for _, d := range registies {
			registryDir := filepath.Join(path, d.Name()) // var/manifests/dev
			repositories, _ := os.ReadDir(registryDir)
			for _, file := range repositories {
				repositoryDir := filepath.Join(registryDir, file.Name()) // var/manifests/dev/registry
				checkManifests(repositoryDir)
				manifests, _ := os.ReadDir(repositoryDir)
				tagsDir := filepath.Join(repositoryDir, "tags")
				tags, _ := os.ReadDir(tagsDir)
				for _, tag := range tags { // ищем ссылки на манифесты в тегах, добавляем в буффер
					data, _ := os.ReadFile(filepath.Join(tagsDir, tag.Name()))
					buffer = append(buffer, string(data))
				}
				var m config.Manifest
				for _, manifest := range manifests {
					data, _ := os.ReadFile(filepath.Join(repositoryDir, manifest.Name()))
					json.Unmarshal(data, &m)
					// ищем манифесты, в которых есть ссылки на blob-ы и копируем ссылку
					if m.MediaType == "application/vnd.docker.distribution.manifest.v2+json" || m.MediaType == "application/vnd.oci.image.manifest.v1+json" {
						for _, layer := range m.Layers {
							layerDigestString := strings.Split(layer.Digest, ":")
							layerDigest := layerDigestString[1]
							blobsBuffer = append(blobsBuffer, layerDigest)
						}
					}
				}
			}
		}

	}(path)
}

func TestCheckManifests(t *testing.T) {
	path := "../var/manifests/local/registry"
	checkManifests(path)
}
