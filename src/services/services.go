// Package services реализует логику обработки запросов к REST API приложения.
package services

import (
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/storage"
	"gorm.io/gorm"
)

func AddRegistry(name string, sql *gorm.DB) error {
	registry := db.Registry{Name: name}
	if err := registry.Add(sql); err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return err
	}
	return nil
}

func DeleteRegistry(data string, sql *gorm.DB, storage *storage.Storage) error {
	if err := storage.DeleteRegistry(data); err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return err
	}
	registy := db.Registry{Name: data}
	if err := registy.Delete(sql); err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return err
	}
	return nil
}

func DeleteImage(name, image, tag string, sql *gorm.DB, storage *storage.Storage) error {
	img := db.Image{Name: image, Tag: tag}
	if err := img.Delete(sql); err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return err
	}
	if err := storage.DeleteImage(name, img.Name, img.Tag, img.Hash); err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return err
	}
	return nil
}

func DeleteRepository(name, image string, sql *gorm.DB, storage *storage.Storage) error {
	repo := db.Repository{Name: image}
	if err := repo.Delete(sql); err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return err
	}
	if err := storage.DeleteRepository(name, repo.Name); err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return err
	}
	return nil
}

func GetImages(image string, sql *gorm.DB) []db.Image {
	repo := db.GetRepository(sql, image)
	data := db.GetImageTags(sql, repo.ID, image)
	return data
}

func DeleteOldestImages() {
	
}
