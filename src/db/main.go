// Package db реализует логику работы с базой данных.
package db

import (
	"sync"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/system"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SQlite абстракция над *gorm.DB.
type SQLite struct {
	Sql   *gorm.DB
	Mutex *sync.Mutex
}

func NewDatabase(sql string, env *config.Env) (SQLite, error) {
	conn, err := gorm.Open(sqlite.Open(sql), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return SQLite{}, err
	}
	var mutex sync.Mutex
	db := SQLite{Sql: conn, Mutex: &mutex}
	if err := automigrate(db.Sql); err != nil {
		return db, err
	}
	if err := setDefaultSettings(db.Sql, env); err != nil {
		return db, err
	}
	return db, nil
}

func CloseDatabase(db *gorm.DB) {
	sqlDB, _ := db.DB()
	if err := sqlDB.Close(); err != nil {
		logrus.Fatal(err)
	}
}

func setDefaultSettings(db *gorm.DB, env *config.Env) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		var settings Settings
		if err := tx.FirstOrCreate(&settings, Settings{TagCount: config.DEFAULT_TAG_EXPIRED_DAYS}).Error; err != nil {
			return err
		}
		var newUser User
		hash := system.Hashed(env.User.Password)
		if err := tx.FirstOrCreate(&newUser, User{Name: env.User.Login, Password: hash}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func automigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Registry{}, &Repository{}, &Image{}, &User{}, &Settings{}); err != nil {
		return err
	}
	return nil
}
