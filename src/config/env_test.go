package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewEnv(t *testing.T) {
	configDir := t.TempDir()
	configBody := []byte(`
server:
  realm: http://127.0.0.1:5050
  service: 127.0.0.1:5050
  issuer: test-registry
  jwt: test-secret

storage:
  type: local

user:
  login: admin
  password: admin
`)
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), configBody, 0600); err != nil {
		t.Fatal(err)
	}

	env, err := NewEnv(configDir, "config")
	if err != nil {
		t.Fatal(err)
	}
	if env.Server.Realm != "http://127.0.0.1:5050" {
		t.Fatalf("unexpected realm: %s", env.Server.Realm)
	}
	if env.Storage.Type != "local" {
		t.Fatalf("unexpected storage type: %s", env.Storage.Type)
	}
	if env.User.Login != "admin" {
		t.Fatalf("unexpected user login: %s", env.User.Login)
	}
}
