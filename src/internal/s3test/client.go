// Package s3test создаёт изолированные buckets для интеграционных тестов.
package s3test

import (
	"context"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

/*
New создаёт тестовый bucket и регистрирует удаление только его содержимого.

Тесты включаются явно через CR_RUN_S3_TESTS=1; credentials берутся из окружения.
*/
func New(t *testing.T) (*minio.Client, string) {
	t.Helper()
	if os.Getenv("CR_RUN_S3_TESTS") != "1" {
		t.Skip("set CR_RUN_S3_TESTS=1 to run S3 integration tests")
	}
	endpoint := os.Getenv("CR_S3_ENDPOINT")
	accessKey := os.Getenv("CR_S3_ACCESS_KEY")
	secretKey := os.Getenv("CR_S3_SECRET_KEY")
	if endpoint == "" || accessKey == "" || secretKey == "" {
		t.Fatal("CR_S3_ENDPOINT, CR_S3_ACCESS_KEY and CR_S3_SECRET_KEY are required")
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: os.Getenv("CR_S3_SSL") == "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	bucket := "cr-test-" + uuid.NewV4().String()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		for object := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				t.Error(object.Err)
				continue
			}
			if err := client.RemoveObject(ctx, bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
				t.Error(err)
			}
		}
		core := minio.Core{Client: client}
		for upload := range client.ListIncompleteUploads(ctx, bucket, "", true) {
			if upload.Err != nil {
				t.Error(upload.Err)
				continue
			}
			if err := core.AbortMultipartUpload(ctx, bucket, upload.Key, upload.UploadID); err != nil {
				t.Error(err)
			}
		}
		if err := client.RemoveBucket(ctx, bucket); err != nil {
			t.Error(err)
		}
	})
	return client, bucket
}
