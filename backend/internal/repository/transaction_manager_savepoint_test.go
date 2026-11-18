package repository_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionManager_ExecuteWithSavepoint(t *testing.T) {
	db := initTestDatabase(t)
	txManager := repository.NewTransactionManager(db, slog.Default())

	t.Run("Rejects nil callback function", func(t *testing.T) {
		err := txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
			res, spErr := tx.ExecuteWithSavepoint(domain.SavepointTaxRefundInventory, nil)
			require.Error(t, spErr)
			assert.True(t, errors.Is(spErr, domain.ErrNilSavepointCallback))
			assert.Equal(t, domain.SavepointStatus(""), res.Status)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("Rejects untrusted or arbitrary savepoint name", func(t *testing.T) {
		err := txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
			untrustedNames := []domain.SavepointName{
				"",
				"arbitrary_savepoint",
				"sp_tax_refund_inventory; DROP TABLE users; --",
				"sp_finance_idempotency", // Not whitelisted for usecase layer
			}
			for _, name := range untrustedNames {
				res, spErr := tx.ExecuteWithSavepoint(name, func() error { return nil })
				require.Error(t, spErr, "Untrusted name %q must be rejected", name)
				assert.True(t, errors.Is(spErr, domain.ErrInvalidSavepointName))
				assert.Equal(t, domain.SavepointStatus(""), res.Status)
			}
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("Successful callback commits and releases savepoint", func(t *testing.T) {
		err := txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
			callbackExecuted := false
			res, spErr := tx.ExecuteWithSavepoint(domain.SavepointTaxRefundInventory, func() error {
				callbackExecuted = true
				return nil
			})
			require.NoError(t, spErr)
			assert.True(t, callbackExecuted)
			assert.Equal(t, domain.SavepointStatusCommitted, res.Status)
			assert.Nil(t, res.CallbackErr)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("Failing callback rolls back and releases savepoint while keeping outer tx healthy", func(t *testing.T) {
		expectedCallbackErr := errors.New("inventory release simulated failure")

		err := txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
			res, spErr := tx.ExecuteWithSavepoint(domain.SavepointTaxRefundInventory, func() error {
				return expectedCallbackErr
			})
			require.NoError(t, spErr, "Savepoint management itself must succeed")
			assert.Equal(t, domain.SavepointStatusRolledBack, res.Status)
			assert.True(t, errors.Is(res.CallbackErr, expectedCallbackErr))

			// Safety verification: The outer transaction must remain completely healthy and usable
			clearingAcc, findErr := tx.FinanceAccountRepository().FindByCode(t.Context(), finance.AccountCodeRazorpayClearing)
			// Whether found or not, the query must execute without transaction-aborted errors ("current transaction is aborted")
			if findErr != nil {
				assert.False(t, errors.Is(findErr, domain.ErrConcurrentPosting))
				assert.NotContains(t, findErr.Error(), "current transaction is aborted")
			}
			_ = clearingAcc

			return nil
		})
		require.NoError(t, err)
	})
}
