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
	// result := sql.Create(&i)
	if sql.Model(&r).Where("name = ?", r.Name).First(&r).RowsAffected == 0 {
		sql.Create(&r)
		logrus.Infof("Создан новый репозиторий %v", r)
	}
	// if result.Error != nil {
	// 	return result.Error
	// }
}

// func GetRepositoryImages(sql *gorm.DB, id int, name string) []Image {
// 	var r []Image
// 	sql.Where("registry_id =? AND name =?", id, name).Find(&r)
// 	return r
// }

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
