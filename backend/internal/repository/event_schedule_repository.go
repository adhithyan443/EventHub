package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventScheduleRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewEventScheduleRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *EventScheduleRepository {
	return &EventScheduleRepository{
		db:     db,
		logger: logger,
	}
}

func (r *EventScheduleRepository) Create(
	schedule *domain.EventSchedule,
) error {
	scheduleModel := toEventScheduleModel(schedule)

	if err := r.db.Create(scheduleModel).Error; err != nil {
		r.logger.Error(
			"event_schedule_create_failed",
			"event_id", schedule.EventID,
			"error", err,
		)

		return err
	}

	*schedule = *toEventScheduleDomain(scheduleModel)

	r.logger.Info(
		"event_schedule_created",
		"event_id", schedule.EventID,
		"schedule_id", schedule.ID,
	)

	return nil
}

func (r *EventScheduleRepository) FindByEventID(
	eventID uuid.UUID,
) (*domain.EventSchedule, error) {
	var scheduleModel models.EventScheduleModel

	if err := r.db.
		Where("event_id = ?", eventID).
		First(&scheduleModel).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEventNotFound
		}

		r.logger.Error(
			"event_schedule_find_by_event_id_failed",
			"event_id", eventID,
			"error", err,
		)

		return nil, err
	}

	return toEventScheduleDomain(&scheduleModel), nil
}

func toEventScheduleModel(
	schedule *domain.EventSchedule,
) *models.EventScheduleModel {
	return &models.EventScheduleModel{
		ID:        schedule.ID,
		EventID:   schedule.EventID,
		EventDate: schedule.EventDate,
		StartTime: schedule.StartTime,
		EndTime:   schedule.EndTime,
		CreatedAt: schedule.CreatedAt,
		UpdatedAt: schedule.UpdatedAt,
	}
}

func toEventScheduleDomain(
	scheduleModel *models.EventScheduleModel,
) *domain.EventSchedule {
	return &domain.EventSchedule{
		ID:        scheduleModel.ID,
		EventID:   scheduleModel.EventID,
		EventDate: scheduleModel.EventDate,
		StartTime: scheduleModel.StartTime,
		EndTime:   scheduleModel.EndTime,
		CreatedAt: scheduleModel.CreatedAt,
		UpdatedAt: scheduleModel.UpdatedAt,
	}
}