package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type BookingModel struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey"`
	BookingNumber string          `gorm:"type:varchar(50);uniqueIndex;not null"`
	UserID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	EventID       uuid.UUID       `gorm:"type:uuid;not null;index"`
	ReservationID *uuid.UUID      `gorm:"type:uuid;index"`
	Subtotal      decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	PlatformFee   decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	Tax           decimal.Decimal `gorm:"type:numeric(10,2);not null;column:tax"`
	Discount      decimal.Decimal `gorm:"type:numeric(10,2);not null;column:discount"`
	Total         decimal.Decimal `gorm:"type:numeric(10,2);not null;column:total"`
	Status        string          `gorm:"type:varchar(30);not null"`
	CreatedAt     time.Time       `gorm:"not null"`
	UpdatedAt     time.Time       `gorm:"not null"`

	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (BookingModel) TableName() string {
	return "bookings"
}
