package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
)

func TestNewAuthServiceCreatesDefaultUser(t *testing.T) {
	ctx := context.Background()
	database, err := db.NewDatabase(
		ctx,
		filepath.Join(t.TempDir(), "registry.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("failed to close SQLite: %v", err)
		}
	})

	env := new(config.Env)
	env.Server.Jwt = "test-token-secret-with-at-least-32-bytes"
	env.DefaultUser.Login = "admin"
	env.DefaultUser.Password = "secure-password"

	authService, err := newAuthService(ctx, database, env)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authService.Login(
		ctx,
		env.DefaultUser.Login,
		env.DefaultUser.Password,
		nil,
	); err != nil {
		t.Fatal(err)
	}
}

func TestNewAuthServiceRejectsInvalidJWTConfig(t *testing.T) {
	ctx := context.Background()
	database, err := db.NewDatabase(
		ctx,
		filepath.Join(t.TempDir(), "registry.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("failed to close SQLite: %v", err)
		}
	})

	env := new(config.Env)
	env.Server.Jwt = "short"
	env.DefaultUser.Login = "admin"
	env.DefaultUser.Password = "secure-password"

	if _, err := newAuthService(ctx, database, env); err == nil {
		t.Fatal("expected invalid JWT configuration error")
	}
}
