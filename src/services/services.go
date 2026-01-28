// Package services реализует логику обработки запросов к REST API приложения.
// Выступает промежуточным слоем работы docker api, файловой системой и базой данных.
package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/storage"

	"github.com/PavelMilanov/container-registry/system"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func AddRegistry(name string, sql *gorm.DB, storage storage.Storage) error {
	registry := db.Registry{Name: name}
	if err := registry.Add(sql); err != nil {
		logrus.Error(err)
		return errors.New("Ошибка при создании реестра")
	}
	if err := storage.AddRegistry(name); err != nil {
		logrus.Error(err)
		return errors.New("Ошибка при создании реестра")
	}
	logrus.WithFields(logrus.Fields{
		"name": registry.Name,
	}).Info("Создан новый реестр")
	return nil
}

func GetRegistries(sql *gorm.DB) ([]db.Registry, error) {
	data, err := db.GetRegistires(sql)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return data, nil
}

func DeleteRegistry(name string, sql *gorm.DB, storage storage.Storage) error {
	if err := storage.DeleteRegistry(name); err != nil {
		logrus.Error(err)
		return err
	}
	registy := db.Registry{Name: name}
	if err := registy.Delete(sql); err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Infof("Удален реестр %+v", registy)
	return nil
}

func DeleteImage(name, image, hash string, sql *gorm.DB, storage storage.Storage) error {
	img := db.Image{Name: image, Hash: hash}
	err := sql.Transaction(func(tx *gorm.DB) error {
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
	if err != nil {
		logrus.Error(err)
		return err
	}
	if err := storage.DeleteImage(name, img.Name, img.Tag, img.Hash); err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

func DeleteRepository(name, image string, sql *gorm.DB, storage storage.Storage) error {
	repo, err := db.GetRepository(sql, "name = ?", image)
	if err != nil {
		logrus.Error(err)
		return err
	}
	if err := sql.Transaction(func(tx *gorm.DB) error {
		repoSize := repo.GetSize(tx, "registry_id = ?", repo.RegistryID)
		if err := repo.Delete(tx); err != nil {
			tx.Rollback()
			logrus.Error(err)
			return err
		}
		registry, _ := db.GetRegistry(tx, "ID = ?", repo.RegistryID)
		registry.Size = repoSize
		registry.SizeAlias = system.ConvertSize(registry.Size)
		if err := registry.UpdateSize(tx); err != nil {
			tx.Rollback()
			logrus.Error(err)
			return err
		}
		return nil
	}); err != nil {
		logrus.Error(err)
		return err
	}
	if err := storage.DeleteRepository(name, repo.Name); err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Infof("Удален репозиторий %+v", repo)
	return nil
}

func GetRepositories(sql *gorm.DB, name string) ([]db.Repository, error) {
	var registry db.Registry
	if err := registry.GetRepositories(sql, name); err != nil {
		logrus.Error(err)
		return registry.Repositories, err
	}
	return registry.Repositories, nil
}

func GetImages(image string, sql *gorm.DB) ([]db.Image, error) {
	repo, err := db.GetRepository(sql, "name = ?", image)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	data := db.GetImageTags(sql, repo.ID, image)
	return data, nil
}

// DeleteOlderImages удаляет старые образы из базы данных и хранилища.
func DeleteOlderImages(sql *gorm.DB, storage storage.Storage) {
	tagCount, err := db.GetCountTag(sql)
	if err != nil {
		logrus.Errorf("Не найден тег: %v", err)
		return
	}
	data, err := db.GetLastTagImages(sql, tagCount)
	if err != nil {
		logrus.Error(err)
		return
	}
	statBefore, err := storage.DiskUsage()
	if err != nil {
		logrus.Errorf("Ошибка получения информации о дисковом пространстве: %v", err)
		return
	}
	for _, item := range data {
		repo, _ := db.GetRepository(sql, "ID = ?", item.RepositoryID)
		DeleteImage(repo.Name, item.Name, item.Hash, sql, storage)
	}
	statAfter, err := storage.DiskUsage()
	if err != nil {
		logrus.Errorf("Ошибка получения информации о дисковом пространстве: %v", err)
		return
	}
	clearSpace := statBefore.Used - statAfter.Used
	logrus.Infof("Удалено %d старых образов\nОчищено пространства %s", len(data), system.HumanizeSize(clearSpace))
}

/*
Registration- реализация регистрации пользователя.

	При успешной регистрации ничего не возвращает.
*/
func Registration(sql *gorm.DB, username, password string) error {
	user := db.User{Name: username, Password: password}
	if err := user.Add(sql); err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

/*
Login - реализация авторизации пользователя.

	При успешной авторизации возвращает токен пользователя.
*/
func Login(sql *gorm.DB, cred *config.Env, username, password string) (string, error) {
	user := db.User{Name: username, Password: password}
	if err := user.Login(sql, cred); err != nil {
		logrus.Error(err)
		return user.Token, err
	}
	return user.Token, nil
}

func GetSettings(sql *gorm.DB, storage storage.Storage) (Settings, error) {
	count, err := db.GetCountTag(sql)
	if err != nil {
		logrus.Error(err)
		return Settings{}, err
	}
	diskStat, err := storage.DiskUsage()
	if err != nil {
		logrus.Error(err)
		return Settings{}, err
	}
	return Settings{
		Count:         count,
		Total:         system.HumanizeSize(diskStat.Total),
		Used:          system.HumanizeSize(diskStat.Used),
		UsedToPercent: int(diskStat.UsedToPercent),
		Version:       config.VERSION,
	}, nil
}

func SetCountTag(sql *gorm.DB, count string) error {
	newCount, err := strconv.Atoi(count)
	if err != nil {
		logrus.Error(err)
		return err
	}
	if err := db.SetCountTag(sql, newCount); err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

/*
SaveManifest - логика сохранения манифеста в базу данных и хранилище.
*/
func SaveManifest(sql *gorm.DB, storage storage.Storage, meta config.Meta, body []byte) error {
	reader := bufio.NewReader(bytes.NewBuffer(body))
	manifestPath := filepath.Join(config.MANIFEST_PATH, meta.Repository, meta.Image, meta.Digest)
	var manifest config.Manifest

	data, err := io.ReadAll(reader)
	if err != nil {
		logrus.Error(err)
		return err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		logrus.Error(err)
		return err
	}
	switch manifest.MediaType {
	case config.MANIFEST_TYPE["docker"]:
		var sum int64
		for _, descriptor := range manifest.Layers {
			blob, _ := storage.GetBlob(descriptor.Digest)
			sum += blob.Size + descriptor.Size
		}
		meta.Size = sum
		meta.Platform = "docker"
	case config.MANIFEST_TYPE["oci"]:
		platforms := []string{}
		for _, item := range manifest.Manifests {
			// ищем манифесты с описанием слоев образов
			// может быть несколько, если была мультиплатформенная сборка
			if item.Platform.Architecture != "unknown" {
				platforms = append(platforms, item.Platform.OS+"/"+item.Platform.Architecture)
				var m2 config.Manifest
				if err := json.Unmarshal(data, &m2); err != nil {
					logrus.Error(err)
					return err
				}
				var sum int64
				for _, descriptor := range m2.Layers {
					blob, _ := storage.GetBlob(descriptor.Digest)
					sum += blob.Size + descriptor.Size
				}
				meta.Size = sum
				meta.Platform = strings.Join(platforms, ",")
			}
		}
	}
	registry, err := db.GetRegistry(sql, "name = ?", meta.Repository)
	if err != nil {
		logrus.Error(err)
	}
	repo := db.Repository{
		Name:       meta.Image,
		RegistryID: registry.ID,
	}
	if err := repo.Add(sql); err != nil {
		logrus.Error(err)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"name": repo.Name,
	}).Info("Создан новый репозиторий")

	image := db.Image{
		Name:         meta.Image,
		Hash:         meta.Digest,
		Tag:          meta.Tag,
		Platform:     meta.Platform,
		Size:         meta.Size,
		SizeAlias:    system.ConvertSize(meta.Size),
		RepositoryID: repo.ID,
	}
	if err := image.Add(sql); err != nil {
		logrus.Error(err)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"image": image.Name,
		"tag":   image.Tag,
	}).Info("Создан новый образ")
	if err := storage.SaveManifest(meta, body, manifestPath); err != nil {
		logrus.Error(err)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"manifest": meta.Digest,
	}).Info("Загружен манифест")

	imgSize := image.GetSize(sql, "repository_id = ?", image.RepositoryID)
	repo.Size = imgSize
	repo.SizeAlias = system.ConvertSize(repo.Size)
	repo.UpdateSize(sql)
	repoSize := repo.GetSize(sql, "registry_id = ?", repo.RegistryID)
	registry.Size = repoSize
	registry.SizeAlias = system.ConvertSize(registry.Size)
	registry.UpdateSize(sql)
	return nil
}
