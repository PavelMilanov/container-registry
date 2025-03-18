package db

type Settings struct {
	ID       int `gorm:"primaryKey"`
	TagCount int
}
