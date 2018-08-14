package database

import (
	"github.com/jinzhu/gorm"
)

type Post struct {
	gorm.Model
	Telegram int    `gorm:"unique"`
	Discord  string `gorm:"unique"`
}

func (Post) TableName() string {
	return "posts"
}

type PostManager struct {
	Data *Post
	DB   *gorm.DB
}

func (pm *PostManager) Create() error {
	return pm.DB.Create(&pm.Data).Error
}

func (pm *PostManager) FindByTelegramPost() error {
	return pm.DB.Model(&Post{}).Where("telegram = ?", pm.Data.Telegram).First(&pm.Data).Error
}
