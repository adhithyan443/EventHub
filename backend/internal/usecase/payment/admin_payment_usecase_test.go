package payment_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAdminPaymentRepo struct {
	domain.PaymentRepository
	lastFilter *domain.AdminPaymentFilter
	result     *domain.AdminPaymentListResult
	err        error
}

func (m *mockAdminPaymentRepo) FindAdminPayments(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) (*domain.AdminPaymentListResult, error) {
	m.lastFilter = &filter
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestNewAdminPaymentUsecase(t *testing.T) {
	t.Run("nil repository returns ErrNilPaymentRepository", func(t *testing.T) {
		uc, err := payment.NewAdminPaymentUsecase(nil, slog.Default())
		assert.Nil(t, uc)
		assert.ErrorIs(t, err, payment.ErrNilPaymentRepository)
	})

	t.Run("nil logger defaults to slog.Default without error", func(t *testing.T) {
		repo := &mockAdminPaymentRepo{}
		uc, err := payment.NewAdminPaymentUsecase(repo, nil)
		require.NoError(t, err)
		assert.NotNil(t, uc)
	})

	t.Run("valid parameters construct successfully", func(t *testing.T) {
		repo := &mockAdminPaymentRepo{}
		uc, err := payment.NewAdminPaymentUsecase(repo, slog.Default())
		require.NoError(t, err)
		assert.NotNil(t, uc)
	})
}

func TestAdminPaymentUsecase_ListPayments(t *testing.T) {
	ctx := context.Background()

	t.Run("normalizes pagination parameters", func(t *testing.T) {
		repo := &mockAdminPaymentRepo{
			result: &domain.AdminPaymentListResult{},
		}
		uc, err := payment.NewAdminPaymentUsecase(repo, slog.Default())
		require.NoError(t, err)

		// Page < 1 and Limit < 1 should default to 1 and 10
		_, err = uc.ListPayments(ctx, domain.AdminPaymentFilter{
			Page:  0,
			Limit: -5,
		})
		require.NoError(t, err)
		require.NotNil(t, repo.lastFilter)
		assert.Equal(t, 1, repo.lastFilter.Page)
		assert.Equal(t, 10, repo.lastFilter.Limit)

		// Limit > 100 should clamp to 100
		_, err = uc.ListPayments(ctx, domain.AdminPaymentFilter{
			Page:  2,
			Limit: 500,
		})
		require.NoError(t, err)
		require.NotNil(t, repo.lastFilter)
		assert.Equal(t, 2, repo.lastFilter.Page)
		assert.Equal(t, 100, repo.lastFilter.Limit)

		// Page > 100000 should clamp to 100000
		_, err = uc.ListPayments(ctx, domain.AdminPaymentFilter{
			Page:  200000,
			Limit: 20,
		})
		require.NoError(t, err)
		require.NotNil(t, repo.lastFilter)
		assert.Equal(t, 100000, repo.lastFilter.Page)
		assert.Equal(t, 20, repo.lastFilter.Limit)
	})

	t.Run("normalizes date parameters to UTC", func(t *testing.T) {
		repo := &mockAdminPaymentRepo{
			result: &domain.AdminPaymentListResult{},
		}
		uc, err := payment.NewAdminPaymentUsecase(repo, slog.Default())
		require.NoError(t, err)

		istZone := time.FixedZone("IST", 5*3600+1800)
		fromIST := time.Date(2026, 11, 1, 10, 30, 0, 0, istZone)
		toIST := time.Date(2026, 11, 10, 18, 30, 0, 0, istZone)

		_, err = uc.ListPayments(ctx, domain.AdminPaymentFilter{
			From: &fromIST,
			To:   &toIST,
		})
		require.NoError(t, err)
		require.NotNil(t, repo.lastFilter)
		require.NotNil(t, repo.lastFilter.From)
		require.NotNil(t, repo.lastFilter.To)
		assert.Equal(t, time.UTC, repo.lastFilter.From.Location())
		assert.Equal(t, time.UTC, repo.lastFilter.To.Location())
		assert.Equal(t, 5, repo.lastFilter.From.Hour())
		assert.Equal(t, 13, repo.lastFilter.To.Hour())
	})

	t.Run("normalizes filter strings", func(t *testing.T) {
		repo := &mockAdminPaymentRepo{
			result: &domain.AdminPaymentListResult{},
		}
		uc, err := payment.NewAdminPaymentUsecase(repo, slog.Default())
		require.NoError(t, err)

		_, err = uc.ListPayments(ctx, domain.AdminPaymentFilter{
			Page:             1,
			Limit:            20,
			Status:           "  success  ",
			AllocationStatus: "  accounted  ",
			Search:           "  john doe  ",
		})
		require.NoError(t, err)
		require.NotNil(t, repo.lastFilter)
		assert.Equal(t, "SUCCESS", repo.lastFilter.Status)
		assert.Equal(t, "ACCOUNTED", repo.lastFilter.AllocationStatus)
		assert.Equal(t, "john doe", repo.lastFilter.Search)
	})

	t.Run("validates allocation status values", func(t *testing.T) {
		repo := &mockAdminPaymentRepo{
			result: &domain.AdminPaymentListResult{},
		}
		uc, err := payment.NewAdminPaymentUsecase(repo, slog.Default())
		require.NoError(t, err)

		validStatuses := []string{
			"ACCOUNTED",
			"UNALLOCATED",
			"NOT_APPLICABLE",
			"ALL",
			"",
		}

		for _, status := range validStatuses {
			_, err := uc.ListPayments(ctx, domain.AdminPaymentFilter{
				AllocationStatus: status,
			})
			assert.NoError(t, err, "expected %q to be valid", status)
		}

		invalidStatuses := []string{
			"INVALID",
			"UNKNOWN",
			"SETTLED",
		}

		for _, status := range invalidStatuses {
			_, err := uc.ListPayments(ctx, domain.AdminPaymentFilter{
				AllocationStatus: status,
			})
			assert.ErrorIs(t, err, payment.ErrInvalidAllocationStatus, "expected %q to be rejected", status)
		}
	})

	t.Run("validates date range", func(t *testing.T) {
		repo := &mockAdminPaymentRepo{
			result: &domain.AdminPaymentListResult{},
		}
		uc, err := payment.NewAdminPaymentUsecase(repo, slog.Default())
		require.NoError(t, err)

		t1 := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

		// From after To should fail
		_, err = uc.ListPayments(ctx, domain.AdminPaymentFilter{
			From: &t1,
			To:   &t2,
		})
		assert.ErrorIs(t, err, payment.ErrInvalidDateRange)

		// From before To should succeed
		_, err = uc.ListPayments(ctx, domain.AdminPaymentFilter{
			From: &t2,
			To:   &t1,
		})
		assert.NoError(t, err)
	})

	t.Run("propagates repository result and errors", func(t *testing.T) {
		expectedErr := errors.New("db error")
		repo := &mockAdminPaymentRepo{
			err: expectedErr,
		}
		uc, err := payment.NewAdminPaymentUsecase(repo, slog.Default())
		require.NoError(t, err)

		res, err := uc.ListPayments(ctx, domain.AdminPaymentFilter{})
		assert.Nil(t, res)
		assert.ErrorIs(t, err, expectedErr)

		// Success case
		expectedResult := &domain.AdminPaymentListResult{
			Payments: []domain.AdminPaymentListItem{
				{
					ID:            uuid.New().String(),
					BookingID:     uuid.New().String(),
					BookingNumber: "BK-12345",
					Amount:        decimal.NewFromInt(100),
					Currency:      "INR",
					Status:        domain.PaymentStatusSuccess,
				},
			},
			TotalRecords: 1,
			Page:         1,
			Limit:        10,
			TotalPages:   1,
		}
		repo.err = nil
		repo.result = expectedResult

		actualResult, err := uc.ListPayments(ctx, domain.AdminPaymentFilter{})
		require.NoError(t, err)
		assert.Equal(t, expectedResult, actualResult)
	})
}
