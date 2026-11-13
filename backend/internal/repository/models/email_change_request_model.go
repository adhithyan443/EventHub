package models

import (
	"time"

	"github.com/google/uuid"
)

type EmailChangeRequestModel struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_email_change_requests_user_id"`
	NewEmail          string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_email_change_requests_new_email"`
	OTPHash           string    `gorm:"type:varchar(64);not null"`
	OTPExpiresAt      time.Time `gorm:"not null"`
	OTPAttempts       int64     `gorm:"default:0;not null"`
	ResendAvailableAt time.Time `gorm:"not null"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
}

func (EmailChangeRequestModel) TableName() string {
	return "email_change_requests"
}
