package db

import (
	"time"

	"gorm.io/gorm"
)

type Image struct {
	Name       string
	Hash       string
	Tag        string
	Size       int
	CreatedAt  string
	UpdatedAt  time.Time `gorm:"autoUpdateTime:false"`
	RegistryID int
}

func (i *Image) Add(sql *gorm.DB) {
	now := time.Now()
	i.CreatedAt = now.Format("2006-01-02 15:04:05")
	//result := sql.Create(&i)
	if sql.Model(&i).Where("name = ? AND tag = ?", i.Name, i.Tag).Updates(&i).RowsAffected == 0 {
		sql.Create(&i)
	}
	// if result.Error != nil {
	// 	return result.Error
	// }
}

func GetRepositoryImages(sql *gorm.DB, id int, name string) []Image {
	var r []Image
	sql.Where("registry_id =? AND name =?", id, name).Find(&r)
	return r
}

func GetImages(sql *gorm.DB) []Registry {
	var r []Registry
	sql.Find(&r)
	return r
}
