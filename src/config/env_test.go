package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewEnv(t *testing.T) {
	configDir := t.TempDir()
	configBody := []byte(`
server:
  realm: http://127.0.0.1:5050
  jwt: test-secret-with-at-least-32-bytes
  token_ttl: 3h

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
	if env.Server.TokenTTL != 3*time.Hour {
		t.Fatalf("unexpected token TTL: %s", env.Server.TokenTTL)
	}
	if env.User.Login != "admin" {
		t.Fatalf("unexpected user login: %s", env.User.Login)
	}
}

func TestNewEnvUsesDefaultTokenTTL(t *testing.T) {
	configDir := t.TempDir()
	configBody := []byte(`
server:
  realm: http://127.0.0.1:5050
  jwt: test-secret-with-at-least-32-bytes

storage:
  type: local

user:
  login: admin
  password: admin
`)
	if err := os.WriteFile(
		filepath.Join(configDir, "config.yaml"),
		configBody,
		0600,
	); err != nil {
		t.Fatal(err)
	}

	env, err := NewEnv(configDir, "config")
	if err != nil {
		t.Fatal(err)
	}
	if env.Server.TokenTTL != DefaultTokenTTL {
		t.Fatalf(
			"token TTL = %s, want %s",
			env.Server.TokenTTL,
			DefaultTokenTTL,
		)
	}
}
