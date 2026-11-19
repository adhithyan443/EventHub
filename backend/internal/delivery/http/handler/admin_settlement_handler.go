package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminSettlementHandler struct {
	settlementUsecase domain.SettlementUsecase
	logger            *slog.Logger
}

func NewAdminSettlementHandler(
	settlementUsecase domain.SettlementUsecase,
	logger *slog.Logger,
) *AdminSettlementHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminSettlementHandler{
		settlementUsecase: settlementUsecase,
		logger:            logger,
	}
}

func (h *AdminSettlementHandler) getAdminUserID(ctx *gin.Context) (uuid.UUID, error) {
	val, exists := ctx.Get("user_id")
	if !exists {
		return uuid.Nil, appErrors.NewUnauthorizedError("unauthorized: missing admin identity")
	}
	userID, ok := val.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, appErrors.NewUnauthorizedError("unauthorized: invalid admin identity")
	}
	return userID, nil
}

// ListSettlements handles GET /api/v1/admin/settlements
func (h *AdminSettlementHandler) ListSettlements(ctx *gin.Context) {
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

	var organizerID *uuid.UUID
	if oVal := strings.TrimSpace(ctx.Query("organizer_id")); oVal != "" {
		if oID, err := uuid.Parse(oVal); err == nil && oID != uuid.Nil {
			organizerID = &oID
		}
	}

	result, err := h.settlementUsecase.ListAdminSettlements(ctx.Request.Context(), page, limit, status, organizerID)
	if err != nil {
		h.logger.ErrorContext(ctx.Request.Context(), "admin_list_settlements_failed", slog.Any("error", err))
		ctx.Error(appErrors.NewInternalError("failed to list settlements"))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetSettlement handles GET /api/v1/admin/settlements/:id
func (h *AdminSettlementHandler) GetSettlement(ctx *gin.Context) {
	idParam := strings.TrimSpace(ctx.Param("id"))
	settlementID, err := uuid.Parse(idParam)
	if err != nil || settlementID == uuid.Nil {
		ctx.Error(appErrors.NewValidationError("invalid settlement ID"))
		return
	}

	settlement, err := h.settlementUsecase.GetAdminSettlement(ctx.Request.Context(), settlementID)
	if err != nil {
		if errors.Is(err, domain.ErrSettlementNotFound) {
			ctx.Error(appErrors.NewNotFoundError("settlement not found"))
			return
		}
		h.logger.ErrorContext(ctx.Request.Context(), "admin_get_settlement_failed", slog.Any("error", err))
		ctx.Error(appErrors.NewInternalError("failed to get settlement details"))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settlement,
	})
}

// ApproveSettlement handles POST /api/v1/admin/settlements/:id/approve
func (h *AdminSettlementHandler) ApproveSettlement(ctx *gin.Context) {
	adminUserID, err := h.getAdminUserID(ctx)
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

	settlement, err := h.settlementUsecase.ApproveSettlement(ctx.Request.Context(), adminUserID, settlementID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSettlementNotFound):
			ctx.Error(appErrors.NewNotFoundError("settlement not found"))
		case errors.Is(err, domain.ErrInvalidSettlementStatusTransition):
			ctx.Error(appErrors.NewConflictError("settlement is not in a state that can be approved"))
		case errors.Is(err, domain.ErrPayoutFailed):
			ctx.JSON(http.StatusBadGateway, gin.H{
				"success": false,
				"error":   "payout execution failed; settlement marked FAILED and can be retried",
				"data":    settlement,
			})
		default:
			h.logger.ErrorContext(ctx.Request.Context(), "admin_approve_settlement_failed", slog.Any("error", err))
			ctx.Error(appErrors.NewInternalError("failed to approve settlement"))
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settlement,
	})
}

type RejectSettlementRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// RejectSettlement handles POST /api/v1/admin/settlements/:id/reject
func (h *AdminSettlementHandler) RejectSettlement(ctx *gin.Context) {
	adminUserID, err := h.getAdminUserID(ctx)
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

	var req RejectSettlementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		ctx.Error(appErrors.NewValidationError("rejection reason is required"))
		return
	}

	settlement, err := h.settlementUsecase.RejectSettlement(ctx.Request.Context(), adminUserID, settlementID, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSettlementNotFound):
			ctx.Error(appErrors.NewNotFoundError("settlement not found"))
		case errors.Is(err, domain.ErrInvalidSettlementStatusTransition):
			ctx.Error(appErrors.NewConflictError("settlement is not in a state that can be rejected"))
		case errors.Is(err, domain.ErrRejectionReasonRequired):
			ctx.Error(appErrors.NewValidationError(err.Error()))
		default:
			h.logger.ErrorContext(ctx.Request.Context(), "admin_reject_settlement_failed", slog.Any("error", err))
			ctx.Error(appErrors.NewInternalError("failed to reject settlement"))
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settlement,
	})
}

// RetrySettlement handles POST /api/v1/admin/settlements/:id/retry
func (h *AdminSettlementHandler) RetrySettlement(ctx *gin.Context) {
	adminUserID, err := h.getAdminUserID(ctx)
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

	settlement, err := h.settlementUsecase.RetrySettlement(ctx.Request.Context(), adminUserID, settlementID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSettlementNotFound):
			ctx.Error(appErrors.NewNotFoundError("settlement not found"))
		case errors.Is(err, domain.ErrInvalidSettlementStatusTransition):
			ctx.Error(appErrors.NewConflictError("settlement is not in FAILED state and cannot be retried"))
		case errors.Is(err, domain.ErrPayoutFailed):
			ctx.JSON(http.StatusBadGateway, gin.H{
				"success": false,
				"error":   "payout retry execution failed; settlement remains FAILED",
				"data":    settlement,
			})
		default:
			h.logger.ErrorContext(ctx.Request.Context(), "admin_retry_settlement_failed", slog.Any("error", err))
			ctx.Error(appErrors.NewInternalError("failed to retry settlement"))
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settlement,
	})
}
