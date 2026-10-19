package domain_test

import (
	"math"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToPaise(t *testing.T) {
	tests := []struct {
		name        string
		amount      decimal.Decimal
		expected    int64
		expectError bool
		errType     error
	}{
		{
			name:     "zero amount",
			amount:   decimal.Zero,
			expected: 0,
		},
		{
			name:     "whole rupees",
			amount:   decimal.NewFromInt(500),
			expected: 50000,
		},
		{
			name:     "fractional rupees with two decimals",
			amount:   decimal.RequireFromString("499.50"),
			expected: 49950,
		},
		{
			name:     "small fraction 99 paise",
			amount:   decimal.RequireFromString("0.99"),
			expected: 99,
		},
		{
			name:     "small fraction 1 paisa",
			amount:   decimal.RequireFromString("0.01"),
			expected: 1,
		},
		{
			name:     "banker's rounding half to even down (2.5 paise to 2 paise)",
			amount:   decimal.RequireFromString("0.025"),
			expected: 2, // 2.5 rounds to even 2
		},
		{
			name:     "banker's rounding half to even up (3.5 paise to 4 paise)",
			amount:   decimal.RequireFromString("0.035"),
			expected: 4, // 3.5 rounds to even 4
		},
		{
			name:        "negative amount returns error",
			amount:      decimal.RequireFromString("-10.00"),
			expectError: true,
			errType:     domain.ErrNegativeAmount,
		},
		{
			name:        "large amount exceeding int64 overflow",
			amount:      decimal.NewFromInt(math.MaxInt64).Mul(decimal.NewFromInt(10)),
			expectError: true,
			errType:     domain.ErrAmountOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ToPaise(tt.amount)
			if tt.expectError {
				require.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestFromPaise(t *testing.T) {
	tests := []struct {
		name        string
		paise       int64
		expected    decimal.Decimal
		expectError bool
	}{
		{
			name:     "zero paise",
			paise:    0,
			expected: decimal.Zero,
		},
		{
			name:     "50000 paise to 500 rupees",
			paise:    50000,
			expected: decimal.RequireFromString("500"),
		},
		{
			name:     "49950 paise to 499.50 rupees",
			paise:    49950,
			expected: decimal.RequireFromString("499.5"),
		},
		{
			name:     "1 paisa to 0.01 rupees",
			paise:    1,
			expected: decimal.RequireFromString("0.01"),
		},
		{
			name:        "negative paise returns error",
			paise:       -100,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.FromPaise(tt.paise)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, domain.ErrNegativeAmount)
			} else {
				require.NoError(t, err)
				assert.True(t, tt.expected.Equal(got), "expected %s, got %s", tt.expected.String(), got.String())
			}
		})
	}
}
