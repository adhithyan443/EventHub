package admindashboard

import (
	"context"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

type AdminDashboardUsecase struct {
	repo   domain.AdminDashboardRepository
	logger *slog.Logger
}

// NewAdminDashboardUsecase creates a new usecase instance for admin dashboard analytics.
func NewAdminDashboardUsecase(
	repo domain.AdminDashboardRepository,
	logger *slog.Logger,
) *AdminDashboardUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminDashboardUsecase{
		repo:   repo,
		logger: logger,
	}
}

// GetDashboardData aggregates platform metrics, financial ledger balances, trends, breakdowns, and recent bookings.
func (u *AdminDashboardUsecase) GetDashboardData(
	ctx context.Context,
	filter domain.AdminDashboardFilter,
) (*domain.AdminDashboardData, error) {
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		u.logger.WarnContext(ctx, "invalid dashboard date range filter: from date after to date")
		return nil, domain.ErrInvalidDateRange
	}

	platformOverview, err := u.repo.GetPlatformOverview(ctx)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get platform overview", slog.Any("error", err))
		return nil, err
	}

	financialOverview, err := u.repo.GetFinancialOverview(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get financial overview", slog.Any("error", err))
		return nil, err
	}

	trends, err := u.repo.GetTrends(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get trends", slog.Any("error", err))
		return nil, err
	}

	statusBreakdown, err := u.repo.GetPaymentStatusBreakdown(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get payment status breakdown", slog.Any("error", err))
		return nil, err
	}

	recentBookings, err := u.repo.GetRecentBookings(ctx, 5)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get recent bookings", slog.Any("error", err))
		return nil, err
	}

	u.logger.InfoContext(ctx, "admin dashboard data retrieved successfully",
		slog.Int64("total_users", platformOverview.TotalUsers),
		slog.Int64("total_events", platformOverview.TotalEvents),
		slog.String("gross_captured", financialOverview.GrossCapturedAmount.String()),
		slog.String("platform_fee_revenue", financialOverview.PlatformFeeRevenue.String()),
		slog.String("organizer_payable", financialOverview.OrganizerPayableLiability.String()),
	)

	return &domain.AdminDashboardData{
		PlatformOverview:       *platformOverview,
		FinancialOverview:      *financialOverview,
		Trends:                 *trends,
		PaymentStatusBreakdown: statusBreakdown,
		RecentBookings:         recentBookings,
	}, nil
}
