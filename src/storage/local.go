package storage

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
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
	digest = strings.Split(digest, ":")[1]
	blobPath := filepath.Join(config.BLOBS_PATH, digest)
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
		return errors.New("Не удалось создать директорию для манифеста")
	}
	if err := os.WriteFile(manifestPath, body, 0644); err != nil {
		return errors.New("Не удалось сохранить файл манифеста")
	}
	// Если это тег (а не digest), создаём символическую ссылку
	if !strings.HasPrefix(meta.Tag, "sha256:") {
		if err := os.MkdirAll(filepath.Dir(tagPath), 0755); err != nil {
			return errors.New("Не удалось создать директорию для тега")
		}
		if err := os.WriteFile(tagPath, []byte(meta.Digest), 0644); err != nil {
			return errors.New("Не удалось сохранить файл тега")
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
func (lc *LocalStorage) DeleteImage(repository, imageName, imageTag, imageHash string) error {
	path := filepath.Join(config.MANIFEST_PATH, repository, imageName, imageHash)
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, imageName, "tags", imageTag)
	if err := os.Remove(tagPath); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return nil
}

/*
DeleteRepository удаляет репозиторий из хранилища.

	name - имя репозитория.
	image - имя образа.
*/
func (lc *LocalStorage) DeleteRepository(name, image string) error {
	if err := os.RemoveAll(filepath.Join(config.MANIFEST_PATH, name, image)); err != nil {
		return err
	}
	return nil
}

/*
GarbageCollection выполняет сборку мусора в хранилище.

	Удаляет все образы и слои, которые не используются ни одним реестром.
*/
func (lc *LocalStorage) GarbageCollection() {
	// получаем список всех blob.
	blobs := func() []string {
		var blobs []string
		digests, _ := os.ReadDir(config.BLOBS_PATH)
		for _, blob := range digests {
			blobs = append(blobs, blob.Name())
		}
		return blobs
	}()
	actualBlobs := inventoryBlobs()
	var buffer []string
	for _, v := range blobs {
		if !slices.Contains(actualBlobs, v) {
			buffer = append(buffer, v)
		}
	}
	// statBefore, err := lc.DiskUsage()
	// if err != nil {
	// 	logrus.Printf("Ошибка получения информации о дисковом пространстве: %v", err)
	// 	return
	// }
	for _, i := range buffer {
		if err := os.Remove(filepath.Join(config.BLOBS_PATH, i)); err != nil {
			logrus.Error(err)
		}
	}
	// statAfter, err := lc.DiskUsage()
	// if err != nil {
	// 	logrus.Printf("Ошибка получения информации о дисковом пространстве: %v", err)
	// 	return
	// }
	// clearSpace := statBefore.Used - statAfter.Used
	// logrus.Infof("Инвентаризация blob произведена. Удалено файлов %d\nОчищено пространства %s", len(buffer), system.HumanizeSize(clearSpace))
}

func (*LocalStorage) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (*LocalStorage) GetManifestList(repository, image string) ([]string, error) {
	return nil, nil
}

/* Возвращает список пространств */
func (*LocalStorage) GetCloudList() ([]string, error) {
	dirs, err := os.ReadDir(config.MANIFEST_PATH)
	if err != nil {
		return nil, err
	}
	var cloud []string
	for _, dir := range dirs {
		if dir.IsDir() {
			cloud = append(cloud, dir.Name())
		}
	}
	return cloud, nil
}
