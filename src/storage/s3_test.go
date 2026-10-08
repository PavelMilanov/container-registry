package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/internal/s3test"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

/*
newTestS3Storage создаёт изолированное S3-хранилище без локального uploads.
*/
func newTestS3Storage(t *testing.T) *S3Storage {
	t.Helper()
	client, bucket := s3test.New(t)
	withTempStoragePaths(t)
	config.TMP_PATH = filepath.Join(t.TempDir(), "must-not-be-created")
	return &S3Storage{S3: client, Bucket: bucket}
}

/*
s3TestDigest вычисляет ожидаемый digest тестовых данных.
*/
func s3TestDigest(data []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

func TestS3BlobUploads(t *testing.T) {
	s := newTestS3Storage(t)
	ctx := context.Background()
	for _, size := range []int{0, 31, s3PartSize + 1024} {
		t.Run(fmt.Sprintf("size-%d", size), func(t *testing.T) {
			data := bytes.Repeat([]byte("x"), size)
			id := uuid.NewV4().String()
			if err := s.StartBlobUpload(ctx, id); err != nil {
				t.Fatal(err)
			}
			cut := size / 2
			if offset, err := s.AppendBlobUpload(ctx, id, 0, bytes.NewReader(data[:cut])); err != nil || offset != int64(cut) {
				t.Fatalf("offset=%d err=%v", offset, err)
			}
			// Новый экземпляр продолжает загрузку по состоянию из S3.
			restarted := &S3Storage{S3: s.S3, Bucket: s.Bucket}
			digest := s3TestDigest(data)
			blob, err := restarted.CompleteBlobUpload(ctx, id, digest, bytes.NewReader(data[cut:]))
			if err != nil {
				t.Fatal(err)
			}
			if blob.Size != int64(size) || blob.Path != "" {
				t.Fatalf("blob=%+v", blob)
			}
			reader, info, err := s.OpenBlob(ctx, digest)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(reader)
			reader.Close()
			if err != nil || !bytes.Equal(got, data) || info.Size != int64(size) {
				t.Fatalf("read err=%v size=%d", err, len(got))
			}
			if err := s.CheckBlob(digest); err != nil {
				t.Fatal(err)
			}
			objects, err := s.listObjects(s.uploadPrefix(id), true)
			if err != nil || len(objects) != 0 {
				t.Fatalf("remaining uploads=%d err=%v", len(objects), err)
			}
			id = uuid.NewV4().String()
			if err := s.StartBlobUpload(ctx, id); err != nil {
				t.Fatal(err)
			}
			if _, err := s.CompleteBlobUpload(ctx, id, digest, bytes.NewReader(data)); err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, err := os.Stat(config.TMP_PATH); !os.IsNotExist(err) {
		t.Fatalf("S3 created local upload directory: %v", err)
	}
}

func TestS3UploadValidationAndCancellation(t *testing.T) {
	s := newTestS3Storage(t)
	ctx := context.Background()
	id := uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.StartBlobUpload(ctx, id); err == nil {
		t.Fatal("duplicate upload accepted")
	}
	if _, err := s.AppendBlobUpload(ctx, id, 1, strings.NewReader("abc")); !errors.Is(err, ErrInvalidOffset) {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.AppendBlobUpload(cancelled, id, 0, strings.NewReader("abc")); err == nil {
		t.Fatal("cancelled upload succeeded")
	}
	if offset, err := s.AppendBlobUpload(ctx, id, 0, strings.NewReader("abc")); err != nil || offset != 3 {
		t.Fatalf("offset=%d err=%v", offset, err)
	}
	if _, err := s.CompleteBlobUpload(ctx, id, "sha256:invalid", strings.NewReader("")); !errors.Is(err, ErrInvalidDigest) {
		t.Fatal(err)
	}
	wrong := s3TestDigest([]byte("wrong"))
	if _, err := s.CompleteBlobUpload(ctx, id, wrong, strings.NewReader("")); !errors.Is(err, ErrDigestMismatch) {
		t.Fatal(err)
	}
	if err := s.CheckBlob(wrong); !errors.Is(err, ErrBlobNotFound) {
		t.Fatal(err)
	}
	if _, err := s.readUploadState(ctx, id); !errors.Is(err, ErrUploadNotFound) {
		t.Fatal(err)
	}
	id = uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.AbortBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.AbortBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendBlobUpload(ctx, id, 0, strings.NewReader("")); !errors.Is(err, ErrUploadNotFound) {
		t.Fatal(err)
	}
	if err := s.StartBlobUpload(ctx, "../../escape"); !errors.Is(err, ErrUploadNotFound) {
		t.Fatal(err)
	}
}

func TestS3UploadSerialization(t *testing.T) {
	s := newTestS3Storage(t)
	ctx := context.Background()
	id := uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, err := s.AppendBlobUpload(ctx, id, 0, strings.NewReader("abc")); errs <- err })
	}
	wg.Wait()
	close(errs)
	succeeded, rejected := 0, 0
	for err := range errs {
		if err == nil {
			succeeded++
		} else if errors.Is(err, ErrInvalidOffset) {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("succeeded=%d rejected=%d", succeeded, rejected)
	}
}

func TestS3CleanupUploads(t *testing.T) {
	s := newTestS3Storage(t)
	ctx := context.Background()
	id := uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if deleted, err := s.CleanupUploads(ctx, 24*time.Hour); err != nil || deleted != 0 {
		t.Fatalf("deleted=%d err=%v", deleted, err)
	}
	t.Run("orphan-multipart", func(t *testing.T) {
		orphan := uuid.NewV4().String()
		core := minio.Core{Client: s.S3}
		orphanKey := s.uploadPrefix(orphan) + "parts/orphan"
		protocolID, err := core.NewMultipartUpload(ctx, s.Bucket, orphanKey, minio.PutObjectOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer core.AbortMultipartUpload(ctx, s.Bucket, orphanKey, protocolID)
		listed, err := core.ListMultipartUploads(ctx, s.Bucket, s.uploadPrefix(orphan), "", "", "", 1000)
		if err != nil {
			t.Fatal(err)
		}
		if len(listed.Uploads) == 0 {
			t.Skip("S3 provider does not list the multipart upload it just created; orphan cleanup requires a working ListMultipartUploads API")
		}
		if removed, err := s.cleanupS3Upload(ctx, orphan, time.Now().Add(time.Hour)); err != nil || !removed {
			t.Fatalf("removed=%v err=%v", removed, err)
		}
		listed, err = core.ListMultipartUploads(ctx, s.Bucket, s.uploadPrefix(orphan), "", "", "", 1000)
		if err != nil || len(listed.Uploads) != 0 {
			t.Fatalf("multipart remains=%d err=%v", len(listed.Uploads), err)
		}
	})
	unlock := s.uploadLocks.Lock(id)
	if deleted, err := s.CleanupUploads(ctx, time.Nanosecond); err != nil || deleted != 0 {
		t.Fatalf("active upload deleted=%d err=%v", deleted, err)
	}
	unlock()
	if removed, err := s.cleanupS3Upload(ctx, id, time.Now().Add(time.Hour)); err != nil || !removed {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	if _, err := s.CleanupUploads(ctx, 0); err == nil {
		t.Fatal("invalid retention accepted")
	}
}

func TestS3CleanupOrphanMultipartAPI(t *testing.T) {
	id := uuid.NewV4().String()
	key := "var/tmp/uploads/" + id + "/parts/orphan"
	var aborted atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Has("list-type"):
			fmt.Fprint(w, `<ListBucketResult><Name>registry</Name><IsTruncated>false</IsTruncated></ListBucketResult>`)
		case r.Method == http.MethodGet && r.URL.Query().Has("uploads"):
			if aborted.Load() {
				fmt.Fprint(w, `<ListMultipartUploadsResult><IsTruncated>false</IsTruncated></ListMultipartUploadsResult>`)
				return
			}
			fmt.Fprintf(w, `<ListMultipartUploadsResult><Bucket>registry</Bucket><IsTruncated>false</IsTruncated><Upload><Key>%s</Key><UploadId>protocol-id</UploadId><Initiated>2020-01-01T00:00:00Z</Initiated></Upload></ListMultipartUploadsResult>`, key)
		case r.Method == http.MethodDelete && r.URL.Query().Get("uploadId") == "protocol-id" && r.URL.Path == "/registry/"+key:
			aborted.Store(true)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/registry/var/tmp/uploads/"+id+"/state.json":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected S3 request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	client, err := minio.New(strings.TrimPrefix(server.URL, "http://"), &minio.Options{Region: "us-east-1", Creds: credentials.NewStaticV4("test", "test-secret", "")})
	if err != nil {
		t.Fatal(err)
	}
	s := &S3Storage{S3: client}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	deleted, err := s.CleanupUploads(ctx, 24*time.Hour)
	if err != nil || deleted != 1 || !aborted.Load() {
		t.Fatalf("deleted=%d aborted=%v err=%v", deleted, aborted.Load(), err)
	}
}

func TestS3BackendConfiguration(t *testing.T) {
	client, bucket := s3test.New(t)
	env := &config.Env{}
	env.Storage.Type = "s3"
	env.Storage.Bucket = bucket
	env.Storage.Prefix = "custom-registry"
	env.Storage.Credentials.Endpoint = os.Getenv("CR_S3_ENDPOINT")
	env.Storage.Credentials.AccessKey = os.Getenv("CR_S3_ACCESS_KEY")
	env.Storage.Credentials.SecretKey = os.Getenv("CR_S3_SECRET_KEY")
	env.Storage.Credentials.SSL = os.Getenv("CR_S3_SSL") == "true"
	backend, err := NewStorage(env)
	if err != nil {
		t.Fatal(err)
	}
	assertPersistentCapabilities(t, backend)
	s := backend.Blobs.(*S3Storage)
	id := uuid.NewV4().String()
	ctx := context.Background()
	if err := backend.Uploads.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := client.StatObject(ctx, bucket, "custom-registry/tmp/uploads/"+id+"/state.json", minio.StatObjectOptions{}); err != nil {
		t.Fatal(err)
	}
	data := []byte("custom prefix")
	digest := s3TestDigest(data)
	if _, err := backend.Uploads.CompleteBlobUpload(ctx, id, digest, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckBlob(digest); err != nil {
		t.Fatal(err)
	}
	env.Storage.Bucket = "absent-" + uuid.NewV4().String()
	if _, err := NewStorage(env); err == nil {
		t.Fatal("missing bucket accepted")
	}
}

func TestS3RejectsCorruptedObjects(t *testing.T) {
	s := newTestS3Storage(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	data := []byte("abc")
	digest := s3TestDigest(data)
	id := uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendBlobUpload(ctx, id, 0, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	state, err := s.readUploadState(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.putBytes(state.Parts[0].Key, []byte("longer")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteBlobUpload(ctx, id, digest, strings.NewReader("")); !errors.Is(err, ErrBlobCorrupted) {
		t.Fatalf("damaged part: %v", err)
	}
	if err := s.AbortBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	key, _ := s.blobKeyFromDigest(digest)
	if _, err := s.putBytes(key, []byte("bad")); err != nil {
		t.Fatal(err)
	}
	id = uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteBlobUpload(ctx, id, digest, bytes.NewReader(data)); !errors.Is(err, ErrBlobCorrupted) {
		t.Fatalf("damaged existing blob: %v", err)
	}
}

func TestS3ResumeReadyUpload(t *testing.T) {
	s := newTestS3Storage(t)
	ctx := context.Background()
	id := uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	data := []byte("verified but not yet published")
	if _, err := s.AppendBlobUpload(ctx, id, 0, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	state, err := s.readUploadState(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := s.assembleUpload(ctx, s.uploadPrefix(id)+"blob", state)
	if err != nil {
		t.Fatal(err)
	}
	state.ReadyDigest = digest
	if err := s.writeUploadState(ctx, id, state); err != nil {
		t.Fatal(err)
	}
	restarted := &S3Storage{S3: s.S3, Bucket: s.Bucket}
	if _, err := restarted.CompleteBlobUpload(ctx, id, s3TestDigest([]byte("wrong")), strings.NewReader("")); !errors.Is(err, ErrDigestMismatch) {
		t.Fatal(err)
	}
	if blob, err := restarted.CompleteBlobUpload(ctx, id, digest, strings.NewReader("")); err != nil || blob.Size != int64(len(data)) {
		t.Fatalf("blob=%+v err=%v", blob, err)
	}
}

func TestS3GCNestedIndexesAndInvalidReference(t *testing.T) {
	s := newTestS3Storage(t)
	ctx := context.Background()
	layer := []byte("reachable layer")
	layerDigest := s3TestDigest(layer)
	key, _ := s.blobKeyFromDigest(layerDigest)
	if _, err := s.putBytes(key, layer); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(s.objectPrefix("manifests"), "dev", "image")
	manifest := []byte(fmt.Sprintf(`{"mediaType":%q,"config":{"digest":%q},"layers":[]}`, config.MANIFEST_TYPE["manifest"], layerDigest))
	digest := s3TestDigest(manifest)
	if _, err := s.putBytes(filepath.Join(repo, digest), manifest); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		index := []byte(fmt.Sprintf(`{"mediaType":%q,"manifests":[{"digest":%q}]}`, config.MANIFEST_TYPE["index"], digest))
		digest = s3TestDigest(index)
		if _, err := s.putBytes(filepath.Join(repo, digest), index); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.putBytes(filepath.Join(repo, "tags", "latest"), []byte(digest)); err != nil {
		t.Fatal(err)
	}
	if err := collectGarbage(context.Background(), s, time.Now().Add(25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckBlob(layerDigest); err != nil {
		t.Fatal(err)
	}
	// Неправильная ссылка должна остановить GC до любых удалений.
	unused := s3TestDigest([]byte("unused manifest"))
	unusedKey := filepath.Join(repo, unused)
	if _, err := s.putBytes(unusedKey, []byte("unused manifest")); err != nil {
		t.Fatal(err)
	}
	missing := s3TestDigest([]byte("missing manifest"))
	if _, err := s.putBytes(filepath.Join(repo, "tags", "broken"), []byte(missing)); err != nil {
		t.Fatal(err)
	}
	if err := collectGarbage(context.Background(), s, time.Now().Add(25*time.Hour)); err == nil {
		t.Fatal("GC accepted missing index reference")
	}
	if _, err := s.S3.StatObject(ctx, s.Bucket, unusedKey, minio.StatObjectOptions{}); err != nil {
		t.Fatalf("GC deleted object before validation: %v", err)
	}
	if err := s.DeleteManifest("dev", "image", "broken"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteManifest("dev", "image", "latest"); err != nil {
		t.Fatal(err)
	}
	if err := collectGarbage(context.Background(), s, time.Now().Add(25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckBlob(layerDigest); !errors.Is(err, ErrBlobNotFound) {
		t.Fatalf("unreferenced blob remains: %v", err)
	}
}

func TestS3ManifestsAndAdministration(t *testing.T) {
	s := newTestS3Storage(t)
	ctx := context.Background()
	if err := s.AddCloud("dev"); err != nil {
		t.Fatal(err)
	}
	if exists, err := s.NamespaceExists(ctx, "dev"); err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
	data := []byte("layer")
	digest := s3TestDigest(data)
	id := uuid.NewV4().String()
	if err := s.StartBlobUpload(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteBlobUpload(ctx, id, digest, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"schemaVersion": 2, "mediaType": config.MANIFEST_TYPE["manifest"], "config": map[string]any{"digest": digest}, "layers": []any{map[string]any{"digest": digest}}})
	manifestDigest := s3TestDigest(body)
	for _, tag := range []string{"latest", "v1"} {
		meta := config.Meta{Repository: "dev", Image: "image", Tag: tag, Digest: manifestDigest}
		if err := s.SaveManifest(meta, body, ""); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := s.GetManifest("dev", "image", "latest"); err != nil || !bytes.Equal(got, body) {
		t.Fatalf("manifest err=%v", err)
	}
	if tags, err := s.GetManifestList("dev", "image"); err != nil || len(tags) != 2 {
		t.Fatalf("tags=%v err=%v", tags, err)
	}
	if repos, err := s.GetRepositoriesList("dev"); err != nil || len(repos) != 1 {
		t.Fatalf("repos=%v err=%v", repos, err)
	}
	if clouds, err := s.GetCloudList(); err != nil || len(clouds) != 1 {
		t.Fatalf("clouds=%v err=%v", clouds, err)
	}
	if err := s.DeleteManifest("dev", "image", "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetManifest("dev", "image", manifestDigest); err != nil {
		t.Fatal(err)
	}
	unused := s3TestDigest([]byte("unused"))
	key, _ := s.blobKeyFromDigest(unused)
	if _, err := s.putBytes(key, []byte("unused")); err != nil {
		t.Fatal(err)
	}
	if err := collectGarbage(context.Background(), s, time.Now().Add(25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckBlob(digest); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckBlob(unused); !errors.Is(err, ErrBlobNotFound) {
		t.Fatal(err)
	}
	if err := s.DeleteOlderTags(0); err != nil {
		t.Fatal(err)
	}
	if tags, err := s.GetManifestList("dev", "image"); err != nil || len(tags) != 0 {
		t.Fatalf("tags=%v err=%v", tags, err)
	}
	if err := s.DeleteRepository("dev", "image"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteCloud("dev"); err != nil {
		t.Fatal(err)
	}
	if exists, err := s.NamespaceExists(ctx, "dev"); err != nil || exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
}
