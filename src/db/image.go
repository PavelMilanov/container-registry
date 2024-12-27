package db

import (
	"time"

	"gorm.io/gorm"
)

type Image struct {
	Name       string
	Hash       string
	Tag        string
	Size       string
	CreatedAt  string
	UpdatedAt  time.Time `gorm:"autoUpdateTime:false"`
	RegistryID int
}

func (i *Image) Add(sql *gorm.DB) error {
	now := time.Now()
	i.CreatedAt = now.Format("2006-01-02 15:04:05")
	result := sql.Create(&i)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func GetImages(sql *gorm.DB) []Registry {
	var r []Registry
	sql.Find(&r)
	return r
}
