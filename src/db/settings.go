package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const getTagCountQuery = `
SELECT tag_count
FROM settings
WHERE id = 1;`

const setTagCountQuery = `
UPDATE settings
SET tag_count = ?
WHERE id = 1;`

var ErrSettingsNotFound = errors.New("settings not found")

// SettingsRepository предоставляет операции с настройками в SQLite.
type SettingsRepository struct {
	connection *sql.DB
}

/*
NewSettingsRepository создаёт repository настроек.

	database - подключение к SQLite.
*/
func NewSettingsRepository(database *SQLite) *SettingsRepository {
	return &SettingsRepository{connection: database.connection}
}

/*
GetTagCount возвращает количество сохраняемых тегов.

	ctx - контекст выполнения операции.
*/
func (r *SettingsRepository) GetTagCount(
	ctx context.Context,
) (int, error) {
	var count int
	if err := r.connection.QueryRowContext(
		ctx,
		getTagCountQuery,
	).Scan(&count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrSettingsNotFound
		}
		return 0, fmt.Errorf("не удалось получить настройки: %w", err)
	}
	return count, nil
}

/*
SetTagCount обновляет количество сохраняемых тегов.

	ctx - контекст выполнения операции.
	count - новое количество сохраняемых тегов.
*/
func (r *SettingsRepository) SetTagCount(
	ctx context.Context,
	count int,
) error {
	result, err := r.connection.ExecContext(ctx, setTagCountQuery, count)
	if err != nil {
		return fmt.Errorf("не удалось обновить настройки: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("не удалось проверить обновление настроек: %w", err)
	}
	if updated == 0 {
		return ErrSettingsNotFound
	}
	return nil
}
