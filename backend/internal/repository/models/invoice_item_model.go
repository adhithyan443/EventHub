package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type InvoiceItemModel struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey"`
	InvoiceID     uuid.UUID       `gorm:"type:uuid;not null;index"`
	BookingItemID *uuid.UUID      `gorm:"type:uuid;index"`
	CategoryName  string          `gorm:"type:varchar(100);not null"`
	SeatLabel     *string         `gorm:"type:varchar(100)"`
	Quantity      int             `gorm:"type:integer;not null"`
	UnitPrice     decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	TotalPrice    decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	CreatedAt     time.Time       `gorm:"not null"`
}

func (InvoiceItemModel) TableName() string {
	return "invoice_items"
}
