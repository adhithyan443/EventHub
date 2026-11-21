package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrganizerPaymentHandler struct {
	usecase *payment.OrganizerPaymentUsecase
	logger  *slog.Logger
}

func NewOrganizerPaymentHandler(
	usecase *payment.OrganizerPaymentUsecase,
	logger *slog.Logger,
) *OrganizerPaymentHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &OrganizerPaymentHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *OrganizerPaymentHandler) getUserID(ctx *gin.Context) (uuid.UUID, error) {
	val, exists := ctx.Get("user_id")
	if !exists {
		return uuid.Nil, appErrors.NewUnauthorizedError("unauthorized: missing user identity")
	}
	userID, ok := val.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, appErrors.NewUnauthorizedError("unauthorized: invalid user identity")
	}
	return userID, nil
}

func (h *OrganizerPaymentHandler) parseFilter(ctx *gin.Context) (domain.OrganizerPaymentFilter, error) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 || parsedPage > 100000 {
			return domain.OrganizerPaymentFilter{}, appErrors.NewValidationError("invalid page parameter")
		}
		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			return domain.OrganizerPaymentFilter{}, appErrors.NewValidationError("invalid limit parameter, must be between 1 and 100")
		}
		limit = parsedLimit
	}

	var eventID *uuid.UUID
	if eventIDParam := strings.TrimSpace(ctx.Query("event_id")); eventIDParam != "" {
		parsedEventID, err := uuid.Parse(eventIDParam)
		if err != nil || parsedEventID == uuid.Nil {
			return domain.OrganizerPaymentFilter{}, appErrors.NewValidationError("invalid event_id parameter, must be a valid non-empty UUID")
		}
		eventID = &parsedEventID
	}

	filter := domain.OrganizerPaymentFilter{
		Page:             page,
		Limit:            limit,
		Status:           strings.TrimSpace(ctx.Query("status")),
		AllocationStatus: strings.TrimSpace(ctx.Query("allocation_status")),
		Search:           strings.TrimSpace(ctx.Query("search")),
		EventID:          eventID,
	}

	if fromParam := strings.TrimSpace(ctx.Query("from")); fromParam != "" {
		fromTime, err := parseDateParam(fromParam, false)
		if err != nil {
			return domain.OrganizerPaymentFilter{}, appErrors.NewValidationError("invalid from date parameter, expected RFC3339 or YYYY-MM-DD")
		}
		filter.From = &fromTime
	}

	if toParam := strings.TrimSpace(ctx.Query("to")); toParam != "" {
		toTime, err := parseDateParam(toParam, true)
		if err != nil {
			return domain.OrganizerPaymentFilter{}, appErrors.NewValidationError("invalid to date parameter, expected RFC3339 or YYYY-MM-DD")
		}
		filter.To = &toTime
	}

	return filter, nil
}

func (h *OrganizerPaymentHandler) mapUsecaseError(ctx *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, payment.ErrUnauthorizedOrganizer),
		errors.Is(err, payment.ErrInvalidUserID),
		errors.Is(err, payment.ErrUnauthorizedEvent):
		ctx.Error(appErrors.NewForbiddenError(err.Error()))
	case errors.Is(err, payment.ErrEventNotFound):
		ctx.Error(appErrors.NewNotFoundError(err.Error()))
	case errors.Is(err, payment.ErrInvalidPaymentStatus),
		errors.Is(err, payment.ErrInvalidAllocationStatus),
		errors.Is(err, payment.ErrInvalidDateRange),
		errors.Is(err, payment.ErrInvalidEventID):
		ctx.Error(appErrors.NewValidationError(err.Error()))
	default:
		h.logger.ErrorContext(ctx.Request.Context(), op+"_failed", slog.Any("error", err))
		ctx.Error(appErrors.NewInternalError("internal server error"))
	}
}

// GetPaymentOverview handles GET /api/v1/organizers/payments/overview
func (h *OrganizerPaymentHandler) GetPaymentOverview(ctx *gin.Context) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	filter, err := h.parseFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	overview, err := h.usecase.GetOrganizerPaymentOverview(ctx.Request.Context(), userID, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "get_organizer_payment_overview", err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    overview,
	})
}

// GetEventPaymentSummaries handles GET /api/v1/organizers/payments/summary-by-event
func (h *OrganizerPaymentHandler) GetEventPaymentSummaries(ctx *gin.Context) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	filter, err := h.parseFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	result, err := h.usecase.GetOrganizerEventPaymentSummaries(ctx.Request.Context(), userID, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "get_organizer_event_payment_summaries", err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"summaries":     result.Summaries,
			"events":        result.Summaries,
			"overview":      result.Overview,
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

// ListPayments handles GET /api/v1/organizers/payments
func (h *OrganizerPaymentHandler) ListPayments(ctx *gin.Context) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	filter, err := h.parseFilter(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	result, err := h.usecase.FindOrganizerPayments(ctx.Request.Context(), userID, filter)
	if err != nil {
		h.mapUsecaseError(ctx, "list_organizer_payments", err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"payments":      result.Payments,
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
