package db

import (
	"errors"

	"github.com/PavelMilanov/container-registry/secure"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type User struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	Password string `gorm:"not null"`
}

func (u *User) Add(sql *gorm.DB) error {
	result := sql.Where("name = ? AND password = ?", u.Name, u.Password).First(&u)
	if result.RowsAffected == 0 {
		hash := secure.Hashed(u.Password)
		u.Password = hash
		sql.Create(&u)
		logrus.Infof("Создан новый пользователь %v", u)
	} else {
		logrus.Errorf("пользователь %v уже существует", u)
		return errors.New("пользователь уже существует")
	}
	return nil
}

func (u *User) Check(sql *gorm.DB) error {
	return nil
}
