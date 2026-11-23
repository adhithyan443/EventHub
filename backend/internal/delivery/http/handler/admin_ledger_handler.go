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
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminLedgerHandler struct {
	ledgerUsecase domain.AdminLedgerUsecase
	logger        *slog.Logger
}

// NewAdminLedgerHandler constructs a new AdminLedgerHandler.
func NewAdminLedgerHandler(
	ledgerUsecase domain.AdminLedgerUsecase,
	logger *slog.Logger,
) *AdminLedgerHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminLedgerHandler{
		ledgerUsecase: ledgerUsecase,
		logger:        logger,
	}
}

// ListJournals handles GET /api/v1/admin/ledger/journals.
func (h *AdminLedgerHandler) ListJournals(ctx *gin.Context) {
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

	search := strings.TrimSpace(ctx.Query("search"))
	if search == "" {
		search = strings.TrimSpace(ctx.Query("reference_id"))
	}

	var eventID *uuid.UUID
	if eventParam := strings.TrimSpace(ctx.Query("event_id")); eventParam != "" {
		if id, err := uuid.Parse(eventParam); err == nil {
			eventID = &id
		}
	}

	var organizerID *uuid.UUID
	if orgParam := strings.TrimSpace(ctx.Query("organizer_id")); orgParam != "" {
		if id, err := uuid.Parse(orgParam); err == nil {
			organizerID = &id
		}
	}

	filter := domain.AdminLedgerFilter{
		Page:             page,
		Limit:            limit,
		TransactionType:  strings.TrimSpace(ctx.Query("transaction_type")),
		Status:           strings.TrimSpace(ctx.Query("status")),
		SettlementStatus: strings.TrimSpace(ctx.Query("settlement_status")),
		Search:           search,
		EventID:          eventID,
		OrganizerID:      organizerID,
	}

	if fromParam := strings.TrimSpace(ctx.Query("from")); fromParam != "" {
		fromTime, err := parseLedgerDateParam(fromParam, false)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid from date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.From = &fromTime
	}

	if toParam := strings.TrimSpace(ctx.Query("to")); toParam != "" {
		toTime, err := parseLedgerDateParam(toParam, true)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid to date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.To = &toTime
	}

	result, err := h.ledgerUsecase.ListJournals(ctx.Request.Context(), filter)
	if err != nil {
		h.logger.ErrorContext(ctx.Request.Context(), "admin_list_journals_failed", slog.Any("error", err))
		if errors.Is(err, domain.ErrInvalidDateRange) {
			ctx.Error(appErrors.NewValidationError(err.Error()))
			return
		}
		ctx.Error(appErrors.NewInternalError("failed to retrieve ledger journals"))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"journals":      result.Journals,
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

// GetJournal handles GET /api/v1/admin/ledger/journals/:id.
func (h *AdminLedgerHandler) GetJournal(ctx *gin.Context) {
	idParam := strings.TrimSpace(ctx.Param("id"))
	journalID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid journal id parameter, must be a valid UUID"))
		return
	}

	detail, err := h.ledgerUsecase.GetJournal(ctx.Request.Context(), journalID)
	if err != nil {
		h.logger.ErrorContext(ctx.Request.Context(), "admin_get_journal_failed",
			slog.String("journal_id", idParam),
			slog.Any("error", err),
		)
		if errors.Is(err, domain.ErrLedgerJournalNotFound) {
			ctx.Error(appErrors.NewNotFoundError("journal not found"))
			return
		}
		ctx.Error(appErrors.NewInternalError("failed to retrieve journal details"))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// GetFinancialOverview handles GET /api/v1/admin/ledger/overview.
func (h *AdminLedgerHandler) GetFinancialOverview(ctx *gin.Context) {
	var filter domain.AdminFinancialFilter

	if fromParam := strings.TrimSpace(ctx.Query("from")); fromParam != "" {
		fromTime, err := parseLedgerDateParam(fromParam, false)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid from date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.From = &fromTime
	}

	if toParam := strings.TrimSpace(ctx.Query("to")); toParam != "" {
		toTime, err := parseLedgerDateParam(toParam, true)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid to date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.To = &toTime
	}

	if eventParam := strings.TrimSpace(ctx.Query("event_id")); eventParam != "" {
		if id, err := uuid.Parse(eventParam); err == nil {
			filter.EventID = &id
		} else {
			ctx.Error(appErrors.NewValidationError("invalid event_id parameter, must be a valid UUID"))
			return
		}
	}

	if orgParam := strings.TrimSpace(ctx.Query("organizer_id")); orgParam != "" {
		if id, err := uuid.Parse(orgParam); err == nil {
			filter.OrganizerID = &id
		} else {
			ctx.Error(appErrors.NewValidationError("invalid organizer_id parameter, must be a valid UUID"))
			return
		}
	}

	filter.TransactionType = strings.TrimSpace(ctx.Query("transaction_type"))
	filter.SettlementStatus = strings.TrimSpace(ctx.Query("settlement_status"))

	overview, err := h.ledgerUsecase.GetFinancialOverview(ctx.Request.Context(), filter)
	if err != nil {
		h.logger.ErrorContext(ctx.Request.Context(), "admin_get_financial_overview_failed", slog.Any("error", err))
		if errors.Is(err, domain.ErrInvalidDateRange) {
			ctx.Error(appErrors.NewValidationError(err.Error()))
			return
		}
		ctx.Error(appErrors.NewInternalError("failed to retrieve financial overview"))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    overview,
	})
}

func parseLedgerDateParam(val string, endOfDay bool) (time.Time, error) {
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
