package models

import (
	"time"

	"github.com/google/uuid"
)

type SeatLayoutModel struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	EventID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	LayoutName string    `gorm:"type:varchar(255);not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (SeatLayoutModel) TableName() string {
	return "seat_layouts"
}
