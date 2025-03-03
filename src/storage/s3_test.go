package storage

import (
	"context"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
)

var testS3 = newS3("192.168.12.27:9000", "YT3ITTiJFuO0cASXNY9g", "dj1RFM02yjhuRuBF75W0swehEX5TU4lYitLddWEm", false)

func TestNewS3(t *testing.T) {
	t.Logf("%+v", testS3.Client)
}
func TestPutObject(t *testing.T) {
	object, err := os.Open("test_blob")
	if err != nil {
		t.Fatal(err)
	}
	defer object.Close()
	objectStat, err := object.Stat()
	if err != nil {
		t.Fatal(err)
	}
	info, err := testS3.Client.PutObject(context.Background(), "registry", "manifests/test-blob", object, objectStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		t.Fatal(err)
	}
	t.Fatal(info)
}

func TestRemoveObject(t *testing.T) {
	opts := minio.RemoveObjectOptions{
		GovernanceBypass: true,
	}

	err := testS3.Client.RemoveObject(context.Background(), "registry", "manifests/test-blob", opts)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStatObject(t *testing.T) {
	reader, err := testS3.Client.StatObject(context.Background(), "registry", "manifests/test-blob", minio.GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(reader)
}
