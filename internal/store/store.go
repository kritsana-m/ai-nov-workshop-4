package store

import (
	"kbtg-ai-workshop-nov/workshop-4/backend/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB(path string) (*gorm.DB, error) {
	d, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := d.AutoMigrate(&models.User{}, &models.Transfer{}, &models.PointLedger{}); err != nil {
		return nil, err
	}
	return d, nil
}

func SetDB(d *gorm.DB) { db = d }

func GetDB() *gorm.DB { return db }
