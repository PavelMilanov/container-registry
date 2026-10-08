package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/sirupsen/logrus"
)

/*
LocalStorage представляет хранилище на основе локальной файловой системы.
*/
type LocalStorage struct {
	uploadLocks uploadLockManager
	maintenance maintenanceGate
}

var _ BlobUploadStore = (*LocalStorage)(nil)
var _ UploadCleaner = (*LocalStorage)(nil)
var _ NamespaceStore = (*LocalStorage)(nil)
var _ BlobStore = (*LocalStorage)(nil)
var _ ManifestStore = (*LocalStorage)(nil)
var _ CloudStore = (*LocalStorage)(nil)
var _ RepositoryStore = (*LocalStorage)(nil)
var _ TagStore = (*LocalStorage)(nil)
var _ GarbageCollector = (*LocalStorage)(nil)
var _ TagPruner = (*LocalStorage)(nil)

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
NamespaceExists проверяет наличие пространства имён registry.

	ctx - контекст выполнения операции.
	name - имя пространства.
*/
func (*LocalStorage) NamespaceExists(
	ctx context.Context,
	name string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !validNamespaceName(name) {
		return false, nil
	}

	info, err := os.Stat(filepath.Join(config.MANIFEST_PATH, name))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf(
			"не удалось проверить пространство %s: %w",
			name,
			err,
		)
	}

	return info.IsDir(), nil
}

/*
CheckBlob проверяет наличие Blob в хранилище.

	digest - контрольная сумма Blob в формате sha256:<hex>.
*/
func (lc *LocalStorage) CheckBlob(digest string) error {
	encodedDigest, err := parseSHA256Digest(digest)
	if err != nil {
		return err
	}

	path := filepath.Join(config.BLOBS_PATH, encodedDigest)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return ErrBlobNotFound
	}
	if err != nil {
		return fmt.Errorf("не удалось проверить Blob %s: %w", digest, err)
	}
	if !info.Mode().IsRegular() {
		return ErrBlobCorrupted
	}

	return nil
}

/*
GetBlob возвращает Blob из хранилища в двоичном виде.

	digest - хэш Blob.
*/
func (lc *LocalStorage) GetBlob(digest string) (config.Blob, error) {
	var data config.Blob

	encodedDigest, err := parseSHA256Digest(digest)
	if err != nil {
		return data, err
	}

	blobPath := filepath.Join(config.BLOBS_PATH, encodedDigest)
	fileInfo, err := os.Stat(blobPath)
	if os.IsNotExist(err) {
		return data, ErrBlobNotFound
	}
	if err != nil {
		return data, fmt.Errorf("не удалось получить Blob %s: %w", digest, err)
	}
	if !fileInfo.Mode().IsRegular() {
		return data, ErrBlobCorrupted
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
	unlock, gateErr := lc.maintenance.lock(context.Background(), false)
	if gateErr != nil {
		return gateErr
	}
	defer unlock()
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
	unlock, gateErr := lc.maintenance.lock(context.Background(), false)
	if gateErr != nil {
		return gateErr
	}
	defer unlock()
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
	unlock, gateErr := lc.maintenance.lock(context.Background(), false)
	if gateErr != nil {
		return gateErr
	}
	defer unlock()
	if err := os.RemoveAll(filepath.Join(config.MANIFEST_PATH, name)); err != nil {
		return err
	}
	return nil
}

/*
DeleteManifest удаляет тег, сохраняя manifest до безопасной сборки мусора.

	cloud - пространство registry.
	repository - имя репозитория.
	tag - удаляемый тег.
*/
func (lc *LocalStorage) DeleteManifest(cloud, repository, tag string) error {
	unlock, err := lc.maintenance.lock(context.Background(), false)
	if err != nil {
		return err
	}
	defer unlock()
	// Граф ссылок OCI index проверяет GC; удаление тега не удаляет manifest.
	return os.Remove(filepath.Join(config.MANIFEST_PATH, cloud, repository, "tags", tag))
}

/*
DeleteRepository удаляет репозиторий из хранилища.

	cloud - название пространства.
	repository - название репозитория.
*/
func (lc *LocalStorage) DeleteRepository(cloud, repository string) error {
	unlock, gateErr := lc.maintenance.lock(context.Background(), false)
	if gateErr != nil {
		return gateErr
	}
	defer unlock()
	if err := os.RemoveAll(filepath.Join(config.MANIFEST_PATH, cloud, repository)); err != nil {
		return err
	}
	return nil
}

/*
GarbageCollection выполняет сборку мусора в хранилище.

	Проверяет граф тегов и свежих manifests; удаляет недостижимые объекты старше 24 часов.
	Запуск ограничен context и timeout; мутации хранилища на время GC блокируются.
*/
func (lc *LocalStorage) GarbageCollection(ctx context.Context) error {
	return runGC(ctx, &lc.maintenance, lc)
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

/*
CleanupUploads удаляет устаревшие незавершённые загрузки Blob.

	ctx - контекст выполнения операции.
	olderThan - минимальный возраст удаляемого временного файла.

Возвращает количество удалённых сессий, включая данные и JSON-состояние.
*/
func (s *LocalStorage) CleanupUploads(
	ctx context.Context,
	olderThan time.Duration,
) (int, error) {
	if olderThan <= 0 {
		return 0, errors.New("upload retention must be greater than zero")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(config.TMP_PATH)
	if err != nil {
		return 0, fmt.Errorf("не удалось прочитать каталог uploads: %w", err)
	}

	deadline := time.Now().Add(-olderThan)
	deleted := 0
	seen := make(map[string]bool)

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}

		if entry.IsDir() {
			continue
		}
		id := strings.SplitN(entry.Name(), ".json", 2)[0]
		if _, err := uuid.Parse(id); err != nil || seen[id] {
			continue
		}
		seen[id] = true

		removed, err := s.cleanupUpload(ctx, id, deadline)
		if err != nil {
			return deleted, err
		}
		if removed {
			deleted++
		}
	}

	return deleted, ctx.Err()
}

/*
cleanupUpload удаляет одну устаревшую загрузку под mutex её uploadID.

	ctx - контекст выполнения операции.
	uploadID - идентификатор загрузки.
	deadline - предельное время изменения удаляемого файла.

Активная сессия пропускается без ожидания. Возраст данных и JSON повторно
проверяется под mutex; свежий файл запрещает удаление всей сессии.
*/
func (s *LocalStorage) cleanupUpload(
	ctx context.Context,
	uploadID string,
	deadline time.Time,
) (bool, error) {
	unlock, ok := s.uploadLocks.TryLock(uploadID)
	if !ok {
		return false, nil
	}
	defer unlock()

	if err := ctx.Err(); err != nil {
		return false, err
	}

	path := filepath.Join(config.TMP_PATH, uploadID)
	files, err := filepath.Glob(path + ".json-*")
	if err != nil {
		return false, err
	}
	files = append(files, path, path+".json")
	found := false
	for _, file := range files {
		info, err := os.Stat(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		found = true
		if !info.ModTime().Before(deadline) {
			return false, nil
		}
	}
	if !found {
		return false, nil
	}
	return true, s.removeUpload(ctx, uploadID)
}
