package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDashboardUsecase struct {
	getDashboardDataFn func(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminDashboardData, error)
}

func (m *mockDashboardUsecase) GetDashboardData(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminDashboardData, error) {
	if m.getDashboardDataFn != nil {
		return m.getDashboardDataFn(ctx, filter)
	}
	return &domain.AdminDashboardData{
		PlatformOverview: domain.AdminPlatformOverview{
			TotalUsers:      50,
			TotalOrganizers: 5,
			TotalEvents:     10,
			ActiveEvents:    3,
			TotalBookings:   25,
			TicketsSold:     60,
		},
		FinancialOverview: domain.AdminFinancialOverview{
			GrossCapturedAmount:       decimal.NewFromInt(10000),
			PlatformFeeRevenue:        decimal.NewFromInt(1000),
			OrganizerPayableLiability: decimal.NewFromInt(8000),
			TotalRefunds:              decimal.NewFromInt(200),
			CompletedSettlements:      decimal.NewFromInt(5000),
			PendingSettlements:        decimal.NewFromInt(1000),
			TaxLiability:              decimal.NewFromInt(800),
			Currency:                  "INR",
		},
		Trends: domain.AdminDashboardTrends{
			RevenueTrends: []domain.AdminRevenueTrendPoint{
				{Period: "2026-08", GrossVolume: decimal.NewFromInt(10000), PlatformFees: decimal.NewFromInt(1000)},
			},
			BookingTrends: []domain.AdminBookingTrendPoint{
				{Period: "2026-08", BookingCount: 25, TicketsSold: 60},
			},
			RefundTrends: []domain.AdminRefundTrendPoint{
				{Period: "2026-08", RefundAmount: decimal.NewFromInt(200), RefundCount: 1},
			},
		},
		PaymentStatusBreakdown: []domain.AdminPaymentStatusBreakdownItem{
			{Status: "SUCCESS", Count: 20, Amount: decimal.NewFromInt(10000)},
		},
		RecentBookings: []domain.AdminRecentBookingItem{
			{
				ID:            uuid.New(),
				BookingNumber: "BK-999",
				CustomerName:  "Bob Test",
				CustomerEmail: "bob@test.com",
				EventTitle:    "Go Conference",
				Amount:        decimal.NewFromInt(500),
				Status:        "CONFIRMED",
				PaymentStatus: "SUCCESS",
				CreatedAt:     time.Now().UTC(),
			},
		},
	}, nil
}

func setupDashboardTestRouter(h *handler.AdminDashboardHandler, logger *slog.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(logger))
	r.GET("/api/v1/admin/dashboard", h.GetDashboard)
	return r
}

func TestAdminDashboardHandler_GetDashboard_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := &mockDashboardUsecase{}
	h := handler.NewAdminDashboardHandler(uc, logger)
	r := setupDashboardTestRouter(h, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var res struct {
		Success bool                      `json:"success"`
		Data    domain.AdminDashboardData `json:"data"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.True(t, res.Success)
	assert.Equal(t, int64(50), res.Data.PlatformOverview.TotalUsers)
	assert.True(t, decimal.NewFromInt(10000).Equal(res.Data.FinancialOverview.GrossCapturedAmount))
	assert.Equal(t, "INR", res.Data.FinancialOverview.Currency)
	assert.Len(t, res.Data.RecentBookings, 1)
	assert.Equal(t, "BK-999", res.Data.RecentBookings[0].BookingNumber)
}

func TestAdminDashboardHandler_GetDashboard_DateRangeParams(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var receivedFilter domain.AdminDashboardFilter
	uc := &mockDashboardUsecase{
		getDashboardDataFn: func(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminDashboardData, error) {
			receivedFilter = filter
			return &domain.AdminDashboardData{}, nil
		},
	}
	h := handler.NewAdminDashboardHandler(uc, logger)
	r := setupDashboardTestRouter(h, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard?from=2026-08-01&to=2026-08-31", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, receivedFilter.From)
	require.NotNil(t, receivedFilter.To)
	assert.Equal(t, 2026, receivedFilter.From.Year())
	assert.Equal(t, time.Month(8), receivedFilter.From.Month())
	assert.Equal(t, 1, receivedFilter.From.Day())
	assert.Equal(t, 31, receivedFilter.To.Day())
	assert.Equal(t, 23, receivedFilter.To.Hour())
}

func TestAdminDashboardHandler_GetDashboard_InvalidDateFormat(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := &mockDashboardUsecase{}
	h := handler.NewAdminDashboardHandler(uc, logger)
	r := setupDashboardTestRouter(h, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard?from=bad-date", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid from date parameter")
}

func TestAdminDashboardHandler_GetDashboard_InvalidDateRange(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := &mockDashboardUsecase{
		getDashboardDataFn: func(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminDashboardData, error) {
			return nil, domain.ErrInvalidDateRange
		},
	}
	h := handler.NewAdminDashboardHandler(uc, logger)
	r := setupDashboardTestRouter(h, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard?from=2026-08-20&to=2026-08-10", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "from date cannot be after to date")
}

func TestAdminDashboardHandler_GetDashboard_InternalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := &mockDashboardUsecase{
		getDashboardDataFn: func(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminDashboardData, error) {
			return nil, errors.New("database unreachable")
		},
	}
	h := handler.NewAdminDashboardHandler(uc, logger)
	r := setupDashboardTestRouter(h, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to retrieve dashboard data")
}
