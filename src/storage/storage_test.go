package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func TestGarbageCollection(t *testing.T) {
	env := config.NewEnv(config.CONFIG_PATH, "test.config")
	s := NewStorage(env)
	s.GarbageCollection()
}

func TestGetManifestDigest(t *testing.T) {
	path := "../var/manifests/"
	func(path string) {
		var manifestOSI []string
		var blobsBuffer []string
		var buffer []string
		registies, _ := os.ReadDir(path)
		for _, d := range registies {
			repoDir := filepath.Join(path, d.Name()) // var/manifests/dev
			repositories, _ := os.ReadDir(repoDir)
			for _, file := range repositories {
				tagDir := filepath.Join(repoDir, file.Name()) // var/manifests/dev/registry
				manifests, _ := os.ReadDir(tagDir)
				tagsDir := filepath.Join(tagDir, "tags")
				tags, _ := os.ReadDir(tagsDir)
				for _, tag := range tags { // ищем ссылки на манифесты в тегах, добавляем в буффер
					data, _ := os.ReadFile(filepath.Join(tagsDir, tag.Name()))
					buffer = append(buffer, string(data))
				}
				for _, manifest := range manifests {
					var m config.Manifest
					data, _ := os.ReadFile(filepath.Join(tagDir, manifest.Name()))
					json.Unmarshal(data, &m)
					fmt.Println(m.Config.Digest, m.MediaType)
					// ищем манифесты, в которых есть ссылки на blob-ы и копируем ссылку
					if m.MediaType == "application/vnd.docker.distribution.manifest.v2+json" || m.MediaType == "application/vnd.oci.image.manifest.v1+json" {
						for _, layer := range m.Layers {
							layerDigestString := strings.Split(layer.Digest, ":")
							layerDigest := layerDigestString[1]
							blobsBuffer = append(blobsBuffer, layerDigest)
						}
						// ищем манифесты, в которых ссылки на манифесты мультиплатформенных сборок
					} else if m.MediaType == "application/vnd.oci.image.index.v1+json" && m.Manifests != nil {
						for _, oci := range m.Manifests {
							manifestOSI = append(manifestOSI, oci.Digest)
						}
					}
				}
			}
		}
	}(path)
}
