package models

import (
	"time"

	"github.com/google/uuid"
)

type EventContactModel struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	EventID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	Name  string `gorm:"type:varchar(100);not null"`
	Phone string `gorm:"type:varchar(20);not null"`
	Email string `gorm:"type:varchar(255);not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (EventContactModel) TableName() string {
	return "event_contacts"
}
