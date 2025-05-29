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
	if err := os.Rename(tmpPath, filepath.Join(config.BLOBS_PATH, strings.Replace(digest, "sha256:", "", 1))); err != nil {
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
	digest = strings.Replace(digest, "sha256:", "", 1)
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
func (lc *LocalStorage) SaveManifest(body []byte, repository, image, reference, calculatedDigest string) (string, error) {
	manifestPath := filepath.Join(config.MANIFEST_PATH, repository, image, calculatedDigest)
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, image, "tags", reference)
	err := os.MkdirAll(filepath.Dir(manifestPath), 0755)
	if err != nil {
		return "", errors.New("Failed to create manifest directory")
	}
	err = os.WriteFile(manifestPath, body, 0644)
	if err != nil {
		return "", errors.New("Failed to save manifest")
	}
	// Если это тег (а не digest), создаём символическую ссылку
	if !strings.HasPrefix(reference, "sha256:") {
		err = os.MkdirAll(filepath.Dir(tagPath), 0755)
		if err != nil {
			return "", errors.New("Failed to create tag directory")
		}
		err = os.WriteFile(tagPath, []byte(calculatedDigest), 0644)
		if err != nil {
			return "", errors.New("Failed to save tag reference")
		}
	}
	return manifestPath, nil
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
func (lc *LocalStorage) AddRegistry(registry string) error {
	if err := os.MkdirAll(filepath.Join(config.MANIFEST_PATH, registry), 0755); err != nil {
		return err
	}
	return nil
}

/*
DeleteRegistry удаляет реестр из хранилища.

	registry - имя реестра.
*/
func (lc *LocalStorage) DeleteRegistry(registry string) error {
	if err := os.RemoveAll(filepath.Join(config.MANIFEST_PATH, registry)); err != nil {
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
	for _, i := range buffer {
		if err := os.Remove(filepath.Join(config.BLOBS_PATH, i)); err != nil {
			logrus.Error(err)
		}
	}
	logrus.Infof("Инвентаризация blob произведена. Удалено файлов %d", len(buffer))
}
