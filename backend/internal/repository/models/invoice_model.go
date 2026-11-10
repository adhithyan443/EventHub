package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type InvoiceModel struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey"`
	InvoiceNumber string          `gorm:"type:varchar(50);uniqueIndex;not null"`
	BookingID     uuid.UUID       `gorm:"type:uuid;uniqueIndex;not null"`
	PaymentID     uuid.UUID       `gorm:"type:uuid;not null;index"`
	UserID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	EventID       uuid.UUID       `gorm:"type:uuid;not null;index"`
	Status        string          `gorm:"type:varchar(30);not null"`
	Currency      string          `gorm:"type:varchar(10);not null;default:'INR'"`

	Subtotal       decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	PlatformFee    decimal.Decimal `gorm:"type:numeric(10,2);not null;default:0"`
	TaxAmount      decimal.Decimal `gorm:"type:numeric(10,2);not null;default:0"`
	DiscountAmount decimal.Decimal `gorm:"type:numeric(10,2);not null;default:0"`
	TotalAmount    decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	PaidAmount     decimal.Decimal `gorm:"type:numeric(10,2);not null"`

	CustomerName  string  `gorm:"type:varchar(100);not null"`
	CustomerEmail string  `gorm:"type:varchar(255);not null"`
	CustomerPhone *string `gorm:"type:varchar(20)"`

	EventTitle    string  `gorm:"type:varchar(255);not null"`
	EventDate     string  `gorm:"type:date;not null"`
	EventTimeStr  *string `gorm:"type:varchar(50)"`
	VenueName     *string `gorm:"type:varchar(255)"`
	VenueAddress  *string `gorm:"type:text"`
	IsOnline      bool    `gorm:"type:boolean;not null;default:false"`
	OnlineDetails *string `gorm:"type:text"`

	IssuerName    string  `gorm:"type:varchar(255);not null"`
	OrganizerName *string `gorm:"type:varchar(255)"`

	IssuedAt  time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`

	Items []InvoiceItemModel `gorm:"foreignKey:InvoiceID"`
}

func (InvoiceModel) TableName() string {
	return "invoices"
}
