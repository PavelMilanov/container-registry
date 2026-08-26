package services

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PavelMilanov/container-registry/db"
	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
)

func newTestAuthService(
	t *testing.T,
) (*AuthService, *db.UserRepository) {
	t.Helper()
	database, err := db.NewDatabase(
		context.Background(),
		filepath.Join(t.TempDir(), "registry.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	users := db.NewUserRepository(database)

	tokens, err := registryauth.NewTokenManager(registryauth.TokenConfig{
		Secret:   []byte("test-token-secret-with-at-least-32-bytes"),
		Issuer:   "test-registry",
		Audience: "registry.example.com",
		TTL:      2 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return NewAuthService(
		users,
		registryauth.NewPasswordHasher(),
		tokens,
	), users
}

func TestAuthServiceRegisterAndLogin(t *testing.T) {
	service, users := newTestAuthService(t)
	ctx := context.Background()
	if err := service.Register(ctx, "pavel", "secure-password"); err != nil {
		t.Fatal(err)
	}

	user, err := users.FindByName(ctx, "pavel")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(user.Password, "$argon2id$") {
		t.Fatalf("password hash = %q", user.Password)
	}

	issued, err := service.Login(ctx, "pavel", "secure-password", nil)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.ValidateToken(issued.Value)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "pavel" {
		t.Fatalf("subject = %q", claims.Subject)
	}
}

func TestAuthServiceRejectsInvalidCredentials(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	if err := service.Register(ctx, "pavel", "secure-password"); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Login(ctx, "pavel", "wrong-password", nil); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v", err)
	}
	if _, err := service.Login(ctx, "missing", "secure-password", nil); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("missing user error = %v", err)
	}
}

func TestAuthServiceEnsureUserDoesNotOverwritePassword(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	if err := service.EnsureUser(ctx, "admin", "first-password"); err != nil {
		t.Fatal(err)
	}
	if err := service.EnsureUser(ctx, "admin", "second-password"); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Login(ctx, "admin", "first-password", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Login(ctx, "admin", "second-password", nil); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("overwritten password error = %v", err)
	}
}
