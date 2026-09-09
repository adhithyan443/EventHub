package models

import (
	"time"

	"github.com/google/uuid"
)

type SeatSectionModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	SeatLayoutID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name         string    `gorm:"type:varchar(100);not null"`
	Price        float64   `gorm:"not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (SeatSectionModel) TableName() string {
	return "seat_sections"
}
