package models

import (
	"time"

	"github.com/google/uuid"
)

type EventSettingModel struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	EventID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	SeatLayoutType      string `gorm:"type:varchar(50);not null"`
	BookingLimitPerUser int    `gorm:"not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (EventSettingModel) TableName() string {
	return "event_settings"
}
