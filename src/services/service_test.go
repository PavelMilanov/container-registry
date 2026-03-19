package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveManifest(t *testing.T) {
	manifestDescriptor := struct {
		Schema int    `json:"schemaVersion"`
		Type   string `json:"mediaType"`
		Config struct {
			Digest string `json:"digest"`
		} `json:"config"`
		Manifests []struct {
			Digest   string `json:"digest"`
			Platform struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
			} `json:"platform"`
		} `json:"manifests"`
		Layers []struct {
			Size int64 `json:"size"`
		} `json:"layers"`
	}{}
	path := "../var/manifests/test"
	dirs, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		if dir.IsDir() {
			files, err := os.ReadDir(filepath.Join(path, dir.Name()))
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range files {
				if !file.IsDir() {
					fmt.Println("Name:", file.Name())
					data, err := os.ReadFile(filepath.Join(path, dir.Name(), file.Name()))
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(data, &manifestDescriptor); err != nil {
						t.Error(err)
					}
					fmt.Println("Data:", manifestDescriptor)
				}
			}
		}
	}
}

// func TestDeleteOlderImages(t *testing.T) {
// 	env, err := config.NewEnv("../conf.d", "config")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	s, err := storage.NewStorage(env)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	sqlite, err := db.NewDatabase("../var/registry.db", env)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	DeleteOlderImages(sqlite.Sql, s)
// }
