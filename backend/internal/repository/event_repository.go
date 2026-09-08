package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewEventRepository(db *gorm.DB, logger *slog.Logger) *EventRepository {
	return &EventRepository{
		db:     db,
		logger: logger,
	}
}

func (r *EventRepository) Create(event *domain.Event) error {
	eventModel := toEventModel(event)

	if err := r.db.Create(eventModel).Error; err != nil {
		r.logger.Error(
			"event_create_failed",
			"event_id", event.ID,
			"organizer_id", event.OrganizerID,
			"error", err,
		)

		return err
	}

	*event = *toEventDomain(eventModel)

	r.logger.Info(
		"event_created",
		"event_id", event.ID,
		"organizer_id", event.OrganizerID,
	)

	return nil
}

func (r *EventRepository) FindByID(id uuid.UUID) (*domain.Event, error) {
	var eventModel models.EventModel

	if err := r.db.
		Where("id = ?", id).
		First(&eventModel).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEventNotFound
		}

		r.logger.Error(
			"event_find_by_id_failed",
			"event_id", id,
			"error", err,
		)

		return nil, err
	}

	return toEventDomain(&eventModel), nil
}

func toEventModel(event *domain.Event) *models.EventModel {
	return &models.EventModel{
		ID:             event.ID,
		OrganizerID:    event.OrganizerID,
		CategoryID:     event.CategoryID,
		VenueID:        event.VenueID,
		EventType:      event.EventType,
		OnlineURL:      event.OnlineURL,
		Title:          event.Title,
		Description:    event.Description,
		BannerURL:      event.BannerURL,
		Language:       event.Language,
		AgeRestriction: event.AgeRestriction,
		Status:         event.Status,
		CreatedAt:      event.CreatedAt,
		UpdatedAt:      event.UpdatedAt,
	}
}

func toEventDomain(
	eventModel *models.EventModel,
) *domain.Event {
	return &domain.Event{
		ID:             eventModel.ID,
		OrganizerID:    eventModel.OrganizerID,
		CategoryID:     eventModel.CategoryID,
		VenueID:        eventModel.VenueID,
		EventType:      eventModel.EventType,
		OnlineURL:      eventModel.OnlineURL,
		Title:          eventModel.Title,
		Description:    eventModel.Description,
		BannerURL:      eventModel.BannerURL,
		Language:       eventModel.Language,
		AgeRestriction: eventModel.AgeRestriction,
		Status:         eventModel.Status,
		CreatedAt:      eventModel.CreatedAt,
		UpdatedAt:      eventModel.UpdatedAt,
	}
}
