// Package services реализует логику обработки запросов к REST API приложения.
// Выступает промежуточным слоем работы docker api, файловой системой и базой данных.
package services

import (
	"errors"
	"path/filepath"
	"strconv"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/storage"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func AddCloud(name string, storage storage.Storage) error {
	if err := storage.AddCloud(name); err != nil {
		logrus.WithFields(logrus.Fields{
			"name": name,
		}).Error(err)
		return errors.New("Ошибка при создании реестра")
	}
	logrus.WithFields(logrus.Fields{
		"name": name,
	}).Info("Создано пространство")
	return nil
}

func GetCloudList(storage storage.Storage) ([]string, error) {
	data, err := storage.GetCloudList()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"name": "clouds",
		}).Error(err)
		return data, err
	}
	return data, nil
}

func GetRepositoriesList(cloud string, storage storage.Storage) ([]string, error) {
	data, err := storage.GetRepositoriesList(cloud)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"name": "repositories",
		}).Error(err)
		return data, err
	}
	return data, nil
}

func GetImagesList(cloud, repo string, storage storage.Storage) ([]string, error) {
	data, err := storage.GetManifestList(cloud, repo)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"cloud":      cloud,
			"repository": repo,
		}).Error(err)
		return data, err
	}
	return data, nil
}

func DeleteCloud(name string, storage storage.Storage) error {
	if err := storage.DeleteCloud(name); err != nil {
		logrus.WithFields(logrus.Fields{
			"name": name,
		}).Error(err)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"name": name,
	}).Info("Удалено пространство")
	return nil
}

func DeleteImage(cloud, repository, tag string, storage storage.Storage) error {
	if err := storage.DeleteManifest(cloud, repository, tag); err != nil {
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

// func DeleteRepository(name, image string, sql *gorm.DB, storage storage.Storage) error {
// 	repo, err := db.GetRepository(sql, "name = ?", image)
// 	if err != nil {
// 		logrus.Error(err)
// 		return err
// 	}
// 	if err := sql.Transaction(func(tx *gorm.DB) error {
// 		repoSize := repo.GetSize(tx, "registry_id = ?", repo.RegistryID)
// 		if err := repo.Delete(tx); err != nil {
// 			tx.Rollback()
// 			logrus.Error(err)
// 			return err
// 		}
// 		registry, _ := db.GetRegistry(tx, "ID = ?", repo.RegistryID)
// 		registry.Size = repoSize
// 		registry.SizeAlias = system.ConvertSize(registry.Size)
// 		if err := registry.UpdateSize(tx); err != nil {
// 			tx.Rollback()
// 			logrus.Error(err)
// 			return err
// 		}
// 		return nil
// 	}); err != nil {
// 		logrus.Error(err)
// 		return err
// 	}
// 	if err := storage.DeleteRepository(name, repo.Name); err != nil {
// 		logrus.Error(err)
// 		return err
// 	}
// 	logrus.Infof("Удален репозиторий %+v", repo)
// 	return nil
// }

// func GetRepositories(sql *gorm.DB, name string) ([]db.Repository, error) {
// 	var registry db.Registry
// 	if err := registry.GetRepositories(sql, name); err != nil {
// 		logrus.Error(err)
// 		return registry.Repositories, err
// 	}
// 	return registry.Repositories, nil
// }

// func GetImages(image string, sql *gorm.DB) ([]db.Image, error) {
// 	repo, err := db.GetRepository(sql, "name = ?", image)
// 	if err != nil {
// 		logrus.Error(err)
// 		return nil, err
// 	}
// 	data := db.GetImageTags(sql, repo.ID, image)
// 	return data, nil
// }

// DeleteOlderImages удаляет старые образы из базы данных и хранилища.
// func DeleteOlderImages(sql *gorm.DB, storage storage.Storage) {
// 	tagCount, err := db.GetCountTag(sql)
// 	if err != nil {
// 		logrus.Errorf("Не найден тег: %v", err)
// 		return
// 	}
// 	data, err := db.GetLastTagImages(sql, tagCount)
// 	if err != nil {
// 		logrus.Error(err)
// 		return
// 	}
// 	// statBefore, err := storage.DiskUsage()
// 	if err != nil {
// 		logrus.Errorf("Ошибка получения информации о дисковом пространстве: %v", err)
// 		return
// 	}
// 	for _, item := range data {
// 		repo, _ := db.GetRepository(sql, "ID = ?", item.RepositoryID)
// 		DeleteImage(repo.Name, item.Name, item.Hash, sql, storage)
// 	}
// 	// statAfter, err := storage.DiskUsage()
// 	if err != nil {
// 		logrus.Errorf("Ошибка получения информации о дисковом пространстве: %v", err)
// 		return
// 	}
// 	// clearSpace := statBefore.Used - statAfter.Used
// 	// logrus.Infof("Удалено %d старых образов\nОчищено пространства %s", len(data), system.HumanizeSize(clearSpace))
// }

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
		logrus.WithFields(logrus.Fields{
			"username": username,
		}).Error(err)
		return user.Token, err
	}
	logrus.WithFields(logrus.Fields{
		"username": username,
	}).Info("Успешная авторизация")
	return user.Token, nil
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
SaveManifest - логика сохранения манифеста в хранилище.
*/
func SaveManifest(storage storage.Storage, meta config.Meta, body []byte) error {
	manifestPath := filepath.Join(config.MANIFEST_PATH, meta.Repository, meta.Image, meta.Digest)
	if err := storage.SaveManifest(meta, body, manifestPath); err != nil {
		logrus.WithFields(logrus.Fields{
			"digest": meta.Digest,
		}).Error(err)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"digest": meta.Digest,
	}).Info("Загружен манифест")
	// reader := bufio.NewReader(bytes.NewBuffer(body))
	// manifestDescriptor := struct {
	// 	Schema int    `json:"schemaVersion"`
	// 	Type   string `json:"mediaType"`
	// 	Config struct {
	// 		Digest string `json:"digest"`
	// 	} `json:"config"`
	// 	Manifests []struct {
	// 		Digest   string `json:"digest"`
	// 		Platform struct {
	// 			Architecture string `json:"architecture"`
	// 			OS           string `json:"os"`
	// 		} `json:"platform"`
	// 	} `json:"manifests"`
	// 	Layers []struct {
	// 		Size int64 `json:"size"`
	// 	} `json:"layers"`
	// }{}
	// data, err := io.ReadAll(reader)
	// if err != nil {
	// 	logrus.Error(err)
	// 	return err
	// }
	// if err := json.Unmarshal(data, &manifestDescriptor); err != nil {
	// 	logrus.Error(err)
	// 	return err
	// }
	return nil
}
