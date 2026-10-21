package domain

import (
	"errors"

	"github.com/shopspring/decimal"
)

var (
	// ErrNegativeAmount indicates a monetary value cannot be negative for gateway operations.
	ErrNegativeAmount = errors.New("monetary amount cannot be negative")

	// ErrAmountOverflow indicates the converted amount exceeds the capacity of a 64-bit integer.
	ErrAmountOverflow = errors.New("monetary amount exceeds maximum supported gateway value")
)

var hundred = decimal.NewFromInt(100)

// ToPaise converts a monetary Decimal (INR rupees) to integer paise for gateway consumption (e.g., Razorpay).
//
// Rounding Policy:
// Uses Banker's Rounding (RoundBank(0), half-to-even) to the nearest integer paise, avoiding
// cumulative upward rounding bias standard in financial calculations.
//
// Validation & Error Handling:
// - Returns ErrNegativeAmount if amount is negative.
// - Returns ErrAmountOverflow if the resulting paise value exceeds math.MaxInt64.
func ToPaise(amount decimal.Decimal) (int64, error) {
	if amount.IsNegative() {
		return 0, ErrNegativeAmount
	}

	paiseDec := amount.Mul(hundred).RoundBank(0)
	if !paiseDec.BigInt().IsInt64() {
		return 0, ErrAmountOverflow
	}

	return paiseDec.IntPart(), nil
}

// FromPaise converts integer paise to Decimal INR rupees with exact decimal division.
//
// Validation & Error Handling:
// - Returns ErrNegativeAmount if paise is negative.
func FromPaise(paise int64) (decimal.Decimal, error) {
	if paise < 0 {
		return decimal.Zero, ErrNegativeAmount
	}

	return decimal.NewFromInt(paise).Div(hundred), nil
}
