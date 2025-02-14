package db

import (
	"errors"

	"github.com/PavelMilanov/container-registry/system"
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
		hash := system.Hashed(u.Password)
		u.Password = hash
		sql.Create(&u)
		logrus.Infof("Создан новый пользователь %+v", u)
	} else {
		logrus.Errorf("пользователь %+v уже существует", u)
		return errors.New("пользователь уже существует")
	}
	return nil
}

func (u *User) Login(sql *gorm.DB, jwtKey []byte) error {
	pwd := system.Hashed(u.Password)
	result := sql.Where("name = ? AND password = ?", u.Name, pwd).First(&u)
	if result.RowsAffected == 0 {
		logrus.Error("неверные логин или пароль")
		return errors.New("неверные логин или пароль")
	}
	newToken, err := system.GenerateJWT(jwtKey)
	if err != nil {
		logrus.Error(err)
		return err
	}
	u.Token = newToken
	sql.Save(&u)
	return nil
}
