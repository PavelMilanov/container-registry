package db

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Registry struct {
	gorm.Model
	ID           int    `gorm:"primaryKey"`
	Name         string `gorm:"unique"`
	Size         int
	CreatedAt    string
	Repositories []Repository `gorm:"constraint:OnDelete:CASCADE;"`
}

func (r *Registry) Add(sql *gorm.DB) {
	now := time.Now()
	r.CreatedAt = now.Format("2006-01-02 15:04:05")
	if sql.Model(&r).Where("name = ?", r.Name).Updates(&r).RowsAffected == 0 {
		sql.Create(&r)
		logrus.Infof("Создан новый реестр %v", r)
	}
}

func (r *Registry) Delete(sql *gorm.DB) {
	result := sql.Select("Repositories", "Images").Where("name = ?", r.Name).Delete(&r)
	if result.Error != nil {
		logrus.Error(result.Error)
	}
	logrus.Debug(r)
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
