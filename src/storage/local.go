package storage

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/sirupsen/logrus"
)

/*
LocalStorage представляет хранилище на основе локальной файловой системы.
*/
type LocalStorage struct {
}

/*
newLocalStorage инициализирует новый экземпляр LocalStorage.
*/
func newLocalStorage() (*LocalStorage, error) {
	if err := os.MkdirAll(config.TMP_PATH, 0755); err != nil {
		return &LocalStorage{}, err
	}
	if err := os.MkdirAll(config.BLOBS_PATH, 0755); err != nil {
		return &LocalStorage{}, err
	}
	if err := os.MkdirAll(config.MANIFEST_PATH, 0755); err != nil {
		return &LocalStorage{}, err
	}
	return &LocalStorage{}, nil
}

/*
CheckBlob проверяет наличие Blob в хранилище.

	uuid - идентификатор Blob.
*/
func (lc *LocalStorage) CheckBlob(uuid string) error {
	path := filepath.Join(config.BLOBS_PATH, strings.Replace(uuid, "sha256:", "", 1))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return errors.New("Blob not found")
	}
	return nil
}

/*
SaveBlob сохраняет Blob в хранилище.

	tmpPath - путь к временному файлу Blob.
	digest - хэш Blob.
*/
func (lc *LocalStorage) SaveBlob(tmpPath, digest string) error {
	if err := os.Rename(tmpPath, filepath.Join(config.BLOBS_PATH, strings.Split(digest, ":")[1])); err != nil {
		return err
	}
	os.Remove(tmpPath)
	return nil
}

/*
GetBlob возвращает Blob из хранилища в двоичном виде.

	digest - хэш Blob.
*/
func (lc *LocalStorage) GetBlob(digest string) (config.Blob, error) {
	var data config.Blob
	encodedDigest := strings.Split(digest, ":")[1]
	blobPath := filepath.Join(config.BLOBS_PATH, encodedDigest)
	file, err := os.Open(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return data, errors.New("Blob not found")
		}
		return data, err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return data, errors.New("Failed to stat blob file")
	}
	data.Digest = digest
	data.Path = blobPath
	data.Size = fileInfo.Size()
	return data, nil
}

/*
SaveManifest сохраняет манифест в хранилище.

	body - содержимое манифеста.
	repository - имя репозитория.
	image - имя образа.
	reference - тег образ.
	calculatedDigest - хэш манифеста.
*/
func (lc *LocalStorage) SaveManifest(meta config.Meta, body []byte, manifestPath string) error {
	tagPath := filepath.Join(config.MANIFEST_PATH, meta.Repository, meta.Image, "tags", meta.Tag)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, body, 0755); err != nil {
		return err
	}
	// Если это тег (а не digest), создаём символическую ссылку
	if !strings.HasPrefix(meta.Tag, "sha256:") {
		if err := os.MkdirAll(filepath.Dir(tagPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(tagPath, []byte(meta.Digest), 0755); err != nil {
			return err
		}
	}
	return nil
}

/*
GetManifest	возращает манифест из хранилища в двоичном виде.

	repository - имя репозитория.
	image - имя образа.
	reference - тег или digest.
*/
func (lc *LocalStorage) GetManifest(repository, image, reference string) ([]byte, error) {
	var manifestPath string
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, image, "tags", reference)
	// Определяем путь к файлу манифеста
	if strings.HasPrefix(reference, "sha256:") {
		// Если reference — это digest
		manifestPath = filepath.Join(config.MANIFEST_PATH, repository, image, reference)
	} else {
		// Если reference — это тег
		tagData, err := os.ReadFile(tagPath)
		if err != nil {
			return []byte{}, errors.New("Tag not found")
		}
		manifestDigest := string(tagData)
		manifestPath = filepath.Join(config.MANIFEST_PATH, repository, image, manifestDigest)
	}
	// Читаем содержимое манифеста
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return []byte{}, errors.New("Manifest not found")
	}
	return data, nil
}

/*
AddRegistry добавляет новый реестр в хранилище.

	registry - имя реестра.
*/
func (lc *LocalStorage) AddCloud(name string) error {
	if err := os.MkdirAll(filepath.Join(config.MANIFEST_PATH, name), 0755); err != nil {
		return err
	}
	return nil
}

/*
DeleteRegistry удаляет реестр из хранилища.

	registry - имя реестра.
*/
func (lc *LocalStorage) DeleteCloud(name string) error {
	if err := os.RemoveAll(filepath.Join(config.MANIFEST_PATH, name)); err != nil {
		return err
	}
	return nil
}

/*
DeleteImage удаляет образ из хранилища.

	repository - имя репозитория.
	imageName - имя образа.
	imageTag - тег образа.
	imageHash - хеш образа.
*/
func (lc *LocalStorage) DeleteManifest(cloud, repository, tag string) error {
	tagPath := filepath.Join(config.MANIFEST_PATH, cloud, repository, "tags", tag)
	data, err := os.ReadFile(tagPath)
	if err != nil {
		return err
	}
	manifestDigest := string(data)
	manifestPath := filepath.Join(config.MANIFEST_PATH, cloud, repository, manifestDigest)
	if err := os.Remove(tagPath); err != nil {
		return err
	}

	// Если оставшиеся теги указывают на этот же digest, файл манифеста удалять нельзя.
	tagsPath := filepath.Join(config.MANIFEST_PATH, cloud, repository, "tags")
	tags, err := os.ReadDir(tagsPath)
	if err == nil {
		for _, item := range tags {
			if item.IsDir() {
				continue
			}
			otherTagPath := filepath.Join(tagsPath, item.Name())
			otherDigest, err := os.ReadFile(otherTagPath)
			if err != nil {
				continue
			}
			if string(otherDigest) == manifestDigest {
				return nil
			}
		}
	}

	if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

/*
DeleteRepository удаляет репозиторий из хранилища.

	cloud - название пространства.
	repository - название репозитория.
*/
func (lc *LocalStorage) DeleteRepository(cloud, repository string) error {
	if err := os.RemoveAll(filepath.Join(config.MANIFEST_PATH, cloud, repository)); err != nil {
		return err
	}
	return nil
}

/*
GarbageCollection выполняет сборку мусора в хранилище.

	Удаляет все образы и слои, которые не используются ни одним реестром.
*/
func (lc *LocalStorage) GarbageCollection() error {
	manifests, err := inventoryManifestsStrict()
	if err != nil {
		return err
	}
	usedBlobs, err := parseUsageBlobsStrict(manifests)
	if err != nil {
		return err
	}
	usedBlobsMap := make(map[string]struct{}, len(usedBlobs))
	for _, b := range usedBlobs {
		usedBlobsMap[b] = struct{}{}
	}
	deleted := 0

	blobs, err := inventoryBlobsStrict()
	if err != nil {
		return err
	}
	for _, blob := range blobs {
		if _, ok := usedBlobsMap[blob]; ok {
			continue
		}

		if err := os.Remove(blob); err != nil {
			logrus.WithField("GarbageCollection", "error").WithError(err).Error()
			continue
		}

		deleted++
	}
	logrus.WithField("GarbageCollection", "deleted").Infof("Удалено %d файлов", deleted)
	return nil
}

func (*LocalStorage) GetManifestList(cloud, repository string) ([]string, error) {
	tagPath := filepath.Join(config.MANIFEST_PATH, cloud, repository, "tags")
	files, err := os.ReadDir(tagPath)
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, file := range files {
		tags = append(tags, file.Name())
	}
	return tags, nil
}

/* Возвращает список пространств */
func (*LocalStorage) GetCloudList() ([]string, error) {
	var cloud []string
	dirs, err := os.ReadDir(config.MANIFEST_PATH)
	if err != nil {
		return cloud, err
	}
	for _, dir := range dirs {
		if dir.IsDir() {
			cloud = append(cloud, dir.Name())
		}
	}
	return cloud, nil
}

func (*LocalStorage) GetRepositoriesList(cloud string) ([]string, error) {
	var data []string
	repos, err := os.ReadDir(filepath.Join(config.MANIFEST_PATH, cloud))
	if err != nil {
		return data, errors.New(cloud + " не найден")
	}
	for _, repo := range repos {
		data = append(data, repo.Name())
	}
	return data, nil
}

func (lc *LocalStorage) DeleteOlderTags(count int) error {
	type tagMeta struct {
		Name    string
		ModTime int64
	}

	deleted := 0
	clouds, err := lc.GetCloudList()
	if err != nil {
		return err
	}

	for _, cloud := range clouds {
		repositories, err := lc.GetRepositoriesList(cloud)
		if err != nil {
			logrus.WithField("cloud", cloud).WithError(err).Warn("не удалось получить список репозиториев")
			continue
		}
		for _, repository := range repositories {
			tags, err := lc.GetManifestList(cloud, repository)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"cloud":      cloud,
					"repository": repository,
				}).WithError(err).Warn("не удалось получить список тегов")
				continue
			}
			if len(tags) <= count {
				continue
			}

			withMeta := make([]tagMeta, 0, len(tags))
			for _, tag := range tags {
				tagPath := filepath.Join(config.MANIFEST_PATH, cloud, repository, "tags", tag)
				info, err := os.Stat(tagPath)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"cloud":      cloud,
						"repository": repository,
						"tag":        tag,
					}).WithError(err).Warn("не удалось прочитать метаданные тега")
					continue
				}
				withMeta = append(withMeta, tagMeta{Name: tag, ModTime: info.ModTime().UnixNano()})
			}

			if len(withMeta) <= count {
				continue
			}

			sort.Slice(withMeta, func(i, j int) bool {
				return withMeta[i].ModTime > withMeta[j].ModTime
			})

			for i := count; i < len(withMeta); i++ {
				tag := withMeta[i].Name
				if err := lc.DeleteManifest(cloud, repository, tag); err != nil {
					logrus.WithFields(logrus.Fields{
						"cloud":      cloud,
						"repository": repository,
						"tag":        tag,
					}).WithError(err).Warn("не удалось удалить старый тег")
					continue
				}
				deleted++
			}
		}
	}
	logrus.WithField("deleted_tags", deleted).Info("Удалены старые теги")
	return nil
}
