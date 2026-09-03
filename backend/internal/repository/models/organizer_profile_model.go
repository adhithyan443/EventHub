package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizerProfileModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrganizerID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	BusinessName string `gorm:"type:varchar(255);not null"`
	BusinessType string `gorm:"type:varchar(100);not null"`
	Description  string `gorm:"type:text"`
	Phone        string `gorm:"type:varchar(20);not null"`
	Email        string `gorm:"type:varchar(255);not null"`
	Website      string `gorm:"type:varchar(255)"`
	LogoURL      string `gorm:"type:text"`
	GSTNumber    string `gorm:"type:varchar(50)"`
	PANNumber    string `gorm:"type:varchar(50);not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}
