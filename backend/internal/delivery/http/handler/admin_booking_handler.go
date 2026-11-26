package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminBookingService defines the use-case interface consumed by AdminBookingHandler.
type AdminBookingService interface {
	ListBookings(ctx context.Context, filter domain.AdminBookingFilter) (*domain.AdminBookingListResult, error)
	GetBookingStats(ctx context.Context, filter domain.AdminBookingFilter) (*domain.AdminBookingStats, error)
	GetBookingDetail(ctx context.Context, bookingID uuid.UUID) (*domain.AdminBookingDetail, error)
	ListAttendees(ctx context.Context, filter domain.AdminAttendeeFilter) (*domain.AdminAttendeeListResult, error)
	GetAttendeeStats(ctx context.Context, filter domain.AdminAttendeeFilter) (*domain.AdminAttendeeStats, error)
}

// AdminBookingHandler handles HTTP requests for platform-wide admin bookings and attendee admissions.
type AdminBookingHandler struct {
	usecase AdminBookingService
	logger  *slog.Logger
}

// NewAdminBookingHandler constructs a new AdminBookingHandler instance.
func NewAdminBookingHandler(
	usecase AdminBookingService,
	logger *slog.Logger,
) *AdminBookingHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminBookingHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *AdminBookingHandler) mapUsecaseError(ctx *gin.Context, op string, err error) {
	reqID := getRequestID(ctx)
	switch {
	case errors.Is(err, booking.ErrEventNotFound),
		errors.Is(err, domain.ErrBookingNotFound):
		h.logger.WarnContext(ctx.Request.Context(), op+"_not_found",
			slog.String("request_id", reqID),
			slog.String("path", ctx.Request.URL.Path),
			slog.Any("error", err),
		)
		ctx.Error(appErrors.NewNotFoundError(err.Error()))
	case errors.Is(err, booking.ErrInvalidBookingStatus),
		errors.Is(err, booking.ErrInvalidPaymentStatus),
		errors.Is(err, booking.ErrInvalidCheckInStatus),
		errors.Is(err, booking.ErrInvalidDateRange),
		errors.Is(err, booking.ErrInvalidPagination),
		errors.Is(err, booking.ErrInvalidBookingID),
		errors.Is(err, booking.ErrInvalidEventID):
		h.logger.WarnContext(ctx.Request.Context(), op+"_validation_error",
			slog.String("request_id", reqID),
			slog.String("path", ctx.Request.URL.Path),
			slog.Any("error", err),
		)
		ctx.Error(appErrors.NewValidationError(err.Error()))
	default:
		h.logger.ErrorContext(ctx.Request.Context(), op+"_internal_error",
			slog.String("request_id", reqID),
			slog.String("path", ctx.Request.URL.Path),
			slog.Any("error", err),
		)
		ctx.Error(appErrors.NewInternalError("internal server error"))
	}
}

func (h *AdminBookingHandler) parseBookingFilter(ctx *gin.Context) (domain.AdminBookingFilter, error) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 || parsedPage > 100000 {
			return domain.AdminBookingFilter{}, appErrors.NewValidationError("invalid page parameter")
		}
		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			return domain.AdminBookingFilter{}, appErrors.NewValidationError("invalid limit parameter, must be between 1 and 100")
		}
		limit = parsedLimit
	}

	var eventID *uuid.UUID
	if eventIDParam := strings.TrimSpace(ctx.Query("event_id")); eventIDParam != "" {
		parsedEventID, err := uuid.Parse(eventIDParam)
		if err != nil || parsedEventID == uuid.Nil {
			return domain.AdminBookingFilter{}, appErrors.NewValidationError("invalid event_id parameter, must be a valid non-empty UUID")
		}
		eventID = &parsedEventID
	}

	var organizerID *uuid.UUID
	if orgIDParam := strings.TrimSpace(ctx.Query("organizer_id")); orgIDParam != "" {
		parsedOrgID, err := uuid.Parse(orgIDParam)
		if err != nil || parsedOrgID == uuid.Nil {
			return domain.AdminBookingFilter{}, appErrors.NewValidationError("invalid organizer_id parameter, must be a valid non-empty UUID")
		}
		organizerID = &parsedOrgID
	}

	filter := domain.AdminBookingFilter{
		Page:          page,
		Limit:         limit,
		BookingStatus: strings.TrimSpace(ctx.Query("booking_status")),
		PaymentStatus: strings.TrimSpace(ctx.Query("payment_status")),
		Search:        strings.TrimSpace(ctx.Query("search")),
		EventID:       eventID,
		OrganizerID:   organizerID,
	}

	if fromParam := strings.TrimSpace(ctx.Query("from")); fromParam != "" {
		fromTime, err := parseDateParam(fromParam, false)
		if err != nil {
			return domain.AdminBookingFilter{}, appErrors.NewValidationError("invalid from date parameter, expected RFC3339 or YYYY-MM-DD")
		}
		filter.From = &fromTime
	}

	if toParam := strings.TrimSpace(ctx.Query("to")); toParam != "" {
		toTime, err := parseDateParam(toParam, true)
		if err != nil {
			return domain.AdminBookingFilter{}, appErrors.NewValidationError("invalid to date parameter, expected RFC3339 or YYYY-MM-DD")
		}
		filter.To = &toTime
	}

	return filter, nil
}

func (h *AdminBookingHandler) parseAttendeeFilter(ctx *gin.Context) (domain.AdminAttendeeFilter, error) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 || parsedPage > 100000 {
			return domain.AdminAttendeeFilter{}, appErrors.NewValidationError("invalid page parameter")
		}
		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			return domain.AdminAttendeeFilter{}, appErrors.NewValidationError("invalid limit parameter, must be between 1 and 100")
		}
		limit = parsedLimit
	}

	var eventID *uuid.UUID
	if eventIDParam := strings.TrimSpace(ctx.Query("event_id")); eventIDParam != "" {
		parsedEventID, err := uuid.Parse(eventIDParam)
		if err != nil || parsedEventID == uuid.Nil {
			return domain.AdminAttendeeFilter{}, appErrors.NewValidationError("invalid event_id parameter, must be a valid non-empty UUID")
		}
		eventID = &parsedEventID
	}

	var organizerID *uuid.UUID
	if orgIDParam := strings.TrimSpace(ctx.Query("organizer_id")); orgIDParam != "" {
		parsedOrgID, err := uuid.Parse(orgIDParam)
		if err != nil || parsedOrgID == uuid.Nil {
			return domain.AdminAttendeeFilter{}, appErrors.NewValidationError("invalid organizer_id parameter, must be a valid non-empty UUID")
		}
		organizerID = &parsedOrgID
	}

	checkInStatus := strings.TrimSpace(ctx.Query("check_in_status"))
	if checkInStatus == "" {
		checkInStatus = strings.TrimSpace(ctx.Query("status"))
	}

	return domain.AdminAttendeeFilter{
		Page:          page,
		Limit:         limit,
		CheckInStatus: checkInStatus,
		Search:        strings.TrimSpace(ctx.Query("search")),
		EventID:       eventID,
		OrganizerID:   organizerID,
	}, nil
}

// ListBookings handles GET /api/v1/admin/bookings
func (h *AdminBookingHandler) ListBookings(ctx *gin.Context) {
	filter, err := h.parseBookingFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	reqCtx := contextWithRequestID(ctx)
	result, err := h.usecase.ListBookings(reqCtx, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "list_admin_bookings", err)
		return
	}

	h.logger.InfoContext(reqCtx, "admin_bookings_listed",
		slog.String("request_id", getRequestID(ctx)),
		slog.Int("page", result.Page),
		slog.Int("limit", result.Limit),
		slog.Int64("total_records", result.TotalRecords),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"bookings":      result.Bookings,
			"page":          result.Page,
			"limit":         result.Limit,
			"total_records": result.TotalRecords,
			"total_pages":   result.TotalPages,
			"pagination": gin.H{
				"page":          result.Page,
				"limit":         result.Limit,
				"total_records": result.TotalRecords,
				"total_pages":   result.TotalPages,
			},
		},
	})
}

// GetBookingStats handles GET /api/v1/admin/bookings/stats
func (h *AdminBookingHandler) GetBookingStats(ctx *gin.Context) {
	filter, err := h.parseBookingFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	reqCtx := contextWithRequestID(ctx)
	stats, err := h.usecase.GetBookingStats(reqCtx, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "get_admin_booking_stats", err)
		return
	}

	h.logger.InfoContext(reqCtx, "admin_booking_stats_retrieved",
		slog.String("request_id", getRequestID(ctx)),
		slog.Int64("total_bookings", stats.TotalBookings),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetBookingDetail handles GET /api/v1/admin/bookings/:id
func (h *AdminBookingHandler) GetBookingDetail(ctx *gin.Context) {
	bookingIDParam := strings.TrimSpace(ctx.Param("id"))
	bookingID, err := uuid.Parse(bookingIDParam)
	if err != nil || bookingID == uuid.Nil {
		ctx.Error(appErrors.NewValidationError("invalid booking ID parameter, must be a valid non-empty UUID"))
		return
	}

	reqCtx := contextWithRequestID(ctx)
	detail, err := h.usecase.GetBookingDetail(reqCtx, bookingID)
	if err != nil {
		h.mapUsecaseError(ctx, "get_admin_booking_detail", err)
		return
	}

	h.logger.InfoContext(reqCtx, "admin_booking_detail_retrieved",
		slog.String("request_id", getRequestID(ctx)),
		slog.String("booking_id", bookingID.String()),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// ListAttendees handles GET /api/v1/admin/attendees
func (h *AdminBookingHandler) ListAttendees(ctx *gin.Context) {
	filter, err := h.parseAttendeeFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	reqCtx := contextWithRequestID(ctx)
	result, err := h.usecase.ListAttendees(reqCtx, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "list_admin_attendees", err)
		return
	}

	h.logger.InfoContext(reqCtx, "admin_attendees_listed",
		slog.String("request_id", getRequestID(ctx)),
		slog.Int("page", result.Page),
		slog.Int("limit", result.Limit),
		slog.Int64("total_records", result.TotalRecords),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"attendees":     result.Attendees,
			"page":          result.Page,
			"limit":         result.Limit,
			"total_records": result.TotalRecords,
			"total_pages":   result.TotalPages,
			"pagination": gin.H{
				"page":          result.Page,
				"limit":         result.Limit,
				"total_records": result.TotalRecords,
				"total_pages":   result.TotalPages,
			},
		},
	})
}

// GetAttendeeStats handles GET /api/v1/admin/attendees/stats
func (h *AdminBookingHandler) GetAttendeeStats(ctx *gin.Context) {
	filter, err := h.parseAttendeeFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	reqCtx := contextWithRequestID(ctx)
	stats, err := h.usecase.GetAttendeeStats(reqCtx, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "get_admin_attendee_stats", err)
		return
	}

	h.logger.InfoContext(reqCtx, "admin_attendee_stats_retrieved",
		slog.String("request_id", getRequestID(ctx)),
		slog.Int64("total_issued", stats.TotalIssuedTickets),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
