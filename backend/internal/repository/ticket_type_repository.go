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

type TicketTypeRepository interface {
	Create(ctx context.Context, ticketType *domain.TicketType) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error)
	FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.TicketType, error)
	Update(ctx context.Context, ticketType *domain.TicketType) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByEventID(ctx context.Context, eventID uuid.UUID) error
}

type ticketTypeRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewTicketTypeRepository(db *gorm.DB, logger *slog.Logger) TicketTypeRepository {
	return &ticketTypeRepository{
		db:     db,
		logger: logger,
	}
}

func (r *ticketTypeRepository) Create(
	ctx context.Context,
	ticketType *domain.TicketType,
) error {
	model := models.TicketTypeModelFromDomain(ticketType)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create ticket type",
			slog.String("event_id", ticketType.EventID.String()),
			slog.String("ticket_type_id", ticketType.ID.String()),
			slog.String("error", err.Error()),
		)

		return err
	}

	r.logger.InfoContext(ctx, "ticket type created",
		slog.String("event_id", ticketType.EventID.String()),
		slog.String("ticket_type_id", ticketType.ID.String()),
	)

	return nil
}

func (r *ticketTypeRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.TicketType, error) {
	var model models.TicketTypeModel

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrTicketTypeNotFound
	}

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to find ticket type",
			slog.String("ticket_type_id", id.String()),
			slog.String("error", err.Error()),
		)

		return nil, err
	}

	return model.ToDomain(), nil
}

func (r *ticketTypeRepository) FindByEventID(
	ctx context.Context,
	eventID uuid.UUID,
) ([]*domain.TicketType, error) {
	var ticketTypeModels []models.TicketTypeModel

	if err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Order("created_at ASC").
		Find(&ticketTypeModels).Error; err != nil {

		r.logger.ErrorContext(ctx, "failed to find ticket types by event",
			slog.String("event_id", eventID.String()),
			slog.String("error", err.Error()),
		)

		return nil, err
	}

	ticketTypes := make([]*domain.TicketType, 0, len(ticketTypeModels))

	for i := range ticketTypeModels {
		ticketTypes = append(ticketTypes, ticketTypeModels[i].ToDomain())
	}

	return ticketTypes, nil
}

func (r *ticketTypeRepository) Update(
	ctx context.Context,
	ticketType *domain.TicketType,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.TicketTypeModel{}).
		Where("id = ?", ticketType.ID).
		Updates(map[string]interface{}{
			"name":               ticketType.Name,
			"price":              ticketType.Price,
			"total_quantity":     ticketType.TotalQuantity,
			"available_quantity": ticketType.AvailableQuantity,
			"description":        ticketType.Description,
			"updated_at":         ticketType.UpdatedAt,
		})

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to update ticket type",
			slog.String("ticket_type_id", ticketType.ID.String()),
			slog.String("error", result.Error.Error()),
		)

		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrTicketTypeNotFound
	}

	return nil
}

func (r *ticketTypeRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	result := r.db.WithContext(ctx).
		Delete(&models.TicketTypeModel{}, "id = ?", id)

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to delete ticket type",
			slog.String("ticket_type_id", id.String()),
			slog.String("error", result.Error.Error()),
		)

		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrTicketTypeNotFound
	}

	return nil
}

func (r *ticketTypeRepository) DeleteByEventID(
	ctx context.Context,
	eventID uuid.UUID,
) error {
	if err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Delete(&models.TicketTypeModel{}).Error; err != nil {

		r.logger.ErrorContext(ctx, "failed to delete ticket types by event",
			slog.String("event_id", eventID.String()),
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}
