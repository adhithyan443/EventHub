package adminledger

import (
	"context"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type AdminLedgerUsecase struct {
	repo   domain.AdminLedgerRepository
	logger *slog.Logger
}

// NewAdminLedgerUsecase creates a new instance of AdminLedgerUsecase.
func NewAdminLedgerUsecase(
	repo domain.AdminLedgerRepository,
	logger *slog.Logger,
) *AdminLedgerUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminLedgerUsecase{
		repo:   repo,
		logger: logger,
	}
}

// ListJournals validates filtering parameters and fetches paginated journals.
func (u *AdminLedgerUsecase) ListJournals(
	ctx context.Context,
	filter domain.AdminLedgerFilter,
) (*domain.AdminLedgerListResult, error) {
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		u.logger.WarnContext(ctx, "invalid ledger date range: from after to")
		return nil, domain.ErrInvalidDateRange
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}

	result, err := u.repo.ListJournals(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to list ledger journals", slog.Any("error", err))
		return nil, err
	}

	u.logger.InfoContext(ctx, "admin ledger journals listed successfully",
		slog.Int("page", result.Page),
		slog.Int("limit", result.Limit),
		slog.Int64("total_records", result.TotalRecords),
		slog.Int("total_pages", result.TotalPages),
	)

	return result, nil
}

// GetJournal fetches full journal details including balanced entries by UUID.
func (u *AdminLedgerUsecase) GetJournal(
	ctx context.Context,
	id uuid.UUID,
) (*domain.AdminLedgerJournalDetail, error) {
	if id == uuid.Nil {
		return nil, domain.ErrLedgerJournalNotFound
	}

	detail, err := u.repo.GetJournalByID(ctx, id)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get ledger journal detail",
			slog.String("journal_id", id.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "admin ledger journal retrieved successfully",
		slog.String("journal_id", id.String()),
		slog.String("transaction_type", detail.TransactionType),
		slog.Bool("is_balanced", detail.IsBalanced),
		slog.Int("entries_count", len(detail.Entries)),
	)

	return detail, nil
}

// GetFinancialOverview retrieves the consolidated platform-wide financial overview and breakdowns.
func (u *AdminLedgerUsecase) GetFinancialOverview(
	ctx context.Context,
	filter domain.AdminFinancialFilter,
) (*domain.AdminFinancialOverviewResponse, error) {
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		u.logger.WarnContext(ctx, "invalid financial date range: from after to")
		return nil, domain.ErrInvalidDateRange
	}

	overview, err := u.repo.GetFinancialOverview(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get admin financial overview", slog.Any("error", err))
		return nil, err
	}

	u.logger.InfoContext(ctx, "admin financial overview retrieved successfully",
		slog.String("gross_collections", overview.Metrics.TotalGrossCollections.String()),
		slog.String("net_collections", overview.Metrics.NetCollections.String()),
		slog.String("platform_revenue", overview.Metrics.PlatformRevenue.String()),
		slog.String("tax_liability", overview.Metrics.TaxLiability.String()),
		slog.String("organizer_earnings", overview.Metrics.OrganizerEarnings.String()),
	)

	return overview, nil
}
