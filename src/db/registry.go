package db

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Registry struct {
	gorm.Model
	ID        int    `gorm:"primaryKey"`
	Name      string `gorm:"unique"`
	Size      int
	CreatedAt string
	UpdatedAt time.Time `gorm:"autoUpdateTime:false"`
	Images    []Image   `gorm:"constraint:OnDelete:CASCADE;"`
}

func (r *Registry) Add(sql *gorm.DB) error {
	now := time.Now()
	r.CreatedAt = now.Format("2006-01-02 15:04:05")
	result := sql.Create(&r)
	if result.Error != nil {
		logrus.Error(result.Error)
		return result.Error
	}
	logrus.Infof("Создан новый реестр %v", r)
	return nil
}

func GetRegistires(sql *gorm.DB) []Registry {
	var r []Registry
	sql.Find(&r)
	return r
}

func (r *Registry) Get(name string, sql *gorm.DB) error {
	result := sql.Where("name = ?", name).First(&r)
	if result.RowsAffected == 0 {
		return result.Error
	}
	return nil
}

func (r *Registry) GetImages(sql *gorm.DB) error {
	result := sql.Preload("Images").Where("name = ?", r.Name).First(&r)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
