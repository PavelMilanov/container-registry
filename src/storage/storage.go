package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/sirupsen/logrus"
)

/*
inventoryBlobs сканирует директорию и возвращает список используемых blob-ов.
*/
func inventoryBlobs() []string {
	var blobsBuffer []string
	var buffer []string
	registies, _ := os.ReadDir(config.MANIFEST_PATH)
	for _, d := range registies {
		registryDir := filepath.Join(config.MANIFEST_PATH, d.Name()) // var/manifests/dev
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
					configDigest := strings.Split(m.Config.Digest, ":")[1]
					blobsBuffer = append(blobsBuffer, configDigest)
					for _, layer := range m.Layers {
						layerDigest := strings.Split(layer.Digest, ":")[1]
						blobsBuffer = append(blobsBuffer, layerDigest)
					}
				}
			}
		}
	}
	return blobsBuffer
}

/*
CheckManifests сканирует директорию и удаляет лишние файлы манифестов.

	path - путь к директории с манифестами для конкретного репозитория.
*/
func checkManifests(path string) {
	var tagList []string
	var buffer []string
	tags, _ := os.ReadDir(filepath.Join(path, "tags"))
	// проходим по тегам и забираем их digest
	for _, file := range tags {
		digest, err := os.ReadFile(filepath.Join(path, "tags", file.Name()))
		if err != nil {
			logrus.Error("Error reading file:", err)
		}
		tagList = append(tagList, string(digest))
		buffer = append(buffer, string(digest))
	}
	var manifest config.Manifest
	// читаем манифесты и сканируем зависимости
	// сохраняем манифесты в буфер
	for _, file := range tagList {
		data, err := os.ReadFile(filepath.Join(path, file))
		if err != nil {
			logrus.Error("Error reading file:", err)
		}
		json.Unmarshal(data, &manifest)
		switch manifest.MediaType {
		// стандартные манифесты с сылками на блобы
		case config.MANIFEST_TYPE["docker"]:
			buffer = append(buffer, file)
		// ищем манифесты, в которых ссылки на манифесты мультиплатформенных сборок
		case config.MANIFEST_TYPE["oci"]:
			for _, item := range manifest.Manifests {
				buffer = append(buffer, item.Digest)
			}
		}
	}
	manifests, _ := os.ReadDir(path)
	//заново проходим по директории и удаляет файлы, которые не содержатся в буфере
	for _, file := range manifests {
		if !file.IsDir() {
			if !slices.Contains(buffer, file.Name()) {
				os.Remove(filepath.Join(path, file.Name()))
			}
		}
	}
}
