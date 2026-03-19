package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestInventoryBlobs(t *testing.T) {
	buffer := []string{}
	path := "../var/tmp/blobs"
	blobs, _ := os.ReadDir(path)
	for _, blob := range blobs {
		buffer = append(buffer, filepath.Join(path, blob.Name()))
	}
	fmt.Println("Количество blobs: ", len(buffer))
}

func TestInventoryManifests(t *testing.T) {
	buffer := []string{}
	path := "../var/tmp/manifests"
	clouds, _ := os.ReadDir(path)
	for _, cloud := range clouds {
		cloudPath := filepath.Join(path, cloud.Name())
		repositories, _ := os.ReadDir(cloudPath)
		for _, repo := range repositories {
			repoPath := filepath.Join(cloudPath, repo.Name())
			fmt.Println("Чтение директории: ", repoPath)
			activeTags := parseActiveTags(repoPath)
			manifests := parseManifests(repoPath)
			for _, manifest := range manifests {
				found := slices.Contains(activeTags, manifest)
				if !found {
					fmt.Println("Удаляем неиспользуемый манифест: ", manifest)
					continue
				}
				buffer = append(buffer, manifest)
			}
		}
	}
	fmt.Println("Количество манифестов: ", len(buffer))
}
