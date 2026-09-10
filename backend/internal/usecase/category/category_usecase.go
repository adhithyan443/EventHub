package category

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

type CategoryUsecase struct {
	categoryRepository domain.CategoryRepository
	logger             *slog.Logger
}

func NewCategoryUsecase(
	categoryRepository domain.CategoryRepository,
	logger *slog.Logger,
) *CategoryUsecase {
	return &CategoryUsecase{
		categoryRepository: categoryRepository,
		logger:             logger,
	}
}

func (u *CategoryUsecase) GetActiveCategories() ([]*domain.Category, error) {
	categories, err := u.categoryRepository.FindActive()
	if err != nil {
		return nil, err
	}

	u.logger.Info(
		"active_categories_fetched",
		"count", len(categories),
	)

	return categories, nil
}
