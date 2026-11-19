package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/settlement"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrganizerSettlementHandler struct {
	balanceUsecase    domain.OrganizerBalanceUsecase
	settlementUsecase domain.SettlementUsecase
	logger            *slog.Logger
}

func NewOrganizerSettlementHandler(
	balanceUsecase domain.OrganizerBalanceUsecase,
	settlementUsecase domain.SettlementUsecase,
	logger *slog.Logger,
) *OrganizerSettlementHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &OrganizerSettlementHandler{
		balanceUsecase:    balanceUsecase,
		settlementUsecase: settlementUsecase,
		logger:            logger,
	}
}

func (h *OrganizerSettlementHandler) getUserID(ctx *gin.Context) (uuid.UUID, error) {
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

// GetBalance handles GET /api/v1/organizers/settlements/balance
func (h *OrganizerSettlementHandler) GetBalance(ctx *gin.Context) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	requestID := ctx.GetString("request_id")
	if requestID == "" {
		requestID = ctx.GetHeader("X-Request-ID")
	}

	h.logger.InfoContext(ctx.Request.Context(), "organizer_settlement_balance_requested",
		slog.String("request_id", requestID),
		slog.String("user_id", userID.String()),
	)

	projection, err := h.balanceUsecase.GetOrganizerBalanceProjection(ctx.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, settlement.ErrUnauthorizedOrganizer),
			errors.Is(err, settlement.ErrInvalidUserID):
			ctx.Error(appErrors.NewForbiddenError(err.Error()))
		default:
			h.logger.ErrorContext(ctx.Request.Context(), "get_organizer_settlement_balance_failed",
				slog.String("request_id", requestID),
				slog.String("user_id", userID.String()),
				slog.Any("error", err),
			)
			ctx.Error(appErrors.NewInternalError("internal server error"))
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    projection,
	})
}

type RequestSettlementPayload struct {
	Amount  decimal.Decimal `json:"amount" binding:"required"`
	EventID *uuid.UUID      `json:"event_id"`
}

// RequestSettlement handles POST /api/v1/organizers/settlements
func (h *OrganizerSettlementHandler) RequestSettlement(ctx *gin.Context) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	var payload RequestSettlementPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.Error(appErrors.NewValidationError("invalid settlement request payload"))
		return
	}

	if payload.Amount.LessThanOrEqual(decimal.Zero) {
		ctx.Error(appErrors.NewValidationError("settlement amount must be greater than zero"))
		return
	}

	settlementRecord, err := h.settlementUsecase.RequestSettlement(ctx.Request.Context(), userID, payload.Amount, payload.EventID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidSettlementAmount):
			ctx.Error(appErrors.NewValidationError(err.Error()))
		case errors.Is(err, domain.ErrInsufficientAvailableBalance):
			ctx.Error(appErrors.NewConflictError(err.Error()))
		case errors.Is(err, domain.ErrOrganizerNotEligibleForSettlement):
			ctx.Error(appErrors.NewForbiddenError("organizer is not currently eligible for settlement"))
		case errors.Is(err, domain.ErrEventNotEligibleForSettlement):
			ctx.Error(appErrors.NewConflictError("selected event is not eligible for settlement or balance is insufficient"))
		case errors.Is(err, settlement.ErrUnauthorizedOrganizer):
			ctx.Error(appErrors.NewForbiddenError(err.Error()))
		default:
			h.logger.ErrorContext(ctx.Request.Context(), "request_settlement_failed",
				slog.String("user_id", userID.String()),
				slog.Any("error", err),
			)
			ctx.Error(appErrors.NewInternalError("failed to process settlement request"))
		}
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    settlementRecord,
	})
}

// ListSettlements handles GET /api/v1/organizers/settlements
func (h *OrganizerSettlementHandler) ListSettlements(ctx *gin.Context) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	page := 1
	limit := 10

	if val := ctx.Query("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		}
	}
	if val := ctx.Query("limit"); val != "" {
		if l, err := strconv.Atoi(val); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	var status *domain.SettlementStatus
	if sVal := strings.TrimSpace(ctx.Query("status")); sVal != "" {
		st := domain.SettlementStatus(strings.ToUpper(sVal))
		status = &st
	}

	result, err := h.settlementUsecase.ListOrganizerSettlements(ctx.Request.Context(), userID, page, limit, status)
	if err != nil {
		switch {
		case errors.Is(err, settlement.ErrUnauthorizedOrganizer):
			ctx.Error(appErrors.NewForbiddenError(err.Error()))
		default:
			h.logger.ErrorContext(ctx.Request.Context(), "list_organizer_settlements_failed",
				slog.String("user_id", userID.String()),
				slog.Any("error", err),
			)
			ctx.Error(appErrors.NewInternalError("failed to list settlements"))
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetSettlement handles GET /api/v1/organizers/settlements/:id
func (h *OrganizerSettlementHandler) GetSettlement(ctx *gin.Context) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	idParam := strings.TrimSpace(ctx.Param("id"))
	settlementID, err := uuid.Parse(idParam)
	if err != nil || settlementID == uuid.Nil {
		ctx.Error(appErrors.NewValidationError("invalid settlement ID"))
		return
	}

	record, err := h.settlementUsecase.GetOrganizerSettlement(ctx.Request.Context(), userID, settlementID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSettlementNotFound):
			ctx.Error(appErrors.NewNotFoundError("settlement not found"))
		case errors.Is(err, settlement.ErrUnauthorizedOrganizer):
			ctx.Error(appErrors.NewForbiddenError(err.Error()))
		default:
			h.logger.ErrorContext(ctx.Request.Context(), "get_organizer_settlement_failed",
				slog.String("settlement_id", settlementID.String()),
				slog.String("user_id", userID.String()),
				slog.Any("error", err),
			)
			ctx.Error(appErrors.NewInternalError("failed to get settlement details"))
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    record,
	})
}
