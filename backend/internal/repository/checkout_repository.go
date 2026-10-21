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

type checkoutRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewCheckoutRepository(
	db *gorm.DB,
	logger *slog.Logger,
) domain.CheckoutRepository {
	return &checkoutRepository{
		db:     db,
		logger: logger,
	}
}

func (r *checkoutRepository) Create(
	ctx context.Context,
	checkout *domain.Checkout,
) error {
	model := models.CheckoutModelFromDomain(checkout)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create checkout",
			slog.String("checkout_id", checkout.ID.String()),
			slog.String("reservation_id", checkout.ReservationID.String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	r.logger.InfoContext(ctx, "checkout created",
		slog.String("checkout_id", checkout.ID.String()),
		slog.String("reservation_id", checkout.ReservationID.String()),
		slog.String("status", checkout.Status),
	)

	return nil
}

func (r *checkoutRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Checkout, error) {
	var model models.CheckoutModel

	err := r.db.WithContext(ctx).
		First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCheckoutNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find checkout",
			slog.String("checkout_id", id.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return model.ToDomain(), nil
}

func (r *checkoutRepository) FindByReservationID(
	ctx context.Context,
	reservationID uuid.UUID,
) (*domain.Checkout, error) {
	var model models.CheckoutModel

	err := r.db.WithContext(ctx).
		Where("reservation_id = ?", reservationID).
		Order("created_at DESC").
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCheckoutNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find checkout by reservation",
			slog.String("reservation_id", reservationID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return model.ToDomain(), nil
}

func (r *checkoutRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status string,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.CheckoutModel{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to update checkout status",
			slog.String("checkout_id", id.String()),
			slog.String("status", status),
			slog.String("error", result.Error.Error()),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrCheckoutNotFound
	}

	r.logger.InfoContext(ctx, "checkout status updated",
		slog.String("checkout_id", id.String()),
		slog.String("status", status),
	)

	return nil
}

func (r *checkoutRepository) ExpireActiveByReservationID(
	ctx context.Context,
	reservationID uuid.UUID,
) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&models.CheckoutModel{}).
		Where(
			"reservation_id = ? AND status = ?",
			reservationID,
			domain.CheckoutStatusActive,
		).
		Updates(map[string]interface{}{
			"status":     domain.CheckoutStatusExpired,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to expire checkout",
			slog.String("reservation_id", reservationID.String()),
			slog.String("error", result.Error.Error()),
		)
		return false, result.Error
	}

	if result.RowsAffected == 0 {
		return false, nil
	}

	r.logger.InfoContext(ctx, "checkout expired",
		slog.String("reservation_id", reservationID.String()),
	)

	return true, nil
}
