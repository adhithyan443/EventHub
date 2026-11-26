package repository

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
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

func (r *CategoryRepository) FindByID(id uuid.UUID) (*domain.Category, error) {
	var categoryModel models.CategoryModel

	if err := r.db.
		Where("id = ?", id).
		First(&categoryModel).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCategoryNotFound
		}

		r.logger.Error(
			"category_find_by_id_failed",
			"category_id", id,
			"error", err,
		)

		return nil, err
	}

	return toCategoryDomain(&categoryModel), nil
}

type categoryRowWithCount struct {
	ID          uuid.UUID `gorm:"column:id"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	Status      string    `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
	EventCount  int64     `gorm:"column:event_count"`
}

func (r *CategoryRepository) FindAll(filter domain.CategoryListFilter) ([]*domain.CategoryWithStats, int64, error) {
	countQuery := r.db.Table("categories AS c")
	dataQuery := r.db.Table("categories AS c").
		Select(`c.id, c.name, c.description, c.status, c.created_at, c.updated_at,
			(SELECT COUNT(1) FROM events AS e WHERE e.category_id = c.id) AS event_count`)

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		condition := "(LOWER(c.name) LIKE ? OR LOWER(COALESCE(c.description, '')) LIKE ?)"
		countQuery = countQuery.Where(condition, searchPattern, searchPattern)
		dataQuery = dataQuery.Where(condition, searchPattern, searchPattern)
	}

	if filter.Status != "" && strings.ToUpper(strings.TrimSpace(filter.Status)) != "ALL" {
		status := strings.ToUpper(strings.TrimSpace(filter.Status))
		countQuery = countQuery.Where("c.status = ?", status)
		dataQuery = dataQuery.Where("c.status = ?", status)
	}

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		r.logger.Error("categories_count_failed", "error", err)
		return nil, 0, err
	}

	dataQuery = dataQuery.Order("c.name ASC")

	if filter.Limit > 0 {
		page := filter.Page
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * filter.Limit
		dataQuery = dataQuery.Limit(filter.Limit).Offset(offset)
	}

	var rows []categoryRowWithCount
	if err := dataQuery.Scan(&rows).Error; err != nil {
		r.logger.Error("categories_fetch_failed", "error", err)
		return nil, 0, err
	}

	result := make([]*domain.CategoryWithStats, 0, len(rows))
	for _, row := range rows {
		result = append(result, &domain.CategoryWithStats{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			Status:      row.Status,
			EventCount:  row.EventCount,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	return result, total, nil
}

func (r *CategoryRepository) FindByName(name string) (*domain.Category, error) {
	var categoryModel models.CategoryModel

	err := r.db.
		Where("LOWER(TRIM(name)) = LOWER(?)", strings.TrimSpace(name)).
		First(&categoryModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCategoryNotFound
		}

		r.logger.Error("category_find_by_name_failed", "name", name, "error", err)
		return nil, err
	}

	return toCategoryDomain(&categoryModel), nil
}

func (r *CategoryRepository) Create(category *domain.Category) error {
	now := time.Now()
	if category.ID == uuid.Nil {
		category.ID = uuid.New()
	}
	if category.CreatedAt.IsZero() {
		category.CreatedAt = now
	}
	if category.UpdatedAt.IsZero() {
		category.UpdatedAt = now
	}

	model := &models.CategoryModel{
		ID:          category.ID,
		Name:        strings.TrimSpace(category.Name),
		Description: strings.TrimSpace(category.Description),
		Status:      category.Status,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}

	if err := r.db.Create(model).Error; err != nil {
		r.logger.Error("category_create_failed", "name", category.Name, "error", err)
		return err
	}

	return nil
}

func (r *CategoryRepository) Update(category *domain.Category) error {
	updates := map[string]interface{}{
		"name":        strings.TrimSpace(category.Name),
		"description": strings.TrimSpace(category.Description),
		"updated_at":  time.Now(),
	}

	res := r.db.Model(&models.CategoryModel{}).
		Where("id = ?", category.ID).
		Updates(updates)

	if res.Error != nil {
		r.logger.Error("category_update_failed", "category_id", category.ID, "error", res.Error)
		return res.Error
	}

	if res.RowsAffected == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}

func (r *CategoryRepository) UpdateStatus(id uuid.UUID, status string) error {
	res := r.db.Model(&models.CategoryModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if res.Error != nil {
		r.logger.Error("category_update_status_failed", "category_id", id, "status", status, "error", res.Error)
		return res.Error
	}

	if res.RowsAffected == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}

func (r *CategoryRepository) CountEventsByCategoryID(id uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Table("events").Where("category_id = ?", id).Count(&count).Error
	if err != nil {
		r.logger.Error("category_count_events_failed", "category_id", id, "error", err)
		return 0, err
	}
	return count, nil
}

func (r *CategoryRepository) GetStatistics() (*domain.CategoryStatistics, error) {
	var total, active, inactive, inUse int64

	if err := r.db.Model(&models.CategoryModel{}).Count(&total).Error; err != nil {
		r.logger.Error("category_stats_total_failed", "error", err)
		return nil, err
	}

	if err := r.db.Model(&models.CategoryModel{}).Where("status = ?", domain.CategoryStatusActive).Count(&active).Error; err != nil {
		r.logger.Error("category_stats_active_failed", "error", err)
		return nil, err
	}

	if err := r.db.Model(&models.CategoryModel{}).Where("status = ?", domain.CategoryStatusInactive).Count(&inactive).Error; err != nil {
		r.logger.Error("category_stats_inactive_failed", "error", err)
		return nil, err
	}

	if err := r.db.Table("events").Distinct("category_id").Count(&inUse).Error; err != nil {
		r.logger.Error("category_stats_in_use_failed", "error", err)
		return nil, err
	}

	return &domain.CategoryStatistics{
		Total:    total,
		Active:   active,
		Inactive: inactive,
		InUse:    inUse,
	}, nil
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
