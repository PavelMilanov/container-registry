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
	// В таблице настроек хранится одна активная запись.
	// Обновляем её явно, чтобы избежать массового UPDATE без WHERE.
	return sql.Model(&Settings{}).Where("id = ?", 1).Update("tag_count", count).Error
}
