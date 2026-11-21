package repository_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinanceAccountRepository_Integration(t *testing.T) {
	db := initTestDatabase(t)
	repo := repository.NewFinanceAccountRepository(db, slog.Default())
	ctx := context.Background()

	t.Run("Successful account creation, FindByID, and FindByCode", func(t *testing.T) {
		accID := uuid.New()
		code := "TEST_ACC_" + uuid.New().String()[:8]
		acc, err := finance.NewAccount(
			accID,
			code,
			"Test Operating Account",
			finance.AccountTypeAsset,
			finance.NormalBalanceDebit,
			finance.CurrencyINR,
			true,
		)
		require.NoError(t, err)

		// 1. Create account
		err = repo.Create(ctx, acc)
		require.NoError(t, err, "Account creation must succeed")

		// 2. Find by ID
		foundByID, err := repo.FindByID(ctx, accID)
		require.NoError(t, err)
		require.NotNil(t, foundByID)
		assert.Equal(t, accID, foundByID.ID())
		assert.Equal(t, code, foundByID.Code())
		assert.Equal(t, "Test Operating Account", foundByID.Name())
		assert.Equal(t, finance.AccountTypeAsset, foundByID.Type())
		assert.Equal(t, finance.NormalBalanceDebit, foundByID.NormalBalance())
		assert.Equal(t, finance.CurrencyINR, foundByID.Currency())
		assert.True(t, foundByID.IsActive())

		// 3. Find by Code
		foundByCode, err := repo.FindByCode(ctx, code)
		require.NoError(t, err)
		require.NotNil(t, foundByCode)
		assert.Equal(t, accID, foundByCode.ID())
		assert.Equal(t, code, foundByCode.Code())

		// 4. Find by Code with surrounding whitespace
		foundWithSpace, err := repo.FindByCode(ctx, "  "+code+"  ")
		require.NoError(t, err)
		require.NotNil(t, foundWithSpace)
		assert.Equal(t, accID, foundWithSpace.ID())
	})

	t.Run("Duplicate account code rejection", func(t *testing.T) {
		code := "DUP_" + uuid.New().String()[:8]
		acc1, err := finance.NewAccount(
			uuid.New(),
			code,
			"First Account",
			finance.AccountTypeLiability,
			finance.NormalBalanceCredit,
			finance.CurrencyINR,
			true,
		)
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, acc1))

		acc2, err := finance.NewAccount(
			uuid.New(),
			code,
			"Conflicting Account",
			finance.AccountTypeLiability,
			finance.NormalBalanceCredit,
			finance.CurrencyINR,
			true,
		)
		require.NoError(t, err)

		err = repo.Create(ctx, acc2)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrFinanceAccountAlreadyExists), "Must map 23505 duplicate code to ErrFinanceAccountAlreadyExists")
	})

	t.Run("Account not found by ID and by Code", func(t *testing.T) {
		nonExistentID := uuid.New()
		foundByID, err := repo.FindByID(ctx, nonExistentID)
		require.Error(t, err)
		assert.Nil(t, foundByID)
		assert.True(t, errors.Is(err, domain.ErrFinanceAccountNotFound), "Must return ErrFinanceAccountNotFound for missing ID")

		nonExistentCode := "NON_EXISTENT_" + uuid.New().String()
		foundByCode, err := repo.FindByCode(ctx, nonExistentCode)
		require.Error(t, err)
		assert.Nil(t, foundByCode)
		assert.True(t, errors.Is(err, domain.ErrFinanceAccountNotFound), "Must return ErrFinanceAccountNotFound for missing Code")
	})

	t.Run("Invalid account rejection", func(t *testing.T) {
		// Nil account
		err := repo.Create(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account cannot be nil")

		// Nil UUID in FindByID
		found, err := repo.FindByID(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Nil(t, found)
		assert.True(t, errors.Is(err, finance.ErrInvalidAccountID))

		// Empty code in FindByCode
		found, err = repo.FindByCode(ctx, "   ")
		require.Error(t, err)
		assert.Nil(t, found)
		assert.True(t, errors.Is(err, finance.ErrInvalidAccountCode))
	})

	t.Run("Context cancellation error propagation", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel

		acc, err := finance.NewAccount(
			uuid.New(),
			"CTX_TEST_"+uuid.New().String()[:8],
			"Ctx Account",
			finance.AccountTypeRevenue,
			finance.NormalBalanceCredit,
			finance.CurrencyINR,
			true,
		)
		require.NoError(t, err)

		err = repo.Create(canceledCtx, acc)
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled) || errors.Is(canceledCtx.Err(), context.Canceled))

		_, err = repo.FindByID(canceledCtx, acc.ID())
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled) || errors.Is(canceledCtx.Err(), context.Canceled))
	})
}
