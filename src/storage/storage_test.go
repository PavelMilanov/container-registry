package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func TestInventoryBlobs(t *testing.T) {
	withTempInventoryPaths(t)

	blobPath := filepath.Join(config.BLOBS_PATH, "blob-a")
	if err := os.WriteFile(blobPath, []byte("blob"), 0644); err != nil {
		t.Fatal(err)
	}

	blobs := inventoryBlobs()
	if len(blobs) != 1 {
		t.Fatalf("len(blobs) = %d, want 1", len(blobs))
	}
	if blobs[0] != blobPath {
		t.Fatalf("blob path = %q, want %q", blobs[0], blobPath)
	}
}

func TestInventoryManifests(t *testing.T) {
	withTempInventoryPaths(t)

	repoPath := filepath.Join(config.MANIFEST_PATH, "dev", "registry")
	activeDigest := "sha256:active"
	unusedDigest := "sha256:unused"
	writeInventoryFile(t, filepath.Join(repoPath, "tags", "latest"), activeDigest)
	writeInventoryFile(t, filepath.Join(repoPath, activeDigest), `{"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)
	writeInventoryFile(t, filepath.Join(repoPath, unusedDigest), `{"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)

	activeTags := parseActiveTags(repoPath)
	if len(activeTags) != 1 || activeTags[0] != activeDigest {
		t.Fatalf("activeTags = %+v, want [%s]", activeTags, activeDigest)
	}

	manifests := parseManifests(repoPath)
	if len(manifests) != 2 {
		t.Fatalf("len(manifests) = %d, want 2", len(manifests))
	}
}

func withTempInventoryPaths(t *testing.T) {
	t.Helper()

	oldDataPath := config.DATA_PATH
	oldManifestPath := config.MANIFEST_PATH
	oldBlobsPath := config.BLOBS_PATH
	oldTmpPath := config.TMP_PATH

	config.DATA_PATH = t.TempDir()
	config.MANIFEST_PATH = filepath.Join(config.DATA_PATH, "manifests")
	config.BLOBS_PATH = filepath.Join(config.DATA_PATH, "blobs")
	config.TMP_PATH = filepath.Join(config.DATA_PATH, "tmp")

	for _, path := range []string{config.MANIFEST_PATH, config.BLOBS_PATH, config.TMP_PATH} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}

	t.Cleanup(func() {
		config.DATA_PATH = oldDataPath
		config.MANIFEST_PATH = oldManifestPath
		config.BLOBS_PATH = oldBlobsPath
		config.TMP_PATH = oldTmpPath
	})
}

func writeInventoryFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func withTempStoragePaths(t *testing.T) {
	t.Helper()

	oldDataPath := config.DATA_PATH
	oldManifestPath := config.MANIFEST_PATH
	oldBlobsPath := config.BLOBS_PATH
	oldTmpPath := config.TMP_PATH

	config.DATA_PATH = t.TempDir()
	config.MANIFEST_PATH = filepath.Join(config.DATA_PATH, "manifests")
	config.BLOBS_PATH = filepath.Join(config.DATA_PATH, "blobs")
	config.TMP_PATH = filepath.Join(config.DATA_PATH, "tmp")

	for _, path := range []string{config.MANIFEST_PATH, config.BLOBS_PATH, config.TMP_PATH} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}

	t.Cleanup(func() {
		config.DATA_PATH = oldDataPath
		config.MANIFEST_PATH = oldManifestPath
		config.BLOBS_PATH = oldBlobsPath
		config.TMP_PATH = oldTmpPath
	})
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestGarbageCollectionDeletesOnlyUnusedBlobs(t *testing.T) {
	withTempStoragePaths(t)

	manifestDigest := "sha256:manifest"
	configBlob := strings.Repeat("a", 64)
	layerBlob := strings.Repeat("b", 64)
	configDigest := "sha256:" + configBlob
	layerDigest := "sha256:" + layerBlob
	repoPath := filepath.Join(config.MANIFEST_PATH, "dev", "postgres")

	writeFile(t, filepath.Join(repoPath, "tags", "latest"), manifestDigest)
	writeFile(t, filepath.Join(repoPath, manifestDigest), `{
		"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"digest":"`+configDigest+`","size":1},
		"layers":[{"digest":"`+layerDigest+`","size":1}]
	}`)
	writeFile(t, filepath.Join(config.BLOBS_PATH, configBlob), "used config")
	writeFile(t, filepath.Join(config.BLOBS_PATH, layerBlob), "used layer")
	writeFile(t, filepath.Join(config.BLOBS_PATH, "unused"), "unused")

	storage := &LocalStorage{}
	if err := storage.GarbageCollection(); err != nil {
		t.Fatal(err)
	}

	if !fileExists(filepath.Join(config.BLOBS_PATH, configBlob)) {
		t.Fatal("used config blob was deleted")
	}
	if !fileExists(filepath.Join(config.BLOBS_PATH, layerBlob)) {
		t.Fatal("used layer blob was deleted")
	}
	if fileExists(filepath.Join(config.BLOBS_PATH, "unused")) {
		t.Fatal("unused blob was not deleted")
	}
}

func TestGarbageCollectionDoesNotDeleteBlobsWhenManifestInventoryFails(t *testing.T) {
	withTempStoragePaths(t)

	blobPath := filepath.Join(config.BLOBS_PATH, "orphan")
	writeFile(t, blobPath, "must stay")
	if err := os.RemoveAll(config.MANIFEST_PATH); err != nil {
		t.Fatal(err)
	}

	storage := &LocalStorage{}
	if err := storage.GarbageCollection(); err == nil {
		t.Fatal("expected error, got nil")
	}
	if !fileExists(blobPath) {
		t.Fatal("blob was deleted after manifest inventory failure")
	}
}

func TestGarbageCollectionDoesNotDeleteBlobsWhenTagsDirMissing(t *testing.T) {
	withTempStoragePaths(t)

	repoPath := filepath.Join(config.MANIFEST_PATH, "dev", "postgres")
	configBlob := strings.Repeat("a", 64)
	writeFile(t, filepath.Join(repoPath, "sha256:manifest"), `{
		"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"digest":"sha256:`+configBlob+`","size":1},
		"layers":[]
	}`)
	blobPath := filepath.Join(config.BLOBS_PATH, configBlob)
	writeFile(t, blobPath, "must stay")

	storage := &LocalStorage{}
	if err := storage.GarbageCollection(); err == nil {
		t.Fatal("expected error, got nil")
	}
	if !fileExists(blobPath) {
		t.Fatal("blob was deleted when tags directory was missing")
	}
}

func TestGarbageCollectionDoesNotDeleteBlobsWhenDigestIsInvalid(t *testing.T) {
	withTempStoragePaths(t)

	manifestDigest := "sha256:manifest"
	repoPath := filepath.Join(config.MANIFEST_PATH, "dev", "postgres")
	writeFile(t, filepath.Join(repoPath, "tags", "latest"), manifestDigest)
	writeFile(t, filepath.Join(repoPath, manifestDigest), `{
		"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"digest":"invalid","size":1},
		"layers":[]
	}`)
	blobPath := filepath.Join(config.BLOBS_PATH, "candidate")
	writeFile(t, blobPath, "must stay")

	storage := &LocalStorage{}
	if err := storage.GarbageCollection(); err == nil {
		t.Fatal("expected error, got nil")
	}
	if !fileExists(blobPath) {
		t.Fatal("blob was deleted after invalid manifest digest")
	}
}
