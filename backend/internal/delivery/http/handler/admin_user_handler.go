package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/user"
	"github.com/gin-gonic/gin"
)

type AdminUserHandler struct {
	adminUserUsecase *user.AdminUserUsecase
	logger           *slog.Logger
}

func NewAdminUserHandler(
	adminUserUsecase *user.AdminUserUsecase,
	logger *slog.Logger,
) *AdminUserHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminUserHandler{
		adminUserUsecase: adminUserUsecase,
		logger:           logger,
	}
}

func (h *AdminUserHandler) ListUsers(ctx *gin.Context) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 {
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

	search := ctx.Query("search")
	role := ctx.Query("role")
	status := ctx.Query("status")

	result, err := h.adminUserUsecase.ListUsers(page, limit, search, role, status)
	if err != nil {
		ctx.Error(appErrors.NewValidationError(err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
