package config

import (
	"testing"
)

func TestNewEnv(t *testing.T) {
	env := NewEnv("../var/")
	t.Logf("config: %+v", env)
}
