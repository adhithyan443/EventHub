package repository

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewCategoryRepository(db *gorm.DB, logger *slog.Logger) *CategoryRepository {
	return &CategoryRepository{
		db:     db,
		logger: logger,
	}
}

func (r *CategoryRepository) FindActive() ([]*domain.Category, error) {
	var categoryModels []models.CategoryModel

	err := r.db.
		Where("status = ?", domain.CategoryStatusActive).
		Order("name ASC").
		Find(&categoryModels).Error

	if err != nil {
		r.logger.Error(
			"active_categories_fetch_failed",
			"error", err,
		)

		return nil, err
	}

	categories := make([]*domain.Category, 0, len(categoryModels))

	for _, categoryModel := range categoryModels {
		categories = append(categories, toCategoryDomain(&categoryModel))
	}

	return categories, nil
}

func toCategoryDomain(model *models.CategoryModel) *domain.Category {
	return &domain.Category{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Status:      model.Status,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}
