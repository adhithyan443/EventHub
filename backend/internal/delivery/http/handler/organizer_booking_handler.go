package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OrganizerBookingService defines the use-case interface consumed by OrganizerBookingHandler.
type OrganizerBookingService interface {
	ListBookings(ctx context.Context, userID uuid.UUID, filter domain.OrganizerBookingFilter) (*domain.OrganizerBookingListResult, error)
	GetBookingStats(ctx context.Context, userID uuid.UUID, filter domain.OrganizerBookingFilter) (*domain.OrganizerBookingStats, error)
	GetBookingDetail(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID) (*domain.OrganizerBookingDetail, error)
	ListAttendees(ctx context.Context, userID uuid.UUID, filter domain.OrganizerAttendeeFilter) (*domain.OrganizerAttendeeListResult, error)
	GetAttendeeStats(ctx context.Context, userID uuid.UUID, filter domain.OrganizerAttendeeFilter) (*domain.OrganizerAttendeeStats, error)
}

// OrganizerBookingHandler handles HTTP requests for organizer bookings and attendee admissions.
type OrganizerBookingHandler struct {
	usecase OrganizerBookingService
	logger  *slog.Logger
}

// NewOrganizerBookingHandler constructs a new OrganizerBookingHandler instance.
func NewOrganizerBookingHandler(
	usecase OrganizerBookingService,
	logger *slog.Logger,
) *OrganizerBookingHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &OrganizerBookingHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func getRequestID(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	reqID := ctx.GetString(middleware.RequestIDKey)
	if reqID == "" {
		reqID = ctx.GetHeader("X-Request-ID")
	}
	return reqID
}

func contextWithRequestID(ctx *gin.Context) context.Context {
	reqID := getRequestID(ctx)
	reqCtx := ctx.Request.Context()
	return booking.WithRequestID(reqCtx, reqID)
}

func (h *OrganizerBookingHandler) mapUsecaseError(ctx *gin.Context, op string, err error) {
	reqID := getRequestID(ctx)
	switch {
	case errors.Is(err, booking.ErrUnauthorizedOrganizer),
		errors.Is(err, booking.ErrUnauthorizedEvent):
		h.logger.WarnContext(ctx.Request.Context(), op+"_forbidden",
			slog.String("request_id", reqID),
			slog.String("path", ctx.Request.URL.Path),
			slog.Any("error", err),
		)
		ctx.Error(appErrors.NewForbiddenError(err.Error()))
	case errors.Is(err, booking.ErrInvalidUserID):
		h.logger.WarnContext(ctx.Request.Context(), op+"_unauthorized",
			slog.String("request_id", reqID),
			slog.String("path", ctx.Request.URL.Path),
			slog.Any("error", err),
		)
		ctx.Error(appErrors.NewUnauthorizedError(err.Error()))
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

func (h *OrganizerBookingHandler) parseBookingFilter(ctx *gin.Context) (domain.OrganizerBookingFilter, error) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 || parsedPage > 100000 {
			return domain.OrganizerBookingFilter{}, appErrors.NewValidationError("invalid page parameter")
		}
		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			return domain.OrganizerBookingFilter{}, appErrors.NewValidationError("invalid limit parameter, must be between 1 and 100")
		}
		limit = parsedLimit
	}

	var eventID *uuid.UUID
	if eventIDParam := strings.TrimSpace(ctx.Query("event_id")); eventIDParam != "" {
		parsedEventID, err := uuid.Parse(eventIDParam)
		if err != nil || parsedEventID == uuid.Nil {
			return domain.OrganizerBookingFilter{}, appErrors.NewValidationError("invalid event_id parameter, must be a valid non-empty UUID")
		}
		eventID = &parsedEventID
	}

	filter := domain.OrganizerBookingFilter{
		Page:          page,
		Limit:         limit,
		BookingStatus: strings.TrimSpace(ctx.Query("booking_status")),
		PaymentStatus: strings.TrimSpace(ctx.Query("payment_status")),
		Search:        strings.TrimSpace(ctx.Query("search")),
		EventID:       eventID,
	}

	if fromParam := strings.TrimSpace(ctx.Query("from")); fromParam != "" {
		fromTime, err := parseDateParam(fromParam, false)
		if err != nil {
			return domain.OrganizerBookingFilter{}, appErrors.NewValidationError("invalid from date parameter, expected RFC3339 or YYYY-MM-DD")
		}
		filter.From = &fromTime
	}

	if toParam := strings.TrimSpace(ctx.Query("to")); toParam != "" {
		toTime, err := parseDateParam(toParam, true)
		if err != nil {
			return domain.OrganizerBookingFilter{}, appErrors.NewValidationError("invalid to date parameter, expected RFC3339 or YYYY-MM-DD")
		}
		filter.To = &toTime
	}

	return filter, nil
}

func (h *OrganizerBookingHandler) parseAttendeeFilter(ctx *gin.Context) (domain.OrganizerAttendeeFilter, error) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 || parsedPage > 100000 {
			return domain.OrganizerAttendeeFilter{}, appErrors.NewValidationError("invalid page parameter")
		}
		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			return domain.OrganizerAttendeeFilter{}, appErrors.NewValidationError("invalid limit parameter, must be between 1 and 100")
		}
		limit = parsedLimit
	}

	var eventID *uuid.UUID
	if eventIDParam := strings.TrimSpace(ctx.Query("event_id")); eventIDParam != "" {
		parsedEventID, err := uuid.Parse(eventIDParam)
		if err != nil || parsedEventID == uuid.Nil {
			return domain.OrganizerAttendeeFilter{}, appErrors.NewValidationError("invalid event_id parameter, must be a valid non-empty UUID")
		}
		eventID = &parsedEventID
	}

	checkInStatus := strings.TrimSpace(ctx.Query("check_in_status"))
	if checkInStatus == "" {
		checkInStatus = strings.TrimSpace(ctx.Query("status"))
	}

	return domain.OrganizerAttendeeFilter{
		Page:          page,
		Limit:         limit,
		CheckInStatus: checkInStatus,
		Search:        strings.TrimSpace(ctx.Query("search")),
		EventID:       eventID,
	}, nil
}

// ListBookings handles GET /api/v1/organizers/bookings
func (h *OrganizerBookingHandler) ListBookings(ctx *gin.Context) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	filter, err := h.parseBookingFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	reqCtx := contextWithRequestID(ctx)
	result, err := h.usecase.ListBookings(reqCtx, userID, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "list_organizer_bookings", err)
		return
	}

	h.logger.InfoContext(reqCtx, "organizer_bookings_listed",
		slog.String("request_id", getRequestID(ctx)),
		slog.String("user_id", userID.String()),
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

// GetBookingStats handles GET /api/v1/organizers/bookings/stats
func (h *OrganizerBookingHandler) GetBookingStats(ctx *gin.Context) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	filter, err := h.parseBookingFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	reqCtx := contextWithRequestID(ctx)
	stats, err := h.usecase.GetBookingStats(reqCtx, userID, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "get_organizer_booking_stats", err)
		return
	}

	h.logger.InfoContext(reqCtx, "organizer_booking_stats_retrieved",
		slog.String("request_id", getRequestID(ctx)),
		slog.String("user_id", userID.String()),
		slog.Int64("total_bookings", stats.TotalBookings),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetBookingDetail handles GET /api/v1/organizers/bookings/:id
func (h *OrganizerBookingHandler) GetBookingDetail(ctx *gin.Context) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	bookingIDParam := strings.TrimSpace(ctx.Param("id"))
	bookingID, err := uuid.Parse(bookingIDParam)
	if err != nil || bookingID == uuid.Nil {
		ctx.Error(appErrors.NewValidationError("invalid booking ID parameter, must be a valid non-empty UUID"))
		return
	}

	reqCtx := contextWithRequestID(ctx)
	detail, err := h.usecase.GetBookingDetail(reqCtx, userID, bookingID)
	if err != nil {
		h.mapUsecaseError(ctx, "get_organizer_booking_detail", err)
		return
	}

	h.logger.InfoContext(reqCtx, "organizer_booking_detail_retrieved",
		slog.String("request_id", getRequestID(ctx)),
		slog.String("user_id", userID.String()),
		slog.String("booking_id", bookingID.String()),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// ListAttendees handles GET /api/v1/organizers/attendees
func (h *OrganizerBookingHandler) ListAttendees(ctx *gin.Context) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	filter, err := h.parseAttendeeFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	reqCtx := contextWithRequestID(ctx)
	result, err := h.usecase.ListAttendees(reqCtx, userID, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "list_organizer_attendees", err)
		return
	}

	h.logger.InfoContext(reqCtx, "organizer_attendees_listed",
		slog.String("request_id", getRequestID(ctx)),
		slog.String("user_id", userID.String()),
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

// GetAttendeeStats handles GET /api/v1/organizers/attendees/stats
func (h *OrganizerBookingHandler) GetAttendeeStats(ctx *gin.Context) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	filter, err := h.parseAttendeeFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	reqCtx := contextWithRequestID(ctx)
	stats, err := h.usecase.GetAttendeeStats(reqCtx, userID, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "get_organizer_attendee_stats", err)
		return
	}

	h.logger.InfoContext(reqCtx, "organizer_attendee_stats_retrieved",
		slog.String("request_id", getRequestID(ctx)),
		slog.String("user_id", userID.String()),
		slog.Int64("total_issued", stats.TotalIssuedTickets),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
