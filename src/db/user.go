package db

import (
	"errors"
	"fmt"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/system"
	"gorm.io/gorm"
)

// User абстракция таблицы users.
type User struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null;unique"`
	Password string `gorm:"not null"`
	Token    string
}

func (u *User) Add(sql *gorm.DB) error {
	result := sql.Where("name = ?", u.Name).First(&u)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			hash := system.Hashed(u.Password)
			u.Password = hash
			sql.Create(&u)
			return nil
		} else {
			return result.Error
		}
	}
	errStr := fmt.Sprintf("Пользователь %s уже существует", u.Name)
	return errors.New(errStr)
}

func (u *User) Login(sql *gorm.DB, cred *config.Env) error {
	pwd := system.Hashed(u.Password)
	result := sql.Where("name = ? AND password = ?", u.Name, pwd).First(&u)
	if result.RowsAffected == 0 {
		return errors.New("неверные логин или пароль")
	}
	newToken, err := system.GenerateJWT(u.Name, cred)
	if err != nil {
		return err
	}
	u.Token = newToken
	sql.Save(&u)
	return nil
}
