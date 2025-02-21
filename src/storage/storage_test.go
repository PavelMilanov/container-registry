package storage

import (
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func TestNewStorage(t *testing.T) {
	// s3 := NewS3storage("192.168.12.27:9001", "jlSDwyyoeI71kbnRC1wK", "sBqscm1bmreddLINb5aQqSU2gq6qbmpeqVZ9OxVK")
	// t.Log(s3)
}

func TestGarbageCollection(t *testing.T) {
	env := config.NewEnv()
	storage := NewStorage(env)
	storage.GarbageCollection()
}
