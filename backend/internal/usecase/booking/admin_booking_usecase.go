package booking

import (
	"context"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

// AdminBookingUsecase provides business logic for platform-wide administrator order and attendee operations.
type AdminBookingUsecase struct {
	bookingRepo   domain.BookingRepository
	ticketRepo    domain.TicketRepository
	organizerRepo domain.OrganizerRepository
	eventRepo     domain.EventRepository
	logger        *slog.Logger
}

// NewAdminBookingUsecase constructs a new AdminBookingUsecase instance.
func NewAdminBookingUsecase(
	bookingRepo domain.BookingRepository,
	ticketRepo domain.TicketRepository,
	organizerRepo domain.OrganizerRepository,
	eventRepo domain.EventRepository,
	logger *slog.Logger,
) (*AdminBookingUsecase, error) {
	if bookingRepo == nil {
		return nil, ErrNilBookingRepository
	}
	if ticketRepo == nil {
		return nil, ErrNilTicketRepository
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminBookingUsecase{
		bookingRepo:   bookingRepo,
		ticketRepo:    ticketRepo,
		organizerRepo: organizerRepo,
		eventRepo:     eventRepo,
		logger:        logger,
	}, nil
}

// ListBookings validates filter criteria and retrieves a paginated platform-wide booking collection.
func (u *AdminBookingUsecase) ListBookings(
	ctx context.Context,
	filter domain.AdminBookingFilter,
) (*domain.AdminBookingListResult, error) {
	reqID := extractRequestID(ctx)

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		u.logger.WarnContext(ctx, "invalid booking date range filter",
			slog.String("request_id", reqID),
			slog.Time("from", *filter.From),
			slog.Time("to", *filter.To),
		)
		return nil, ErrInvalidDateRange
	}

	if filter.BookingStatus != "" && !strings.EqualFold(filter.BookingStatus, "ALL") {
		validStatuses := map[string]bool{
			"CONFIRMED":           true,
			"PENDING":             true,
			"CANCELLED":           true,
			"PARTIALLY_CANCELLED": true,
		}
		if !validStatuses[strings.ToUpper(strings.TrimSpace(filter.BookingStatus))] {
			return nil, ErrInvalidBookingStatus
		}
	}

	if filter.PaymentStatus != "" && !strings.EqualFold(filter.PaymentStatus, "ALL") {
		validPaymentStatuses := map[string]bool{
			"SUCCESS":        true,
			"PENDING":        true,
			"FAILED":         true,
			"REFUNDED":       true,
			"FULL":           true,
			"PARTIAL":        true,
			"PENDING_REFUND": true,
		}
		if !validPaymentStatuses[strings.ToUpper(strings.TrimSpace(filter.PaymentStatus))] {
			return nil, ErrInvalidPaymentStatus
		}
	}

	if filter.Page < 1 || filter.Limit < 1 || filter.Limit > 100 {
		return nil, ErrInvalidPagination
	}

	res, err := u.bookingRepo.FindAdminBookings(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to fetch admin bookings",
			slog.String("request_id", reqID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return res, nil
}

// GetBookingStats calculates aggregate financial and booking metrics platform-wide.
func (u *AdminBookingUsecase) GetBookingStats(
	ctx context.Context,
	filter domain.AdminBookingFilter,
) (*domain.AdminBookingStats, error) {
	reqID := extractRequestID(ctx)

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, ErrInvalidDateRange
	}

	if filter.BookingStatus != "" && !strings.EqualFold(filter.BookingStatus, "ALL") {
		validStatuses := map[string]bool{
			"CONFIRMED":           true,
			"PENDING":             true,
			"CANCELLED":           true,
			"PARTIALLY_CANCELLED": true,
		}
		if !validStatuses[strings.ToUpper(strings.TrimSpace(filter.BookingStatus))] {
			return nil, ErrInvalidBookingStatus
		}
	}

	if filter.PaymentStatus != "" && !strings.EqualFold(filter.PaymentStatus, "ALL") {
		validPaymentStatuses := map[string]bool{
			"SUCCESS":        true,
			"PENDING":        true,
			"FAILED":         true,
			"REFUNDED":       true,
			"FULL":           true,
			"PARTIAL":        true,
			"PENDING_REFUND": true,
		}
		if !validPaymentStatuses[strings.ToUpper(strings.TrimSpace(filter.PaymentStatus))] {
			return nil, ErrInvalidPaymentStatus
		}
	}

	stats, err := u.bookingRepo.GetAdminBookingStats(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to calculate admin booking stats",
			slog.String("request_id", reqID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return stats, nil
}

// GetBookingDetail retrieves the complete order breakdown for any booking across the platform.
func (u *AdminBookingUsecase) GetBookingDetail(
	ctx context.Context,
	bookingID uuid.UUID,
) (*domain.AdminBookingDetail, error) {
	reqID := extractRequestID(ctx)

	if bookingID == uuid.Nil {
		return nil, ErrInvalidBookingID
	}

	detail, err := u.bookingRepo.FindAdminBookingDetail(ctx, bookingID)
	if err != nil {
		u.logger.WarnContext(ctx, "failed to fetch admin booking detail",
			slog.String("request_id", reqID),
			slog.String("booking_id", bookingID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	return detail, nil
}

// ListAttendees retrieves a paginated platform-wide list of ticket admission passes.
func (u *AdminBookingUsecase) ListAttendees(
	ctx context.Context,
	filter domain.AdminAttendeeFilter,
) (*domain.AdminAttendeeListResult, error) {
	reqID := extractRequestID(ctx)

	if filter.CheckInStatus != "" && !strings.EqualFold(filter.CheckInStatus, "ALL") {
		validCheckInStatuses := map[string]bool{
			"VALID":              true,
			"PENDING":            true,
			"USED":               true,
			"CHECKED_IN":         true,
			"CANCELLED":          true,
			"REFUNDED":           true,
			"CANCELLED_REFUNDED": true,
		}
		if !validCheckInStatuses[strings.ToUpper(strings.TrimSpace(filter.CheckInStatus))] {
			return nil, ErrInvalidCheckInStatus
		}
	}

	if filter.Page < 1 || filter.Limit < 1 || filter.Limit > 100 {
		return nil, ErrInvalidPagination
	}

	res, err := u.ticketRepo.FindAdminAttendees(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to fetch admin attendees",
			slog.String("request_id", reqID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return res, nil
}

// GetAttendeeStats calculates platform-wide gate admission statistics and check-in rates.
func (u *AdminBookingUsecase) GetAttendeeStats(
	ctx context.Context,
	filter domain.AdminAttendeeFilter,
) (*domain.AdminAttendeeStats, error) {
	reqID := extractRequestID(ctx)

	if filter.CheckInStatus != "" && !strings.EqualFold(filter.CheckInStatus, "ALL") {
		validCheckInStatuses := map[string]bool{
			"VALID":              true,
			"PENDING":            true,
			"USED":               true,
			"CHECKED_IN":         true,
			"CANCELLED":          true,
			"REFUNDED":           true,
			"CANCELLED_REFUNDED": true,
		}
		if !validCheckInStatuses[strings.ToUpper(strings.TrimSpace(filter.CheckInStatus))] {
			return nil, ErrInvalidCheckInStatus
		}
	}

	stats, err := u.ticketRepo.GetAdminAttendeeStats(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to calculate admin attendee stats",
			slog.String("request_id", reqID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return stats, nil
}
