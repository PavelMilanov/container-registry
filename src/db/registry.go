package db

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Registry абстракция таблицы registies.
type Registry struct {
	ID           int    `gorm:"primaryKey"`
	Name         string `gorm:"unique"`
	Size         int64
	SizeAlias    string
	CreatedAt    string
	Repositories []Repository `gorm:"foreignKey:RegistryID;references:ID;constraint:OnDelete:CASCADE;"`
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
	}
	return nil
}

func (r *Registry) Delete(sql *gorm.DB) error {
	res := sql.Where("name = ?", r.Name).Delete(&Registry{})
	if res.Error != nil {
		logrus.Error(res.Error)
		return res.Error
	}
	return nil
}

func GetRegistires(sql *gorm.DB) ([]Registry, error) {
	var r []Registry
	res := sql.Find(&r)
	if res.Error != nil {
		logrus.Error(res.Error)
		return nil, res.Error
	}
	return r, nil
}

func (r *Registry) GetRepositories(sql *gorm.DB, name string) error {
	res := sql.Preload("Repositories").Where("name = ?", name).First(&r)
	if res.Error != nil {
		logrus.Error(res.Error)
		return res.Error
	}
	return nil
}

func (r *Registry) GetImages(sql *gorm.DB) error {
	res := sql.Preload("Images").Where("name = ?", r.Name).First(&r)
	if res.Error != nil {
		logrus.Error(res.Error)
		return res.Error
	}
	return nil
}

func (r *Registry) UpdateSize(sql *gorm.DB) error {
	res := sql.Raw("UPDATE registries SET size = ?, size_alias = ? WHERE id = ?", r.Size, r.SizeAlias, r.ID).Scan(&r)
	if res.Error != nil {
		logrus.Error(res.Error)
		return res.Error
	}
	return nil
}

func GetRegistry(sql *gorm.DB, condition string, args ...interface{}) (*Registry, error) {
	var r Registry
	if err := sql.Where(condition, args...).First(&r).Error; err != nil {
		logrus.Error(err)
		return nil, err
	}
	return &r, nil
}
