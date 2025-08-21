package db

import (
	"gorm.io/gorm"
)

type Settings struct {
	ID       int `gorm:"primaryKey"`
	TagCount int
}

func GetCountTag(sql *gorm.DB) (int, error) {
	var settings Settings
	if err := sql.First(&settings).Error; err != nil {
		return 0, err
	}
	return settings.TagCount, nil
}

func SetCountTag(sql *gorm.DB, count int) error {
	var settings Settings
	result := sql.Raw("UPDATE settings SET tag_count = ?", count).Scan(&settings)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
