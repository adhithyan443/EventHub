package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type BookingModel struct {
	ID             uuid.UUID       `gorm:"type:uuid;primaryKey"`
	BookingNumber  string          `gorm:"type:varchar(30);uniqueIndex;not null"`
	UserID         uuid.UUID       `gorm:"type:uuid;not null;index"`
	EventID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	ReservationID  *uuid.UUID      `gorm:"type:uuid;index"`
	Subtotal       decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	PlatformFee    decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	TaxAmount      decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	DiscountAmount decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	TotalAmount    decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	Status         string          `gorm:"type:varchar(20);not null"`
	CreatedAt      time.Time       `gorm:"not null"`
	UpdatedAt      time.Time       `gorm:"not null"`

	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (BookingModel) TableName() string {
	return "bookings"
}
