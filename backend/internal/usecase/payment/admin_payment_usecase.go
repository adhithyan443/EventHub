package payment

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

var (
	ErrNilPaymentRepository    = errors.New("payment repository cannot be nil")
	ErrInvalidAllocationStatus = errors.New("invalid allocation status filter")
	ErrInvalidDateRange        = errors.New("from date cannot be after to date")
)

type AdminPaymentUsecase struct {
	paymentRepo domain.PaymentRepository
	logger      *slog.Logger
}

func NewAdminPaymentUsecase(
	paymentRepo domain.PaymentRepository,
	logger *slog.Logger,
) (*AdminPaymentUsecase, error) {
	if paymentRepo == nil {
		return nil, ErrNilPaymentRepository
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminPaymentUsecase{
		paymentRepo: paymentRepo,
		logger:      logger,
	}, nil
}

func (u *AdminPaymentUsecase) ListPayments(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) (*domain.AdminPaymentListResult, error) {
	const maxSafePage = 100_000
	if filter.Page < 1 {
		filter.Page = 1
	} else if filter.Page > maxSafePage {
		filter.Page = maxSafePage
	}

	if filter.Limit < 1 {
		filter.Limit = 10
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}

	filter.Status = strings.ToUpper(strings.TrimSpace(filter.Status))
	filter.AllocationStatus = strings.ToUpper(strings.TrimSpace(filter.AllocationStatus))
	filter.Search = strings.TrimSpace(filter.Search)

	if filter.AllocationStatus != "" && filter.AllocationStatus != "ALL" {
		switch domain.AllocationStatus(filter.AllocationStatus) {
		case domain.AllocationStatusAccounted,
			domain.AllocationStatusUnallocated,
			domain.AllocationStatusNotApplicable:
			// Valid
		default:
			return nil, ErrInvalidAllocationStatus
		}
	}

	if filter.From != nil {
		fromUTC := filter.From.UTC()
		filter.From = &fromUTC
	}
	if filter.To != nil {
		toUTC := filter.To.UTC()
		filter.To = &toUTC
	}

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, ErrInvalidDateRange
	}

	u.logger.InfoContext(ctx, "admin_payment_listing_requested",
		slog.Int("page", filter.Page),
		slog.Int("limit", filter.Limit),
		slog.String("status", filter.Status),
		slog.String("allocation_status", filter.AllocationStatus),
		slog.Bool("has_search", filter.Search != ""),
	)

	return u.paymentRepo.FindAdminPayments(ctx, filter)
}
