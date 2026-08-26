package services

import (
	"errors"
	"testing"
)

type fakeCloudStore struct {
	added   string
	deleted string
	clouds  []string
	err     error
}

func (f *fakeCloudStore) AddCloud(name string) error {
	f.added = name
	return f.err
}

func (f *fakeCloudStore) DeleteCloud(name string) error {
	f.deleted = name
	return f.err
}

func (f *fakeCloudStore) GetCloudList() ([]string, error) {
	return f.clouds, f.err
}

type fakeGarbageCollector struct {
	called bool
	err    error
}

func (f *fakeGarbageCollector) GarbageCollection() error {
	f.called = true
	return f.err
}

func TestCloudServicesUseCloudCapability(t *testing.T) {
	store := &fakeCloudStore{clouds: []string{"dev"}}
	if err := AddCloud("test", store); err != nil {
		t.Fatal(err)
	}
	if store.added != "test" {
		t.Fatalf("added cloud = %q", store.added)
	}

	clouds, err := GetCloudList(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(clouds) != 1 || clouds[0] != "dev" {
		t.Fatalf("clouds = %#v", clouds)
	}
}

func TestGarbageCollectionUsesDedicatedCapability(t *testing.T) {
	expectedErr := errors.New("collection failed")
	collector := &fakeGarbageCollector{err: expectedErr}

	err := GarbageCollection(collector)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
	if !collector.called {
		t.Fatal("garbage collector was not called")
	}
}
