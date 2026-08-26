package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

/*
newTestDatabase создаёт изолированную SQLite для теста.
*/
func newTestDatabase(t *testing.T) *SQLite {
	t.Helper()

	database, err := NewDatabase(
		context.Background(),
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
	return database
}

func TestNewDatabaseCreatesDefaultSettings(t *testing.T) {
	database := newTestDatabase(t)
	settings := NewSettingsRepository(database)

	count, err := settings.GetTagCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("tag count = %d, want 0", count)
	}
}

func TestSettingsRepository(t *testing.T) {
	database := newTestDatabase(t)
	settings := NewSettingsRepository(database)
	ctx := context.Background()

	if err := settings.SetTagCount(ctx, 12); err != nil {
		t.Fatal(err)
	}
	count, err := settings.GetTagCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 12 {
		t.Fatalf("tag count = %d, want 12", count)
	}
}

func TestUserRepository(t *testing.T) {
	database := newTestDatabase(t)
	users := NewUserRepository(database)
	ctx := context.Background()

	user := User{Name: "pavel", Password: "first-hash"}
	if err := users.Create(ctx, &user); err != nil {
		t.Fatal(err)
	}
	if user.ID == 0 {
		t.Fatal("created user ID is empty")
	}

	duplicate := User{Name: user.Name, Password: "other-hash"}
	if err := users.Create(ctx, &duplicate); !errors.Is(err, ErrUserExists) {
		t.Fatalf("duplicate error = %v, want ErrUserExists", err)
	}

	if err := users.UpdatePassword(ctx, user.ID, "second-hash"); err != nil {
		t.Fatal(err)
	}
	stored, err := users.FindByName(ctx, user.Name)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Password != "second-hash" {
		t.Fatalf("password = %q, want second-hash", stored.Password)
	}

	if _, err := users.FindByName(ctx, "missing"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("missing user error = %v, want ErrUserNotFound", err)
	}
	if err := users.UpdatePassword(ctx, -1, "hash"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("missing update error = %v, want ErrUserNotFound", err)
	}
}
