package storage

import (
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func TestNewStorageBuildsLocalBackend(t *testing.T) {
	withTempStoragePaths(t)
	env := &config.Env{}
	env.Storage.Type = "local"

	backend, err := NewStorage(env)
	if err != nil {
		t.Fatal(err)
	}
	assertPersistentCapabilities(t, backend)
	if backend.Uploads == nil {
		t.Fatal("local backend does not provide BlobUploadStore")
	}
	if backend.UploadCleaner == nil {
		t.Fatal("local backend does not provide UploadCleaner")
	}
}

func TestBackendFromS3LeavesUnsupportedCapabilitiesEmpty(t *testing.T) {
	backend := backendFromS3(&S3Storage{})

	assertPersistentCapabilities(t, backend)
	if backend.Uploads != nil {
		t.Fatal("S3 backend unexpectedly provides BlobUploadStore")
	}
	if backend.UploadCleaner != nil {
		t.Fatal("S3 backend unexpectedly provides UploadCleaner")
	}
}

func TestNewStorageRejectsUnknownBackend(t *testing.T) {
	env := &config.Env{}
	env.Storage.Type = "unknown"

	backend, err := NewStorage(env)
	if err == nil {
		t.Fatal("unknown backend error is nil")
	}
	if backend != (Backend{}) {
		t.Fatalf("backend = %#v, want zero value", backend)
	}
}

func assertPersistentCapabilities(t *testing.T, backend Backend) {
	t.Helper()

	capabilities := map[string]any{
		"blobs":            backend.Blobs,
		"manifests":        backend.Manifests,
		"namespaces":       backend.Namespaces,
		"clouds":           backend.Clouds,
		"repositories":     backend.Repositories,
		"tags":             backend.Tags,
		"garbageCollector": backend.GarbageCollector,
		"tagPruner":        backend.TagPruner,
	}
	for name, capability := range capabilities {
		if capability == nil {
			t.Errorf("backend capability %s is nil", name)
		}
	}
}
