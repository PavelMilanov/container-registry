package storage

import (
	"os"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/minio/minio-go"
	"github.com/sirupsen/logrus"
)

type S3 struct {
	Client *minio.Client
}

func newS3(endpoint string, accessKey string, privateKey string) *S3 {
	s3Client, err := minio.New(endpoint, accessKey, privateKey, false)
	if err != nil {
		logrus.Fatal(err)
	}
	err = s3Client.MakeBucket(config.BACKET_NAME, os.Getenv("TZ"))
	if err != nil {
		exists, errBucketExists := s3Client.BucketExists(config.BACKET_NAME)
		if errBucketExists == nil && exists {
			logrus.Debugf("Подключен bucket %s", config.BACKET_NAME)
		} else {
			logrus.Error(err)
		}
	} else {
		logrus.Infof("Успешно инициализирован bucket %s", config.BACKET_NAME)
	}
	return &S3{
		Client: s3Client,
	}
}
