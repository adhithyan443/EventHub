package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventCancellationRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewEventCancellationRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *EventCancellationRepository {
	return &EventCancellationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *EventCancellationRepository) Create(
	cancellation *domain.EventCancellation,
) error {
	cancellationModel := toEventCancellationModel(cancellation)

	if err := r.db.Create(cancellationModel).Error; err != nil {
		r.logger.Error(
			"event_cancellation_create_failed",
			"event_id", cancellation.EventID,
			"error", err,
		)

		return err
	}

	*cancellation = *toEventCancellationDomain(cancellationModel)

	r.logger.Info(
		"event_cancellation_created",
		"event_id", cancellation.EventID,
		"cancellation_id", cancellation.ID,
	)

	return nil
}

func (r *EventCancellationRepository) FindByEventID(
	eventID uuid.UUID,
) (*domain.EventCancellation, error) {
	var cancellationModel models.EventCancellationModel

	if err := r.db.
		Where("event_id = ?", eventID).
		First(&cancellationModel).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEventNotFound
		}

		r.logger.Error(
			"event_cancellation_find_by_event_id_failed",
			"event_id", eventID,
			"error", err,
		)

		return nil, err
	}

	return toEventCancellationDomain(&cancellationModel), nil
}

func toEventCancellationModel(
	cancellation *domain.EventCancellation,
) *models.EventCancellationModel {
	return &models.EventCancellationModel{
		ID:                        cancellation.ID,
		EventID:                   cancellation.EventID,
		CancellationAllowed:       cancellation.CancellationAllowed,
		CancellationDeadlineHours: cancellation.CancellationDeadlineHours,
		CancelledAt:               cancellation.CancelledAt,
		CancellationReason:        cancellation.CancellationReason,
		CreatedAt:                 cancellation.CreatedAt,
		UpdatedAt:                 cancellation.UpdatedAt,
	}
}

func toEventCancellationDomain(
	cancellationModel *models.EventCancellationModel,
) *domain.EventCancellation {
	return &domain.EventCancellation{
		ID:                        cancellationModel.ID,
		EventID:                   cancellationModel.EventID,
		CancellationAllowed:       cancellationModel.CancellationAllowed,
		CancellationDeadlineHours: cancellationModel.CancellationDeadlineHours,
		CancelledAt:               cancellationModel.CancelledAt,
		CancellationReason:        cancellationModel.CancellationReason,
		CreatedAt:                 cancellationModel.CreatedAt,
		UpdatedAt:                 cancellationModel.UpdatedAt,
	}
}
