package repository

import (
	"errors"
	"log/slog"
	"time"

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

func (r *EventCancellationRepository) Update(
	cancellation *domain.EventCancellation,
) error {
	if cancellation == nil {
		return errors.New("event cancellation cannot be nil")
	}

	cancellationModel := toEventCancellationModel(cancellation)

	result := r.db.
		Model(&models.EventCancellationModel{}).
		Where("event_id = ?", cancellation.EventID).
		Updates(map[string]interface{}{
			"cancellation_allowed":        cancellationModel.CancellationAllowed,
			"cancellation_deadline_hours": cancellationModel.CancellationDeadlineHours,
			"refund_policy":               cancellationModel.RefundPolicy,
			"refund_percentage":           cancellationModel.RefundPercentage,
			"updated_at":                  time.Now(),
		})

	if result.Error != nil {
		r.logger.Error(
			"event_cancellation_update_failed",
			"event_id", cancellation.EventID,
			"error", result.Error,
		)

		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn(
			"event_cancellation_update_not_found",
			"event_id", cancellation.EventID,
		)

		return domain.ErrEventNotFound
	}

	r.logger.Info(
		"event_cancellation_updated",
		"event_id", cancellation.EventID,
		"cancellation_id", cancellation.ID,
	)

	return nil
}

func toEventCancellationModel(
	cancellation *domain.EventCancellation,
) *models.EventCancellationModel {
	return &models.EventCancellationModel{
		ID:                        cancellation.ID,
		EventID:                   cancellation.EventID,
		CancellationAllowed:       cancellation.CancellationAllowed,
		CancellationDeadlineHours: cancellation.CancellationDeadlineHours,
		RefundPolicy:              cancellation.RefundPolicy,
		RefundPercentage:          cancellation.RefundPercentage,
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
		RefundPolicy:              cancellationModel.RefundPolicy,
		RefundPercentage:          cancellationModel.RefundPercentage,
		CancelledAt:               cancellationModel.CancelledAt,
		CancellationReason:        cancellationModel.CancellationReason,
		CreatedAt:                 cancellationModel.CreatedAt,
		UpdatedAt:                 cancellationModel.UpdatedAt,
	}
}
