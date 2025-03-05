package db

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Registry абстракция таблицы registies.
type Registry struct {
	gorm.Model
	ID           int    `gorm:"primaryKey"`
	Name         string `gorm:"unique"`
	Size         int
	CreatedAt    string
	Repositories []Repository `gorm:"constraint:OnDelete:CASCADE;"`
}

func (r *Registry) Add(sql *gorm.DB) error {
	now := time.Now()
	r.CreatedAt = now.Format("2006-01-02 15:04:05")
	if sql.Model(&r).Where("name = ?", r.Name).Updates(&r).RowsAffected == 0 {
		result := sql.Create(&r)
		if result.Error != nil {
			logrus.Error(result.Error)
			return result.Error
		}
		logrus.Infof("Создан новый реестр %+v", r)
	}
	return nil
}

func (r *Registry) Delete(sql *gorm.DB) error {
	result := sql.Raw("DELETE FROM registries WHERE name = ?", r.Name).Scan(&r)
	if result.Error != nil {
		logrus.Error(result.Error)
		return result.Error
	}
	logrus.Infof("Удален реестр %+v", r)
	return nil
}

func GetRegistires(sql *gorm.DB) []Registry {
	var r []Registry
	result := sql.Find(&r)
	if result.Error != nil {
		logrus.Error(result.Error)
	}
	return r
}

func (r *Registry) Get(sql *gorm.DB, name string) error {
	result := sql.Where("name = ?", name).First(&r)
	if result.RowsAffected == 0 {
		logrus.Error(result.Error)
		return result.Error
	}
	return nil
}

func (r *Registry) GetRepositories(sql *gorm.DB, name string) error {
	result := sql.Preload("Repositories").Where("name = ?", name).First(&r)
	if result.Error != nil {
		logrus.Error(result.Error)
		return result.Error
	}
	return nil
}

func (r *Registry) GetImages(sql *gorm.DB) error {
	result := sql.Preload("Images").Where("name = ?", r.Name).First(&r)
	if result.Error != nil {
		logrus.Error(result.Error)
		return result.Error
	}
	return nil
}
