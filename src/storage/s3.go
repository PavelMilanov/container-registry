package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

/*
S3Storage представляет хранилище на основе облачной системы S3.
*/
type S3Storage struct {
	S3          *minio.Client
	Bucket      string
	Prefix      string
	uploadLocks uploadLockManager
	maintenance maintenanceGate
}

var _ NamespaceStore = (*S3Storage)(nil)
var _ BlobStore = (*S3Storage)(nil)
var _ ManifestStore = (*S3Storage)(nil)
var _ CloudStore = (*S3Storage)(nil)
var _ RepositoryStore = (*S3Storage)(nil)
var _ TagStore = (*S3Storage)(nil)
var _ GarbageCollector = (*S3Storage)(nil)
var _ TagPruner = (*S3Storage)(nil)
var _ BlobUploadStore = (*S3Storage)(nil)
var _ UploadCleaner = (*S3Storage)(nil)
var _ BlobReader = (*S3Storage)(nil)

/*
newS3Storage создает новый экземпляр S3Storage.

	env - конфигурация окружения.
*/
func newS3Storage(env *config.Env) (*S3Storage, error) {
	s3Client, err := minio.New(env.Storage.Credentials.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(env.Storage.Credentials.AccessKey, env.Storage.Credentials.SecretKey, ""),
		Secure: env.Storage.Credentials.SSL,
	})
	if err != nil {
		return &S3Storage{}, err
	}
	store := &S3Storage{S3: s3Client, Bucket: env.Storage.Bucket, Prefix: env.Storage.Prefix}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bucketExists, err := s3Client.BucketExists(ctx, store.bucketName())
	if err != nil {
		return &S3Storage{}, err
	}
	if !bucketExists {
		return nil, fmt.Errorf("S3 bucket %q не создан", store.bucketName())
	}
	return store, nil
}

/*
bucketName возвращает настроенный bucket или имя по умолчанию.
*/
func (s *S3Storage) bucketName() string {
	if s.Bucket != "" {
		return s.Bucket
	}
	return config.BACKET_NAME
}

/*
objectPrefix возвращает стабильный S3-префикс независимо от локального DATA_PATH.

По умолчанию используется var для совместимости с существующими ключами.
*/
func (s *S3Storage) objectPrefix(directory string) string {
	prefix := s.Prefix
	if prefix == "" {
		prefix = "var"
	}
	return filepathToS3(filepath.Join(prefix, directory))
}

/*
NamespaceExists проверяет наличие пространства имён registry.

	ctx - контекст выполнения операции.
	name - имя пространства.
*/
func (s *S3Storage) NamespaceExists(
	ctx context.Context,
	name string,
) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !validNamespaceName(name) {
		return false, nil
	}

	prefix := filepath.Join(s.objectPrefix("manifests"), name) + "/"
	objects := s.S3.ListObjects(
		ctx,
		s.bucketName(),
		minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		},
	)

	for object := range objects {
		if object.Err != nil {
			return false, fmt.Errorf(
				"не удалось проверить пространство %s: %w",
				name,
				object.Err,
			)
		}
		if object.Key != "" {
			return true, nil
		}
	}

	return false, nil
}

/*
CheckBlob проверяет наличие Blob в хранилище.

	uuid - идентификатор Blob.
*/
func (s *S3Storage) CheckBlob(uuid string) error {
	key, err := s.blobKeyFromDigest(uuid)
	if err != nil {
		return err
	}
	if _, err := s.S3.StatObject(context.Background(), s.bucketName(), key, minio.StatObjectOptions{}); err != nil {
		if isS3NotFound(err) {
			return ErrBlobNotFound
		}
		return err
	}
	return nil
}

/*
GetBlob возвращает метаданные Blob без скачивания его содержимого.

	digest - хэш Blob.
*/
func (s *S3Storage) GetBlob(digest string) (config.Blob, error) {
	return s.statBlob(context.Background(), digest)
}

/*
statBlob возвращает метаданные S3 Blob без скачивания на локальный диск.
*/
func (s *S3Storage) statBlob(ctx context.Context, digest string) (config.Blob, error) {
	var data config.Blob
	key, err := s.blobKeyFromDigest(digest)
	if err != nil {
		return data, err
	}

	info, err := s.S3.StatObject(ctx, s.bucketName(), key, minio.StatObjectOptions{})
	if err != nil {
		if isS3NotFound(err) {
			return data, ErrBlobNotFound
		}
		return data, err
	}
	data.Digest = digest
	data.Size = info.Size
	return data, nil
}

/*
OpenBlob открывает поток S3 Blob; вызывающий код обязан закрыть его.

	ctx - контекст запроса, отменяющий обращения к S3.
	digest - контрольная сумма Blob.
*/
func (s *S3Storage) OpenBlob(ctx context.Context, digest string) (io.ReadSeekCloser, config.Blob, error) {
	info, err := s.statBlob(ctx, digest)
	if err != nil {
		return nil, info, err
	}
	key, err := s.blobKeyFromDigest(digest)
	if err != nil {
		return nil, info, err
	}
	object, err := s.S3.GetObject(ctx, s.bucketName(), key, minio.GetObjectOptions{})
	return object, info, err
}

/*
SaveManifest сохраняет манифест в хранилище.

	body - содержимое манифеста.
*/
func (s *S3Storage) SaveManifest(meta config.Meta, body []byte, manifestPath string) error {
	unlock, gateErr := s.maintenance.lock(context.Background(), false)
	if gateErr != nil {
		return gateErr
	}
	defer unlock()
	// Путь из services относится к LocalStorage. S3 использует собственный prefix.
	manifestPath = filepath.Join(s.objectPrefix("manifests"), meta.Repository, meta.Image, meta.Digest)
	if _, err := s.putBytes(manifestPath, body); err != nil {
		return err
	}
	if !strings.HasPrefix(meta.Tag, "sha256:") {
		tagPath := filepath.Join(s.objectPrefix("manifests"), meta.Repository, meta.Image, "tags", meta.Tag)
		if _, err := s.putBytes(tagPath, []byte(meta.Digest)); err != nil {
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
func (s *S3Storage) GetManifest(repository, image, reference string) ([]byte, error) {
	manifestPath := ""
	tagPath := filepath.Join(s.objectPrefix("manifests"), repository, image, "tags", reference)
	if strings.HasPrefix(reference, "sha256:") {
		manifestPath = filepath.Join(s.objectPrefix("manifests"), repository, image, reference)
	} else {
		tagData, err := s.ReadFile(tagPath)
		if err != nil {
			return nil, errors.New("Tag not found")
		}
		manifestPath = filepath.Join(s.objectPrefix("manifests"), repository, image, string(tagData))
	}
	data, err := s.ReadFile(manifestPath)
	if err != nil {
		if isS3NotFound(err) {
			return nil, errors.New("Manifest not found")
		}
		return nil, err
	}
	return data, nil
}

/*
AddRegistry добавляет новый реестр в хранилище.

	registry - имя реестра.
*/
func (s *S3Storage) AddCloud(name string) error {
	unlock, gateErr := s.maintenance.lock(context.Background(), false)
	if gateErr != nil {
		return gateErr
	}
	defer unlock()
	_, err := s.putBytes(filepath.Join(s.objectPrefix("manifests"), name, ".keep"), nil)
	return err
}

/*
DeleteRegistry удаляет реестр из хранилища.

	registry - имя реестра.
*/
func (s *S3Storage) DeleteCloud(name string) error {
	unlock, gateErr := s.maintenance.lock(context.Background(), false)
	if gateErr != nil {
		return gateErr
	}
	defer unlock()
	return s.removePrefix(filepath.Join(s.objectPrefix("manifests"), name) + "/")
}

/*
DeleteManifest удаляет тег, сохраняя manifest до безопасной сборки мусора.

	cloud - пространство registry.
	repository - имя репозитория.
	tag - удаляемый тег.
*/
func (s *S3Storage) DeleteManifest(cloud, repository, tag string) error {
	unlock, gateErr := s.maintenance.lock(context.Background(), false)
	if gateErr != nil {
		return gateErr
	}
	defer unlock()
	tagPath := filepath.Join(s.objectPrefix("manifests"), cloud, repository, "tags", tag)
	if _, err := s.ReadFile(tagPath); err != nil {
		return err
	}
	// Манифест может быть достижим через OCI index другого тега. Его удаляет GC.
	return s.S3.RemoveObject(context.Background(), s.bucketName(), tagPath, minio.RemoveObjectOptions{})
}

/*
DeleteRepository удаляет репозиторий из хранилища.

	name - имя репозитория.
	image - имя образа.
*/
func (s *S3Storage) DeleteRepository(name, image string) error {
	unlock, gateErr := s.maintenance.lock(context.Background(), false)
	if gateErr != nil {
		return gateErr
	}
	defer unlock()
	return s.removePrefix(filepath.Join(s.objectPrefix("manifests"), name, image) + "/")
}

/*
GarbageCollection выполняет сборку мусора в хранилище.

	Проверяет граф тегов и свежих manifests; удаляет недостижимые объекты старше 24 часов.
	Запуск ограничен context и timeout; мутации хранилища на время GC блокируются.
*/
func (s *S3Storage) GarbageCollection(ctx context.Context) error {
	return runGC(ctx, &s.maintenance, s)
}

func (s *S3Storage) ReadFile(path string) ([]byte, error) {
	reader, err := s.S3.GetObject(context.Background(), s.bucketName(), path, minio.GetObjectOptions{})
	if err != nil {
		return []byte{}, err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		if isS3NotFound(err) {
			return []byte{}, err
		}
		return []byte{}, err
	}
	return data, nil
}

func (s *S3Storage) GetManifestList(cloud, repository string) ([]string, error) {
	objects, err := s.listTagObjects(cloud, repository)
	if err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(objects))
	prefix := filepath.Join(s.objectPrefix("manifests"), cloud, repository, "tags") + "/"
	for _, object := range objects {
		tag := strings.TrimPrefix(object.Key, prefix)
		if tag == "" || strings.Contains(tag, "/") {
			continue
		}
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags, nil
}

/* Возвращает список пространств */
func (s *S3Storage) GetCloudList() ([]string, error) {
	prefix := s.objectPrefix("manifests") + "/"
	objects, err := s.listObjects(prefix, false)
	if err != nil {
		return nil, err
	}
	clouds := make([]string, 0, len(objects))
	seen := make(map[string]struct{})
	for _, object := range objects {
		name := firstKeyPart(strings.TrimPrefix(object.Key, prefix))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		clouds = append(clouds, name)
	}
	sort.Strings(clouds)
	return clouds, nil
}

func (s *S3Storage) GetRepositoriesList(cloud string) ([]string, error) {
	prefix := filepath.Join(s.objectPrefix("manifests"), cloud) + "/"
	objects, err := s.listObjects(prefix, false)
	if err != nil {
		return nil, err
	}
	if len(objects) == 0 {
		return nil, errors.New(cloud + " не найден")
	}

	repos := make([]string, 0, len(objects))
	seen := make(map[string]struct{})
	for _, object := range objects {
		name := firstKeyPart(strings.TrimPrefix(object.Key, prefix))
		if name == "" || name == ".keep" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		repos = append(repos, name)
	}
	sort.Strings(repos)
	return repos, nil
}

func (s *S3Storage) DeleteOlderTags(count int) error {
	if count < 0 {
		return errors.New("tag retention must not be negative")
	}
	deleted := 0
	clouds, err := s.GetCloudList()
	if err != nil {
		return err
	}

	for _, cloud := range clouds {
		repositories, err := s.GetRepositoriesList(cloud)
		if err != nil {
			logrus.WithField("cloud", cloud).WithError(err).Warn("не удалось получить список репозиториев")
			continue
		}
		for _, repository := range repositories {
			tags, err := s.listTagObjects(cloud, repository)
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

			sort.Slice(tags, func(i, j int) bool {
				return tags[i].LastModified.After(tags[j].LastModified)
			})

			for i := count; i < len(tags); i++ {
				tag := filepath.Base(tags[i].Key)
				if err := s.DeleteManifest(cloud, repository, tag); err != nil {
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

func (s *S3Storage) listTagObjects(cloud, repository string) ([]minio.ObjectInfo, error) {
	return s.listObjects(filepath.Join(s.objectPrefix("manifests"), cloud, repository, "tags")+"/", false)
}

func (s *S3Storage) listObjects(prefix string, recursive bool) ([]minio.ObjectInfo, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var objects []minio.ObjectInfo
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: recursive}
	for object := range s.S3.ListObjects(ctx, s.bucketName(), opts) {
		if object.Err != nil {
			return nil, object.Err
		}
		if object.Key == "" {
			continue
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func (s *S3Storage) removePrefix(prefix string) error {
	objects, err := s.listObjects(prefix, true)
	if err != nil {
		return err
	}
	var resultErr error
	for _, object := range objects {
		resultErr = errors.Join(resultErr, s.S3.RemoveObject(context.Background(), s.bucketName(), object.Key, minio.RemoveObjectOptions{}))
	}
	return resultErr
}

func (s *S3Storage) putBytes(key string, data []byte) (minio.UploadInfo, error) {
	reader := bytes.NewReader(data)
	return s.S3.PutObject(context.Background(), s.bucketName(), key, reader, reader.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
}

func firstKeyPart(value string) string {
	value = strings.Trim(value, "/")
	if value == "" {
		return ""
	}
	return strings.Split(value, "/")[0]
}

func (s *S3Storage) blobKeyFromDigest(digest string) (string, error) {
	encoded, err := parseSHA256Digest(digest)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.objectPrefix("blobs"), encoded), nil
}

func isS3NotFound(err error) bool {
	if err == nil {
		return false
	}
	resp := minio.ToErrorResponse(err)
	return resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket" || resp.StatusCode == 404
}
