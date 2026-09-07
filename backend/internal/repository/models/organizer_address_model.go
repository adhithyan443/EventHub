package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizerAddressModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrganizerID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	AddressLine string `gorm:"type:text;not null"`
	City        string `gorm:"type:varchar(100);not null"`
	State       string `gorm:"type:varchar(100);not null"`
	Country     string `gorm:"type:varchar(100);not null"`
	PostalCode  string `gorm:"type:varchar(20);not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (OrganizerAddressModel) TableName() string {
	return "organizer_addresses"
}
