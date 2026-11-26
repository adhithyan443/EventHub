package category

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type CategoryUsecase struct {
	categoryRepository domain.CategoryRepository
	logger             *slog.Logger
}

func NewCategoryUsecase(
	categoryRepository domain.CategoryRepository,
	logger *slog.Logger,
) *CategoryUsecase {
	if logger == nil {
		logger = slog.Default()
	}
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

func (u *CategoryUsecase) AdminListCategories(ctx context.Context, filter domain.CategoryListFilter) (*domain.CategoryListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Status = strings.TrimSpace(filter.Status)

	categories, total, err := u.categoryRepository.FindAll(filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "admin_list_categories_failed", "error", err)
		return nil, err
	}

	stats, err := u.categoryRepository.GetStatistics()
	if err != nil {
		u.logger.ErrorContext(ctx, "admin_category_stats_failed", "error", err)
		return nil, err
	}

	u.logger.InfoContext(ctx, "admin_categories_listed",
		"total", total,
		"page", filter.Page,
		"limit", filter.Limit,
		"search", filter.Search,
		"status", filter.Status,
	)

	return &domain.CategoryListResult{
		Categories: categories,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		Statistics: *stats,
	}, nil
}

func (u *CategoryUsecase) AdminCreateCategory(ctx context.Context, name, description string) (*domain.Category, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" || len(trimmedName) < 2 || len(trimmedName) > 100 {
		u.logger.WarnContext(ctx, "category_create_invalid_name", "name", name)
		return nil, domain.ErrCategoryInvalidName
	}

	trimmedDesc := strings.TrimSpace(description)
	if len(trimmedDesc) > 500 {
		u.logger.WarnContext(ctx, "category_create_invalid_description_length", "len", len(trimmedDesc))
		return nil, domain.ErrCategoryInvalidName
	}

	existing, err := u.categoryRepository.FindByName(trimmedName)
	if err == nil && existing != nil {
		u.logger.WarnContext(ctx, "category_create_duplicate_name", "name", trimmedName)
		return nil, domain.ErrCategoryNameDuplicate
	}
	if err != nil && !errors.Is(err, domain.ErrCategoryNotFound) {
		u.logger.ErrorContext(ctx, "category_create_lookup_failed", "error", err)
		return nil, err
	}

	newCategory := &domain.Category{
		ID:          uuid.New(),
		Name:        trimmedName,
		Description: trimmedDesc,
		Status:      domain.CategoryStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := u.categoryRepository.Create(newCategory); err != nil {
		u.logger.ErrorContext(ctx, "category_create_failed", "name", trimmedName, "error", err)
		return nil, err
	}

	u.logger.InfoContext(ctx, "category_created",
		"category_id", newCategory.ID,
		"name", newCategory.Name,
	)

	return newCategory, nil
}

func (u *CategoryUsecase) AdminUpdateCategory(ctx context.Context, id uuid.UUID, name, description string) (*domain.Category, error) {
	if id == uuid.Nil {
		return nil, domain.ErrCategoryNotFound
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" || len(trimmedName) < 2 || len(trimmedName) > 100 {
		u.logger.WarnContext(ctx, "category_update_invalid_name", "category_id", id, "name", name)
		return nil, domain.ErrCategoryInvalidName
	}

	trimmedDesc := strings.TrimSpace(description)
	if len(trimmedDesc) > 500 {
		u.logger.WarnContext(ctx, "category_update_invalid_description_length", "category_id", id, "len", len(trimmedDesc))
		return nil, domain.ErrCategoryInvalidName
	}

	existing, err := u.categoryRepository.FindByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			u.logger.WarnContext(ctx, "category_update_not_found", "category_id", id)
		} else {
			u.logger.ErrorContext(ctx, "category_update_fetch_failed", "category_id", id, "error", err)
		}
		return nil, err
	}

	byName, err := u.categoryRepository.FindByName(trimmedName)
	if err == nil && byName != nil && byName.ID != id {
		u.logger.WarnContext(ctx, "category_update_duplicate_name", "category_id", id, "name", trimmedName)
		return nil, domain.ErrCategoryNameDuplicate
	}
	if err != nil && !errors.Is(err, domain.ErrCategoryNotFound) {
		u.logger.ErrorContext(ctx, "category_update_name_lookup_failed", "error", err)
		return nil, err
	}

	existing.Name = trimmedName
	existing.Description = trimmedDesc
	existing.UpdatedAt = time.Now()

	if err := u.categoryRepository.Update(existing); err != nil {
		u.logger.ErrorContext(ctx, "category_update_save_failed", "category_id", id, "error", err)
		return nil, err
	}

	u.logger.InfoContext(ctx, "category_updated",
		"category_id", existing.ID,
		"name", existing.Name,
	)

	return existing, nil
}

func (u *CategoryUsecase) AdminToggleCategoryStatus(ctx context.Context, id uuid.UUID, targetStatus string) (*domain.Category, error) {
	if id == uuid.Nil {
		return nil, domain.ErrCategoryNotFound
	}

	status := strings.ToUpper(strings.TrimSpace(targetStatus))
	if status != domain.CategoryStatusActive && status != domain.CategoryStatusInactive {
		u.logger.WarnContext(ctx, "category_toggle_invalid_status", "category_id", id, "status", targetStatus)
		return nil, domain.ErrCategoryInvalidStatus
	}

	existing, err := u.categoryRepository.FindByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			u.logger.WarnContext(ctx, "category_toggle_not_found", "category_id", id)
		} else {
			u.logger.ErrorContext(ctx, "category_toggle_fetch_failed", "category_id", id, "error", err)
		}
		return nil, err
	}

	oldStatus := existing.Status
	existing.Status = status
	existing.UpdatedAt = time.Now()

	if err := u.categoryRepository.UpdateStatus(id, status); err != nil {
		u.logger.ErrorContext(ctx, "category_toggle_save_failed", "category_id", id, "error", err)
		return nil, err
	}

	u.logger.InfoContext(ctx, "category_status_toggled",
		"category_id", id,
		"old_status", oldStatus,
		"new_status", status,
	)

	return existing, nil
}
