package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/user"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminUserHandler struct {
	adminUserUsecase *user.AdminUserUsecase
	logger           *slog.Logger
}

type BlockUserRequest struct {
	Reason string `json:"reason"`
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

func (h *AdminUserHandler) GetUser(ctx *gin.Context) {
	idParam := strings.TrimSpace(ctx.Param("id"))
	userID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid user id"))
		return
	}

	result, err := h.adminUserUsecase.GetUser(userID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func (h *AdminUserHandler) BlockUser(ctx *gin.Context) {
	adminIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}
	adminID, ok := adminIDVal.(uuid.UUID)
	if !ok {
		ctx.Error(appErrors.NewUnauthorizedError("invalid admin session"))
		return
	}

	idParam := strings.TrimSpace(ctx.Param("id"))
	targetID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid user id"))
		return
	}

	var req BlockUserRequest
	// JSON body is optional, so ignore EOF/empty body errors
	_ = ctx.ShouldBindJSON(&req)

	result, err := h.adminUserUsecase.BlockUser(adminID, targetID, req.Reason)
	if err != nil {
		ctx.Error(err)
		return
	}

	// Immediately invalidate in-memory status cache for this user
	middleware.InvalidateUserStatusCache(targetID)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "user blocked successfully",
		"data":    result,
	})
}

func (h *AdminUserHandler) UnblockUser(ctx *gin.Context) {
	adminIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}
	adminID, ok := adminIDVal.(uuid.UUID)
	if !ok {
		ctx.Error(appErrors.NewUnauthorizedError("invalid admin session"))
		return
	}

	idParam := strings.TrimSpace(ctx.Param("id"))
	targetID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid user id"))
		return
	}

	result, err := h.adminUserUsecase.UnblockUser(adminID, targetID)
	if err != nil {
		ctx.Error(err)
		return
	}

	// Immediately invalidate in-memory status cache for this user
	middleware.InvalidateUserStatusCache(targetID)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "user unblocked successfully",
		"data":    result,
	})
}
