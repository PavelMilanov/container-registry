package system

import (
	"testing"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/storage"
)

func TestGarbageCollection(t *testing.T) {
	env := config.NewEnv()
	storage := storage.NewStorage(env)
	GarbageCollection(storage)
}
