package storage

import (
	"os"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/minio/minio-go"
	"github.com/sirupsen/logrus"
)

type S3storage struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

func NewS3storage(url string, accessKey string, secretKey string) *S3storage {
	minioClient, err := minio.New(url, accessKey, secretKey, false)
	if err != nil {
		logrus.Fatal(err)
	}
	err = minioClient.MakeBucket(config.BACKET_NAME, os.Getenv("TZ"))
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(config.BACKET_NAME)
		if errBucketExists == nil && exists {
			logrus.Debugf("Подключен bucket %s", config.BACKET_NAME)
		} else {
			logrus.Error(err)
		}
	} else {
		logrus.Infof("Успешно инициализирован bucket %s", config.BACKET_NAME)
	}
	return &S3storage{
		Endpoint:        url,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	}
}
