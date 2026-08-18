package database

import (
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
	)
}
