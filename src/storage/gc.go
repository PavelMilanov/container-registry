package storage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/sirupsen/logrus"
)

const gcRetention = 24 * time.Hour
const gcTimeout = 10 * time.Minute

type gcObject struct {
	Key      string
	Modified time.Time
}
type gcRoot struct {
	Repository string
	Digest     string
}
type gcSnapshot struct {
	Manifests map[string]gcObject
	Blobs     map[string]gcObject
	Roots     []gcRoot
}

type gcBackend interface {
	gcInventory(context.Context) (gcSnapshot, error)
	gcRead(context.Context, string) ([]byte, error)
	gcDelete(context.Context, string) error
	gcManifestKey(string, string) string
	gcBlobKey(string) (string, error)
}

/*
collectGarbage выполняет общий mark-and-sweep без удаления во время сканирования.

Теги и свежие манифесты являются корнями. Неизвестные файлы и uploads не трогаются.
Любая ошибка mark останавливает GC до первого удаления; ошибки sweep возвращаются.
*/
func collectGarbage(ctx context.Context, backend gcBackend, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	snapshot, err := backend.gcInventory(ctx)
	if err != nil {
		return err
	}
	for _, objects := range []map[string]gcObject{snapshot.Manifests, snapshot.Blobs} {
		for key, object := range objects {
			if object.Modified.IsZero() {
				return fmt.Errorf("неизвестный возраст объекта GC: %s", key)
			}
		}
	}
	deadline := now.Add(-gcRetention)
	marked := make(map[string]bool)
	visiting := make(map[string]bool)
	usedBlobs := make(map[string]bool)
	var visit func(string, string, int) error
	visit = func(repository, digest string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := parseSHA256Digest(digest); err != nil {
			return fmt.Errorf("некорректная ссылка manifest %q: %w", digest, err)
		}
		key := backend.gcManifestKey(repository, digest)
		if visiting[key] {
			return fmt.Errorf("цикл ссылок manifest: %s", key)
		}
		if marked[key] {
			return nil
		}
		if depth > 128 {
			return fmt.Errorf("слишком глубокий граф manifest: %s", key)
		}
		if _, ok := snapshot.Manifests[key]; !ok {
			return fmt.Errorf("manifest отсутствует: %s", key)
		}
		data, err := backend.gcRead(ctx, key)
		if err != nil {
			return fmt.Errorf("чтение manifest %s: %w", key, err)
		}
		if actual := fmt.Sprintf("sha256:%x", sha256.Sum256(data)); actual != digest {
			return fmt.Errorf("digest manifest %s не соответствует содержимому: %w", key, ErrBlobCorrupted)
		}
		var body struct {
			MediaType string `json:"mediaType"`
			Config    struct {
				Digest string `json:"digest"`
			} `json:"config"`
			Layers []struct {
				Digest string `json:"digest"`
			} `json:"layers"`
			Manifests []struct {
				Digest string `json:"digest"`
			} `json:"manifests"`
			Subject *struct {
				Digest string `json:"digest"`
			} `json:"subject"`
		}
		if err := json.Unmarshal(data, &body); err != nil {
			return fmt.Errorf("JSON manifest %s: %w", key, err)
		}
		visiting[key] = true
		defer delete(visiting, key)
		markBlob := func(digest string) error {
			blob, err := backend.gcBlobKey(digest)
			if err != nil {
				return fmt.Errorf("digest Blob в %s: %w", key, err)
			}
			if _, ok := snapshot.Blobs[blob]; !ok {
				return fmt.Errorf("Blob %s из %s отсутствует", digest, key)
			}
			usedBlobs[blob] = true
			return nil
		}
		switch body.MediaType {
		case config.MANIFEST_TYPE["manifest"], "application/vnd.docker.distribution.manifest.v2+json":
			if err := markBlob(body.Config.Digest); err != nil {
				return err
			}
			for _, layer := range body.Layers {
				if err := markBlob(layer.Digest); err != nil {
					return err
				}
			}
		case config.MANIFEST_TYPE["index"], "application/vnd.docker.distribution.manifest.list.v2+json":
			for _, child := range body.Manifests {
				if err := visit(repository, child.Digest, depth+1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("неподдерживаемый mediaType %q в %s", body.MediaType, key)
		}
		if body.Subject != nil {
			if err := visit(repository, body.Subject.Digest, depth+1); err != nil {
				return err
			}
		}
		marked[key] = true
		return nil
	}
	for _, root := range snapshot.Roots {
		if err := visit(root.Repository, root.Digest, 0); err != nil {
			return err
		}
	}
	// Свежие untagged manifests также защищают свой граф (push по digest до index/tag).
	for key, object := range snapshot.Manifests {
		if object.Modified.Before(deadline) {
			continue
		}
		root := splitGCManifestKey(key)
		if err := visit(root.Repository, root.Digest, 0); err != nil {
			return err
		}
	}
	logger := logrus.WithFields(logrus.Fields{"task": "garbage_collection", "manifests": len(snapshot.Manifests), "blobs": len(snapshot.Blobs), "reachable_manifests": len(marked), "reachable_blobs": len(usedBlobs), "retention": gcRetention})
	logger.Info("Сканирование GC завершено")
	var candidates []string
	for key, object := range snapshot.Manifests {
		if !marked[key] && object.Modified.Before(deadline) {
			candidates = append(candidates, key)
		}
	}
	for key, object := range snapshot.Blobs {
		if !usedBlobs[key] && object.Modified.Before(deadline) {
			candidates = append(candidates, key)
		}
	}
	// Сначала удаляем metadata, затем данные, на которые metadata могли ссылаться.
	sort.Slice(candidates, func(i, j int) bool {
		_, left := snapshot.Manifests[candidates[i]]
		_, right := snapshot.Manifests[candidates[j]]
		if left != right {
			return left
		}
		return candidates[i] < candidates[j]
	})
	deleted := 0
	var resultErr error
	for _, key := range candidates {
		if _, blob := snapshot.Blobs[key]; blob && resultErr != nil {
			// Не оставляем неудалённый manifest без его Blob при ошибке удаления metadata.
			break
		}
		if err := ctx.Err(); err != nil {
			resultErr = errors.Join(resultErr, err)
			break
		}
		if err := backend.gcDelete(ctx, key); err != nil {
			logger.WithField("key", key).WithError(err).Error("Не удалось удалить объект GC")
			resultErr = errors.Join(resultErr, fmt.Errorf("удаление %s: %w", key, err))
		} else {
			deleted++
		}
	}
	logger.WithFields(logrus.Fields{"candidates": len(candidates), "deleted": deleted}).Info("Удаление объектов GC завершено")
	return errors.Join(resultErr, ctx.Err())
}
