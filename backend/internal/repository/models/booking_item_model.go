package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BookingItemModel struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey"`
	BookingID    uuid.UUID       `gorm:"type:uuid;not null;index"`
	SeatID       *uuid.UUID      `gorm:"type:uuid;index"`
	TicketTypeID *uuid.UUID      `gorm:"type:uuid;index"`
	Quantity     int             `gorm:"not null"`
	UnitPrice    decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	TotalPrice   decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	CreatedAt    time.Time       `gorm:"not null"`
	UpdatedAt    time.Time       `gorm:"not null"`
}

func (BookingItemModel) TableName() string {
	return "booking_items"
}
