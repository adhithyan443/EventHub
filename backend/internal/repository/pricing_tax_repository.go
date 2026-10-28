package repository

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PricingTaxRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewPricingTaxRepository(db *gorm.DB, logger *slog.Logger) *PricingTaxRepository {
	return &PricingTaxRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PricingTaxRepository) FindActiveAt(ctx context.Context, t time.Time) (*domain.PricingTaxConfiguration, error) {
	var m models.PricingTaxModel
	err := r.db.WithContext(ctx).
		Where("status != ?", domain.PricingTaxStatusCancelled).
		Where("effective_from <= ?", t).
		Where("effective_to IS NULL OR effective_to > ?", t).
		Order("effective_from DESC, version DESC").
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPricingConfigurationUnavailable
		}
		r.logger.ErrorContext(ctx, "failed to query active pricing configuration",
			slog.Time("timestamp", t),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	return m.ToDomain(), nil
}

func (r *PricingTaxRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.PricingTaxConfiguration, error) {
	var m models.PricingTaxModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPricingConfigurationUnavailable
		}
		return nil, err
	}
	return m.ToDomain(), nil
}

func (r *PricingTaxRepository) FindLatest(ctx context.Context) (*domain.PricingTaxConfiguration, error) {
	var m models.PricingTaxModel
	err := r.db.WithContext(ctx).Order("version DESC").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPricingConfigurationUnavailable
		}
		return nil, err
	}
	return m.ToDomain(), nil
}

func (r *PricingTaxRepository) FindTimeline(ctx context.Context) ([]domain.PricingTaxConfiguration, error) {
	var list []models.PricingTaxModel
	err := r.db.WithContext(ctx).
		Where("status != ?", domain.PricingTaxStatusCancelled).
		Order("effective_from ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}

	result := make([]domain.PricingTaxConfiguration, len(list))
	for i := range list {
		result[i] = *list[i].ToDomain()
	}
	return result, nil
}

func (r *PricingTaxRepository) FindHistory(ctx context.Context, page, limit int) ([]domain.PricingTaxConfiguration, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int64
	if err := r.db.WithContext(ctx).Model(&models.PricingTaxModel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []models.PricingTaxModel
	err := r.db.WithContext(ctx).
		Order("version DESC").
		Offset(offset).
		Limit(limit).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.PricingTaxConfiguration, len(list))
	for i := range list {
		result[i] = *list[i].ToDomain()
	}
	return result, total, nil
}

func (r *PricingTaxRepository) Create(ctx context.Context, config *domain.PricingTaxConfiguration) error {
	m := models.PricingTaxModelFromDomain(config)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create pricing configuration",
			slog.Int("version", config.Version),
			slog.String("error", err.Error()),
		)
		return err
	}
	return nil
}

func (r *PricingTaxRepository) Update(ctx context.Context, config *domain.PricingTaxConfiguration) error {
	m := models.PricingTaxModelFromDomain(config)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to update pricing configuration",
			slog.String("id", config.ID.String()),
			slog.Int("version", config.Version),
			slog.String("error", err.Error()),
		)
		return err
	}
	return nil
}

func (r *PricingTaxRepository) AcquireAdvisoryLock(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?)", domain.PricingTaxAdvisoryLockID).Error
}
