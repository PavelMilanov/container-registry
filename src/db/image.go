package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Image абстракция таблицы images.
type Image struct {
	ID           int `gorm:"primaryKey"`
	Name         string
	Hash         string
	Tag          string
	Platform     string
	Size         int64
	SizeAlias    string
	CreatedAt    string
	RepositoryID int `gorm:"constraint:OnDelete:CASCADE;"`
}

func (i *Image) Add(sql *gorm.DB) {
	now := time.Now()
	i.CreatedAt = now.Format("2006-01-02 15:04:05")
	if sql.Model(&i).Where("name = ? AND tag = ?", i.Name, i.Tag).Updates(&i).RowsAffected == 0 {
		sql.Create(&i)
	}
}

func (i *Image) Delete(sql *gorm.DB) error {
	sql.Where("name = ? AND tag = ?", i.Name, i.Tag).First(&i)
	result := sql.Delete(&i)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func GetImageTags(sql *gorm.DB, id int, name string) []Image {
	var i []Image
	sql.Where("repository_id =? AND name =?", id, name).Find(&i)
	return i
}

func GetLastTagImages(sql *gorm.DB, count int) []Image {
	var images []Image
	sql.Raw(`WITH ranked AS (
  SELECT
  *,
    ROW_NUMBER() OVER (PARTITION BY name ORDER BY created_at DESC) AS rn
  FROM images
)
SELECT *
FROM ranked
WHERE rn <= ?`, count).Scan(&images)
	return images
}

func GetImage(sql *gorm.DB, condition string, args ...interface{}) (*Image, error) {
	var i Image
	if err := sql.Where(condition, args...).First(&i).Error; err != nil {
		return nil, err
	}
	return &i, nil
}

func GetImages(sql *gorm.DB, condition string, args ...interface{}) ([]Image, error) {
	var images []Image
	if err := sql.Where(condition, args...).Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

func (i *Image) GetSize(sql *gorm.DB, condition string, args ...interface{}) int64 {
	var size int64
	script := fmt.Sprintf("select SUM(size) from images WHERE %s", condition)
	sql.Raw(script, args...).Scan(&size)
	return size
}
