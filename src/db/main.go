// Package db реализует логику работы с базой данных.
package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/PavelMilanov/container-registry/config"
	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY,
    tag_count INTEGER NOT NULL DEFAULT 0
);`

const ensureDefaultSettingsQuery = `
INSERT INTO settings (id, tag_count)
VALUES (1, ?)
ON CONFLICT (id) DO NOTHING;`

// SQLite управляет соединением с SQLite.
type SQLite struct {
	connection *sql.DB
}

/*
NewDatabase открывает SQLite и подготавливает схему приложения.

	ctx - контекст инициализации.
	path - путь к файлу SQLite.
*/
func NewDatabase(
	ctx context.Context,
	path string,
) (*SQLite, error) {
	connection, err := sql.Open(
		"sqlite3",
		path+"?_foreign_keys=on&_busy_timeout=5000",
	)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть SQLite: %w", err)
	}

	connection.SetMaxOpenConns(1)
	connection.SetMaxIdleConns(1)

	if err := connection.PingContext(ctx); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("не удалось подключиться к SQLite: %w", err)
	}

	database := &SQLite{connection: connection}
	if err := database.migrate(ctx); err != nil {
		_ = connection.Close()
		return nil, err
	}

	return database, nil
}

/*
Close закрывает соединение с SQLite.
*/
func (d *SQLite) Close() error {
	return d.connection.Close()
}

/*
migrate создаёт таблицы и начальную запись настроек.

	ctx - контекст выполнения миграции.
*/
func (d *SQLite) migrate(ctx context.Context) error {
	transaction, err := d.connection.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("не удалось начать миграцию SQLite: %w", err)
	}
	defer transaction.Rollback()

	if _, err := transaction.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("не удалось создать схему SQLite: %w", err)
	}
	if _, err := transaction.ExecContext(
		ctx,
		ensureDefaultSettingsQuery,
		config.DEFAULT_TAG_EXPIRED_DAYS,
	); err != nil {
		return fmt.Errorf("не удалось создать настройки SQLite: %w", err)
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("не удалось завершить миграцию SQLite: %w", err)
	}
	return nil
}
