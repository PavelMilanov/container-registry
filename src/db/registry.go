package db

import (
	"time"

	"gorm.io/gorm"
)

type Registry struct {
	gorm.Model
	ID        int `gorm:"primaryKey"`
	Name      string
	Size      string
	CreatedAt string
	UpdatedAt time.Time `gorm:"autoUpdateTime:false"`
	Images    []Image   `gorm:"constraint:OnDelete:CASCADE;"`
}

func (r *Registry) Add(sql *gorm.DB) error {
	now := time.Now()
	r.CreatedAt = now.Format("2006-01-02 15:04:05")
	result := sql.Create(&r)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func GetRegistires(sql *gorm.DB) []Registry {
	var r []Registry
	sql.Find(&r)
	return r
}

func (r *Registry) Get(name string, sql *gorm.DB) *Registry {
	sql.Where("name = ?", name).First(&r)
	return r
}
