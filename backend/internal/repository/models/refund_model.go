package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type RefundModel struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey"`
	BookingID       uuid.UUID       `gorm:"type:uuid;not null;index"`
	PaymentID       uuid.UUID       `gorm:"type:uuid;not null;index"`
	GatewayRefundID *string         `gorm:"type:varchar(100);uniqueIndex"`
	Amount          decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	Status          string          `gorm:"type:varchar(30);not null;default:'PENDING'"`
	Reason          string          `gorm:"type:varchar(255);not null;default:'CUSTOMER_CANCELLATION'"`
	CreatedAt       time.Time       `gorm:"not null"`
	UpdatedAt       time.Time       `gorm:"not null"`
	ProcessedAt     *time.Time      `gorm:"type:timestamptz"`
}

func (RefundModel) TableName() string {
	return "refunds"
}
