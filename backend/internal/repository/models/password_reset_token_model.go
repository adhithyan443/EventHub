package models

import (
	"time"

	"github.com/google/uuid"
)

type PasswordResetTokenModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"type:varchar(255);not null"`
	ExpiresAt time.Time `gorm:"not null"`
	UsedAt    *time.Time
	CreatedAt time.Time `gorm:"not null"`
}

func (PasswordResetTokenModel) TableName() string {
	return "password_reset_tokens"
}
