package config

import (
	"errors"

	"github.com/spf13/viper"
)

/*
Env описывает конфигурацию приложения.
*/
type Env struct {
	Server  server
	Storage storage
}

/*
server описывает конфигурацию сервера.
*/
type server struct {
	Realm   string `mapstructure:"realm"`
	Service string `mapstructure:"service"`
	Issuer  string `mapstructure:"issuer"`
	Jwt     string `mapstructure:"jwt"`
}

/*
storage описывает конфигурацию хранилища.
*/
type storage struct {
	Type        string      `mapstructure:"type"`
	Credentials credentials `mapstructure:"credentials,omitzero"`
}

/*
credentials описывает параметры подключения к S3.
*/
type credentials struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	SSL       bool   `mapstructure:"ssl"`
}

/*
NewEnv инициализирует переменные из файла конфигурации.

	path - путь к файлу конфигурации. (относительный)
	file - название файла. (без расширения)

	Пример: NewEnv("var/conf.d", "config")
*/
func NewEnv(path, file string) (*Env, error) {
	var env Env
	viper.SetConfigName(file) // имя файла без расширения
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	err := viper.ReadInConfig()
	if err != nil {
		return &env, err

	}

	err = viper.Unmarshal(&env)
	if err != nil {
		return &env, err
	}
	switch env.Storage.Type {
	case "s3":
		if env.Storage.Credentials.Endpoint == "" || env.Storage.Credentials.AccessKey == "" || env.Storage.Credentials.SecretKey == "" {
			return &env, errors.New("не указан конфиг для подключения к S3 storage")
		}
	}
	return &env, nil
}
