package storage

import (
	"context"

	"github.com/PavelMilanov/container-registry/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

type S3 struct {
	Client *minio.Client
}

func newS3(endpoint string, accessKey string, privateKey string, ssl bool) *S3 {
	s3Client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, privateKey, ""),
		Secure: ssl,
	})
	if err != nil {
		logrus.Fatal(err)
	}
	_, errBucketExists := s3Client.BucketExists(context.Background(), config.BACKET_NAME)
	if errBucketExists != nil {
		logrus.Error(err)
	}
	// logrus.Debugf("Подключен bucket %s", config.BACKET_NAME)

	return &S3{
		Client: s3Client,
	}
}
