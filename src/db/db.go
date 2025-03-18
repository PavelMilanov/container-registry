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

func NewDatabase(sql string) SQLite {
	conn, err := gorm.Open(sqlite.Open(sql), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Silent)})
	if err != nil {
		logrus.Fatal(err.Error())
	}
	var mutex sync.Mutex
	db := SQLite{Sql: conn, Mutex: &mutex}
	automigrate(db.Sql)
	setDefaultSettings(db.Sql)
	return db
}

func CloseDatabase(db *gorm.DB) {
	sqlDB, _ := db.DB()
	if err := sqlDB.Close(); err != nil {
		logrus.Fatal("Ошибка при закрытии соединения с базой данных:", err)
	}
}

func setDefaultSettings(db *gorm.DB) {
	var settings Settings
	if err := db.FirstOrCreate(&settings, Settings{TagCount: config.DEFAULT_TAG_EXPIRED_DAYS}).Error; err != nil {
		logrus.Fatal(err)
		return
	}
}

func automigrate(db *gorm.DB) {
	if err := db.AutoMigrate(&Registry{}, &Repository{}, &Image{}, &User{}, &Settings{}); err != nil {
		logrus.Fatalf("%s", err)
		return
	}
}
