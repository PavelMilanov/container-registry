package db

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Repository struct {
	ID         int    `gorm:"primaryKey"`
	Name       string `gorm:"unique"`
	CreatedAt  string
	Images     []Image `gorm:"constraint:OnDelete:CASCADE;"`
	RegistryID int
}

func (r *Repository) Add(sql *gorm.DB) {
	now := time.Now()
	r.CreatedAt = now.Format("2006-01-02 15:04:05")
	if sql.Model(&r).Where("name = ?", r.Name).First(&r).RowsAffected == 0 {
		sql.Create(&r)
		logrus.Infof("Создан новый репозиторий %+v", r)
	}
}

func (r *Repository) Delete(sql *gorm.DB) error {
	sql.Preload("Images").Where("name = ?", r.Name).First(&r)
	result := sql.Delete(&r)
	if result.Error != nil {
		logrus.Error(result.Error)
		return result.Error
	}
	logrus.Infof("Удален репозиторий %+v", r)
	go func() {
		for _, image := range r.Images {
			sql.Delete(&image)
		}
	}()
	return nil
}

func GetRepositories(sql *gorm.DB) []Repository {
	var r []Repository
	sql.Find(&r)
	return r
}

func GetRepository(sql *gorm.DB, name string) Repository {
	var r Repository
	sql.Where("name =?", name).First(&r)
	return r
}
