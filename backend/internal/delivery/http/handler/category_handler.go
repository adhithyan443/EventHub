package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/category"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CategoryHandler struct {
	categoryUsecase *category.CategoryUsecase
	logger          *slog.Logger
}

func NewCategoryHandler(
	categoryUsecase *category.CategoryUsecase,
	logger ...*slog.Logger,
) *CategoryHandler {
	var l *slog.Logger
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	} else {
		l = slog.Default()
	}
	return &CategoryHandler{
		categoryUsecase: categoryUsecase,
		logger:          l,
	}
}

func (h *CategoryHandler) GetCategories(ctx *gin.Context) {
	categories, err := h.categoryUsecase.GetActiveCategories()
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"categories": categories,
		},
	})
}

type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ToggleCategoryStatusRequest struct {
	Status string `json:"status"`
}

func (h *CategoryHandler) AdminListCategories(ctx *gin.Context) {
	page := 1
	limit := 10

	if value := strings.TrimSpace(ctx.Query("page")); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 {
			_ = ctx.Error(appErrors.NewValidationError("invalid page parameter"))
			return
		}
		page = parsedPage
	}

	if value := strings.TrimSpace(ctx.Query("limit")); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			_ = ctx.Error(appErrors.NewValidationError("invalid limit parameter, must be between 1 and 100"))
			return
		}
		limit = parsedLimit
	}

	filter := domain.CategoryListFilter{
		Page:   page,
		Limit:  limit,
		Search: strings.TrimSpace(ctx.Query("search")),
		Status: strings.TrimSpace(ctx.Query("status")),
	}

	result, err := h.categoryUsecase.AdminListCategories(ctx.Request.Context(), filter)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func (h *CategoryHandler) AdminCreateCategory(ctx *gin.Context) {
	var req CreateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		_ = ctx.Error(appErrors.NewValidationError("category name is required"))
		return
	}

	created, err := h.categoryUsecase.AdminCreateCategory(ctx.Request.Context(), req.Name, req.Description)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNameDuplicate) {
			_ = ctx.Error(appErrors.NewConflictError("category name already exists"))
			return
		}
		if errors.Is(err, domain.ErrCategoryInvalidName) {
			_ = ctx.Error(appErrors.NewValidationError("invalid category name, must be between 2 and 100 characters"))
			return
		}
		_ = ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Category created successfully",
		"data":    created,
	})
}

func (h *CategoryHandler) AdminUpdateCategory(ctx *gin.Context) {
	idParam := strings.TrimSpace(ctx.Param("id"))
	categoryID, err := uuid.Parse(idParam)
	if err != nil {
		_ = ctx.Error(appErrors.NewValidationError("invalid category id"))
		return
	}

	var req UpdateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		_ = ctx.Error(appErrors.NewValidationError("category name is required"))
		return
	}

	updated, err := h.categoryUsecase.AdminUpdateCategory(ctx.Request.Context(), categoryID, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			_ = ctx.Error(appErrors.NewNotFoundError("category not found"))
			return
		}
		if errors.Is(err, domain.ErrCategoryNameDuplicate) {
			_ = ctx.Error(appErrors.NewConflictError("category name already exists"))
			return
		}
		if errors.Is(err, domain.ErrCategoryInvalidName) {
			_ = ctx.Error(appErrors.NewValidationError("invalid category name, must be between 2 and 100 characters"))
			return
		}
		_ = ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category updated successfully",
		"data":    updated,
	})
}

func (h *CategoryHandler) AdminToggleStatus(ctx *gin.Context) {
	idParam := strings.TrimSpace(ctx.Param("id"))
	categoryID, err := uuid.Parse(idParam)
	if err != nil {
		_ = ctx.Error(appErrors.NewValidationError("invalid category id"))
		return
	}

	var req ToggleCategoryStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	if strings.TrimSpace(req.Status) == "" {
		_ = ctx.Error(appErrors.NewValidationError("status is required"))
		return
	}

	updated, err := h.categoryUsecase.AdminToggleCategoryStatus(ctx.Request.Context(), categoryID, req.Status)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			_ = ctx.Error(appErrors.NewNotFoundError("category not found"))
			return
		}
		if errors.Is(err, domain.ErrCategoryInvalidStatus) {
			_ = ctx.Error(appErrors.NewValidationError("invalid category status, must be ACTIVE or INACTIVE"))
			return
		}
		_ = ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category status updated successfully",
		"data":    updated,
	})
}
