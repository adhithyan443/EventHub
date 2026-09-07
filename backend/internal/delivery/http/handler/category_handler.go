package handler

import (
	"net/http"

	"github.com/adhithyan443/EventHub/backend/internal/usecase/category"
	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	categoryUsecase *category.CategoryUsecase
}

func NewCategoryHandler(
	categoryUsecase *category.CategoryUsecase,
) *CategoryHandler {
	return &CategoryHandler{
		categoryUsecase: categoryUsecase,
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
