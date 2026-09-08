package models

import (
	"time"

	"github.com/google/uuid"
)

type EventModel struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	OrganizerID uuid.UUID `gorm:"type:uuid;not null;index"`
	CategoryID  uuid.UUID `gorm:"type:uuid;not null;index"`
	VenueID     uuid.UUID `gorm:"type:uuid;not null;index"`

	Title          string `gorm:"type:varchar(255);not null"`
	Description    string `gorm:"type:text;not null"`
	BannerURL      string `gorm:"type:varchar(500)"`
	Language       string `gorm:"type:varchar(100)"`
	AgeRestriction int    `gorm:"not null;default:0"`

	Status string `gorm:"type:varchar(30);not null;index"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (EventModel) TableName() string {
	return "events"
}
