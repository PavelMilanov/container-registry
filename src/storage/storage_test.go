package storage

import (
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func TestGarbageCollection(t *testing.T) {
	env := config.NewEnv(config.CONFIG_PATH, "test.config")
	s := NewStorage(env)
	s.GarbageCollection()
}
