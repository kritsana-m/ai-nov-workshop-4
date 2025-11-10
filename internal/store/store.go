package store

import (
	"kbtg-ai-workshop-nov/workshop-4/backend/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

// InitDB opens a SQLite database at the provided path and applies AutoMigrate for
// the known models. Returns the gorm.DB instance for use by the application.
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

// SetDB sets the package-level DB instance used by handlers.
func SetDB(d *gorm.DB) { db = d }

// GetDB returns the currently configured *gorm.DB instance (may be nil).
func GetDB() *gorm.DB { return db }
