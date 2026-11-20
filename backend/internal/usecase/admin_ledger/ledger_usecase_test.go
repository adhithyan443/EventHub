package adminledger_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	adminledger "github.com/adhithyan443/EventHub/backend/internal/usecase/admin_ledger"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLedgerRepository struct {
	listJournalsFn   func(ctx context.Context, filter domain.AdminLedgerFilter) (*domain.AdminLedgerListResult, error)
	getJournalByIDFn func(ctx context.Context, id uuid.UUID) (*domain.AdminLedgerJournalDetail, error)
}

func (m *mockLedgerRepository) ListJournals(ctx context.Context, filter domain.AdminLedgerFilter) (*domain.AdminLedgerListResult, error) {
	if m.listJournalsFn != nil {
		return m.listJournalsFn(ctx, filter)
	}
	return &domain.AdminLedgerListResult{}, nil
}

func (m *mockLedgerRepository) GetJournalByID(ctx context.Context, id uuid.UUID) (*domain.AdminLedgerJournalDetail, error) {
	if m.getJournalByIDFn != nil {
		return m.getJournalByIDFn(ctx, id)
	}
	return nil, domain.ErrLedgerJournalNotFound
}

func TestAdminLedgerUsecase_ListJournals(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("invalid date range returns ErrInvalidDateRange", func(t *testing.T) {
		repo := &mockLedgerRepository{}
		uc := adminledger.NewAdminLedgerUsecase(repo, logger)

		from := time.Now()
		to := from.Add(-24 * time.Hour)
		filter := domain.AdminLedgerFilter{From: &from, To: &to}

		result, err := uc.ListJournals(context.Background(), filter)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrInvalidDateRange))
	})

	t.Run("sanitizes default pagination", func(t *testing.T) {
		var capturedFilter domain.AdminLedgerFilter
		repo := &mockLedgerRepository{
			listJournalsFn: func(ctx context.Context, filter domain.AdminLedgerFilter) (*domain.AdminLedgerListResult, error) {
				capturedFilter = filter
				return &domain.AdminLedgerListResult{
					Page:         filter.Page,
					Limit:        filter.Limit,
					TotalRecords: 0,
					TotalPages:   1,
				}, nil
			},
		}
		uc := adminledger.NewAdminLedgerUsecase(repo, logger)

		filter := domain.AdminLedgerFilter{Page: -1, Limit: 500}
		res, err := uc.ListJournals(context.Background(), filter)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, 1, capturedFilter.Page)
		assert.Equal(t, 10, capturedFilter.Limit)
	})

	t.Run("returns result on success", func(t *testing.T) {
		repo := &mockLedgerRepository{
			listJournalsFn: func(ctx context.Context, filter domain.AdminLedgerFilter) (*domain.AdminLedgerListResult, error) {
				return &domain.AdminLedgerListResult{
					Journals: []domain.AdminLedgerJournalListItem{
						{
							ID:              uuid.New(),
							TransactionType: "PAYMENT",
							TotalDebit:      decimal.NewFromInt(100),
							TotalCredit:     decimal.NewFromInt(100),
							IsBalanced:      true,
						},
					},
					Page:         1,
					Limit:        10,
					TotalRecords: 1,
					TotalPages:   1,
				}, nil
			},
		}
		uc := adminledger.NewAdminLedgerUsecase(repo, logger)

		res, err := uc.ListJournals(context.Background(), domain.AdminLedgerFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.TotalRecords)
		assert.Len(t, res.Journals, 1)
		assert.True(t, res.Journals[0].IsBalanced)
	})
}

func TestAdminLedgerUsecase_GetJournal(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("uuid nil returns ErrLedgerJournalNotFound", func(t *testing.T) {
		repo := &mockLedgerRepository{}
		uc := adminledger.NewAdminLedgerUsecase(repo, logger)

		detail, err := uc.GetJournal(context.Background(), uuid.Nil)
		assert.Nil(t, detail)
		assert.True(t, errors.Is(err, domain.ErrLedgerJournalNotFound))
	})

	t.Run("returns journal detail with entries", func(t *testing.T) {
		journalID := uuid.New()
		repo := &mockLedgerRepository{
			getJournalByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.AdminLedgerJournalDetail, error) {
				return &domain.AdminLedgerJournalDetail{
					ID:              journalID,
					TransactionType: "PAYMENT",
					TotalDebit:      decimal.NewFromInt(500),
					TotalCredit:     decimal.NewFromInt(500),
					IsBalanced:      true,
					Entries: []domain.AdminLedgerEntryItem{
						{
							ID:          uuid.New(),
							AccountCode: "1010",
							AccountName: "Razorpay Clearing",
							EntryType:   "DEBIT",
							Amount:      decimal.NewFromInt(500),
						},
						{
							ID:          uuid.New(),
							AccountCode: "2010",
							AccountName: "Organizer Payable",
							EntryType:   "CREDIT",
							Amount:      decimal.NewFromInt(500),
						},
					},
				}, nil
			},
		}
		uc := adminledger.NewAdminLedgerUsecase(repo, logger)

		detail, err := uc.GetJournal(context.Background(), journalID)
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, journalID, detail.ID)
		assert.True(t, detail.IsBalanced)
		assert.Len(t, detail.Entries, 2)
	})
}
