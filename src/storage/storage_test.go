package storage

import (
	"fmt"
	"os"
	"path/filepath"
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
		var buffer []string
		registies, _ := os.ReadDir(path)
		fmt.Println(registies)
		for _, d := range registies {
			repoDir := filepath.Join(path, d.Name()) // var/manifests/dev
			repositories, _ := os.ReadDir(repoDir)
			for _, file := range repositories {
				tagDir := filepath.Join(repoDir, file.Name()) // var/manifests/dev/registry
				fmt.Println(tagDir)
				// manifests, _ := os.ReadDir(tagDir)
				tagsDir := filepath.Join(tagDir, "tags")
				tags, _ := os.ReadDir(tagsDir)
				for _, tag := range tags { // ищем ссылки на манифесты в тегах, добавляем в буффер
					data, _ := os.ReadFile(filepath.Join(tagsDir, tag.Name()))
					buffer = append(buffer, string(data))
				}
				// for _, manifest := range manifests {
				// 	var m config.Manifest
				// 	data, _ := os.ReadFile(filepath.Join(tagDir, manifest.Name()))
				// 	json.Unmarshal(data, &m)
				// 	fmt.Println(m.MediaType)
				// }

			}
		}
	}(path)
}
