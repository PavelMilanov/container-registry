package db

import (
	"sync"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLite struct {
	Sql   *gorm.DB
	Mutex *sync.Mutex
}

func NewDatabase(sql string) SQLite {
	conn, err := gorm.Open(sqlite.Open(sql), &gorm.Config{PrepareStmt: true})
	if err != nil {
		logrus.Fatal("Ошибка при подключении к базе данных")
	}
	var mutex sync.Mutex
	db := SQLite{Sql: conn, Mutex: &mutex}

	logrus.Info("Соединение с базой данных установлено")
	automigrate(db.Sql)
	return db
}

func CloseDatabase(db *gorm.DB) {
	sqlDB, _ := db.DB()
	if err := sqlDB.Close(); err != nil {
		logrus.Fatal("Ошибка при закрытии соединения с базой данных:", err)
	}
}

func automigrate(db *gorm.DB) {
	if err := db.AutoMigrate(&Registry{}, &Image{}); err != nil {
		logrus.Fatalf("%s", err)
		return
	}
}
