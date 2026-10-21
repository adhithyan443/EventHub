package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SeatSectionModel struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey"`
	SeatLayoutID uuid.UUID       `gorm:"type:uuid;not null;index"`
	Name         string          `gorm:"type:varchar(100);not null"`
	Price        decimal.Decimal `gorm:"type:numeric(10,2);not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (SeatSectionModel) TableName() string {
	return "seat_sections"
}
