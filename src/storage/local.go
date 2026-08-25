package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/sirupsen/logrus"
)

/*
LocalStorage представляет хранилище на основе локальной файловой системы.
*/
type LocalStorage struct {
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

/*
StartBlobUpload создаёт пустой временный файл для последующей загрузки Blob.

	ctx - контекст выполнения операции.
	uploadID - идентификатор загрузки.
*/
func (s *LocalStorage) StartBlobUpload(
	ctx context.Context,
	uploadID string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path, err := localUploadPath(uploadID)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		0600,
	)
	if err != nil {
		return err
	}

	return file.Close()
}

/*
AppendBlobUpload дописывает очередную часть Blob во временный файл.

	ctx - контекст выполнения операции.
	uploadID - идентификатор загрузки.
	expectedOffset - ожидаемая позиция начала записи в байтах.
	body - поток с очередной частью Blob.

Возвращает размер временного файла после успешной записи.
*/
func (s *LocalStorage) AppendBlobUpload(
	ctx context.Context,
	uploadID string,
	expectedOffset int64,
	body io.Reader,
) (newOffset int64, resultErr error) {
	path, err := localUploadPath(uploadID)
	if err != nil {
		return 0, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrUploadNotFound
		}
		return 0, err
	}

	oldSize := int64(0)
	defer func() {
		if err := file.Close(); resultErr == nil && err != nil {
			_ = os.Truncate(path, oldSize)
			newOffset = oldSize
			resultErr = fmt.Errorf("не удалось закрыть upload %s: %w", uploadID, err)
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return 0, err
	}

	oldSize = info.Size()
	if oldSize != expectedOffset {
		return oldSize, ErrInvalidOffset
	}

	if _, err := file.Seek(expectedOffset, io.SeekStart); err != nil {
		return oldSize, err
	}

	written, err := copyWithContext(ctx, file, body)
	if err != nil {
		_ = file.Truncate(oldSize)
		return oldSize, err
	}

	return oldSize + written, nil
}

/*
CompleteBlobUpload завершает загрузку Blob и перемещает его в постоянное хранилище.

	ctx - контекст выполнения операции.
	uploadID - идентификатор загрузки.
	expectedDigest - ожидаемая контрольная сумма Blob в формате sha256:<hex>.
	finalBody - поток с заключительной частью Blob.

Метод дописывает заключительную часть, проверяет SHA-256 и после успешной
проверки перемещает временный файл из TMP_PATH в BLOBS_PATH.
Возвращает метаданные сохранённого Blob.
*/
func (s *LocalStorage) CompleteBlobUpload(
	ctx context.Context,
	uploadID string,
	expectedDigest string,
	finalBody io.Reader,
) (config.Blob, error) {
	var result config.Blob

	encodedDigest, err := parseSHA256Digest(expectedDigest)
	if err != nil {
		return result, err
	}

	uploadPath, err := localUploadPath(uploadID)
	if err != nil {
		return result, err
	}

	info, err := os.Stat(uploadPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, ErrUploadNotFound
		}
		return result, err
	}

	finalSize, err := s.AppendBlobUpload(
		ctx,
		uploadID,
		info.Size(),
		finalBody,
	)
	if err != nil {
		return result, err
	}

	calculatedDigest, err := calculateFileDigest(ctx, uploadPath)
	if err != nil {
		return result, err
	}

	if calculatedDigest != expectedDigest {
		_ = os.Remove(uploadPath)
		return result, ErrDigestMismatch
	}

	finalPath := filepath.Join(config.BLOBS_PATH, encodedDigest)

	if existing, err := os.Stat(finalPath); err == nil {
		if !existing.Mode().IsRegular() {
			return result, ErrBlobCorrupted
		}
		if existing.Size() != finalSize {
			return result, ErrBlobCorrupted
		}

		existingDigest, err := calculateFileDigest(ctx, finalPath)
		if err != nil {
			return result, err
		}
		if existingDigest != expectedDigest {
			return result, ErrBlobCorrupted
		}

		if err := os.Remove(uploadPath); err != nil {
			return result, err
		}

		return config.Blob{
			Digest: expectedDigest,
			Path:   finalPath,
			Size:   existing.Size(),
		}, nil
	} else if !os.IsNotExist(err) {
		return result, err
	}

	if err := os.Rename(uploadPath, finalPath); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return result, fmt.Errorf(
				"%w: TMP_PATH=%s, BLOBS_PATH=%s",
				ErrCrossDevice,
				config.TMP_PATH,
				config.BLOBS_PATH,
			)
		}
		return result, err
	}

	return config.Blob{
		Digest: expectedDigest,
		Path:   finalPath,
		Size:   finalSize,
	}, nil
}

/*
AbortBlobUpload отменяет загрузку Blob и удаляет её временный файл.

	ctx - контекст выполнения операции.
	uploadID - идентификатор загрузки.
*/
func (s *LocalStorage) AbortBlobUpload(
	ctx context.Context,
	uploadID string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path, err := localUploadPath(uploadID)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}

	return err
}

/*
CleanupUploads удаляет устаревшие незавершённые загрузки Blob.

	ctx - контекст выполнения операции.
	olderThan - минимальный возраст удаляемого временного файла.

Возвращает количество удалённых временных файлов.
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

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}

		if entry.IsDir() {
			continue
		}
		if _, err := uuid.Parse(entry.Name()); err != nil {
			continue
		}

		path := filepath.Join(config.TMP_PATH, entry.Name())
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return deleted, fmt.Errorf(
				"не удалось прочитать upload %s: %w",
				entry.Name(),
				err,
			)
		}

		if info.ModTime().Before(deadline) {
			if err := os.Remove(path); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return deleted, fmt.Errorf(
					"не удалось удалить upload %s: %w",
					entry.Name(),
					err,
				)
			}
			deleted++
		}
	}

	return deleted, nil
}
