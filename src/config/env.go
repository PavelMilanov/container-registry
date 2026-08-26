package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

const (
	DefaultTokenIssuer  = "container-registry"
	DefaultTokenService = "container-registry"
	DefaultTokenTTL     = 2 * time.Hour
)

/*
Env описывает конфигурацию приложения.
*/
type Env struct {
	Server  server
	Storage storage
	User    user
}

/*
server описывает конфигурацию сервера.
*/
type server struct {
	Realm    string        `mapstructure:"realm"`
	Jwt      string        `mapstructure:"jwt"`
	TokenTTL time.Duration `mapstructure:"token_ttl"`
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
user описывает параметры для суперпользователя.
*/
type user struct {
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
	reader.SetDefault("server.token_ttl", DefaultTokenTTL)

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
	if env.User.Login == "" || env.User.Password == "" {
		return &env, errors.New("не указаны логин или пароль для суперпользователя")
	}
	return &env, nil
}
