package finance

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccount(t *testing.T) {
	validID := uuid.New()

	t.Run("valid asset account succeeds", func(t *testing.T) {
		acc, err := NewAccount(
			validID,
			AccountCodeRazorpayClearing,
			"Razorpay Clearing Account",
			AccountTypeAsset,
			NormalBalanceDebit,
			CurrencyINR,
			true,
		)
		require.NoError(t, err)
		assert.Equal(t, validID, acc.ID())
		assert.Equal(t, AccountCodeRazorpayClearing, acc.Code())
		assert.Equal(t, "Razorpay Clearing Account", acc.Name())
		assert.Equal(t, AccountTypeAsset, acc.Type())
		assert.Equal(t, NormalBalanceDebit, acc.NormalBalance())
		assert.Equal(t, CurrencyINR, acc.Currency())
		assert.True(t, acc.IsActive())
	})

	t.Run("valid liability control account succeeds", func(t *testing.T) {
		acc, err := NewAccount(
			validID,
			AccountCodeOrganizerPayable,
			"Organizer Payable Control Account",
			AccountTypeLiability,
			NormalBalanceCredit,
			CurrencyINR,
			true,
		)
		require.NoError(t, err)
		assert.Equal(t, AccountTypeLiability, acc.Type())
		assert.Equal(t, NormalBalanceCredit, acc.NormalBalance())
	})

	t.Run("valid platform fee revenue account succeeds", func(t *testing.T) {
		acc, err := NewAccount(
			validID,
			AccountCodePlatformFeeRevenue,
			"Platform Fee Revenue",
			AccountTypeRevenue,
			NormalBalanceCredit,
			CurrencyINR,
			true,
		)
		require.NoError(t, err)
		assert.Equal(t, AccountTypeRevenue, acc.Type())
		assert.Equal(t, NormalBalanceCredit, acc.NormalBalance())
	})

	t.Run("valid expense account succeeds", func(t *testing.T) {
		acc, err := NewAccount(
			validID,
			AccountCodePaymentGatewayExpense,
			"Payment Gateway Processing Fees",
			AccountTypeExpense,
			NormalBalanceDebit,
			CurrencyINR,
			true,
		)
		require.NoError(t, err)
		assert.Equal(t, AccountTypeExpense, acc.Type())
		assert.Equal(t, NormalBalanceDebit, acc.NormalBalance())
	})

	t.Run("valid equity account succeeds", func(t *testing.T) {
		acc, err := NewAccount(
			validID,
			"3010",
			"Retained Earnings",
			AccountTypeEquity,
			NormalBalanceCredit,
			CurrencyINR,
			true,
		)
		require.NoError(t, err)
		assert.Equal(t, AccountTypeEquity, acc.Type())
		assert.Equal(t, NormalBalanceCredit, acc.NormalBalance())
	})

	t.Run("invalid account type rejected", func(t *testing.T) {
		_, err := NewAccount(
			validID,
			"9999",
			"Invalid Type",
			"UNKNOWN_TYPE",
			NormalBalanceDebit,
			CurrencyINR,
			true,
		)
		assert.ErrorIs(t, err, ErrInvalidAccountType)
	})

	t.Run("invalid normal balance rejected", func(t *testing.T) {
		_, err := NewAccount(
			validID,
			"1010",
			"Invalid Balance",
			AccountTypeAsset,
			"UNKNOWN_BALANCE",
			CurrencyINR,
			true,
		)
		assert.ErrorIs(t, err, ErrInvalidNormalBalance)
	})

	t.Run("normal balance mismatch rejected", func(t *testing.T) {
		// Asset with CREDIT balance must be rejected
		_, err := NewAccount(
			validID,
			"1010",
			"Asset as Credit",
			AccountTypeAsset,
			NormalBalanceCredit,
			CurrencyINR,
			true,
		)
		assert.ErrorIs(t, err, ErrNormalBalanceMismatch)

		// Liability with DEBIT balance must be rejected
		_, err2 := NewAccount(
			validID,
			"2010",
			"Liability as Debit",
			AccountTypeLiability,
			NormalBalanceDebit,
			CurrencyINR,
			true,
		)
		assert.ErrorIs(t, err2, ErrNormalBalanceMismatch)
	})

	t.Run("unsupported currency rejected", func(t *testing.T) {
		_, err := NewAccount(
			validID,
			AccountCodeRazorpayClearing,
			"Razorpay USD",
			AccountTypeAsset,
			NormalBalanceDebit,
			"USD",
			true,
		)
		assert.ErrorIs(t, err, ErrUnsupportedCurrency)
	})

	t.Run("nil uuid rejected", func(t *testing.T) {
		_, err := NewAccount(
			uuid.Nil,
			"1010",
			"Test",
			AccountTypeAsset,
			NormalBalanceDebit,
			CurrencyINR,
			true,
		)
		assert.ErrorIs(t, err, ErrInvalidAccountID)
	})

	t.Run("empty code rejected", func(t *testing.T) {
		_, err := NewAccount(
			validID,
			"   ",
			"Test",
			AccountTypeAsset,
			NormalBalanceDebit,
			CurrencyINR,
			true,
		)
		assert.ErrorIs(t, err, ErrInvalidAccountCode)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		_, err := NewAccount(
			validID,
			"1010",
			"   ",
			AccountTypeAsset,
			NormalBalanceDebit,
			CurrencyINR,
			true,
		)
		assert.ErrorIs(t, err, ErrInvalidAccountName)
	})

	t.Run("json serialization and deserialization", func(t *testing.T) {
		acc, err := NewAccount(
			validID,
			AccountCodeRazorpayClearing,
			"Razorpay Clearing Account",
			AccountTypeAsset,
			NormalBalanceDebit,
			CurrencyINR,
			true,
		)
		require.NoError(t, err)

		data, err := json.Marshal(acc)
		require.NoError(t, err)

		var unmarshaled Account
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, acc.ID(), unmarshaled.ID())
		assert.Equal(t, acc.Code(), unmarshaled.Code())
		assert.Equal(t, acc.Name(), unmarshaled.Name())
		assert.Equal(t, acc.Type(), unmarshaled.Type())
		assert.Equal(t, acc.NormalBalance(), unmarshaled.NormalBalance())
		assert.Equal(t, acc.Currency(), unmarshaled.Currency())
		assert.Equal(t, acc.IsActive(), unmarshaled.IsActive())
	})
}
