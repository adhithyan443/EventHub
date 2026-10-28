package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculatePricing(t *testing.T) {
	now := time.Now().UTC()

	activeConfig := &PricingTaxConfiguration{
		ID:                 uuid.New(),
		Version:            2,
		Status:             PricingTaxStatusActive,
		PlatformFeeEnabled: true,
		PlatformFeePercent: decimal.NewFromFloat(5.00), // 5%
		TaxEnabled:         false,
		TaxPercent:         decimal.Zero,
		TaxBasis:           TaxBasisUnvalidated,
		TaxRulesValidated:  false,
		FreeTicketExempt:   true,
		EffectiveFrom:      now.Add(-24 * time.Hour),
		EffectiveTo:        nil,
	}

	t.Run("nil configuration returns fail-closed error", func(t *testing.T) {
		items := []SeatReservationItem{
			{Quantity: 1, Price: decimal.NewFromInt(100)},
		}
		_, err := CalculatePricing(items, nil)
		require.ErrorIs(t, err, ErrPricingConfigurationUnavailable)
	})

	t.Run("invalid item with zero or negative quantity returns error", func(t *testing.T) {
		items := []SeatReservationItem{
			{Quantity: 0, Price: decimal.NewFromInt(100)},
		}
		_, err := CalculatePricing(items, activeConfig)
		require.ErrorIs(t, err, ErrInvalidReservationRequest)

		itemsNegative := []SeatReservationItem{
			{Quantity: -1, Price: decimal.NewFromInt(100)},
		}
		_, err = CalculatePricing(itemsNegative, activeConfig)
		require.ErrorIs(t, err, ErrInvalidReservationRequest)
	})

	t.Run("invalid item with negative price returns error", func(t *testing.T) {
		items := []SeatReservationItem{
			{Quantity: 1, Price: decimal.NewFromInt(-50)},
		}
		_, err := CalculatePricing(items, activeConfig)
		require.ErrorIs(t, err, ErrInvalidReservationRequest)
	})

	t.Run("pure free-ticket booking is completely exempt from fees and taxes", func(t *testing.T) {
		items := []SeatReservationItem{
			{Quantity: 2, Price: decimal.Zero},
			{Quantity: 1, Price: decimal.Zero},
		}

		result, err := CalculatePricing(items, activeConfig)
		require.NoError(t, err)

		assert.True(t, result.Subtotal.IsZero())
		assert.True(t, result.PaidSubtotal.IsZero())
		assert.True(t, result.PlatformFee.IsZero())
		assert.True(t, result.TaxAmount.IsZero())
		assert.True(t, result.TotalAmount.IsZero())
		assert.Equal(t, 3, result.FreeTicketCount)
		assert.Equal(t, 0, result.PaidTicketCount)

		// Convert to paise for gateway
		paise, err := ToPaise(result.TotalAmount)
		require.NoError(t, err)
		assert.Equal(t, int64(0), paise)
	})

	t.Run("pure paid booking calculates fee on aggregate subtotal", func(t *testing.T) {
		// 2 tickets @ 250 = 500
		// 1 ticket @ 500 = 500
		// PaidSubtotal = 1000
		// Platform fee at 5% = 50.00
		// Total = 1050.00
		items := []SeatReservationItem{
			{Quantity: 2, Price: decimal.NewFromInt(250)},
			{Quantity: 1, Price: decimal.NewFromInt(500)},
		}

		result, err := CalculatePricing(items, activeConfig)
		require.NoError(t, err)

		assert.True(t, result.Subtotal.Equal(decimal.NewFromInt(1000)))
		assert.True(t, result.PaidSubtotal.Equal(decimal.NewFromInt(1000)))
		assert.True(t, result.PlatformFee.Equal(decimal.NewFromInt(50)))
		assert.True(t, result.TaxAmount.IsZero())
		assert.True(t, result.TotalAmount.Equal(decimal.NewFromInt(1050)))
		assert.Equal(t, 0, result.FreeTicketCount)
		assert.Equal(t, 3, result.PaidTicketCount)

		// Convert to paise
		paise, err := ToPaise(result.TotalAmount)
		require.NoError(t, err)
		assert.Equal(t, int64(105000), paise)
	})

	t.Run("mixed-ticket booking isolates free tickets and calculates fee only on paid tickets", func(t *testing.T) {
		// 2 Free tickets @ 0
		// 1 Paid VIP ticket @ 750
		// Subtotal = 750
		// PaidSubtotal = 750
		// Platform fee at 5% on 750 = 37.50
		// Total = 787.50
		items := []SeatReservationItem{
			{Quantity: 2, Price: decimal.Zero},
			{Quantity: 1, Price: decimal.NewFromInt(750)},
		}

		result, err := CalculatePricing(items, activeConfig)
		require.NoError(t, err)

		assert.True(t, result.Subtotal.Equal(decimal.NewFromInt(750)))
		assert.True(t, result.PaidSubtotal.Equal(decimal.NewFromInt(750)))
		assert.True(t, result.PlatformFee.Equal(decimal.NewFromFloat(37.50)))
		assert.True(t, result.TaxAmount.IsZero())
		assert.True(t, result.TotalAmount.Equal(decimal.NewFromFloat(787.50)))
		assert.Equal(t, 2, result.FreeTicketCount)
		assert.Equal(t, 1, result.PaidTicketCount)

		paise, err := ToPaise(result.TotalAmount)
		require.NoError(t, err)
		assert.Equal(t, int64(78750), paise)
	})

	t.Run("banker's rounding half to even on fee calculation", func(t *testing.T) {
		// Test half to even rounding:
		// Subtotal = 101, fee = 2.5% -> 101 * 0.025 = 2.525 -> rounds to even: 2.52
		// Subtotal = 103, fee = 2.5% -> 103 * 0.025 = 2.575 -> rounds to even: 2.58
		configHalfEven := &PricingTaxConfiguration{
			PlatformFeeEnabled: true,
			PlatformFeePercent: decimal.NewFromFloat(2.50),
		}

		items1 := []SeatReservationItem{
			{Quantity: 1, Price: decimal.NewFromInt(101)},
		}
		res1, err := CalculatePricing(items1, configHalfEven)
		require.NoError(t, err)
		assert.Equal(t, "2.52", res1.PlatformFee.StringFixed(2))

		items2 := []SeatReservationItem{
			{Quantity: 1, Price: decimal.NewFromInt(103)},
		}
		res2, err := CalculatePricing(items2, configHalfEven)
		require.NoError(t, err)
		assert.Equal(t, "2.58", res2.PlatformFee.StringFixed(2))
	})

	t.Run("disabled platform fee results in zero fee", func(t *testing.T) {
		disabledConfig := &PricingTaxConfiguration{
			PlatformFeeEnabled: false,
			PlatformFeePercent: decimal.NewFromFloat(5.00),
		}
		items := []SeatReservationItem{
			{Quantity: 1, Price: decimal.NewFromInt(500)},
		}
		res, err := CalculatePricing(items, disabledConfig)
		require.NoError(t, err)
		assert.True(t, res.PlatformFee.IsZero())
		assert.True(t, res.TotalAmount.Equal(decimal.NewFromInt(500)))
	})

	t.Run("tax enabled with validation calculates correctly based on basis", func(t *testing.T) {
		// Subtotal = 1000, Platform fee = 5% (50.00)
		// Tax = 18%
		// Basis: PlatformFeeOnly -> Tax = 50 * 0.18 = 9.00
		configPlatformFeeOnly := &PricingTaxConfiguration{
			PlatformFeeEnabled: true,
			PlatformFeePercent: decimal.NewFromFloat(5.00),
			TaxEnabled:         true,
			TaxPercent:         decimal.NewFromFloat(18.00),
			TaxBasis:           TaxBasisPlatformFeeOnly,
			TaxRulesValidated:  true,
		}

		items := []SeatReservationItem{
			{Quantity: 1, Price: decimal.NewFromInt(1000)},
		}
		res, err := CalculatePricing(items, configPlatformFeeOnly)
		require.NoError(t, err)
		assert.Equal(t, "50.00", res.PlatformFee.StringFixed(2))
		assert.Equal(t, "9.00", res.TaxAmount.StringFixed(2))
		assert.Equal(t, "1059.00", res.TotalAmount.StringFixed(2))

		// Basis: SubtotalOnly -> Tax = 1000 * 0.18 = 180.00
		configSubtotalOnly := &PricingTaxConfiguration{
			PlatformFeeEnabled: true,
			PlatformFeePercent: decimal.NewFromFloat(5.00),
			TaxEnabled:         true,
			TaxPercent:         decimal.NewFromFloat(18.00),
			TaxBasis:           TaxBasisSubtotalOnly,
			TaxRulesValidated:  true,
		}
		resSub, err := CalculatePricing(items, configSubtotalOnly)
		require.NoError(t, err)
		assert.Equal(t, "180.00", resSub.TaxAmount.StringFixed(2))
		assert.Equal(t, "1230.00", resSub.TotalAmount.StringFixed(2))

		// Basis: TotalBeforeTax -> Tax = (1000 + 50) * 0.18 = 189.00
		configTotalBeforeTax := &PricingTaxConfiguration{
			PlatformFeeEnabled: true,
			PlatformFeePercent: decimal.NewFromFloat(5.00),
			TaxEnabled:         true,
			TaxPercent:         decimal.NewFromFloat(18.00),
			TaxBasis:           TaxBasisTotalBeforeTax,
			TaxRulesValidated:  true,
		}
		resTot, err := CalculatePricing(items, configTotalBeforeTax)
		require.NoError(t, err)
		assert.Equal(t, "189.00", resTot.TaxAmount.StringFixed(2))
		assert.Equal(t, "1239.00", resTot.TotalAmount.StringFixed(2))
	})

	t.Run("unvalidated tax is not calculated even if tax_enabled is true", func(t *testing.T) {
		configUnvalidated := &PricingTaxConfiguration{
			PlatformFeeEnabled: true,
			PlatformFeePercent: decimal.NewFromFloat(5.00),
			TaxEnabled:         true,
			TaxPercent:         decimal.NewFromFloat(18.00),
			TaxBasis:           TaxBasisUnvalidated,
			TaxRulesValidated:  false, // NOT validated
		}

		items := []SeatReservationItem{
			{Quantity: 1, Price: decimal.NewFromInt(1000)},
		}
		res, err := CalculatePricing(items, configUnvalidated)
		require.NoError(t, err)
		assert.True(t, res.TaxAmount.IsZero())
		assert.Equal(t, "1050.00", res.TotalAmount.StringFixed(2))
	})
}
