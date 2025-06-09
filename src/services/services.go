// Package services реализует логику обработки запросов к REST API приложения.
package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/PavelMilanov/container-registry/system"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func AddRegistry(name string, sql *gorm.DB, storage storage.Storage) error {
	if err := storage.AddRegistry(name); err != nil {
		return err
	}
	registry := db.Registry{Name: name}
	if err := registry.Add(sql); err != nil {
		return err
	}
	return nil
}

func DeleteRegistry(name string, sql *gorm.DB, storage storage.Storage) error {
	if err := storage.DeleteRegistry(name); err != nil {
		return err
	}
	registy := db.Registry{Name: name}
	if err := registy.Delete(sql); err != nil {
		return err
	}
	return nil
}

func DeleteImage(name, image, tag string, sql *gorm.DB, storage storage.Storage) error {
	img := db.Image{Name: image, Tag: tag}
	sql.Transaction(func(tx *gorm.DB) error {
		if err := img.Delete(tx); err != nil {
			tx.Rollback()
			return err
		}
		imgSize := img.GetSize(tx, "repository_id = ?", img.RepositoryID)
		repo, _ := db.GetRepository(tx, "ID = ?", img.RepositoryID)
		repo.Size = imgSize
		repo.SizeAlias = system.ConvertSize(repo.Size)
		if err := repo.UpdateSize(tx); err != nil {
			tx.Rollback()
			return err
		}
		repoSize := repo.GetSize(tx, "registry_id = ?", repo.RegistryID)
		registry, _ := db.GetRegistry(tx, "ID = ?", repo.RegistryID)
		registry.Size = repoSize
		registry.SizeAlias = system.ConvertSize(registry.Size)
		if err := registry.UpdateSize(tx); err != nil {
			tx.Rollback()
			return err
		}
		return nil
	})
	if err := storage.DeleteImage(name, img.Name, img.Tag, img.Hash); err != nil {
		return err
	}
	return nil
}

func DeleteRepository(name, image string, sql *gorm.DB, storage storage.Storage) error {
	repo := db.Repository{Name: image}
	sql.Transaction(func(tx *gorm.DB) error {
		if err := repo.Delete(tx); err != nil {
			tx.Rollback()
			return err
		}
		repoSize := repo.GetSize(tx, "registry_id = ?", repo.RegistryID)
		registry, _ := db.GetRegistry(tx, "ID = ?", repo.RegistryID)
		registry.Size = repoSize
		registry.SizeAlias = system.ConvertSize(registry.Size)
		if err := registry.UpdateSize(tx); err != nil {
			tx.Rollback()
			return err
		}
		return nil
	})

	if err := storage.DeleteRepository(name, repo.Name); err != nil {
		return err
	}
	return nil
}

func GetImages(image string, sql *gorm.DB) []db.Image {
	repo, _ := db.GetRepository(sql, "name = ?", image)
	data := db.GetImageTags(sql, repo.ID, image)
	return data
}

// DeleteOlderImages удаляет старые образы из базы данных и хранилища.
func DeleteOlderImages(sql *gorm.DB, storage storage.Storage) {
	tagCount, err := db.GetCountTag(sql)
	if err != nil {
		logrus.Printf("Не найден тег: %v", err)
		return
	}
	data := db.GetLastTagImages(sql, tagCount)
	for _, item := range data {
		repo, _ := db.GetRepository(sql, "ID = ?", item.RepositoryID)
		DeleteImage(repo.Name, item.Name, item.Tag, sql, storage)
	}
}

// SaveManifestToDB сохраняет манифест в базу данных и обновляет зависимости.
// mediaType - тип медиа-файла (например, "application/vnd.docker.distribution.manifest.v2+json").
// link - ссылка на манифест.
// tag - тег образа.
// sql - экземпляр базы данных.
func SaveManifestToDB(mediaType, link, tag string, sql *gorm.DB) error {
	resizeRegistry := func(repository, imageName, manifestFile, platform string, sum int64) {
		registry, err := db.GetRegistry(sql, "name = ?", repository)
		if err != nil {
			logrus.Error(err)
		}
		repo := db.Repository{
			Name:       imageName,
			RegistryID: registry.ID,
		}
		repo.Add(sql)
		image := db.Image{
			Name:         imageName,
			Hash:         manifestFile,
			Tag:          tag,
			Platform:     platform,
			Size:         sum,
			SizeAlias:    system.ConvertSize(sum),
			RepositoryID: repo.ID,
		}
		image.Add(sql)
		imgSize := image.GetSize(sql, "repository_id = ?", image.RepositoryID)
		repo.Size = imgSize
		repo.SizeAlias = system.ConvertSize(repo.Size)
		repo.UpdateSize(sql)
		repoSize := repo.GetSize(sql, "registry_id = ?", repo.RegistryID)
		registry.Size = repoSize
		registry.SizeAlias = system.ConvertSize(registry.Size)
		registry.UpdateSize(sql)
	}
	path, manifestFile := filepath.Split(link) // var/manifests/dev/alpine/ sha256:33fe5b4ced5027766381b0c5578efa7217c5cc4498b10d1ab7275182197933c8
	repository := strings.Split(path, "/")[2]
	imageName := strings.Split(path, "/")[3]
	var manifest config.Manifest
	body, err := os.ReadFile(link)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		return err
	}
	switch mediaType {
	case config.MANIFEST_TYPE["docker"]:
		var sum int64
		for _, descriptor := range manifest.Layers {
			sum += descriptor.Size
		}
		resizeRegistry(repository, imageName, manifestFile, "docker", sum)
	case config.MANIFEST_TYPE["oci"]:
		platforms := []string{}
		sizes := []int64{}
		for _, item := range manifest.Manifests {
			// ищем манифесты с описанием слоев образов
			// может быть несколько, если была мультиплатформенная сборка
			if item.Platform.Architecture != "unknown" {
				platforms = append(platforms, item.Platform.OS+"/"+item.Platform.Architecture)
				body, err := os.ReadFile(path + item.Digest)
				if err != nil {
					return err
				}
				var m2 config.Manifest
				if err := json.Unmarshal(body, &m2); err != nil {
					return err
				}
				var sum int64
				for _, descriptor := range m2.Layers {
					sum += descriptor.Size
				}
				sizes = append(sizes, sum)
			}
		}
		resizeRegistry(repository, imageName, manifestFile, strings.Join(platforms, ","), sizes[0])
	}
	return nil
}
