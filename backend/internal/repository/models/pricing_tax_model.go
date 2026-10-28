package models

import (
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PricingTaxModel struct {
	ID                       uuid.UUID       `gorm:"type:uuid;primaryKey"`
	Version                  int             `gorm:"type:integer;not null;uniqueIndex"`
	Status                   string          `gorm:"type:varchar(20);not null"`
	PlatformFeeEnabled       bool            `gorm:"type:boolean;not null;default:false"`
	PlatformFeePercent       decimal.Decimal `gorm:"type:numeric(5,2);not null;default:0.00"`
	TaxEnabled               bool            `gorm:"type:boolean;not null;default:false"`
	TaxPercent               decimal.Decimal `gorm:"type:numeric(5,2);not null;default:0.00"`
	TaxBasis                 string          `gorm:"type:varchar(32);not null;default:'UNVALIDATED'"`
	TaxRulesValidated        bool            `gorm:"type:boolean;not null;default:false"`
	TaxLegalSignoffReference *string         `gorm:"type:varchar(128)"`
	FreeTicketExempt         bool            `gorm:"type:boolean;not null;default:true"`
	EffectiveFrom            time.Time       `gorm:"type:timestamptz;not null"`
	EffectiveTo              *time.Time      `gorm:"type:timestamptz"`
	CancelledAt              *time.Time      `gorm:"type:timestamptz"`
	CancelledBy              *uuid.UUID      `gorm:"type:uuid"`
	CreatedBy                *uuid.UUID      `gorm:"type:uuid"`
	ChangeReason             string          `gorm:"type:varchar(255);not null"`
	CreatedAt                time.Time       `gorm:"type:timestamptz;not null"`
	UpdatedAt                time.Time       `gorm:"type:timestamptz;not null"`
}

func (PricingTaxModel) TableName() string {
	return "pricing_tax_configurations"
}

func PricingTaxModelFromDomain(d *domain.PricingTaxConfiguration) *PricingTaxModel {
	if d == nil {
		return nil
	}
	return &PricingTaxModel{
		ID:                       d.ID,
		Version:                  d.Version,
		Status:                   d.Status,
		PlatformFeeEnabled:       d.PlatformFeeEnabled,
		PlatformFeePercent:       d.PlatformFeePercent,
		TaxEnabled:               d.TaxEnabled,
		TaxPercent:               d.TaxPercent,
		TaxBasis:                 d.TaxBasis,
		TaxRulesValidated:        d.TaxRulesValidated,
		TaxLegalSignoffReference: d.TaxLegalSignoffReference,
		FreeTicketExempt:         d.FreeTicketExempt,
		EffectiveFrom:            d.EffectiveFrom,
		EffectiveTo:              d.EffectiveTo,
		CancelledAt:              d.CancelledAt,
		CancelledBy:              d.CancelledBy,
		CreatedBy:                d.CreatedBy,
		ChangeReason:             d.ChangeReason,
		CreatedAt:                d.CreatedAt,
		UpdatedAt:                d.UpdatedAt,
	}
}

func (m *PricingTaxModel) ToDomain() *domain.PricingTaxConfiguration {
	if m == nil {
		return nil
	}
	return &domain.PricingTaxConfiguration{
		ID:                       m.ID,
		Version:                  m.Version,
		Status:                   m.Status,
		PlatformFeeEnabled:       m.PlatformFeeEnabled,
		PlatformFeePercent:       m.PlatformFeePercent,
		TaxEnabled:               m.TaxEnabled,
		TaxPercent:               m.TaxPercent,
		TaxBasis:                 m.TaxBasis,
		TaxRulesValidated:        m.TaxRulesValidated,
		TaxLegalSignoffReference: m.TaxLegalSignoffReference,
		FreeTicketExempt:         m.FreeTicketExempt,
		EffectiveFrom:            m.EffectiveFrom,
		EffectiveTo:              m.EffectiveTo,
		CancelledAt:              m.CancelledAt,
		CancelledBy:              m.CancelledBy,
		CreatedBy:                m.CreatedBy,
		ChangeReason:             m.ChangeReason,
		CreatedAt:                m.CreatedAt,
		UpdatedAt:                m.UpdatedAt,
	}
}
