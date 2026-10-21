package models

import (
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type CheckoutModel struct {
	ID             uuid.UUID       `gorm:"type:uuid;primaryKey"`
	ReservationID  uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex"`
	UserID         uuid.UUID       `gorm:"type:uuid;not null;index"`
	Subtotal       decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	PlatformFee    decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	TaxAmount      decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	DiscountAmount decimal.Decimal `gorm:"type:numeric(10,2);not null;default:0"`
	TotalAmount    decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	Status         string          `gorm:"type:varchar(20);not null;index"`
	ExpiresAt      time.Time       `gorm:"not null;index"`
	CreatedAt      time.Time       `gorm:"not null"`
	UpdatedAt      time.Time       `gorm:"not null"`
}

func (CheckoutModel) TableName() string {
	return "checkouts"
}

func CheckoutModelFromDomain(checkout *domain.Checkout) *CheckoutModel {
	return &CheckoutModel{
		ID:             checkout.ID,
		ReservationID:  checkout.ReservationID,
		UserID:         checkout.UserID,
		Subtotal:       checkout.Subtotal,
		PlatformFee:    checkout.PlatformFee,
		TaxAmount:      checkout.TaxAmount,
		DiscountAmount: checkout.DiscountAmount,
		TotalAmount:    checkout.TotalAmount,
		Status:         checkout.Status,
		ExpiresAt:      checkout.ExpiresAt,
		CreatedAt:      checkout.CreatedAt,
		UpdatedAt:      checkout.UpdatedAt,
	}
}

func (m *CheckoutModel) ToDomain() *domain.Checkout {
	return &domain.Checkout{
		ID:             m.ID,
		ReservationID:  m.ReservationID,
		UserID:         m.UserID,
		Subtotal:       m.Subtotal,
		PlatformFee:    m.PlatformFee,
		TaxAmount:      m.TaxAmount,
		DiscountAmount: m.DiscountAmount,
		TotalAmount:    m.TotalAmount,
		Status:         m.Status,
		ExpiresAt:      m.ExpiresAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// BeforeCreate ensures GORM has a UUID if the caller hasn't assigned one.
func (m *CheckoutModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
