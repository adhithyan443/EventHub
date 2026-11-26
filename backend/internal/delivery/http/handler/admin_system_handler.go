package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	systemmonitoring "github.com/adhithyan443/EventHub/backend/internal/usecase/system_monitoring"
	"github.com/gin-gonic/gin"
)

// AdminSystemHandler handles HTTP requests for admin system monitoring and application logs.
type AdminSystemHandler struct {
	usecase domain.SystemMonitoringUsecase
	logger  *slog.Logger
}

// NewAdminSystemHandler creates a new AdminSystemHandler instance.
func NewAdminSystemHandler(
	usecase domain.SystemMonitoringUsecase,
	logger *slog.Logger,
) *AdminSystemHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminSystemHandler{
		usecase: usecase,
		logger:  logger,
	}
}

// GetLogs handles GET /api/v1/admin/system/logs
func (h *AdminSystemHandler) GetLogs(ctx *gin.Context) {
	reqID := getRequestID(ctx)

	page := 1
	limit := 50

	if val := ctx.Query("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p >= 1 {
			page = p
		} else {
			ctx.Error(appErrors.NewValidationError("invalid page parameter, must be a positive integer"))
			return
		}
	}

	if val := ctx.Query("limit"); val != "" {
		if l, err := strconv.Atoi(val); err == nil && l >= 1 && l <= 200 {
			limit = l
		} else {
			ctx.Error(appErrors.NewValidationError("invalid limit parameter, must be between 1 and 200"))
			return
		}
	}

	filter := domain.SystemLogFilter{
		Level:  strings.TrimSpace(ctx.Query("level")),
		Search: strings.TrimSpace(ctx.Query("search")),
		Page:   page,
		Limit:  limit,
	}

	reqCtx := contextWithRequestID(ctx)
	result, err := h.usecase.GetLogs(reqCtx, filter)
	if err != nil {
		switch {
		case errors.Is(err, systemmonitoring.ErrInvalidLogLevel),
			errors.Is(err, systemmonitoring.ErrInvalidPagination):
			h.logger.WarnContext(reqCtx, "admin_get_logs_validation_error",
				slog.String("request_id", reqID),
				slog.Any("error", err),
			)
			ctx.Error(appErrors.NewValidationError(err.Error()))
		default:
			h.logger.ErrorContext(reqCtx, "admin_get_logs_internal_error",
				slog.String("request_id", reqID),
				slog.Any("error", err),
			)
			ctx.Error(appErrors.NewInternalError("failed to retrieve system logs"))
		}
		return
	}

	h.logger.InfoContext(reqCtx, "admin_system_logs_retrieved",
		slog.String("request_id", reqID),
		slog.Int("count", len(result.Logs)),
		slog.Int64("total_records", result.TotalRecords),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":          result.Logs,
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
