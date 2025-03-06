package main

import (
	"os"
	"testing"
)

func TestNewStorage(t *testing.T) {
	t.Log(s)
}

func TestGarbageCollection(t *testing.T) {
	s.GarbageCollection()
	os.Remove("test.db")
}
