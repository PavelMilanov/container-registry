package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const createUserQuery = `
INSERT INTO users (name, password)
VALUES (?, ?)
ON CONFLICT (name) DO NOTHING
RETURNING id;`

const findUserByNameQuery = `
SELECT id, name, password
FROM users
WHERE name = ?
LIMIT 1;`

const updatePasswordQuery = `
UPDATE users
SET password = ?
WHERE id = ?;`

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

// User описывает пользователя приложения.
type User struct {
	ID       int
	Name     string
	Password string
}

// UserRepository предоставляет операции с пользователями в SQLite.
type UserRepository struct {
	connection *sql.DB
}

/*
NewUserRepository создаёт repository пользователей.

	database - подключение к SQLite.
*/
func NewUserRepository(database *SQLite) *UserRepository {
	return &UserRepository{connection: database.connection}
}

/*
Create создаёт пользователя с готовым хешем пароля.

	ctx - контекст выполнения операции.
	user - сохраняемый пользователь.
*/
func (r *UserRepository) Create(
	ctx context.Context,
	user *User,
) error {
	var userID int64
	err := r.connection.QueryRowContext(
		ctx,
		createUserQuery,
		user.Name,
		user.Password,
	).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s", ErrUserExists, user.Name)
	}
	if err != nil {
		return fmt.Errorf(
			"не удалось создать пользователя %s: %w",
			user.Name,
			err,
		)
	}

	user.ID = int(userID)
	return nil
}

/*
FindByName возвращает пользователя по имени.

	ctx - контекст выполнения операции.
	name - имя пользователя.
*/
func (r *UserRepository) FindByName(
	ctx context.Context,
	name string,
) (User, error) {
	var user User
	var userID int64
	err := r.connection.QueryRowContext(
		ctx,
		findUserByNameQuery,
		name,
	).Scan(
		&userID,
		&user.Name,
		&user.Password,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf(
			"не удалось получить пользователя %s: %w",
			name,
			err,
		)
	}

	user.ID = int(userID)
	return user, nil
}

/*
UpdatePassword обновляет хеш пароля пользователя.

	ctx - контекст выполнения операции.
	userID - идентификатор пользователя.
	passwordHash - новый хеш пароля.
*/
func (r *UserRepository) UpdatePassword(
	ctx context.Context,
	userID int,
	passwordHash string,
) error {
	result, err := r.connection.ExecContext(
		ctx,
		updatePasswordQuery,
		passwordHash,
		userID,
	)
	if err != nil {
		return fmt.Errorf("не удалось обновить пароль: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("не удалось проверить обновление пароля: %w", err)
	}
	if updated == 0 {
		return ErrUserNotFound
	}
	return nil
}
