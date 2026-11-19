package payment

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrNilOrganizerRepository = errors.New("organizer repository cannot be nil")
	ErrUnauthorizedOrganizer  = errors.New("organizer access required or account not active")
	ErrInvalidUserID          = errors.New("invalid or empty user ID")
	ErrInvalidPaymentStatus   = errors.New("invalid payment status filter")
	ErrInvalidEventID         = errors.New("invalid event ID")
	ErrUnauthorizedEvent      = errors.New("event does not belong to authenticated organizer")
	ErrEventNotFound          = errors.New("event not found")
)

type OrganizerPaymentUsecase struct {
	paymentRepo   domain.PaymentRepository
	organizerRepo domain.OrganizerRepository
	eventRepo     domain.EventRepository
	logger        *slog.Logger
}

func NewOrganizerPaymentUsecase(
	paymentRepo domain.PaymentRepository,
	organizerRepo domain.OrganizerRepository,
	eventRepo domain.EventRepository,
	logger *slog.Logger,
) (*OrganizerPaymentUsecase, error) {
	if paymentRepo == nil {
		return nil, ErrNilPaymentRepository
	}
	if organizerRepo == nil {
		return nil, ErrNilOrganizerRepository
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &OrganizerPaymentUsecase{
		paymentRepo:   paymentRepo,
		organizerRepo: organizerRepo,
		eventRepo:     eventRepo,
		logger:        logger,
	}, nil
}

func (u *OrganizerPaymentUsecase) resolveAndValidateOrganizer(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.Organizer, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}

	organizer, err := u.organizerRepo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerNotFound) {
			u.logger.WarnContext(ctx, "organizer payment access denied: organizer profile not found",
				slog.String("user_id", userID.String()),
			)
			return nil, ErrUnauthorizedOrganizer
		}
		u.logger.ErrorContext(ctx, "failed to lookup organizer by user ID",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	status := strings.ToUpper(strings.TrimSpace(organizer.Status))
	if status != "ACTIVE" && status != "APPROVED" {
		u.logger.WarnContext(ctx, "organizer payment access denied: unapproved or inactive organizer status",
			slog.String("user_id", userID.String()),
			slog.String("organizer_id", organizer.ID.String()),
			slog.String("status", organizer.Status),
		)
		return nil, ErrUnauthorizedOrganizer
	}

	return organizer, nil
}

func (u *OrganizerPaymentUsecase) validateAndNormalizeFilter(
	ctx context.Context,
	organizerID uuid.UUID,
	filter *domain.OrganizerPaymentFilter,
) error {
	// Strictly scope the query to the authenticated organizer ID.
	// Never accept organizer ID from frontend.
	filter.OrganizerID = organizerID

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
	if filter.Status != "" && filter.Status != "ALL" {
		switch domain.PaymentStatus(filter.Status) {
		case domain.PaymentStatusPending,
			domain.PaymentStatusSuccess,
			domain.PaymentStatusFailed,
			domain.PaymentStatusRefundRequired:
			// Valid
		default:
			return ErrInvalidPaymentStatus
		}
	}

	filter.AllocationStatus = strings.ToUpper(strings.TrimSpace(filter.AllocationStatus))
	if filter.AllocationStatus != "" && filter.AllocationStatus != "ALL" {
		switch domain.AllocationStatus(filter.AllocationStatus) {
		case domain.AllocationStatusAccounted,
			domain.AllocationStatusUnallocated,
			domain.AllocationStatusNotApplicable:
			// Valid
		default:
			return ErrInvalidAllocationStatus
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
		return ErrInvalidDateRange
	}

	filter.Search = strings.TrimSpace(filter.Search)

	// Validate event filtering ownership to prevent cross-organizer leakage
	if filter.EventID != nil {
		if *filter.EventID == uuid.Nil {
			return ErrInvalidEventID
		}
		if u.eventRepo != nil {
			event, err := u.eventRepo.FindByID(*filter.EventID)
			if err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					return ErrEventNotFound
				}
				u.logger.ErrorContext(ctx, "failed to lookup event for organizer payment filter",
					slog.String("event_id", filter.EventID.String()),
					slog.Any("error", err),
				)
				return err
			}
			if event.OrganizerID != organizerID {
				u.logger.WarnContext(ctx, "security: organizer attempted to filter payments by foreign event",
					slog.String("authenticated_organizer_id", organizerID.String()),
					slog.String("requested_event_id", filter.EventID.String()),
					slog.String("actual_event_owner_id", event.OrganizerID.String()),
				)
				return ErrUnauthorizedEvent
			}
		}
	}

	return nil
}

func (u *OrganizerPaymentUsecase) GetOrganizerPaymentOverview(
	ctx context.Context,
	userID uuid.UUID,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerPaymentOverview, error) {
	organizer, err := u.resolveAndValidateOrganizer(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := u.validateAndNormalizeFilter(ctx, organizer.ID, &filter); err != nil {
		return nil, err
	}

	u.logger.InfoContext(ctx, "organizer_payment_overview_requested",
		slog.String("organizer_id", organizer.ID.String()),
		slog.String("status", filter.Status),
		slog.String("allocation_status", filter.AllocationStatus),
		slog.Bool("has_search", filter.Search != ""),
		slog.Bool("has_event_id", filter.EventID != nil),
	)

	return u.paymentRepo.GetOrganizerPaymentOverview(ctx, filter)
}

func (u *OrganizerPaymentUsecase) GetOrganizerEventPaymentSummaries(
	ctx context.Context,
	userID uuid.UUID,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerEventPaymentSummaryResult, error) {
	organizer, err := u.resolveAndValidateOrganizer(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := u.validateAndNormalizeFilter(ctx, organizer.ID, &filter); err != nil {
		return nil, err
	}

	u.logger.InfoContext(ctx, "organizer_event_payment_summaries_requested",
		slog.String("organizer_id", organizer.ID.String()),
		slog.Int("page", filter.Page),
		slog.Int("limit", filter.Limit),
		slog.String("status", filter.Status),
		slog.String("allocation_status", filter.AllocationStatus),
		slog.Bool("has_search", filter.Search != ""),
		slog.Bool("has_event_id", filter.EventID != nil),
	)

	return u.paymentRepo.GetOrganizerEventPaymentSummaries(ctx, filter)
}

func (u *OrganizerPaymentUsecase) FindOrganizerPayments(
	ctx context.Context,
	userID uuid.UUID,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerPaymentListResult, error) {
	organizer, err := u.resolveAndValidateOrganizer(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := u.validateAndNormalizeFilter(ctx, organizer.ID, &filter); err != nil {
		return nil, err
	}

	u.logger.InfoContext(ctx, "organizer_payment_listing_requested",
		slog.String("organizer_id", organizer.ID.String()),
		slog.Int("page", filter.Page),
		slog.Int("limit", filter.Limit),
		slog.String("status", filter.Status),
		slog.String("allocation_status", filter.AllocationStatus),
		slog.Bool("has_search", filter.Search != ""),
		slog.Bool("has_event_id", filter.EventID != nil),
	)

	return u.paymentRepo.FindOrganizerPayments(ctx, filter)
}
