package storage

import (
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func initConfig() *config.Env {
	env, _ := config.NewEnv("../", "test.config")
	return env
}

func TestNewS3Storage(t *testing.T) {
	env := initConfig()
	testS3, err := NewStorage(env)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(testS3)
}

func TestCheckBlob(t *testing.T) {
	env := initConfig()
	testS3, err := NewStorage(env)
	if err != nil {
		t.Fatal(err)
	}
	blob := "test_blob"
	err = testS3.CheckBlob(blob)
	if err != nil {
		t.Fatal(err)
	}
}
