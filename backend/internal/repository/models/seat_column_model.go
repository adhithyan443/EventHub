package models

import (
	"time"

	"github.com/google/uuid"
)

type SeatColumnModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	SeatLayoutID uuid.UUID `gorm:"type:uuid;not null;index"`
	Label        string    `gorm:"type:varchar(50);not null"`
	OrderIndex   int       `gorm:"not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (SeatColumnModel) TableName() string {
	return "seat_columns"
}
