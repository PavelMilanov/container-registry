package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Env struct {
	Server  server
	Storage storage
}

type server struct {
	Debug bool   `mapstructure:"debug"`
	Url   string `mapstructure:"url"`
	Jwt   string `mapstructure:"jwt"`
}

type storage struct {
	Type string `mapstructure:"type"`
}

func NewEnv() *Env {
	env := Env{}
	viper.SetConfigName("config") // имя файла без расширения
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Fatal("не найден файл конфигурации : ", err)
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		logrus.Fatal("не загружен файл конфигурации: ", err)
	}

	return &env
}
