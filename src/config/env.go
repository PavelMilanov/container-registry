package config

import (
	"errors"

	"github.com/spf13/viper"
)

const (
	DefaultTokenIssuer  = "container-registry"
	DefaultTokenService = "container-registry"
)

/*
Env описывает конфигурацию приложения.
*/
type Env struct {
	Server      server
	Storage     storage
	DefaultUser defaultUser `mapstructure:"default_user"`
}

/*
server описывает конфигурацию сервера.
*/
type server struct {
	Realm string `mapstructure:"realm"`
	Jwt   string `mapstructure:"jwt"`
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
defaultUser описывает параметры пользователя, создаваемого при первом запуске.
*/
type defaultUser struct {
	Login    string `mapstructure:"login"`
	Password string `mapstructure:"password"`
}

/*
NewEnv инициализирует переменные из файла конфигурации.

	path - путь к файлу конфигурации. (относительный)
	file - название файла. (без расширения)

	Пример: NewEnv("var/conf.d", "config")
*/
func NewEnv(path, file string) (*Env, error) {
	var env Env
	reader := viper.New()
	reader.SetConfigName(file) // имя файла без расширения
	reader.SetConfigType("yaml")
	reader.AddConfigPath(path)
	err := reader.ReadInConfig()
	if err != nil {
		return &env, err

	}

	err = reader.Unmarshal(&env)
	if err != nil {
		return &env, err
	}
	switch env.Storage.Type {
	case "s3":
		if env.Storage.Credentials.Endpoint == "" || env.Storage.Credentials.AccessKey == "" || env.Storage.Credentials.SecretKey == "" {
			return &env, errors.New("не указан конфиг для подключения к S3 storage")
		}
	}
	if env.DefaultUser.Login == "" || env.DefaultUser.Password == "" {
		return &env, errors.New("не указаны login или password для default_user")
	}
	return &env, nil
}
