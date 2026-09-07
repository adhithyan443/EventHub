package models

import (
	"time"

	"github.com/google/uuid"
)

type VenueModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	GooglePlaceID string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Name          string    `gorm:"type:varchar(255);not null"`
	Address       string    `gorm:"type:text;not null"`
	City          string    `gorm:"type:varchar(100);not null"`
	State         string    `gorm:"type:varchar(100);not null"`
	Country       string    `gorm:"type:varchar(100);not null"`
	PostalCode    string    `gorm:"type:varchar(20)"`
	Latitude      float64   `gorm:"not null"`
	Longitude     float64   `gorm:"not null"`
	CreatedAt     time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"not null;autoUpdateTime"`
}

func (VenueModel) TableName() string {
	return "venues"
}
