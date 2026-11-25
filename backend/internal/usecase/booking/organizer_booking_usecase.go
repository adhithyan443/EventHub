package booking

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrNilBookingRepository   = errors.New("booking repository cannot be nil")
	ErrNilTicketRepository    = errors.New("ticket repository cannot be nil")
	ErrNilOrganizerRepository = errors.New("organizer repository cannot be nil")
	ErrUnauthorizedOrganizer  = errors.New("organizer access required or account not approved")
	ErrInvalidUserID          = errors.New("invalid or empty user ID")
	ErrInvalidBookingID       = errors.New("invalid or empty booking ID")
	ErrInvalidEventID         = errors.New("invalid event ID")
	ErrUnauthorizedEvent      = errors.New("event does not belong to authenticated organizer")
	ErrEventNotFound          = errors.New("event not found")
	ErrInvalidBookingStatus   = errors.New("invalid booking status filter")
	ErrInvalidPaymentStatus   = errors.New("invalid payment status filter")
	ErrInvalidCheckInStatus   = errors.New("invalid check-in status filter")
	ErrInvalidDateRange       = errors.New("invalid date range: 'from' cannot be after 'to'")
	ErrInvalidPagination      = errors.New("invalid pagination parameters")
)

// OrganizerBookingUsecase provides business logic for organizer order and attendee operations.
type OrganizerBookingUsecase struct {
	bookingRepo   domain.BookingRepository
	ticketRepo    domain.TicketRepository
	organizerRepo domain.OrganizerRepository
	eventRepo     domain.EventRepository
	logger        *slog.Logger
}

// NewOrganizerBookingUsecase constructs a new OrganizerBookingUsecase instance.
func NewOrganizerBookingUsecase(
	bookingRepo domain.BookingRepository,
	ticketRepo domain.TicketRepository,
	organizerRepo domain.OrganizerRepository,
	eventRepo domain.EventRepository,
	logger *slog.Logger,
) (*OrganizerBookingUsecase, error) {
	if bookingRepo == nil {
		return nil, ErrNilBookingRepository
	}
	if ticketRepo == nil {
		return nil, ErrNilTicketRepository
	}
	if organizerRepo == nil {
		return nil, ErrNilOrganizerRepository
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &OrganizerBookingUsecase{
		bookingRepo:   bookingRepo,
		ticketRepo:    ticketRepo,
		organizerRepo: organizerRepo,
		eventRepo:     eventRepo,
		logger:        logger,
	}, nil
}

// resolveAndValidateOrganizer retrieves the organizer profile associated with the authenticated user ID
// and enforces approved organizer account status (lifecycle: SUBMITTED -> UNDER_REVIEW -> APPROVED).
func (u *OrganizerBookingUsecase) resolveAndValidateOrganizer(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.Organizer, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}

	organizer, err := u.organizerRepo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerNotFound) {
			u.logger.WarnContext(ctx, "organizer access denied: organizer profile not found",
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
	if status != "APPROVED" {
		u.logger.WarnContext(ctx, "organizer access denied: unapproved organizer status",
			slog.String("user_id", userID.String()),
			slog.String("organizer_id", organizer.ID.String()),
			slog.String("status", organizer.Status),
		)
		return nil, ErrUnauthorizedOrganizer
	}

	return organizer, nil
}

// validateEventOwnership verifies that if an event ID is requested, the event exists
// and belongs to the authenticated organizer.
func (u *OrganizerBookingUsecase) validateEventOwnership(
	ctx context.Context,
	organizerID uuid.UUID,
	eventID *uuid.UUID,
) error {
	if eventID == nil {
		return nil
	}
	if *eventID == uuid.Nil {
		return ErrInvalidEventID
	}

	if u.eventRepo != nil {
		event, err := u.eventRepo.FindByID(*eventID)
		if err != nil {
			if errors.Is(err, domain.ErrEventNotFound) {
				return ErrEventNotFound
			}
			u.logger.ErrorContext(ctx, "failed to lookup event for organizer filter validation",
				slog.String("event_id", eventID.String()),
				slog.Any("error", err),
			)
			return err
		}
		if event.OrganizerID != organizerID {
			u.logger.WarnContext(ctx, "security: organizer attempted to access foreign event",
				slog.String("authenticated_organizer_id", organizerID.String()),
				slog.String("requested_event_id", eventID.String()),
				slog.String("actual_event_owner_id", event.OrganizerID.String()),
			)
			return ErrUnauthorizedEvent
		}
	}
	return nil
}

// validateAndNormalizeBookingFilter validates and sanitizes query filters for organizer bookings.
func (u *OrganizerBookingUsecase) validateAndNormalizeBookingFilter(
	ctx context.Context,
	organizerID uuid.UUID,
	filter *domain.OrganizerBookingFilter,
) error {
	// Strictly scope query to authenticated organizer ID
	filter.OrganizerID = organizerID

	const maxSafePage = 100_000
	const defaultLimit = 10
	const maxLimit = 100

	if filter.Page < 0 || filter.Page > maxSafePage {
		return ErrInvalidPagination
	}
	if filter.Limit < 0 || filter.Limit > maxLimit {
		return ErrInvalidPagination
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = defaultLimit
	}

	filter.BookingStatus = strings.ToUpper(strings.TrimSpace(filter.BookingStatus))
	if filter.BookingStatus != "" && filter.BookingStatus != "ALL" {
		switch filter.BookingStatus {
		case domain.BookingStatusPending,
			domain.BookingStatusConfirmed,
			domain.BookingStatusCancelled,
			domain.BookingStatusPartiallyCancelled:
			// Valid
		default:
			return ErrInvalidBookingStatus
		}
	}

	filter.PaymentStatus = strings.ToUpper(strings.TrimSpace(filter.PaymentStatus))
	if filter.PaymentStatus != "" && filter.PaymentStatus != "ALL" {
		switch filter.PaymentStatus {
		case "PENDING", "SUCCESS", "FAILED", "REFUNDED", "PARTIAL", "FULL", "PENDING_REFUND":
			// Valid
		default:
			return ErrInvalidPaymentStatus
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

	return u.validateEventOwnership(ctx, organizerID, filter.EventID)
}

// validateAndNormalizeAttendeeFilter validates and sanitizes query filters for event attendees.
func (u *OrganizerBookingUsecase) validateAndNormalizeAttendeeFilter(
	ctx context.Context,
	organizerID uuid.UUID,
	filter *domain.OrganizerAttendeeFilter,
) error {
	// Strictly scope query to authenticated organizer ID
	filter.OrganizerID = organizerID

	const maxSafePage = 100_000
	const defaultLimit = 10
	const maxLimit = 100

	if filter.Page < 0 || filter.Page > maxSafePage {
		return ErrInvalidPagination
	}
	if filter.Limit < 0 || filter.Limit > maxLimit {
		return ErrInvalidPagination
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = defaultLimit
	}

	filter.CheckInStatus = strings.ToUpper(strings.TrimSpace(filter.CheckInStatus))
	if filter.CheckInStatus != "" && filter.CheckInStatus != "ALL" {
		switch filter.CheckInStatus {
		case "USED", "CHECKED_IN", "VALID", "PENDING", "CANCELLED", "REFUNDED", "CANCELLED_REFUNDED":
			// Valid
		default:
			return ErrInvalidCheckInStatus
		}
	}

	filter.Search = strings.TrimSpace(filter.Search)

	return u.validateEventOwnership(ctx, organizerID, filter.EventID)
}

// ListBookings returns paginated booking records for the authenticated organizer.
func (u *OrganizerBookingUsecase) ListBookings(
	ctx context.Context,
	userID uuid.UUID,
	filter domain.OrganizerBookingFilter,
) (*domain.OrganizerBookingListResult, error) {
	organizer, err := u.resolveAndValidateOrganizer(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := u.validateAndNormalizeBookingFilter(ctx, organizer.ID, &filter); err != nil {
		return nil, err
	}

	u.logger.InfoContext(ctx, "organizer_bookings_listing_requested",
		slog.String("organizer_id", organizer.ID.String()),
		slog.Int("page", filter.Page),
		slog.Int("limit", filter.Limit),
		slog.String("booking_status", filter.BookingStatus),
		slog.String("payment_status", filter.PaymentStatus),
		slog.Bool("has_search", filter.Search != ""),
		slog.Bool("has_event_id", filter.EventID != nil),
	)

	return u.bookingRepo.FindOrganizerBookings(ctx, filter)
}

// GetBookingStats calculates aggregate financial and booking metrics for the organizer's scope.
func (u *OrganizerBookingUsecase) GetBookingStats(
	ctx context.Context,
	userID uuid.UUID,
	filter domain.OrganizerBookingFilter,
) (*domain.OrganizerBookingStats, error) {
	organizer, err := u.resolveAndValidateOrganizer(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := u.validateAndNormalizeBookingFilter(ctx, organizer.ID, &filter); err != nil {
		return nil, err
	}

	u.logger.InfoContext(ctx, "organizer_booking_stats_requested",
		slog.String("organizer_id", organizer.ID.String()),
		slog.String("booking_status", filter.BookingStatus),
		slog.String("payment_status", filter.PaymentStatus),
		slog.Bool("has_search", filter.Search != ""),
		slog.Bool("has_event_id", filter.EventID != nil),
	)

	return u.bookingRepo.GetOrganizerBookingStats(ctx, filter)
}

// GetBookingDetail retrieves complete booking order details with ownership verification.
func (u *OrganizerBookingUsecase) GetBookingDetail(
	ctx context.Context,
	userID uuid.UUID,
	bookingID uuid.UUID,
) (*domain.OrganizerBookingDetail, error) {
	organizer, err := u.resolveAndValidateOrganizer(ctx, userID)
	if err != nil {
		return nil, err
	}

	if bookingID == uuid.Nil {
		return nil, ErrInvalidBookingID
	}

	detail, err := u.bookingRepo.FindOrganizerBookingDetail(ctx, organizer.ID, bookingID)
	if err != nil {
		if errors.Is(err, domain.ErrBookingNotFound) {
			u.logger.WarnContext(ctx, "organizer booking detail not found or unauthorized",
				slog.String("organizer_id", organizer.ID.String()),
				slog.String("booking_id", bookingID.String()),
			)
			return nil, domain.ErrBookingNotFound
		}
		u.logger.ErrorContext(ctx, "failed to query organizer booking detail",
			slog.String("organizer_id", organizer.ID.String()),
			slog.String("booking_id", bookingID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "organizer_booking_detail_retrieved",
		slog.String("organizer_id", organizer.ID.String()),
		slog.String("booking_id", bookingID.String()),
	)

	return detail, nil
}

// ListAttendees retrieves a paginated list of individual ticket admissions for the organizer's events.
func (u *OrganizerBookingUsecase) ListAttendees(
	ctx context.Context,
	userID uuid.UUID,
	filter domain.OrganizerAttendeeFilter,
) (*domain.OrganizerAttendeeListResult, error) {
	organizer, err := u.resolveAndValidateOrganizer(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := u.validateAndNormalizeAttendeeFilter(ctx, organizer.ID, &filter); err != nil {
		return nil, err
	}

	u.logger.InfoContext(ctx, "organizer_attendees_listing_requested",
		slog.String("organizer_id", organizer.ID.String()),
		slog.Int("page", filter.Page),
		slog.Int("limit", filter.Limit),
		slog.String("check_in_status", filter.CheckInStatus),
		slog.Bool("has_search", filter.Search != ""),
		slog.Bool("has_event_id", filter.EventID != nil),
	)

	return u.ticketRepo.FindOrganizerAttendees(ctx, filter)
}

// GetAttendeeStats aggregates gate admission metrics and check-in rates for the organizer's events.
func (u *OrganizerBookingUsecase) GetAttendeeStats(
	ctx context.Context,
	userID uuid.UUID,
	filter domain.OrganizerAttendeeFilter,
) (*domain.OrganizerAttendeeStats, error) {
	organizer, err := u.resolveAndValidateOrganizer(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := u.validateAndNormalizeAttendeeFilter(ctx, organizer.ID, &filter); err != nil {
		return nil, err
	}

	u.logger.InfoContext(ctx, "organizer_attendee_stats_requested",
		slog.String("organizer_id", organizer.ID.String()),
		slog.String("check_in_status", filter.CheckInStatus),
		slog.Bool("has_search", filter.Search != ""),
		slog.Bool("has_event_id", filter.EventID != nil),
	)

	return u.ticketRepo.GetOrganizerAttendeeStats(ctx, filter)
}
