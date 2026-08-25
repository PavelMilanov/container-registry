package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

// User абстракция таблицы users.
type User struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null;unique"`
	Password string `gorm:"not null"`
}

// UserRepository предоставляет операции с пользователями в базе данных.
type UserRepository struct {
	sql *gorm.DB
}

/*
NewUserRepository создаёт repository пользователей.

	sql - подключение к базе данных.
*/
func NewUserRepository(sql *gorm.DB) *UserRepository {
	return &UserRepository{sql: sql}
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
	var count int64
	if err := r.sql.WithContext(ctx).
		Model(&User{}).
		Where("name = ?", user.Name).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: %s", ErrUserExists, user.Name)
	}

	if err := r.sql.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("не удалось создать пользователя %s: %w", user.Name, err)
	}
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
	err := r.sql.WithContext(ctx).
		Where("name = ?", name).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
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
	result := r.sql.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", userID).
		Update("password", passwordHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}
