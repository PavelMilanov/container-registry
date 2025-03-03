package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Env описывает конфигурацию приложения.
type Env struct {
	Server  server
	Storage storage
}

// server описывает конфигурацию сервера.
type server struct {
	Url string `mapstructure:"url"`
	Jwt string `mapstructure:"jwt"`
}

// storage описывает конфигурацию хранилища.
type storage struct {
	Type        string      `mapstructure:"type"`
	Credentials credentials `mapstructure:"credentials,omitzero"`
}

type credentials struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	SSL       bool   `mapstructure:"ssl"`
}

func NewEnv(path string) *Env {
	env := Env{}
	viper.SetConfigName("config") // имя файла без расширения
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Fatal("не найден файл конфигурации : ", err)
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		logrus.Fatal("не загружен файл конфигурации: ", err)
	}
	switch env.Storage.Type {
	case "s3":
		if env.Storage.Credentials.Endpoint == "" || env.Storage.Credentials.AccessKey == "" || env.Storage.Credentials.SecretKey == "" {
			logrus.Fatal("не указан конфиг для подключения к S3 storage")
		}
	}
	return &env
}
