package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/gin-gonic/gin"
)

type AdminDashboardHandler struct {
	dashboardUsecase domain.AdminDashboardUsecase
	logger           *slog.Logger
}

// NewAdminDashboardHandler creates a new AdminDashboardHandler.
func NewAdminDashboardHandler(
	dashboardUsecase domain.AdminDashboardUsecase,
	logger *slog.Logger,
) *AdminDashboardHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminDashboardHandler{
		dashboardUsecase: dashboardUsecase,
		logger:           logger,
	}
}

// GetDashboard handles GET /api/v1/admin/dashboard.
func (h *AdminDashboardHandler) GetDashboard(ctx *gin.Context) {
	var filter domain.AdminDashboardFilter

	if fromParam := strings.TrimSpace(ctx.Query("from")); fromParam != "" {
		fromTime, err := parseDashboardDateParam(fromParam, false)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid from date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.From = &fromTime
	}

	if toParam := strings.TrimSpace(ctx.Query("to")); toParam != "" {
		toTime, err := parseDashboardDateParam(toParam, true)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid to date parameter, expected RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.To = &toTime
	}

	data, err := h.dashboardUsecase.GetDashboardData(ctx.Request.Context(), filter)
	if err != nil {
		h.logger.ErrorContext(ctx.Request.Context(), "admin_get_dashboard_failed", slog.Any("error", err))
		if errors.Is(err, domain.ErrInvalidDateRange) {
			ctx.Error(appErrors.NewValidationError(err.Error()))
			return
		}
		ctx.Error(appErrors.NewInternalError("failed to retrieve dashboard data"))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

func parseDashboardDateParam(val string, endOfDay bool) (time.Time, error) {
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
