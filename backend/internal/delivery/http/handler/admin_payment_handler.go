package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminPaymentHandler struct {
	adminPaymentUsecase *payment.AdminPaymentUsecase
	logger              *slog.Logger
}

func NewAdminPaymentHandler(
	adminPaymentUsecase *payment.AdminPaymentUsecase,
	logger *slog.Logger,
) *AdminPaymentHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminPaymentHandler{
		adminPaymentUsecase: adminPaymentUsecase,
		logger:              logger,
	}
}

func (h *AdminPaymentHandler) ListPayments(ctx *gin.Context) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 || parsedPage > 100000 {
			ctx.Error(appErrors.NewValidationError("invalid page parameter"))
			return
		}
		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			ctx.Error(appErrors.NewValidationError("invalid limit parameter, must be between 1 and 100"))
			return
		}
		limit = parsedLimit
	}

	var eventID *uuid.UUID
	if eventIDParam := strings.TrimSpace(ctx.Query("event_id")); eventIDParam != "" {
		parsedEventID, err := uuid.Parse(eventIDParam)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid event_id parameter, must be a valid UUID"))
			return
		}
		eventID = &parsedEventID
	}

	filter := domain.AdminPaymentFilter{
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
			ctx.Error(appErrors.NewValidationError("invalid from date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.From = &fromTime
	}

	if toParam := strings.TrimSpace(ctx.Query("to")); toParam != "" {
		toTime, err := parseDateParam(toParam, true)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid to date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.To = &toTime
	}

	result, err := h.adminPaymentUsecase.ListPayments(ctx.Request.Context(), filter)
	if err != nil {
		h.logger.ErrorContext(ctx.Request.Context(), "admin_list_payments_failed", slog.Any("error", err))
		if errors.Is(err, payment.ErrInvalidAllocationStatus) || errors.Is(err, payment.ErrInvalidDateRange) {
			ctx.Error(appErrors.NewValidationError(err.Error()))
			return
		}
		ctx.Error(appErrors.NewInternalError("internal server error"))
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

func (h *AdminPaymentHandler) GetEventPaymentSummaries(ctx *gin.Context) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 || parsedPage > 100000 {
			ctx.Error(appErrors.NewValidationError("invalid page parameter"))
			return
		}
		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			ctx.Error(appErrors.NewValidationError("invalid limit parameter, must be between 1 and 100"))
			return
		}
		limit = parsedLimit
	}

	var eventID *uuid.UUID
	if eventIDParam := strings.TrimSpace(ctx.Query("event_id")); eventIDParam != "" {
		parsedEventID, err := uuid.Parse(eventIDParam)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid event_id parameter, must be a valid UUID"))
			return
		}
		eventID = &parsedEventID
	}

	filter := domain.AdminPaymentFilter{
		Page:    page,
		Limit:   limit,
		Search:  strings.TrimSpace(ctx.Query("search")),
		EventID: eventID,
	}

	if fromParam := strings.TrimSpace(ctx.Query("from")); fromParam != "" {
		fromTime, err := parseDateParam(fromParam, false)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid from date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.From = &fromTime
	}

	if toParam := strings.TrimSpace(ctx.Query("to")); toParam != "" {
		toTime, err := parseDateParam(toParam, true)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid to date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.To = &toTime
	}

	result, err := h.adminPaymentUsecase.GetEventPaymentSummaries(ctx.Request.Context(), filter)
	if err != nil {
		h.logger.ErrorContext(ctx.Request.Context(), "admin_get_event_payment_summaries_failed", slog.Any("error", err))
		if errors.Is(err, payment.ErrInvalidDateRange) {
			ctx.Error(appErrors.NewValidationError(err.Error()))
			return
		}
		ctx.Error(appErrors.NewInternalError("internal server error"))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"summaries":       result.Summaries,
			"events":          result.Summaries,
			"overall_summary": result.OverallSummary,
			"page":            result.Page,
			"limit":           result.Limit,
			"total_records":   result.TotalRecords,
			"total_pages":     result.TotalPages,
			"pagination": gin.H{
				"page":          result.Page,
				"limit":         result.Limit,
				"total_records": result.TotalRecords,
				"total_pages":   result.TotalPages,
			},
		},
	})
}

func parseDateParam(val string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, val); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", val); err == nil {
		if endOfDay {
			return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC), nil
		}
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, errors.New("unsupported date format")
}
