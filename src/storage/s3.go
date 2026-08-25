package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
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
	S3 *minio.Client
}

var _ NamespaceStore = (*S3Storage)(nil)

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
	bucketExists, err := s3Client.BucketExists(context.Background(), config.BACKET_NAME)
	if err != nil {
		return &S3Storage{}, err
	}
	if !bucketExists {
		return &S3Storage{}, errors.New("Backet не создан")

	}
	return &S3Storage{
		S3: s3Client,
	}, nil
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
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !validNamespaceName(name) {
		return false, nil
	}

	prefix := filepath.Join(config.MANIFEST_PATH, name) + "/"
	objects := s.S3.ListObjects(
		ctx,
		config.BACKET_NAME,
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
	key, err := blobKeyFromDigest(uuid)
	if err != nil {
		return err
	}
	if _, err := s.S3.StatObject(context.Background(), config.BACKET_NAME, key, minio.StatObjectOptions{}); err != nil {
		if isS3NotFound(err) {
			return errors.New("Blob not found")
		}
		return err
	}
	return nil
}

/*
GetBlob возвращает Blob из хранилища в двоичном виде.

	digest - хэш Blob.
*/
func (s *S3Storage) GetBlob(digest string) (config.Blob, error) {
	var data config.Blob
	key, err := blobKeyFromDigest(digest)
	if err != nil {
		return data, err
	}

	reader, err := s.S3.GetObject(context.Background(), config.BACKET_NAME, key, minio.GetObjectOptions{})
	if err != nil {
		if isS3NotFound(err) {
			return data, errors.New("Blob not found")
		}
		return data, err
	}
	defer reader.Close()

	info, err := reader.Stat()
	if err != nil {
		if isS3NotFound(err) {
			return data, errors.New("Blob not found")
		}
		return data, err
	}

	tmpName := digestValue(digest)
	if err := os.MkdirAll(config.TMP_PATH, 0755); err != nil {
		return data, err
	}
	tmpPath := filepath.Join(config.TMP_PATH, tmpName)
	file, err := os.Create(tmpPath)
	if err != nil {
		return data, err
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		return data, err
	}
	if err := file.Close(); err != nil {
		return data, err
	}

	timer := time.NewTimer(5 * time.Second)
	go func() {
		<-timer.C
		_ = os.Remove(tmpPath)
	}()

	data.Digest = digest
	data.Path = tmpPath
	data.Size = info.Size
	return data, nil
}

/*
SaveManifest сохраняет манифест в хранилище.

	body - содержимое манифеста.
*/
func (s *S3Storage) SaveManifest(meta config.Meta, body []byte, manifestPath string) error {
	if _, err := s.putBytes(manifestPath, body); err != nil {
		return err
	}
	if !strings.HasPrefix(meta.Tag, "sha256:") {
		tagPath := filepath.Join(config.MANIFEST_PATH, meta.Repository, meta.Image, "tags", meta.Tag)
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
	tagPath := filepath.Join(config.MANIFEST_PATH, repository, image, "tags", reference)
	if strings.HasPrefix(reference, "sha256:") {
		manifestPath = filepath.Join(config.MANIFEST_PATH, repository, image, reference)
	} else {
		tagData, err := s.ReadFile(tagPath)
		if err != nil {
			return nil, errors.New("Tag not found")
		}
		manifestPath = filepath.Join(config.MANIFEST_PATH, repository, image, string(tagData))
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
	_, err := s.putBytes(filepath.Join(config.MANIFEST_PATH, name, ".keep"), nil)
	return err
}

/*
DeleteRegistry удаляет реестр из хранилища.

	registry - имя реестра.
*/
func (s *S3Storage) DeleteCloud(name string) error {
	return s.removePrefix(filepath.Join(config.MANIFEST_PATH, name) + "/")
}

/*
DeleteImage удаляет образ из хранилища.

	repository - имя репозитория.
	imageName - имя образа.
	imageTag - тег образа.
	imageHash - хеш образа.
*/
func (s *S3Storage) DeleteManifest(cloud, repository, tag string) error {
	tagPath := filepath.Join(config.MANIFEST_PATH, cloud, repository, "tags", tag)
	data, err := s.ReadFile(tagPath)
	if err != nil {
		return err
	}
	manifestDigest := string(data)
	manifestPath := filepath.Join(config.MANIFEST_PATH, cloud, repository, manifestDigest)

	if err := s.S3.RemoveObject(context.Background(), config.BACKET_NAME, tagPath, minio.RemoveObjectOptions{}); err != nil {
		return err
	}

	tags, err := s.listTagObjects(cloud, repository)
	if err != nil {
		return err
	}
	for _, item := range tags {
		otherDigest, err := s.ReadFile(item.Key)
		if err != nil {
			logrus.WithField("tag", item.Key).WithError(err).Warn("не удалось прочитать tag-файл")
			continue
		}
		if string(otherDigest) == manifestDigest {
			return nil
		}
	}

	if err := s.S3.RemoveObject(context.Background(), config.BACKET_NAME, manifestPath, minio.RemoveObjectOptions{}); err != nil && !isS3NotFound(err) {
		return err
	}
	return nil
}

/*
DeleteRepository удаляет репозиторий из хранилища.

	name - имя репозитория.
	image - имя образа.
*/
func (s *S3Storage) DeleteRepository(name, image string) error {
	return s.removePrefix(filepath.Join(config.MANIFEST_PATH, name, image) + "/")
}

/*
GarbageCollection выполняет сборку мусора в хранилище.

	Удаляет все образы и слои, которые не используются ни одним реестром.
*/
func (s *S3Storage) GarbageCollection() error {
	manifests, err := s.inventoryManifests()
	if err != nil {
		return err
	}
	usedBlobs, err := s.parseUsageBlobs(manifests)
	if err != nil {
		return err
	}
	usedBlobsMap := make(map[string]struct{}, len(usedBlobs))
	for _, blob := range usedBlobs {
		usedBlobsMap[blob] = struct{}{}
	}

	blobs, err := s.inventoryBlobs()
	if err != nil {
		return err
	}
	deleted := 0
	for _, blob := range blobs {
		if _, ok := usedBlobsMap[blob]; ok {
			continue
		}
		if err := s.S3.RemoveObject(context.Background(), config.BACKET_NAME, blob, minio.RemoveObjectOptions{}); err != nil {
			logrus.WithField("GarbageCollection", "error").WithError(err).Error()
			continue
		}
		deleted++
	}
	logrus.WithField("GarbageCollection", "deleted").Infof("Удалено %d файлов", deleted)
	return nil
}

func (s *S3Storage) ReadFile(path string) ([]byte, error) {
	reader, err := s.S3.GetObject(context.Background(), config.BACKET_NAME, path, minio.GetObjectOptions{})
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
	prefix := filepath.Join(config.MANIFEST_PATH, cloud, repository, "tags") + "/"
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
	prefix := config.MANIFEST_PATH + "/"
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
	prefix := filepath.Join(config.MANIFEST_PATH, cloud) + "/"
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

func (s *S3Storage) inventoryBlobs() ([]string, error) {
	objects, err := s.listObjects(config.BLOBS_PATH+"/", true)
	if err != nil {
		return nil, err
	}
	blobs := make([]string, 0, len(objects))
	for _, object := range objects {
		if object.Key == "" || strings.HasSuffix(object.Key, "/") {
			continue
		}
		blobs = append(blobs, object.Key)
	}
	logrus.WithField("GarbageCollection", "blobs").Infof("Количество слоев: %d", len(blobs))
	return blobs, nil
}

func (s *S3Storage) inventoryManifests() ([]string, error) {
	objects, err := s.listObjects(config.MANIFEST_PATH+"/", true)
	if err != nil {
		return nil, err
	}

	type repoInventory struct {
		tags      []minio.ObjectInfo
		manifests []string
	}
	repos := make(map[string]*repoInventory)
	for _, object := range objects {
		if object.Key == "" || strings.HasSuffix(object.Key, "/") || filepath.Base(object.Key) == ".keep" {
			continue
		}
		relative := strings.TrimPrefix(object.Key, config.MANIFEST_PATH+"/")
		parts := strings.Split(relative, "/")
		if len(parts) < 3 {
			continue
		}
		repoKey := filepath.Join(parts[0], parts[1])
		inventory := repos[repoKey]
		if inventory == nil {
			inventory = &repoInventory{}
			repos[repoKey] = inventory
		}
		if len(parts) == 4 && parts[2] == "tags" {
			inventory.tags = append(inventory.tags, object)
			continue
		}
		if len(parts) == 3 {
			inventory.manifests = append(inventory.manifests, object.Key)
		}
	}

	var activeManifests []string
	for repoKey, inventory := range repos {
		if len(inventory.manifests) > 0 && len(inventory.tags) == 0 {
			return nil, fmt.Errorf("не удалось прочитать директорию тегов %s: теги не найдены", filepath.Join(config.MANIFEST_PATH, repoKey, "tags"))
		}

		activeTags, err := s.parseActiveTags(filepath.Join(config.MANIFEST_PATH, repoKey), inventory.tags)
		if err != nil {
			return nil, err
		}
		activeTagSet := make(map[string]struct{}, len(activeTags))
		for _, tag := range activeTags {
			activeTagSet[tag] = struct{}{}
		}

		for _, manifest := range inventory.manifests {
			digest := filepath.Base(manifest)
			if _, found := activeTagSet[digest]; !found {
				if err := s.S3.RemoveObject(context.Background(), config.BACKET_NAME, manifest, minio.RemoveObjectOptions{}); err != nil {
					return nil, fmt.Errorf("не удалось удалить неиспользуемый манифест %s: %w", manifest, err)
				}
				continue
			}
			activeManifests = append(activeManifests, manifest)
		}
	}

	logrus.WithField("GarbageCollection", "manifests").Infof("Количество манифестов: %d", len(activeManifests))
	return activeManifests, nil
}

func (s *S3Storage) parseActiveTags(repoPath string, tags []minio.ObjectInfo) ([]string, error) {
	var activeTags []string
	for _, tag := range tags {
		data, err := s.ReadFile(tag.Key)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать tag-файл %s: %w", tag.Key, err)
		}
		digest := string(data)
		manifestPath := filepath.Join(repoPath, digest)
		manifestData, err := s.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать манифест %s: %w", digest, err)
		}
		var body struct {
			MediaType string `json:"mediaType"`
		}
		if err := json.Unmarshal(manifestData, &body); err != nil {
			return nil, fmt.Errorf("не удалось распарсить mediaType манифеста %s: %w", digest, err)
		}
		switch body.MediaType {
		case config.MANIFEST_TYPE["manifest"]:
			activeTags = append(activeTags, digest)
		case config.MANIFEST_TYPE["index"]:
			var index config.Index
			if err := json.Unmarshal(manifestData, &index); err != nil {
				return nil, fmt.Errorf("не удалось распарсить index-манифест %s: %w", digest, err)
			}
			activeTags = append(activeTags, digest)
			for _, manifest := range index.Manifests {
				activeTags = append(activeTags, manifest.Digest)
			}
		default:
			return nil, fmt.Errorf("неизвестный mediaType %q в манифесте %s", body.MediaType, digest)
		}
	}
	return activeTags, nil
}

func (s *S3Storage) parseUsageBlobs(links []string) ([]string, error) {
	var buffer []string
	for _, link := range links {
		file, err := s.ReadFile(link)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать манифест %s: %w", link, err)
		}
		var body struct {
			MediaType string `json:"mediaType"`
		}
		if err := json.Unmarshal(file, &body); err != nil {
			return nil, fmt.Errorf("не удалось распарсить mediaType манифеста %s: %w", link, err)
		}
		switch body.MediaType {
		case config.MANIFEST_TYPE["manifest"]:
			var manifest config.Manifest
			if err := json.Unmarshal(file, &manifest); err != nil {
				return nil, fmt.Errorf("не удалось распарсить manifest %s: %w", link, err)
			}
			configBlob, err := blobPathFromDigest(manifest.Config.Digest)
			if err != nil {
				return nil, fmt.Errorf("некорректный config digest в %s: %w", link, err)
			}
			buffer = append(buffer, configBlob)
			for _, layer := range manifest.Layers {
				layerBlob, err := blobPathFromDigest(layer.Digest)
				if err != nil {
					return nil, fmt.Errorf("некорректный layer digest в %s: %w", link, err)
				}
				buffer = append(buffer, layerBlob)
			}
		case config.MANIFEST_TYPE["index"]:
			continue
		default:
			return nil, fmt.Errorf("неизвестный mediaType %q в манифесте %s", body.MediaType, link)
		}
	}

	uniqueBlobs := make(map[string]struct{}, len(buffer))
	for _, blob := range buffer {
		uniqueBlobs[blob] = struct{}{}
	}
	result := make([]string, 0, len(uniqueBlobs))
	for blob := range uniqueBlobs {
		result = append(result, blob)
	}
	logrus.WithField("GarbageCollection", "blobs").Infof("Количество используемых слоев: %d", len(result))
	return result, nil
}

func (s *S3Storage) listTagObjects(cloud, repository string) ([]minio.ObjectInfo, error) {
	return s.listObjects(filepath.Join(config.MANIFEST_PATH, cloud, repository, "tags")+"/", false)
}

func (s *S3Storage) listObjects(prefix string, recursive bool) ([]minio.ObjectInfo, error) {
	var objects []minio.ObjectInfo
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: recursive}
	for object := range s.S3.ListObjects(context.Background(), config.BACKET_NAME, opts) {
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
	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: true}
		for object := range s.S3.ListObjects(context.Background(), config.BACKET_NAME, opts) {
			if object.Err != nil {
				logrus.Error(object.Err)
				continue
			}
			objectsCh <- object
		}
	}()
	errCh := s.S3.RemoveObjects(context.Background(), config.BACKET_NAME, objectsCh, minio.RemoveObjectsOptions{})
	for err := range errCh {
		return err.Err
	}
	return nil
}

func (s *S3Storage) putBytes(key string, data []byte) (minio.UploadInfo, error) {
	reader := bytes.NewReader(data)
	return s.S3.PutObject(context.Background(), config.BACKET_NAME, key, reader, reader.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
}

func firstKeyPart(value string) string {
	value = strings.Trim(value, "/")
	if value == "" {
		return ""
	}
	return strings.Split(value, "/")[0]
}

func blobKeyFromDigest(digest string) (string, error) {
	algorithm, encoded, ok := strings.Cut(digest, ":")
	if !ok {
		encoded = digest
		algorithm = "sha256"
	}
	if algorithm != "sha256" || encoded == "" {
		return "", fmt.Errorf("digest должен иметь формат sha256:<hex>, получено %q", digest)
	}
	return filepath.Join(config.BLOBS_PATH, encoded), nil
}

func digestValue(digest string) string {
	if _, encoded, ok := strings.Cut(digest, ":"); ok {
		return encoded
	}
	return digest
}

func isS3NotFound(err error) bool {
	if err == nil {
		return false
	}
	resp := minio.ToErrorResponse(err)
	return resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket" || resp.StatusCode == 404
}
