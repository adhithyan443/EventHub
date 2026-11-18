package finance

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// CurrencyINR is the sole supported currency for v1.
const CurrencyINR = "INR"

var (
	hundredDec = decimal.NewFromInt(100)
	zeroDec    = decimal.Zero.Truncate(2)
)

// Money is an immutable monetary value object with exact two-decimal precision.
// It uses shopspring/decimal to prevent binary floating-point rounding errors.
type Money struct {
	amount   decimal.Decimal
	currency string
}

// NewMoney creates a new Money value object in INR.
// It strictly rejects amounts with fractional paise (more than two decimal places)
// to prevent silent precision loss in financial transactions.
func NewMoney(amount decimal.Decimal, currency string) (Money, error) {
	cleanCurrency := strings.ToUpper(strings.TrimSpace(currency))
	if cleanCurrency != CurrencyINR {
		return Money{}, ErrUnsupportedCurrency
	}

	// Double-entry ledger amounts must represent exact paise.
	// Truncating at 2 decimal places must equal the original amount; otherwise, fractional paise exist.
	if !amount.Truncate(2).Equal(amount) {
		return Money{}, ErrFractionalPaise
	}

	return Money{
		amount:   amount.Truncate(2),
		currency: CurrencyINR,
	}, nil
}

// NewMoneyFromDecimalRounded creates a Money value object applying Banker's (half-to-even)
// rounding to two decimal places. This is intended for calculated amounts (such as fees, commissions, or taxes).
func NewMoneyFromDecimalRounded(amount decimal.Decimal, currency string) (Money, error) {
	cleanCurrency := strings.ToUpper(strings.TrimSpace(currency))
	if cleanCurrency != CurrencyINR {
		return Money{}, ErrUnsupportedCurrency
	}

	return Money{
		amount:   amount.RoundBank(2),
		currency: CurrencyINR,
	}, nil
}

// NewPositiveMoney creates a Money value object that must be strictly positive (> 0).
func NewPositiveMoney(amount decimal.Decimal, currency string) (Money, error) {
	m, err := NewMoney(amount, currency)
	if err != nil {
		return Money{}, err
	}

	if !m.IsPositive() {
		return Money{}, ErrNonPositiveMoney
	}

	return m, nil
}

// NewMoneyFromPaise converts integer paise to a Money value object in INR.
func NewMoneyFromPaise(paise int64, currency string) (Money, error) {
	cleanCurrency := strings.ToUpper(strings.TrimSpace(currency))
	if cleanCurrency != CurrencyINR {
		return Money{}, ErrUnsupportedCurrency
	}

	amount := decimal.NewFromInt(paise).Div(hundredDec).Truncate(2)
	return Money{
		amount:   amount,
		currency: CurrencyINR,
	}, nil
}

// ZeroINR returns a zero-valued Money object in INR.
func ZeroINR() Money {
	return Money{
		amount:   zeroDec,
		currency: CurrencyINR,
	}
}

// Amount returns the underlying exact decimal.Decimal value.
func (m Money) Amount() decimal.Decimal {
	return m.amount
}

// Currency returns the currency ISO code (e.g. "INR").
func (m Money) Currency() string {
	return m.currency
}

// IsPositive returns true if amount > 0.
func (m Money) IsPositive() bool {
	return m.amount.IsPositive()
}

// IsNegative returns true if amount < 0.
func (m Money) IsNegative() bool {
	return m.amount.IsNegative()
}

// IsZero returns true if amount == 0.
func (m Money) IsZero() bool {
	return m.amount.IsZero()
}

// Add safely adds two Money objects with the same currency.
// Adding two exact two-decimal numbers preserves exactness without rounding.
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}

	return Money{
		amount:   m.amount.Add(other.amount),
		currency: m.currency,
	}, nil
}

// Sub safely subtracts another Money object with the same currency.
// Subtracting two exact two-decimal numbers preserves exactness without rounding.
func (m Money) Sub(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}

	return Money{
		amount:   m.amount.Sub(other.amount),
		currency: m.currency,
	}, nil
}

// Equal returns true if both Money objects have identical currency and amount.
func (m Money) Equal(other Money) bool {
	return m.currency == other.currency && m.amount.Equal(other.amount)
}

// GreaterThan returns true if this amount is strictly greater than other.
func (m Money) GreaterThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, ErrCurrencyMismatch
	}
	return m.amount.GreaterThan(other.amount), nil
}

// LessThan returns true if this amount is strictly less than other.
func (m Money) LessThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, ErrCurrencyMismatch
	}
	return m.amount.LessThan(other.amount), nil
}

// Cmp compares two Money objects: -1 if m < other, 0 if m == other, 1 if m > other.
func (m Money) Cmp(other Money) (int, error) {
	if m.currency != other.currency {
		return 0, ErrCurrencyMismatch
	}
	return m.amount.Cmp(other.amount), nil
}

// ToPaise converts the INR monetary value to integer paise for payment gateways.
func (m Money) ToPaise() (int64, error) {
	if m.amount.IsNegative() {
		return 0, ErrNegativeMoney
	}

	paiseDec := m.amount.Mul(hundredDec).Truncate(0)
	if !paiseDec.BigInt().IsInt64() {
		return 0, fmt.Errorf("money amount exceeds maximum 64-bit integer paise")
	}

	return paiseDec.IntPart(), nil
}

// String returns the formatted monetary string, e.g. "INR 1000.00".
func (m Money) String() string {
	return fmt.Sprintf("%s %s", m.currency, m.amount.StringFixed(2))
}

// MarshalJSON serializes Money as a structured JSON object.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}{
		Amount:   m.amount.StringFixed(2),
		Currency: m.currency,
	})
}

// UnmarshalJSON deserializes Money from a structured JSON object.
func (m *Money) UnmarshalJSON(data []byte) error {
	var aux struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	dec, err := decimal.NewFromString(aux.Amount)
	if err != nil {
		return err
	}

	parsed, err := NewMoney(dec, aux.Currency)
	if err != nil {
		return err
	}

	*m = parsed
	return nil
}
