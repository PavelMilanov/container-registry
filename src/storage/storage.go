package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/sirupsen/logrus"
)

/*
inventoryBlobs сканирует директорию и возвращает список слоев в хранилище.

Returns:
  - []string: ссылки на слои.
*/
func inventoryBlobs() []string {
	var buffer []string
	blobs, _ := os.ReadDir(config.BLOBS_PATH)
	for _, blob := range blobs {
		buffer = append(buffer, filepath.Join(config.BLOBS_PATH, blob.Name()))
	}
	logrus.WithField("GarbageCollection", "blobs").Infof("Количество слоев: %d", len(buffer))
	return buffer
}

func inventoryBlobsStrict() ([]string, error) {
	var buffer []string
	blobs, err := os.ReadDir(config.BLOBS_PATH)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать директорию слоев %s: %w", config.BLOBS_PATH, err)
	}
	for _, blob := range blobs {
		if blob.IsDir() {
			continue
		}
		buffer = append(buffer, filepath.Join(config.BLOBS_PATH, blob.Name()))
	}
	logrus.WithField("GarbageCollection", "blobs").Infof("Количество слоев: %d", len(buffer))
	return buffer, nil
}

/*
inventoryManifests сканирует директории и удаляет неиспользуемые файлы манифестов.

Returns:
  - []string: ссылки на используемые манифесты.
*/
func inventoryManifests() []string {
	var buffer []string
	path := config.MANIFEST_PATH
	clouds, err := os.ReadDir(path)
	if err != nil {
		logrus.WithField("GarbageCollection", "error").
			WithError(err).
			Errorf("не удалось прочитать директорию манифестов: %s", path)
		return buffer
	}
	for _, cloud := range clouds {
		cloudPath := filepath.Join(path, cloud.Name())
		repositories, err := os.ReadDir(cloudPath)
		if err != nil {
			logrus.WithField("GarbageCollection", "error").
				WithError(err).
				Errorf("не удалось прочитать директорию: %s", cloudPath)
			continue
		}
		for _, repo := range repositories {
			repoPath := filepath.Join(cloudPath, repo.Name())
			logrus.WithField("GarbageCollection", "scan").
				Debugf("чтение директории: %s", repoPath)
			activeTags := parseActiveTags(repoPath)
			activeTagSet := make(map[string]struct{}, len(activeTags))
			for _, tag := range activeTags {
				activeTagSet[tag] = struct{}{}
			}
			manifests := parseManifests(repoPath)
			for _, manifest := range manifests {
				fileLink := filepath.Join(repoPath, manifest)
				if _, found := activeTagSet[manifest]; !found {
					if err := os.Remove(fileLink); err != nil {
						logrus.WithField("GarbageCollection", "error").
							WithError(err).
							Errorf("не удалось удалить файл: %s", fileLink)
					}
					continue
				}
				buffer = append(buffer, fileLink)
			}
		}
	}
	logrus.WithField("GarbageCollection", "manifests").Infof("Количество манифестов: %d", len(buffer))
	return buffer
}

func inventoryManifestsStrict() ([]string, error) {
	var buffer []string
	path := config.MANIFEST_PATH
	clouds, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать директорию манифестов %s: %w", path, err)
	}
	for _, cloud := range clouds {
		if !cloud.IsDir() {
			continue
		}
		cloudPath := filepath.Join(path, cloud.Name())
		repositories, err := os.ReadDir(cloudPath)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать директорию %s: %w", cloudPath, err)
		}
		for _, repo := range repositories {
			if !repo.IsDir() {
				continue
			}
			repoPath := filepath.Join(cloudPath, repo.Name())
			logrus.WithField("GarbageCollection", "scan").
				Debugf("чтение директории: %s", repoPath)

			activeTags, err := parseActiveTagsStrict(repoPath)
			if err != nil {
				return nil, err
			}
			activeTagSet := make(map[string]struct{}, len(activeTags))
			for _, tag := range activeTags {
				activeTagSet[tag] = struct{}{}
			}

			manifests, err := parseManifestsStrict(repoPath)
			if err != nil {
				return nil, err
			}
			for _, manifest := range manifests {
				fileLink := filepath.Join(repoPath, manifest)
				if _, found := activeTagSet[manifest]; !found {
					if err := os.Remove(fileLink); err != nil {
						return nil, fmt.Errorf("не удалось удалить неиспользуемый манифест %s: %w", fileLink, err)
					}
					continue
				}
				buffer = append(buffer, fileLink)
			}
		}
	}
	logrus.WithField("GarbageCollection", "manifests").Infof("Количество манифестов: %d", len(buffer))
	return buffer, nil
}

/*
parseActiveTags читает директорию с тегами и возвращает список активных тегов.

Params:
  - path: путь к директории с тегами. Формат: <manifest>/<repo>/tags/<tag>

Returns:
  - []string: список активных тегов
*/
func parseActiveTags(path string) []string {
	var tagsLink []string
	var activeTags []string
	tagPath := filepath.Join(path, "tags")
	tags, _ := os.ReadDir(tagPath)
	for _, tag := range tags {
		filePath := filepath.Join(tagPath, tag.Name())
		logrus.WithField("GarbageCollection", "scan").Debug("Чтение файла: ", filePath)
		data, err := os.ReadFile(filePath)
		if err != nil {
			logrus.WithField("GarbageCollection", "scan").WithError(err).Warn("Ошибка чтения файла: ", filePath)
			continue
		}
		tagsLink = append(tagsLink, string(data))
	}

	for _, file := range tagsLink {
		logrus.WithField("GarbageCollection", "scan").Debug("Чтение файла: ", file)
		body := struct {
			MediaType string `json:"mediaType"`
		}{}
		data, err := os.ReadFile(filepath.Join(path, file))
		if err != nil {
			logrus.WithField("GarbageCollection", "scan").WithError(err).Warn("Ошибка чтения файла: ", file)
			break
		}
		json.Unmarshal(data, &body)
		switch body.MediaType {
		case config.MANIFEST_TYPE["manifest"]:
			activeTags = append(activeTags, file)
		case config.MANIFEST_TYPE["index"]:
			var index config.Index
			json.Unmarshal(data, &index)
			activeTags = append(activeTags, file) // добавляем сам тег
			for _, manifest := range index.Manifests {
				activeTags = append(activeTags, manifest.Digest)
			}
		default:
			logrus.WithField("GarbageCollection", "scan").Warn("Неизвестный тип медиа: ", file)
		}
	}
	return activeTags
}

func parseActiveTagsStrict(path string) ([]string, error) {
	var tagsLink []string
	var activeTags []string
	tagPath := filepath.Join(path, "tags")
	tags, err := os.ReadDir(tagPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать директорию тегов %s: %w", tagPath, err)
	}
	for _, tag := range tags {
		if tag.IsDir() {
			continue
		}
		filePath := filepath.Join(tagPath, tag.Name())
		logrus.WithField("GarbageCollection", "scan").Debug("Чтение файла: ", filePath)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать tag-файл %s: %w", filePath, err)
		}
		tagsLink = append(tagsLink, string(data))
	}

	for _, file := range tagsLink {
		logrus.WithField("GarbageCollection", "scan").Debug("Чтение файла: ", file)
		data, err := os.ReadFile(filepath.Join(path, file))
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать манифест %s: %w", file, err)
		}
		var body struct {
			MediaType string `json:"mediaType"`
		}
		if err := json.Unmarshal(data, &body); err != nil {
			return nil, fmt.Errorf("не удалось распарсить mediaType манифеста %s: %w", file, err)
		}
		switch body.MediaType {
		case config.MANIFEST_TYPE["manifest"]:
			activeTags = append(activeTags, file)
		case config.MANIFEST_TYPE["index"]:
			var index config.Index
			if err := json.Unmarshal(data, &index); err != nil {
				return nil, fmt.Errorf("не удалось распарсить index-манифест %s: %w", file, err)
			}
			activeTags = append(activeTags, file)
			for _, manifest := range index.Manifests {
				activeTags = append(activeTags, manifest.Digest)
			}
		default:
			return nil, fmt.Errorf("неизвестный mediaType %q в манифесте %s", body.MediaType, file)
		}
	}
	return activeTags, nil
}

/*
parseManifests сканирует директорию и возвращает список используемых манифестов.

Returns:
  - []string: список манифестов.
*/
func parseManifests(path string) []string {
	var manifests []string
	files, _ := os.ReadDir(path)
	for _, file := range files {
		if !file.IsDir() {
			logrus.WithField("GarbageCollection", "scan").Debug("Чтение файла: ", file.Name())
			manifests = append(manifests, file.Name())
		}
	}
	return manifests
}

func parseManifestsStrict(path string) ([]string, error) {
	var manifests []string
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать директорию манифестов %s: %w", path, err)
	}
	for _, file := range files {
		if !file.IsDir() {
			logrus.WithField("GarbageCollection", "scan").Debug("Чтение файла: ", file.Name())
			manifests = append(manifests, file.Name())
		}
	}
	return manifests, nil
}

/*
parseUsageBlobs сканирует список ссылок на файлы и возвращает список используемых слоев.

Returns:
  - []string: ссылки на используемые слои.
*/
func parseUsageBlobs(links []string) []string {
	var buffer []string
	var manifest config.Manifest
	// var index config.Index
	body := struct {
		MediaType string `json:"mediaType"`
	}{}
	for _, link := range links {
		file, err := os.ReadFile(link)
		if err != nil {
			logrus.WithField("GarbageCollection", "scan").WithError(err).Warn("Ошибка чтения файла: ", link)
			continue
		}
		json.Unmarshal(file, &body)
		switch body.MediaType {
		case config.MANIFEST_TYPE["manifest"]:
			json.Unmarshal(file, &manifest)
			configBlob := strings.Split(manifest.Config.Digest, ":")[1]
			buffer = append(buffer, filepath.Join(config.BLOBS_PATH, configBlob))
			for _, layer := range manifest.Layers {
				layerBlob := strings.Split(layer.Digest, ":")[1]
				buffer = append(buffer, filepath.Join(config.BLOBS_PATH, layerBlob))
			}
		case config.MANIFEST_TYPE["index"]:
			continue
		// здесь ссылки на манифесты типа manifest
		// никак не обрабатываются???
		default:
			logrus.WithField("GarbageCollection", "scan").Warn("Неизвестный тип файла: ", link)
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
	return result
}

func parseUsageBlobsStrict(links []string) ([]string, error) {
	var buffer []string
	for _, link := range links {
		file, err := os.ReadFile(link)
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

func blobPathFromDigest(digest string) (string, error) {
	algorithm, encoded, ok := strings.Cut(digest, ":")
	if !ok || algorithm != "sha256" || !isSHA256Hex(encoded) {
		return "", fmt.Errorf("digest должен иметь формат sha256:<hex>, получено %q", digest)
	}
	return filepath.Join(config.BLOBS_PATH, encoded), nil
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, ch := range value {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
			continue
		}
		return false
	}
	return true
}
