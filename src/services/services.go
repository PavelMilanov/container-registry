// Package services реализует логику обработки запросов к REST API приложения.
// Выступает промежуточным слоем работы docker api, файловой системой и базой данных.
package services

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/storage"

	"github.com/sirupsen/logrus"
)

// SettingsStore предоставляет сервисам операции с настройками приложения.
type SettingsStore interface {
	GetTagCount(ctx context.Context) (int, error)
	SetTagCount(ctx context.Context, count int) error
}

func AddCloud(name string, store storage.CloudStore) error {
	if err := store.AddCloud(name); err != nil {
		logrus.WithField("name", name).Error(err)
		return errors.New("Ошибка при создании реестра")
	}
	logrus.WithField("name", name).Info("Создано пространство")
	return nil
}

func GetCloudList(store storage.CloudStore) ([]string, error) {
	data, err := store.GetCloudList()
	if err != nil {
		logrus.WithField("task", "clouds").Error(err)
		return data, err
	}
	return data, nil
}

func GetRepositoriesList(cloud string, store storage.RepositoryStore) ([]string, error) {
	data, err := store.GetRepositoriesList(cloud)
	if err != nil {
		logrus.WithField("task", "repositories").Error(err)
		return data, err
	}
	return data, nil
}

func GetImagesList(cloud, repo string, store storage.TagStore) ([]string, error) {
	data, err := store.GetManifestList(cloud, repo)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"cloud":      cloud,
			"repository": repo,
		}).Error(err)
		return data, err
	}
	return data, nil
}

func DeleteCloud(name string, store storage.CloudStore) error {
	if err := store.DeleteCloud(name); err != nil {
		logrus.WithField("name", name).Error(err)
		return err
	}
	logrus.WithField("name", name).Info("Удалено пространство")
	return nil
}

func DeleteImage(cloud, repository, tag string, store storage.TagStore) error {
	if err := store.DeleteManifest(cloud, repository, tag); err != nil {
		logrus.WithFields(logrus.Fields{
			"cloud":      cloud,
			"repository": repository,
			"tag":        tag,
		}).Error(err)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"cloud":      cloud,
		"repository": repository,
		"tag":        tag,
	}).Info("Удален манифест")
	return nil
}

func DeleteRepository(cloud, repository string, store storage.RepositoryStore) error {
	if err := store.DeleteRepository(cloud, repository); err != nil {
		logrus.WithFields(logrus.Fields{
			"cloud":      cloud,
			"repository": repository,
		}).Error(err)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"cloud":      cloud,
		"repository": repository,
	}).Info("Удален репозиторий")
	return nil
}

func SetCountTag(
	ctx context.Context,
	settings SettingsStore,
	count string,
) error {
	newCount, err := strconv.Atoi(count)
	if err != nil {
		logrus.Error(err)
		return err
	}
	if newCount <= 0 {
		logrus.Error("значение tag должно быть больше 0")
		return errors.New("значение tag должно быть больше 0")
	}
	if err := settings.SetTagCount(ctx, newCount); err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

func GetCountTag(
	ctx context.Context,
	settings SettingsStore,
) (int, error) {
	count, err := settings.GetTagCount(ctx)
	if err != nil {
		logrus.Error(err)
		return 0, err
	}
	return count, nil
}

func DeleteOlderTags(
	ctx context.Context,
	settings SettingsStore,
	pruner storage.TagPruner,
) error {
	tagCount, err := settings.GetTagCount(ctx)
	if err != nil {
		return err
	}
	if tagCount <= 0 {
		return errors.New("значение tag_count должно быть больше 0")
	}
	return pruner.DeleteOlderTags(tagCount)
}

/*
SaveManifest - логика сохранения манифеста в хранилище.
*/
func SaveManifest(store storage.ManifestStore, meta config.Meta, body []byte) error {
	manifestPath := filepath.Join(config.MANIFEST_PATH, meta.Repository, meta.Image, meta.Digest)
	if err := store.SaveManifest(meta, body, manifestPath); err != nil {
		logrus.WithField("digest", meta.Digest).Error(err)
		return err
	}
	logrus.WithField("digest", meta.Digest).Info("Загружен манифест")
	return nil
}

func GarbageCollection(collector storage.GarbageCollector) error {
	return collector.GarbageCollection()
}
