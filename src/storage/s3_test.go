package storage

import (
	"os"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func initConfig(t *testing.T) *config.Env {
	t.Helper()
	env, err := config.NewEnv("../", "test.config")
	if err != nil {
		t.Skipf("s3 test config not found: %v", err)
	}
	return env
}

func TestNewS3Storage(t *testing.T) {
	if os.Getenv("CR_RUN_S3_TESTS") != "1" {
		t.Skip("set CR_RUN_S3_TESTS=1 to run S3 integration tests")
	}
	env := initConfig(t)
	testS3, err := NewStorage(env)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(testS3)
}

func TestCheckBlob(t *testing.T) {
	if os.Getenv("CR_RUN_S3_TESTS") != "1" {
		t.Skip("set CR_RUN_S3_TESTS=1 to run S3 integration tests")
	}
	env := initConfig(t)
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
