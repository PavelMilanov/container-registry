package system

import (
	"testing"

	"github.com/PavelMilanov/container-registry/config"
)

var env = config.NewEnv(config.CONFIG_PATH, "config")

func TestGenerateJWT(t *testing.T) {
	token, err := GenerateJWT("test", env)
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(token)
}

func TestValidateJWT(t *testing.T) {
	token, _ := GenerateJWT("test", env)
	if !ValidateJWT(token, []byte(env.Server.Jwt)) {
		t.Error("токен не валиден")
	}
}
