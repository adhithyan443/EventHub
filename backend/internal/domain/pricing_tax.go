package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	PricingTaxStatusActive     = "ACTIVE"
	PricingTaxStatusScheduled  = "SCHEDULED"
	PricingTaxStatusSuperseded = "SUPERSEDED"
	PricingTaxStatusCancelled  = "CANCELLED"
)

const (
	TaxBasisUnvalidated     = "UNVALIDATED"
	TaxBasisPlatformFeeOnly = "PLATFORM_FEE_ONLY"
	TaxBasisSubtotalOnly    = "SUBTOTAL_ONLY"
	TaxBasisTotalBeforeTax  = "TOTAL_BEFORE_TAX"
)

// PricingTaxAdvisoryLockID is a fixed PostgreSQL advisory lock key (CRC32 of "eventhub_pricing_tax_configuration")
// used to strictly serialize all configuration mutations across concurrent transactions.
const PricingTaxAdvisoryLockID int64 = 749201851

var (
	ErrPricingConfigurationUnavailable   = errors.New("pricing and tax configuration is temporarily unavailable")
	ErrConcurrentConfigurationModification = errors.New("pricing configuration was modified concurrently by another administrator")
	ErrInvalidPricingConfiguration       = errors.New("invalid pricing configuration parameters")
	ErrTaxBasisUnvalidated               = errors.New("tax cannot be enabled until tax basis and legal classification are validated with a legal sign-off reference")
	ErrInvalidEffectiveDate              = errors.New("scheduled configuration effective date must be in the future")
	ErrCannotCancelActiveConfiguration   = errors.New("cannot cancel a configuration that is currently active or in the past")
	ErrPricingTimelineDiscontinuous      = errors.New("pricing timeline integrity check failed: discontinuous intervals or gaps detected")
	ErrScheduledConfigurationNotFound    = errors.New("scheduled configuration not found")
)

// PricingTaxConfiguration represents an effective-dated snapshot of platform fee and tax rules.
type PricingTaxConfiguration struct {
	ID                       uuid.UUID       `json:"id"`
	Version                  int             `json:"version"`
	Status                   string          `json:"status"`
	PlatformFeeEnabled       bool            `json:"platform_fee_enabled"`
	PlatformFeePercent       decimal.Decimal `json:"platform_fee_percent"`
	TaxEnabled               bool            `json:"tax_enabled"`
	TaxPercent               decimal.Decimal `json:"tax_percent"`
	TaxBasis                 string          `json:"tax_basis"`
	TaxRulesValidated        bool            `json:"tax_rules_validated"`
	TaxLegalSignoffReference *string         `json:"tax_legal_signoff_reference,omitempty"`
	FreeTicketExempt         bool            `json:"free_ticket_exempt"`
	EffectiveFrom            time.Time       `json:"effective_from"`
	EffectiveTo              *time.Time      `json:"effective_to,omitempty"`
	CancelledAt              *time.Time      `json:"cancelled_at,omitempty"`
	CancelledBy              *uuid.UUID      `json:"cancelled_by,omitempty"`
	CreatedBy                *uuid.UUID      `json:"created_by,omitempty"`
	ChangeReason             string          `json:"change_reason"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
}

// ResolveStatus evaluates the configuration lifecycle state at reference timestamp at.
// The half-open interval [effective_from, effective_to) defines active validity:
//   - at < effective_from                 => SCHEDULED
//   - effective_from <= at < effective_to => ACTIVE
//   - at >= effective_to                  => SUPERSEDED
// If cancelled, it is only CANCELLED at or after CancelledAt (or if static status is CANCELLED without CancelledAt).
// For historical queries before CancelledAt, the cancellation had not taken place yet,
// reflecting its pre-cancellation lifecycle state at that historical timestamp.
func (c *PricingTaxConfiguration) ResolveStatus(at time.Time) string {
	if c.CancelledAt != nil {
		if !at.Before(*c.CancelledAt) {
			return PricingTaxStatusCancelled
		}
	} else if c.Status == PricingTaxStatusCancelled {
		return PricingTaxStatusCancelled
	}

	if at.Before(c.EffectiveFrom) {
		return PricingTaxStatusScheduled
	}
	if c.EffectiveTo == nil || at.Before(*c.EffectiveTo) {
		return PricingTaxStatusActive
	}
	return PricingTaxStatusSuperseded
}

// PricingBreakdown captures the computed monetary breakdown for a reservation.
type PricingBreakdown struct {
	Subtotal        decimal.Decimal `json:"subtotal"`
	PaidSubtotal    decimal.Decimal `json:"paid_subtotal"`
	PlatformFee     decimal.Decimal `json:"platform_fee"`
	TaxAmount       decimal.Decimal `json:"tax_amount"`
	DiscountAmount  decimal.Decimal `json:"discount_amount"`
	TotalAmount     decimal.Decimal `json:"total_amount"`
	PaidTicketCount int             `json:"paid_ticket_count"`
	FreeTicketCount int             `json:"free_ticket_count"`
}

// PricingTaxConfigurationRepository defines storage operations for pricing & tax configurations.
type PricingTaxConfigurationRepository interface {
	// FindActiveAt returns the single active configuration effective at timestamp t.
	FindActiveAt(ctx context.Context, t time.Time) (*PricingTaxConfiguration, error)

	// FindByID returns a configuration by its UUID.
	FindByID(ctx context.Context, id uuid.UUID) (*PricingTaxConfiguration, error)

	// FindLatest returns the latest configuration by version.
	FindLatest(ctx context.Context) (*PricingTaxConfiguration, error)

	// FindTimeline returns all non-cancelled configurations ordered by effective_from ASC,
	// with statuses resolved relative to the evaluation timestamp asOf.
	FindTimeline(ctx context.Context, asOf time.Time) ([]PricingTaxConfiguration, error)

	// FindHistory returns paginated configuration audit records ordered by version DESC,
	// with statuses resolved relative to the evaluation timestamp asOf.
	FindHistory(ctx context.Context, page, limit int, asOf time.Time) ([]PricingTaxConfiguration, int64, error)

	// Create inserts a new configuration.
	Create(ctx context.Context, config *PricingTaxConfiguration) error

	// Update updates an existing configuration record.
	Update(ctx context.Context, config *PricingTaxConfiguration) error

	// AcquireAdvisoryLock acquires an exclusive transaction-level PostgreSQL advisory lock.
	AcquireAdvisoryLock(ctx context.Context) error
}

var hundredDec = decimal.NewFromInt(100)

// CalculatePricing calculates fee, tax, and total based on reservation line items and active pricing config.
// Rounding follows domain banker's rounding (RoundBank(2), half-to-even) to the nearest cent/paisa.
func CalculatePricing(
	items []SeatReservationItem,
	config *PricingTaxConfiguration,
) (PricingBreakdown, error) {
	if config == nil {
		return PricingBreakdown{}, ErrPricingConfigurationUnavailable
	}

	subtotal := decimal.Zero
	paidSubtotal := decimal.Zero
	freeCount := 0
	paidCount := 0

	for _, item := range items {
		if item.Quantity <= 0 || item.Price.IsNegative() {
			return PricingBreakdown{}, ErrInvalidReservationRequest
		}

		itemTotal := item.Total()
		subtotal = subtotal.Add(itemTotal)

		if item.Price.IsPositive() {
			paidSubtotal = paidSubtotal.Add(itemTotal)
			paidCount += item.Quantity
		} else {
			freeCount += item.Quantity
		}
	}

	platformFee := decimal.Zero
	taxAmount := decimal.Zero
	discountAmount := decimal.Zero // Reserved for future promo/voucher engine

	// Platform fee: Percentage applied to aggregate paid-ticket subtotal.
	// Free tickets contribute 0 to paidSubtotal and are exempt by default.
	if config.PlatformFeeEnabled && config.PlatformFeePercent.IsPositive() && paidSubtotal.IsPositive() {
		platformFee = paidSubtotal.Mul(config.PlatformFeePercent).Div(hundredDec).RoundBank(2)
	}

	// Tax calculation: Only computed if tax is enabled and legally validated.
	if config.TaxEnabled && config.TaxPercent.IsPositive() && config.TaxRulesValidated {
		var taxableBasis decimal.Decimal

		switch config.TaxBasis {
		case TaxBasisPlatformFeeOnly:
			taxableBasis = platformFee
		case TaxBasisSubtotalOnly:
			taxableBasis = paidSubtotal
		case TaxBasisTotalBeforeTax:
			taxableBasis = paidSubtotal.Add(platformFee)
		default:
			taxableBasis = decimal.Zero
		}

		if taxableBasis.IsPositive() {
			taxAmount = taxableBasis.Mul(config.TaxPercent).Div(hundredDec).RoundBank(2)
		}
	}

	totalAmount := subtotal.Add(platformFee).Add(taxAmount).Sub(discountAmount)

	return PricingBreakdown{
		Subtotal:        subtotal,
		PaidSubtotal:    paidSubtotal,
		PlatformFee:     platformFee,
		TaxAmount:       taxAmount,
		DiscountAmount:  discountAmount,
		TotalAmount:     totalAmount,
		PaidTicketCount: paidCount,
		FreeTicketCount: freeCount,
	}, nil
}
