package events_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDecimalMinPriceCalculation(t *testing.T) {
	prices := []decimal.Decimal{
		decimal.RequireFromString("499.00"),
		decimal.RequireFromString("299.50"),
		decimal.RequireFromString("799.99"),
		decimal.RequireFromString("299.49"),
	}

	minPrice := decimal.NewFromInt(-1)

	for _, p := range prices {
		if minPrice.IsNegative() || p.LessThan(minPrice) {
			minPrice = p
		}
	}

	if minPrice.IsNegative() {
		minPrice = decimal.Zero
	}

	assert.Equal(t, "299.49", minPrice.String())
	assert.Equal(t, 299.49, minPrice.InexactFloat64())
}

func TestDecimalMinPriceCalculation_EmptyPrices(t *testing.T) {
	var prices []decimal.Decimal

	minPrice := decimal.NewFromInt(-1)

	for _, p := range prices {
		if minPrice.IsNegative() || p.LessThan(minPrice) {
			minPrice = p
		}
	}

	if minPrice.IsNegative() {
		minPrice = decimal.Zero
	}

	assert.True(t, decimal.Zero.Equal(minPrice))
	assert.Equal(t, 0.0, minPrice.InexactFloat64())
}

func TestDecimalNegativeValidation(t *testing.T) {
	negativePrice := decimal.RequireFromString("-0.01")
	zeroPrice := decimal.Zero
	positivePrice := decimal.RequireFromString("100.00")

	assert.True(t, negativePrice.IsNegative())
	assert.False(t, zeroPrice.IsNegative())
	assert.False(t, positivePrice.IsNegative())
}

func TestDecimalPrecisionPreservation(t *testing.T) {
	// Verify that fractional paise precision like 199.99 does not lose precision
	price := decimal.RequireFromString("199.99")
	qty := decimal.NewFromInt(7)

	total := price.Mul(qty)
	expected := decimal.RequireFromString("1399.93")

	assert.True(t, expected.Equal(total), "expected %s, got %s", expected.String(), total.String())
}
