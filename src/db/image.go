package db

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Image абстракция таблицы images.
type Image struct {
	ID           int `gorm:"primaryKey"`
	Name         string
	Hash         string
	Tag          string
	Size         int
	SizeAlias    string
	CreatedAt    string
	RepositoryID int `gorm:"constraint:OnDelete:CASCADE;"`
}

func (i *Image) Add(sql *gorm.DB) {
	now := time.Now()
	i.CreatedAt = now.Format("2006-01-02 15:04:05")
	if sql.Model(&i).Where("name = ? AND tag = ?", i.Name, i.Tag).Updates(&i).RowsAffected == 0 {
		sql.Create(&i)
		logrus.Infof("Добавлен новый образ %+v", i)
	}
}

func (i *Image) Delete(sql *gorm.DB) error {
	sql.Where("name = ? AND tag = ?", i.Name, i.Tag).First(&i)
	result := sql.Delete(&i)
	if result.Error != nil {
		logrus.Error(result.Error)
		return result.Error
	}
	logrus.Infof("Удален образ %+v", i)
	return nil
}

func GetImageTags(sql *gorm.DB, id int, name string) []Image {
	var i []Image
	sql.Where("repository_id =? AND name =?", id, name).Find(&i)
	return i
}
