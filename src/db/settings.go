package db

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Settings struct {
	ID       int `gorm:"primaryKey"`
	TagCount int
}

func GetCountTag(sql *gorm.DB) (int, error) {
	var settings Settings
	if err := sql.First(&settings).Error; err != nil {
		logrus.Error(err)
		return 0, err
	}
	return settings.TagCount, nil
}

func SetCountTag(sql *gorm.DB, count int) error {
	var settings Settings
	result := sql.Raw("UPDATE settings SET tag_count = ?", count).Scan(&settings)
	if result.Error != nil {
		logrus.Error(result.Error)
		return result.Error
	}
	return nil
}
