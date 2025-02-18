package config

import (
	"testing"
)

func TestNewEnv(t *testing.T) {
	env := Env{}
	t.Logf("config: %+v", env)
}
