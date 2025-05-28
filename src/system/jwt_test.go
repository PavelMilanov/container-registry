package system

import (
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

func initConfig() *config.Env {
	env, _ := config.NewEnv("../", "test.config")
	return env
}

func TestGenerateJWT(t *testing.T) {
	env := initConfig()
	token, err := GenerateJWT("test", env)
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(token)
}

func TestValidateJWT(t *testing.T) {
	env := initConfig()
	token, _ := GenerateJWT("test", env)
	if !ValidateJWT(token, []byte(env.Server.Jwt)) {
		t.Error("токен не валиден")
	}
}
