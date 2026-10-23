package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type PaymentModel struct {
	ID              string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BookingID       string          `gorm:"type:uuid;not null;index"`
	TransactionID   *string         `gorm:"type:varchar(100);uniqueIndex"`
	Provider        string          `gorm:"type:varchar(30);not null"`
	ProviderOrderID *string         `gorm:"type:varchar(100);uniqueIndex"`
	PaymentMethod   string          `gorm:"type:varchar(30);not null"`
	Amount          decimal.Decimal `gorm:"type:decimal(10,2);not null"`
	Status          string          `gorm:"type:varchar(20);not null"`
	PaidAt          *time.Time      `gorm:"type:timestamp"`
	CreatedAt       time.Time       `gorm:"not null;default:now()"`
	UpdatedAt       time.Time       `gorm:"not null;default:now()"`
}

func (PaymentModel) TableName() string {
	return "payments"
}
