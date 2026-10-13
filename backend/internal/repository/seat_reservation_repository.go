package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type seatReservationRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewSeatReservationRepository(
	db *gorm.DB,
	logger *slog.Logger,
) domain.SeatReservationRepository {
	return &seatReservationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *seatReservationRepository) Create(
	ctx context.Context,
	reservation *domain.SeatReservation,
) error {
	model := models.SeatReservationModelFromDomain(reservation)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create reservation",
			"reservation_id", reservation.ID,
			"error", err,
		)
		return err
	}

	r.logger.InfoContext(ctx, "reservation created",
		"reservation_id", reservation.ID,
		"event_id", reservation.EventID,
		"status", reservation.Status,
	)

	return nil
}

func (r *seatReservationRepository) CreateItem(
	ctx context.Context,
	item *domain.SeatReservationItem,
) error {
	model := models.SeatReservationItemModelFromDomain(item)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create reservation item",
			"reservation_id", item.ReservationID,
			"reservation_item_id", item.ID,
			"error", err,
		)
		return err
	}

	return nil
}

func (r *seatReservationRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.SeatReservation, error) {
	var model models.SeatReservationModel

	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSeatReservationNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find reservation",
			"reservation_id", id,
			"error", err,
		)
		return nil, err
	}

	return model.ToDomain(), nil
}

func (r *seatReservationRepository) FindItemsByReservationID(
	ctx context.Context,
	reservationID uuid.UUID,
) ([]domain.SeatReservationItem, error) {
	var modelsList []models.SeatReservationItemModel

	err := r.db.WithContext(ctx).
		Where("reservation_id = ?", reservationID).
		Find(&modelsList).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to find reservation items",
			"reservation_id", reservationID,
			"error", err,
		)
		return nil, err
	}

	items := make([]domain.SeatReservationItem, 0, len(modelsList))
	for i := range modelsList {
		items = append(items, *modelsList[i].ToDomain())
	}

	return items, nil
}

func (r *seatReservationRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status string,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.SeatReservationModel{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to update reservation status",
			"reservation_id", id,
			"error", result.Error,
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrSeatReservationNotFound
	}

	return nil
}
