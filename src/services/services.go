// Package services реализует логику обработки запросов к REST API приложения.
package services

import (
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/PavelMilanov/container-registry/system"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func AddRegistry(name string, sql *gorm.DB) error {
	registry := db.Registry{Name: name}
	if err := registry.Add(sql); err != nil {
		return err
	}
	return nil
}

func DeleteRegistry(data string, sql *gorm.DB, storage *storage.Storage) error {
	if err := storage.DeleteRegistry(data); err != nil {
		return err
	}
	registy := db.Registry{Name: data}
	if err := registy.Delete(sql); err != nil {
		return err
	}
	return nil
}

func DeleteImage(name, image, tag string, sql *gorm.DB, storage *storage.Storage) error {
	img := db.Image{Name: image, Tag: tag}
	sql.Transaction(func(tx *gorm.DB) error {
		if err := img.Delete(tx); err != nil {
			tx.Rollback()
			return err
		}
		repo, _ := db.GetRepository(tx, "ID = ?", img.RepositoryID)
		registry, _ := db.GetRegistry(tx, "ID = ?", repo.RegistryID)
		repo.Size -= img.Size
		repo.SizeAlias = system.ConvertSize(repo.Size)
		if err := repo.UpdateSize(tx); err != nil {
			tx.Rollback()
			return err
		}
		registry.Size -= repo.Size
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

func DeleteRepository(name, image string, sql *gorm.DB, storage *storage.Storage) error {
	repo := db.Repository{Name: image}
	sql.Transaction(func(tx *gorm.DB) error {
		if err := repo.Delete(tx); err != nil {
			tx.Rollback()
			return err
		}
		registry, _ := db.GetRegistry(tx, "ID = ?", repo.RegistryID)
		registry.Size -= repo.Size
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
func DeleteOlderImages(sql *gorm.DB, storage *storage.Storage) {
	tagCount, err := db.GetCountTag(sql)
	if err != nil {
		logrus.Printf("Failed to get tag count: %v", err)
		return
	}
	data := db.GetLastTagImages(sql, tagCount)
	for _, item := range data {
		repo, _ := db.GetRepository(sql, "ID = ?", item.RepositoryID)
		storage.DeleteImage(repo.Name, item.Name, item.Tag, item.Hash)
		sql.Delete(&item)
	}
}
