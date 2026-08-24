package models

import (
	"time"

	"github.com/google/uuid"
)

type PendingRegistrationModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	FullName     string    `gorm:"type:varchar(100);not null"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Phone        string    `gorm:"type:varchar(20);not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	OTPHash      string    `gorm:"type:varchar(64);not null"`
	OTPExpiresAt time.Time `gorm:"not null"`
	OTPAttempts  int       `gorm:"not null;default:0"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

func (PendingRegistrationModel) TableName() string {
	return "pending_registrations"
}
