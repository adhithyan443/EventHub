package database

import (
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(

		&models.UserModel{},
		&models.RefreshTokenModel{},
		&models.PasswordResetTokenModel{},
	)
}
