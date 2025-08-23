package config

import "testing"

func TestNewEnv(t *testing.T) {
	env, err := NewEnv("../conf.d", "config")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v", env)
}
