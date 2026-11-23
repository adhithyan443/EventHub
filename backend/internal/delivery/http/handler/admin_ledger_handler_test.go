package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLedgerUsecase struct {
	listJournalsFn         func(ctx context.Context, filter domain.AdminLedgerFilter) (*domain.AdminLedgerListResult, error)
	getJournalFn           func(ctx context.Context, id uuid.UUID) (*domain.AdminLedgerJournalDetail, error)
	getFinancialOverviewFn func(ctx context.Context, filter domain.AdminFinancialFilter) (*domain.AdminFinancialOverviewResponse, error)
}

func (m *mockLedgerUsecase) ListJournals(ctx context.Context, filter domain.AdminLedgerFilter) (*domain.AdminLedgerListResult, error) {
	if m.listJournalsFn != nil {
		return m.listJournalsFn(ctx, filter)
	}
	return &domain.AdminLedgerListResult{
		Journals:     []domain.AdminLedgerJournalListItem{},
		Page:         1,
		Limit:        10,
		TotalRecords: 0,
		TotalPages:   1,
	}, nil
}

func (m *mockLedgerUsecase) GetJournal(ctx context.Context, id uuid.UUID) (*domain.AdminLedgerJournalDetail, error) {
	if m.getJournalFn != nil {
		return m.getJournalFn(ctx, id)
	}
	return nil, domain.ErrLedgerJournalNotFound
}

func (m *mockLedgerUsecase) GetFinancialOverview(ctx context.Context, filter domain.AdminFinancialFilter) (*domain.AdminFinancialOverviewResponse, error) {
	if m.getFinancialOverviewFn != nil {
		return m.getFinancialOverviewFn(ctx, filter)
	}
	return &domain.AdminFinancialOverviewResponse{}, nil
}

func setupAdminLedgerTestRouter(h *handler.AdminLedgerHandler, logger *slog.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler(logger))

	admin := router.Group("/api/v1/admin/ledger")
	{
		admin.GET("/overview", h.GetFinancialOverview)
		admin.GET("/journals", h.ListJournals)
		admin.GET("/journals/:id", h.GetJournal)
	}
	return router
}

func TestAdminLedgerHandler_ListJournals(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("returns journals list on success", func(t *testing.T) {
		journalID := uuid.New()
		usecase := &mockLedgerUsecase{
			listJournalsFn: func(ctx context.Context, filter domain.AdminLedgerFilter) (*domain.AdminLedgerListResult, error) {
				return &domain.AdminLedgerListResult{
					Journals: []domain.AdminLedgerJournalListItem{
						{
							ID:              journalID,
							TransactionType: "PAYMENT",
							TotalDebit:      decimal.NewFromInt(1000),
							TotalCredit:     decimal.NewFromInt(1000),
							IsBalanced:      true,
							Status:          "POSTED",
						},
					},
					Page:         1,
					Limit:        10,
					TotalRecords: 1,
					TotalPages:   1,
				}, nil
			},
		}

		h := handler.NewAdminLedgerHandler(usecase, logger)
		router := setupAdminLedgerTestRouter(h, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ledger/journals?page=1&limit=10", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Journals []struct {
					ID              uuid.UUID `json:"id"`
					TransactionType string    `json:"transaction_type"`
					IsBalanced      bool      `json:"is_balanced"`
				} `json:"journals"`
				TotalRecords int64 `json:"total_records"`
			} `json:"data"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int64(1), resp.Data.TotalRecords)
		assert.Equal(t, journalID, resp.Data.Journals[0].ID)
	})

	t.Run("rejects invalid page", func(t *testing.T) {
		usecase := &mockLedgerUsecase{}
		h := handler.NewAdminLedgerHandler(usecase, logger)
		router := setupAdminLedgerTestRouter(h, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ledger/journals?page=0", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects invalid date range", func(t *testing.T) {
		usecase := &mockLedgerUsecase{
			listJournalsFn: func(ctx context.Context, filter domain.AdminLedgerFilter) (*domain.AdminLedgerListResult, error) {
				return nil, domain.ErrInvalidDateRange
			},
		}
		h := handler.NewAdminLedgerHandler(usecase, logger)
		router := setupAdminLedgerTestRouter(h, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ledger/journals?from=2026-10-10&to=2026-10-01", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestAdminLedgerHandler_GetJournal(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("returns journal detail on success", func(t *testing.T) {
		journalID := uuid.New()
		usecase := &mockLedgerUsecase{
			getJournalFn: func(ctx context.Context, id uuid.UUID) (*domain.AdminLedgerJournalDetail, error) {
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
					},
				}, nil
			},
		}

		h := handler.NewAdminLedgerHandler(usecase, logger)
		router := setupAdminLedgerTestRouter(h, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ledger/journals/"+journalID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				ID         uuid.UUID `json:"id"`
				IsBalanced bool      `json:"is_balanced"`
				Entries    []struct {
					AccountCode string `json:"account_code"`
				} `json:"entries"`
			} `json:"data"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, journalID, resp.Data.ID)
		assert.True(t, resp.Data.IsBalanced)
		assert.Len(t, resp.Data.Entries, 1)
	})

	t.Run("returns 404 for unknown journal", func(t *testing.T) {
		usecase := &mockLedgerUsecase{
			getJournalFn: func(ctx context.Context, id uuid.UUID) (*domain.AdminLedgerJournalDetail, error) {
				return nil, domain.ErrLedgerJournalNotFound
			},
		}

		h := handler.NewAdminLedgerHandler(usecase, logger)
		router := setupAdminLedgerTestRouter(h, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ledger/journals/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		usecase := &mockLedgerUsecase{}
		h := handler.NewAdminLedgerHandler(usecase, logger)
		router := setupAdminLedgerTestRouter(h, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ledger/journals/not-a-uuid", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestAdminLedgerHandler_GetFinancialOverview(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("returns 200 with financial overview data", func(t *testing.T) {
		eventID := uuid.New()
		organizerID := uuid.New()

		usecase := &mockLedgerUsecase{
			getFinancialOverviewFn: func(ctx context.Context, filter domain.AdminFinancialFilter) (*domain.AdminFinancialOverviewResponse, error) {
				return &domain.AdminFinancialOverviewResponse{
					Metrics: domain.AdminFinancialMetrics{
						TotalGrossCollections:    decimal.NewFromInt(50000),
						TotalRefunds:             decimal.NewFromInt(2000),
						NetCollections:           decimal.NewFromInt(48000),
						PlatformRevenue:          decimal.NewFromInt(5000),
						TaxLiability:             decimal.NewFromInt(900),
						OrganizerEarnings:        decimal.NewFromInt(42100),
						OrganizerPayouts:         decimal.NewFromInt(30000),
						PendingSettlements:       decimal.NewFromInt(5000),
						PaymentGatewayFees:       decimal.NewFromInt(1000),
						AvailablePlatformBalance: decimal.NewFromInt(4000),
						Currency:                 "INR",
					},
					EventBreakdown: []domain.AdminFinancialEventBreakdownItem{
						{
							EventID:          eventID,
							EventTitle:       "Tech Summit 2026",
							GrossCollections: decimal.NewFromInt(50000),
							NetCollections:   decimal.NewFromInt(48000),
						},
					},
					OrganizerBreakdown: []domain.AdminFinancialOrganizerBreakdownItem{
						{
							OrganizerID:      organizerID,
							OrganizerName:    "Global Tech Org",
							GrossCollections: decimal.NewFromInt(50000),
							NetEarnings:      decimal.NewFromInt(42100),
						},
					},
				}, nil
			},
		}

		h := handler.NewAdminLedgerHandler(usecase, logger)
		router := setupAdminLedgerTestRouter(h, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ledger/overview?from=2026-01-01&to=2026-12-31", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp struct {
			Success bool                                  `json:"success"`
			Data    domain.AdminFinancialOverviewResponse `json:"data"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.True(t, resp.Data.Metrics.TotalGrossCollections.Equal(decimal.NewFromInt(50000)))
		assert.True(t, resp.Data.Metrics.AvailablePlatformBalance.Equal(decimal.NewFromInt(4000)))
		assert.Len(t, resp.Data.EventBreakdown, 1)
		assert.Len(t, resp.Data.OrganizerBreakdown, 1)
	})

	t.Run("returns 400 for invalid date format", func(t *testing.T) {
		usecase := &mockLedgerUsecase{}
		h := handler.NewAdminLedgerHandler(usecase, logger)
		router := setupAdminLedgerTestRouter(h, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ledger/overview?from=not-a-date", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 400 for invalid event UUID", func(t *testing.T) {
		usecase := &mockLedgerUsecase{}
		h := handler.NewAdminLedgerHandler(usecase, logger)
		router := setupAdminLedgerTestRouter(h, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ledger/overview?event_id=not-a-uuid", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
