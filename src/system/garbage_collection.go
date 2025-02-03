package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelMilanov/container-registry/storage"
)

func GarbageCollection() {
	var digests = []string{}
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
	fmt.Println(digests)
}
