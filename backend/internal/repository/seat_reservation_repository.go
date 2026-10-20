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
	"gorm.io/gorm/clause"
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

func (r *seatReservationRepository) FindByIDForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (*domain.SeatReservation, error) {
	var model models.SeatReservationModel

	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&model, "id = ?", id).Error
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

func (r *seatReservationRepository) ExpireIfActive(
	ctx context.Context,
	id uuid.UUID,
	now time.Time,
) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&models.SeatReservationModel{}).
		Where(
			"id = ? AND status = ? AND expires_at <= ?",
			id,
			domain.ReservationStatusActive,
			now,
		).
		Updates(map[string]interface{}{
			"status":     domain.ReservationStatusExpired,
			"updated_at": now,
		})

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to expire reservation",
			slog.String("reservation_id", id.String()),
			slog.String("error", result.Error.Error()),
		)
		return false, result.Error
	}

	if result.RowsAffected == 0 {
		return false, nil
	}

	r.logger.InfoContext(ctx, "reservation expired",
		slog.String("reservation_id", id.String()),
	)

	return true, nil
}

func (r *seatReservationRepository) CancelIfActive(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
) (bool, error) {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&models.SeatReservationModel{}).
		Where(
			"id = ? AND user_id = ? AND status = ? AND expires_at > ?",
			id,
			userID,
			domain.ReservationStatusActive,
			now,
		).
		Updates(map[string]interface{}{
			"status":     domain.ReservationStatusCancelled,
			"updated_at": now,
		})

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to cancel reservation",
			slog.String("reservation_id", id.String()),
			slog.String("error", result.Error.Error()),
		)
		return false, result.Error
	}

	if result.RowsAffected == 0 {
		return false, nil
	}

	r.logger.InfoContext(ctx, "reservation cancelled",
		slog.String("reservation_id", id.String()),
		slog.String("user_id", userID.String()),
	)

	return true, nil
}

func (r *seatReservationRepository) FindExpiredActive(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]domain.SeatReservation, error) {
	if limit <= 0 {
		return nil, domain.ErrInvalidReservationRequest
	}

	var reservationModels []models.SeatReservationModel

	err := r.db.WithContext(ctx).
		Where(
			"status = ? AND expires_at <= ?",
			domain.ReservationStatusActive,
			now,
		).
		Order("expires_at ASC").
		Limit(limit).
		Find(&reservationModels).Error

	if err != nil {
		r.logger.ErrorContext(
			ctx,
			"failed to find expired reservations",
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	reservations := make(
		[]domain.SeatReservation,
		0,
		len(reservationModels),
	)

	for _, model := range reservationModels {
		reservations = append(reservations, domain.SeatReservation{
			ID:        model.ID,
			UserID:    model.UserID,
			EventID:   model.EventID,
			Status:    model.Status,
			ExpiresAt: model.ExpiresAt,
			CreatedAt: model.CreatedAt,
			UpdatedAt: model.UpdatedAt,
		})
	}

	return reservations, nil
}
