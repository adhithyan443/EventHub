package finance

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMoney(t *testing.T) {
	t.Run("valid INR amount succeeds", func(t *testing.T) {
		m, err := NewMoney(decimal.NewFromFloat(100.50), "INR")
		require.NoError(t, err)
		assert.Equal(t, "INR", m.Currency())
		assert.True(t, m.Amount().Equal(decimal.NewFromFloat(100.50)))
	})

	t.Run("lowercase inr is normalized to uppercase", func(t *testing.T) {
		m, err := NewMoney(decimal.NewFromFloat(50.00), "inr")
		require.NoError(t, err)
		assert.Equal(t, "INR", m.Currency())
	})

	t.Run("unsupported currency rejected", func(t *testing.T) {
		currencies := []string{"USD", "EUR", "GBP", "JPY", ""}
		for _, curr := range currencies {
			_, err := NewMoney(decimal.NewFromFloat(100), curr)
			assert.ErrorIs(t, err, ErrUnsupportedCurrency, "currency %s should be rejected", curr)
		}
	})

	t.Run("fractional paise rejected in NewMoney", func(t *testing.T) {
		// NewMoney must not silently round fractional paise; it must reject them.
		_, err1 := NewMoney(decimal.RequireFromString("100.555"), "INR")
		assert.ErrorIs(t, err1, ErrFractionalPaise)

		_, err2 := NewMoney(decimal.RequireFromString("100.545"), "INR")
		assert.ErrorIs(t, err2, ErrFractionalPaise)

		_, err3 := NewMoney(decimal.RequireFromString("0.001"), "INR")
		assert.ErrorIs(t, err3, ErrFractionalPaise)
	})

	t.Run("NewMoneyFromDecimalRounded applies Banker rounding to two decimals", func(t *testing.T) {
		// 100.555 -> 100.56 (rounds to nearest even)
		m1, err := NewMoneyFromDecimalRounded(decimal.RequireFromString("100.555"), "INR")
		require.NoError(t, err)
		assert.Equal(t, "100.56", m1.Amount().StringFixed(2))

		// 100.545 -> 100.54 (rounds to nearest even)
		m2, err := NewMoneyFromDecimalRounded(decimal.RequireFromString("100.545"), "INR")
		require.NoError(t, err)
		assert.Equal(t, "100.54", m2.Amount().StringFixed(2))
	})
}

func TestNewPositiveMoney(t *testing.T) {
	t.Run("positive amount succeeds", func(t *testing.T) {
		m, err := NewPositiveMoney(decimal.NewFromFloat(0.01), "INR")
		require.NoError(t, err)
		assert.True(t, m.IsPositive())
	})

	t.Run("fractional paise in positive money rejected", func(t *testing.T) {
		_, err := NewPositiveMoney(decimal.RequireFromString("10.555"), "INR")
		assert.ErrorIs(t, err, ErrFractionalPaise)
	})

	t.Run("zero amount rejected", func(t *testing.T) {
		_, err := NewPositiveMoney(decimal.Zero, "INR")
		assert.ErrorIs(t, err, ErrNonPositiveMoney)
	})

	t.Run("negative amount rejected", func(t *testing.T) {
		_, err := NewPositiveMoney(decimal.NewFromFloat(-10.50), "INR")
		assert.ErrorIs(t, err, ErrNonPositiveMoney)
	})
}

func TestNewMoneyFromPaise(t *testing.T) {
	t.Run("converts paise to rupees accurately", func(t *testing.T) {
		m, err := NewMoneyFromPaise(10050, "INR")
		require.NoError(t, err)
		assert.Equal(t, "100.50", m.Amount().StringFixed(2))
	})

	t.Run("unsupported currency rejected", func(t *testing.T) {
		_, err := NewMoneyFromPaise(10050, "USD")
		assert.ErrorIs(t, err, ErrUnsupportedCurrency)
	})
}

func TestZeroINR(t *testing.T) {
	z := ZeroINR()
	assert.Equal(t, "INR", z.Currency())
	assert.True(t, z.IsZero())
	assert.False(t, z.IsPositive())
	assert.False(t, z.IsNegative())
	assert.Equal(t, "0.00", z.Amount().StringFixed(2))
}

func TestMoney_Arithmetic(t *testing.T) {
	m100, _ := NewMoney(decimal.NewFromInt(100), "INR")
	m50, _ := NewMoney(decimal.NewFromInt(50), "INR")

	t.Run("safe addition preserves exactness", func(t *testing.T) {
		sum, err := m100.Add(m50)
		require.NoError(t, err)
		assert.Equal(t, "150.00", sum.Amount().StringFixed(2))
	})

	t.Run("safe subtraction preserves exactness", func(t *testing.T) {
		diff, err := m100.Sub(m50)
		require.NoError(t, err)
		assert.Equal(t, "50.00", diff.Amount().StringFixed(2))
	})

	t.Run("currency mismatch on add or sub rejected", func(t *testing.T) {
		other := Money{amount: decimal.NewFromInt(50), currency: "USD"}
		_, errAdd := m100.Add(other)
		assert.ErrorIs(t, errAdd, ErrCurrencyMismatch)

		_, errSub := m100.Sub(other)
		assert.ErrorIs(t, errSub, ErrCurrencyMismatch)
	})
}

func TestMoney_Comparison(t *testing.T) {
	m100, _ := NewMoney(decimal.NewFromInt(100), "INR")
	m100Alt, _ := NewMoney(decimal.RequireFromString("100.00"), "INR")
	m50, _ := NewMoney(decimal.NewFromInt(50), "INR")

	assert.True(t, m100.Equal(m100Alt))
	assert.False(t, m100.Equal(m50))

	gt, err := m100.GreaterThan(m50)
	require.NoError(t, err)
	assert.True(t, gt)

	lt, err := m50.LessThan(m100)
	require.NoError(t, err)
	assert.True(t, lt)

	cmp, err := m100.Cmp(m50)
	require.NoError(t, err)
	assert.Equal(t, 1, cmp)

	cmpRev, err := m50.Cmp(m100)
	require.NoError(t, err)
	assert.Equal(t, -1, cmpRev)

	cmpEqual, err := m100.Cmp(m100Alt)
	require.NoError(t, err)
	assert.Equal(t, 0, cmpEqual)
}

func TestMoney_ToPaise(t *testing.T) {
	t.Run("converts valid INR to paise", func(t *testing.T) {
		m, _ := NewMoney(decimal.RequireFromString("499.99"), "INR")
		paise, err := m.ToPaise()
		require.NoError(t, err)
		assert.Equal(t, int64(49999), paise)
	})

	t.Run("negative amount rejected for paise conversion", func(t *testing.T) {
		m, _ := NewMoney(decimal.NewFromInt(-100), "INR")
		_, err := m.ToPaise()
		assert.ErrorIs(t, err, ErrNegativeMoney)
	})
}

func TestMoney_String(t *testing.T) {
	m, _ := NewMoney(decimal.RequireFromString("1234.50"), "INR")
	assert.Equal(t, "INR 1234.50", m.String())
}

func TestMoney_JSON(t *testing.T) {
	m, err := NewMoney(decimal.RequireFromString("1250.75"), "INR")
	require.NoError(t, err)

	data, err := json.Marshal(m)
	require.NoError(t, err)

	var unmarshaled Money
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.True(t, m.Equal(unmarshaled))
}
