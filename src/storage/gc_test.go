package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

type memoryGC struct {
	snapshot     gcSnapshot
	data         map[string][]byte
	deleted      []string
	deleteErr    error
	inventoryErr error
}

func (m *memoryGC) gcInventory(context.Context) (gcSnapshot, error) {
	return m.snapshot, m.inventoryErr
}
func (m *memoryGC) gcRead(ctx context.Context, key string) ([]byte, error) {
	return m.data[key], ctx.Err()
}
func (m *memoryGC) gcDelete(ctx context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = append(m.deleted, key)
	return ctx.Err()
}
func (m *memoryGC) gcManifestKey(repo, digest string) string { return filepath.Join(repo, digest) }
func (m *memoryGC) gcBlobKey(digest string) (string, error) {
	encoded, err := parseSHA256Digest(digest)
	return "blobs/" + encoded, err
}

/*
newMemoryGC создаёт inventory для проверки mark-and-sweep без внешнего хранилища.
*/
func newMemoryGC() *memoryGC {
	return &memoryGC{snapshot: gcSnapshot{Manifests: make(map[string]gcObject), Blobs: make(map[string]gcObject)}, data: make(map[string][]byte)}
}

/*
manifest добавляет manifest с digest, соответствующим его содержимому.
*/
func (m *memoryGC) manifest(body string, modified time.Time) string {
	digest := testBlobDigest([]byte(body))
	key := m.gcManifestKey("manifests/dev/image", digest)
	m.snapshot.Manifests[key] = gcObject{key, modified}
	m.data[key] = []byte(body)
	return digest
}

func TestGCMarkSweepProtectsNestedGraph(t *testing.T) {
	now := time.Now()
	old := now.Add(-25 * time.Hour)
	m := newMemoryGC()
	blobDigest := "sha256:" + strings.Repeat("a", 64)
	blobKey, _ := m.gcBlobKey(blobDigest)
	m.snapshot.Blobs[blobKey] = gcObject{blobKey, old}
	child := m.manifest(fmt.Sprintf(`{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"digest":%q},"layers":[]}`, blobDigest), old)
	for range 3 {
		child = m.manifest(fmt.Sprintf(`{"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"digest":%q}]}`, child), old)
	}
	m.snapshot.Roots = []gcRoot{{"manifests/dev/image", child}}
	orphan := "blobs/" + strings.Repeat("b", 64)
	m.snapshot.Blobs[orphan] = gcObject{orphan, old}
	if err := collectGarbage(context.Background(), m, now); err != nil {
		t.Fatal(err)
	}
	if len(m.deleted) != 1 || m.deleted[0] != orphan {
		t.Fatal(m.deleted)
	}
}

func TestGCProtectsFreshUntaggedGraphAndBlobs(t *testing.T) {
	now := time.Now()
	m := newMemoryGC()
	digest := "sha256:" + strings.Repeat("a", 64)
	key, _ := m.gcBlobKey(digest)
	m.snapshot.Blobs[key] = gcObject{key, now.Add(-48 * time.Hour)}
	m.manifest(fmt.Sprintf(`{"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"digest":%q},"layers":[]}`, digest), now)
	orphan := "blobs/" + strings.Repeat("b", 64)
	m.snapshot.Blobs[orphan] = gcObject{orphan, now}
	if err := collectGarbage(context.Background(), m, now); err != nil {
		t.Fatal(err)
	}
	if len(m.deleted) != 0 {
		t.Fatal(m.deleted)
	}
}

func TestGCScanFailuresNeverDelete(t *testing.T) {
	for _, failure := range []string{"inventory", "missing-child", "missing-blob", "modified-manifest", "unsupported-media", "invalid-tag", "missing-age"} {
		t.Run(failure, func(t *testing.T) {
			now := time.Now()
			old := now.Add(-48 * time.Hour)
			m := newMemoryGC()
			candidate := "blobs/" + strings.Repeat("b", 64)
			m.snapshot.Blobs[candidate] = gcObject{candidate, old}
			digest := "sha256:" + strings.Repeat("a", 64)
			switch failure {
			case "inventory":
				m.inventoryErr = errors.New("scan failed")
			case "missing-child":
				digest = m.manifest(fmt.Sprintf(`{"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[{"digest":%q}]}`, digest), old)
			case "missing-blob":
				digest = m.manifest(fmt.Sprintf(`{"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"digest":%q}}`, digest), old)
			case "modified-manifest":
				digest = m.manifest(`{"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[]}`, old)
				m.data[m.gcManifestKey("manifests/dev/image", digest)] = []byte("modified")
			case "unsupported-media":
				digest = m.manifest(`{"mediaType":"unknown"}`, old)
			case "invalid-tag":
				digest = "../../escape"
			case "missing-age":
				m.snapshot.Blobs[candidate] = gcObject{Key: candidate}
			}
			m.snapshot.Roots = []gcRoot{{"manifests/dev/image", digest}}
			if err := collectGarbage(context.Background(), m, now); err == nil {
				t.Fatal("expected scan error")
			}
			if len(m.deleted) != 0 {
				t.Fatal(m.deleted)
			}
		})
	}
}

/*
TestGCBackendContract проверяет одинаковый граф, grace period и fail-closed в Local/S3.
Возраст моделируется часами GC; внешнее хранилище использует изолированный bucket.
*/
func TestGCBackendContract(t *testing.T) {
	for _, name := range []string{"local", "s3"} {
		t.Run(name, func(t *testing.T) {
			var backend gcBackend
			var repository string
			write := func(key, body string) {
				t.Helper()
				if s, ok := backend.(*S3Storage); ok {
					_, err := s.S3.PutObject(context.Background(), s.bucketName(), key, strings.NewReader(body), int64(len(body)), minio.PutObjectOptions{})
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				if err := os.MkdirAll(filepath.Dir(key), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(key, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if name == "local" {
				withTempStoragePaths(t)
				backend = &LocalStorage{}
			} else {
				backend = newTestS3Storage(t)
			}
			blob, _ := backend.gcBlobKey(testBlobDigest([]byte("used")))
			// Выводим repository из manifest-key backend, не из глобального пути S3.
			if s, ok := backend.(*S3Storage); ok {
				repository = filepath.Join(s.objectPrefix("manifests"), "dev", "image")
			} else {
				repository = filepath.Join(filepath.Dir(filepath.Dir(blob)), "manifests", "dev", "image")
			}
			write(blob, "used")
			orphan, _ := backend.gcBlobKey(testBlobDigest([]byte("orphan")))
			write(orphan, "orphan")
			unknown := filepath.Join(filepath.Dir(orphan), "operator-notes.txt")
			write(unknown, "not a Blob")
			childBody := fmt.Sprintf(`{"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"digest":%q},"layers":[]}`, testBlobDigest([]byte("used")))
			childDigest := testBlobDigest([]byte(childBody))
			childKey := backend.gcManifestKey(repository, childDigest)
			write(childKey, childBody)
			parentBody := fmt.Sprintf(`{"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[{"digest":%q}]}`, childDigest)
			parentDigest := testBlobDigest([]byte(parentBody))
			parentKey := backend.gcManifestKey(repository, parentDigest)
			write(parentKey, parentBody)
			ctx := context.Background()
			// Пока тег не появился, свежий untagged index сохраняет граф и orphan Blob.
			if err := collectGarbage(ctx, backend, time.Now()); err != nil {
				t.Fatal(err)
			}
			if _, err := backend.gcRead(ctx, orphan); err != nil {
				t.Fatal(err)
			}
			write(filepath.Join(repository, "tags", "latest"), parentDigest)
			future := time.Now().Add(25 * time.Hour)
			if err := collectGarbage(ctx, backend, future); err != nil {
				t.Fatal(err)
			}
			if _, err := backend.gcRead(ctx, orphan); err == nil {
				t.Fatal("old orphan retained")
			}
			for _, key := range []string{blob, childKey, parentKey, unknown} {
				if _, err := backend.gcRead(ctx, key); err != nil {
					t.Fatalf("reachable/unknown object %s: %v", key, err)
				}
			}
			write(orphan, "orphan")
			write(childKey, "corrupted")
			if err := collectGarbage(ctx, backend, future); err == nil {
				t.Fatal("expected corrupted manifest error")
			}
			if _, err := backend.gcRead(ctx, orphan); err != nil {
				t.Fatal("deletion happened before mark finished", err)
			}
			write(childKey, childBody)
			if err := backend.(TagStore).DeleteManifest("dev", "image", "latest"); err != nil {
				t.Fatal(err)
			}
			// Удаление тега не удаляет manifest синхронно: граф может использовать другой index.
			if _, err := backend.gcRead(ctx, parentKey); err != nil {
				t.Fatal(err)
			}
			if err := collectGarbage(ctx, backend, future); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{blob, childKey, parentKey, orphan} {
				if _, err := backend.gcRead(ctx, key); err == nil {
					t.Fatalf("untagged old object retained: %s", key)
				}
			}
		})
	}
}

func TestGCDeletionFailureAndCanceledContext(t *testing.T) {
	m := newMemoryGC()
	key := "blobs/" + strings.Repeat("a", 64)
	m.snapshot.Blobs[key] = gcObject{key, time.Now().Add(-48 * time.Hour)}
	m.deleteErr = errors.New("delete failed")
	if err := collectGarbage(context.Background(), m, time.Now()); !errors.Is(err, m.deleteErr) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := collectGarbage(ctx, m, time.Now()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(m.deleted) != 0 {
		t.Fatal(m.deleted)
	}
}

/*
TestGCManifestDeletionFailurePreservesBlobs запрещает удалять данные при ошибке metadata.
*/
func TestGCManifestDeletionFailurePreservesBlobs(t *testing.T) {
	m := newMemoryGC()
	old := time.Now().Add(-48 * time.Hour)
	m.manifest(`{"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[]}`, old)
	key := "blobs/" + strings.Repeat("a", 64)
	m.snapshot.Blobs[key] = gcObject{key, old}
	backend := &recordingGC{memoryGC: m}
	if err := collectGarbage(context.Background(), backend, time.Now()); err == nil {
		t.Fatal("expected deletion error")
	}
	if len(backend.attempts) != 1 || strings.HasPrefix(backend.attempts[0], "blobs/") {
		t.Fatal(backend.attempts)
	}
}

type recordingGC struct {
	*memoryGC
	attempts []string
}

/*
gcDelete фиксирует попытки удаления и моделирует ошибку первого manifest.
*/
func (m *recordingGC) gcDelete(ctx context.Context, key string) error {
	m.attempts = append(m.attempts, key)
	return errors.New("manifest deletion denied")
}

func TestMaintenanceGateExcludesWritersAndHonorsTimeout(t *testing.T) {
	var gate maintenanceGate
	mutation, err := gate.lock(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := gate.lock(ctx, true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	mutation()
	gc, err := gate.lock(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel2()
	if _, err := gate.lock(ctx2, false); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	gc()
}
