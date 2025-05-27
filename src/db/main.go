// Package db реализует логику работы с базой данных.
package db

import (
	"sync"

	"github.com/PavelMilanov/container-registry/config"
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

func NewDatabase(sql string) (SQLite, error) {
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
	if err := setDefaultSettings(db.Sql); err != nil {
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

func setDefaultSettings(db *gorm.DB) error {
	var settings Settings
	if err := db.FirstOrCreate(&settings, Settings{TagCount: config.DEFAULT_TAG_EXPIRED_DAYS}).Error; err != nil {
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
