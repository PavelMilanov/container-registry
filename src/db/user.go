package db

import (
	"errors"

	"github.com/PavelMilanov/container-registry/secure"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type User struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null,unique"`
	Password string `gorm:"not null"`
	Token    string
}

func (u *User) Add(sql *gorm.DB) error {
	result := sql.Where("name = ? AND password = ?", u.Name, u.Password).First(&u)
	if result.RowsAffected == 0 {
		hash := secure.Hashed(u.Password)
		u.Password = hash
		token, err := secure.GenerateJWT()
		if err != nil {
			logrus.Error(err)
			return err
		}
		u.Token = token
		sql.Create(&u)
		logrus.Infof("Создан новый пользователь %+v", u)
	} else {
		logrus.Errorf("пользователь %+v уже существует", u)
		return errors.New("пользователь уже существует")
	}
	return nil
}

func (u *User) Login(sql *gorm.DB) error {
	pwd := secure.Hashed(u.Password)
	result := sql.Where("name = ? AND password = ?", u.Name, pwd).First(&u)
	if result.RowsAffected == 0 {
		logrus.Error("неверные логин или пароль")
		return errors.New("неверные логин или пароль")
	}
	return nil
}
