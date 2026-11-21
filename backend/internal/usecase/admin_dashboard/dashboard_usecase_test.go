package admindashboard_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	admindashboard "github.com/adhithyan443/EventHub/backend/internal/usecase/admin_dashboard"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDashboardRepo struct {
	platformOverviewFn       func(ctx context.Context) (*domain.AdminPlatformOverview, error)
	financialOverviewFn      func(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminFinancialOverview, error)
	trendsFn                 func(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminDashboardTrends, error)
	paymentStatusBreakdownFn func(ctx context.Context, filter domain.AdminDashboardFilter) ([]domain.AdminPaymentStatusBreakdownItem, error)
	recentBookingsFn         func(ctx context.Context, limit int) ([]domain.AdminRecentBookingItem, error)
}

func (m *mockDashboardRepo) GetPlatformOverview(ctx context.Context) (*domain.AdminPlatformOverview, error) {
	if m.platformOverviewFn != nil {
		return m.platformOverviewFn(ctx)
	}
	return &domain.AdminPlatformOverview{
		TotalUsers:      100,
		TotalOrganizers: 10,
		TotalEvents:     20,
		ActiveEvents:    5,
		TotalBookings:   50,
		TicketsSold:     120,
	}, nil
}

func (m *mockDashboardRepo) GetFinancialOverview(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminFinancialOverview, error) {
	if m.financialOverviewFn != nil {
		return m.financialOverviewFn(ctx, filter)
	}
	return &domain.AdminFinancialOverview{
		GrossCapturedAmount:       decimal.NewFromInt(15000),
		PlatformFeeRevenue:        decimal.NewFromInt(1500),
		OrganizerPayableLiability: decimal.NewFromInt(12000),
		TotalRefunds:              decimal.NewFromInt(500),
		CompletedSettlements:      decimal.NewFromInt(8000),
		PendingSettlements:        decimal.NewFromInt(2000),
		TaxLiability:              decimal.NewFromInt(1000),
		Currency:                  "INR",
	}, nil
}

func (m *mockDashboardRepo) GetTrends(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminDashboardTrends, error) {
	if m.trendsFn != nil {
		return m.trendsFn(ctx, filter)
	}
	return &domain.AdminDashboardTrends{
		RevenueTrends: []domain.AdminRevenueTrendPoint{
			{Period: "2026-08", GrossVolume: decimal.NewFromInt(15000), PlatformFees: decimal.NewFromInt(1500)},
		},
		BookingTrends: []domain.AdminBookingTrendPoint{
			{Period: "2026-08", BookingCount: 50, TicketsSold: 120},
		},
		RefundTrends: []domain.AdminRefundTrendPoint{
			{Period: "2026-08", RefundAmount: decimal.NewFromInt(500), RefundCount: 2},
		},
	}, nil
}

func (m *mockDashboardRepo) GetPaymentStatusBreakdown(ctx context.Context, filter domain.AdminDashboardFilter) ([]domain.AdminPaymentStatusBreakdownItem, error) {
	if m.paymentStatusBreakdownFn != nil {
		return m.paymentStatusBreakdownFn(ctx, filter)
	}
	return []domain.AdminPaymentStatusBreakdownItem{
		{Status: "SUCCESS", Count: 45, Amount: decimal.NewFromInt(15000)},
		{Status: "PENDING", Count: 3, Amount: decimal.NewFromInt(600)},
		{Status: "FAILED", Count: 2, Amount: decimal.NewFromInt(400)},
		{Status: "REFUND_REQUIRED", Count: 0, Amount: decimal.Zero},
	}, nil
}

func (m *mockDashboardRepo) GetRecentBookings(ctx context.Context, limit int) ([]domain.AdminRecentBookingItem, error) {
	if m.recentBookingsFn != nil {
		return m.recentBookingsFn(ctx, limit)
	}
	return []domain.AdminRecentBookingItem{
		{
			ID:            uuid.New(),
			BookingNumber: "BK-1001",
			CustomerID:    uuid.New(),
			CustomerName:  "Alice Doe",
			CustomerEmail: "alice@example.com",
			EventID:       uuid.New(),
			EventTitle:    "Tech Summit 2026",
			Amount:        decimal.NewFromInt(500),
			Status:        "CONFIRMED",
			PaymentStatus: "SUCCESS",
			CreatedAt:     time.Now().UTC(),
		},
	}, nil
}

func TestAdminDashboardUsecase_Success(t *testing.T) {
	repo := &mockDashboardRepo{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := admindashboard.NewAdminDashboardUsecase(repo, logger)

	data, err := uc.GetDashboardData(context.Background(), domain.AdminDashboardFilter{})
	require.NoError(t, err)
	require.NotNil(t, data)

	// Platform overview checks
	assert.Equal(t, int64(100), data.PlatformOverview.TotalUsers)
	assert.Equal(t, int64(10), data.PlatformOverview.TotalOrganizers)
	assert.Equal(t, int64(20), data.PlatformOverview.TotalEvents)
	assert.Equal(t, int64(5), data.PlatformOverview.ActiveEvents)
	assert.Equal(t, int64(50), data.PlatformOverview.TotalBookings)
	assert.Equal(t, int64(120), data.PlatformOverview.TicketsSold)

	// Financial overview checks
	assert.True(t, decimal.NewFromInt(15000).Equal(data.FinancialOverview.GrossCapturedAmount))
	assert.True(t, decimal.NewFromInt(1500).Equal(data.FinancialOverview.PlatformFeeRevenue))
	assert.True(t, decimal.NewFromInt(12000).Equal(data.FinancialOverview.OrganizerPayableLiability))
	assert.True(t, decimal.NewFromInt(500).Equal(data.FinancialOverview.TotalRefunds))
	assert.True(t, decimal.NewFromInt(8000).Equal(data.FinancialOverview.CompletedSettlements))
	assert.True(t, decimal.NewFromInt(2000).Equal(data.FinancialOverview.PendingSettlements))
	assert.True(t, decimal.NewFromInt(1000).Equal(data.FinancialOverview.TaxLiability))
	assert.Equal(t, "INR", data.FinancialOverview.Currency)

	// Trends checks
	assert.Len(t, data.Trends.RevenueTrends, 1)
	assert.Equal(t, "2026-08", data.Trends.RevenueTrends[0].Period)
	assert.Len(t, data.Trends.BookingTrends, 1)
	assert.Len(t, data.Trends.RefundTrends, 1)

	// Payment status breakdown checks
	assert.Len(t, data.PaymentStatusBreakdown, 4)
	assert.Equal(t, "SUCCESS", data.PaymentStatusBreakdown[0].Status)

	// Recent bookings checks
	assert.Len(t, data.RecentBookings, 1)
	assert.Equal(t, "BK-1001", data.RecentBookings[0].BookingNumber)
	assert.Equal(t, "Alice Doe", data.RecentBookings[0].CustomerName)
}

func TestAdminDashboardUsecase_InvalidDateRange(t *testing.T) {
	repo := &mockDashboardRepo{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := admindashboard.NewAdminDashboardUsecase(repo, logger)

	from := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)

	data, err := uc.GetDashboardData(context.Background(), domain.AdminDashboardFilter{
		From: &from,
		To:   &to,
	})
	assert.ErrorIs(t, err, domain.ErrInvalidDateRange)
	assert.Nil(t, data)
}

func TestAdminDashboardUsecase_PlatformOverviewError(t *testing.T) {
	repo := &mockDashboardRepo{
		platformOverviewFn: func(ctx context.Context) (*domain.AdminPlatformOverview, error) {
			return nil, errors.New("db connection failure")
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := admindashboard.NewAdminDashboardUsecase(repo, logger)

	data, err := uc.GetDashboardData(context.Background(), domain.AdminDashboardFilter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failure")
	assert.Nil(t, data)
}

func TestAdminDashboardUsecase_FinancialOverviewError(t *testing.T) {
	repo := &mockDashboardRepo{
		financialOverviewFn: func(ctx context.Context, filter domain.AdminDashboardFilter) (*domain.AdminFinancialOverview, error) {
			return nil, errors.New("ledger table corrupt")
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := admindashboard.NewAdminDashboardUsecase(repo, logger)

	data, err := uc.GetDashboardData(context.Background(), domain.AdminDashboardFilter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ledger table corrupt")
	assert.Nil(t, data)
}
