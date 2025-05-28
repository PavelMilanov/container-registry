package db

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Repository абстракция таблицы repositories.
type Repository struct {
	ID         int    `gorm:"primaryKey"`
	Name       string `gorm:"unique"`
	CreatedAt  string
	Size       int64
	SizeAlias  string
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

func (r *Repository) UpdateSize(sql *gorm.DB) error {
	result := sql.Raw("UPDATE repositories SET size = ?, size_alias = ? WHERE id = ?", r.Size, r.SizeAlias, r.ID).Scan(&r)
	if result.Error != nil {
		logrus.Error(result.Error)
		return result.Error
	}
	return nil
}

func (r *Repository) GetSize(sql *gorm.DB, condition string, args ...interface{}) int64 {
	var size int64
	script := fmt.Sprintf("select SUM(size) from repositories WHERE %s", condition)
	sql.Raw(script, args...).Scan(&size)
	return size
}

func GetRepository(sql *gorm.DB, condition string, args ...interface{}) (*Repository, error) {
	var r Repository
	if err := sql.Where(condition, args...).First(&r).Error; err != nil {
		logrus.Error(err)
		return nil, err
	}
	return &r, nil
}
