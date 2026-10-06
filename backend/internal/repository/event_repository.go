package repository

import (
	"errors"
	"log/slog"
	"strings"

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

func (r *EventRepository) Update(event *domain.Event) error {
	if event == nil {
		return errors.New("event cannot be nil")
	}

	model := toEventModel(event)

	result := r.db.
		Model(&models.EventModel{}).
		Where("id = ?", event.ID).
		Updates(map[string]interface{}{
			"organizer_id":         model.OrganizerID,
			"category_id":          model.CategoryID,
			"venue_id":             model.VenueID,
			"event_type":           model.EventType,
			"online_url":           model.OnlineURL,
			"title":                model.Title,
			"description":          model.Description,
			"banner_url":           model.BannerURL,
			"language":             model.Language,
			"age_restriction":      model.AgeRestriction,
			"visibility":           model.Visibility,
			"highlights":           model.Highlights,
			"rules":                model.Rules,
			"attendee_information": model.AttendeeInformation,
			"status":               model.Status,
			"updated_at":           model.UpdatedAt,
		})

	if result.Error != nil {
		r.logger.Error(
			"event_update_failed",
			"event_id", event.ID,
			"error", result.Error,
		)

		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn(
			"event_update_not_found",
			"event_id", event.ID,
		)

		return domain.ErrEventNotFound
	}

	r.logger.Info(
		"event_updated",
		"event_id", event.ID,
		"organizer_id", event.OrganizerID,
	)

	return nil
}

func (r *EventRepository) Delete(id uuid.UUID) error {
	result := r.db.Where("id = ?", id).Delete(&models.EventModel{})
	if result.Error != nil {
		r.logger.Error(
			"event_delete_failed",
			"event_id", id,
			"error", result.Error,
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn(
			"event_delete_not_found",
			"event_id", id,
		)
		return domain.ErrEventNotFound
	}

	r.logger.Info("event_deleted", "event_id", id)
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

func (r *EventRepository) FindByOrganizerID(
	organizerID uuid.UUID,
	page int,
	limit int,
	status string,
	search string,
) ([]*domain.OrganizerEvent, int64, error) {

	var events []*domain.OrganizerEvent
	var total int64

	offset := (page - 1) * limit

	countQuery := r.db.
		Table("events AS e").
		Where("e.organizer_id = ?::uuid", organizerID)

	if status != "" {
		countQuery = countQuery.Where("e.status = ?", status)
	}
	if search != "" {
		countQuery = countQuery.Where(
			"LOWER(e.title) LIKE ?",
			"%"+strings.ToLower(search)+"%",
		)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		r.logger.Error(
			"event_list_count_failed",
			"organizer_id", organizerID,
			"error", err,
		)
		return nil, 0, err
	}

	listQuery := r.db.
		Table("events AS e").
		Select(`
			e.id,
			e.organizer_id,
			e.category_id,
			c.name AS category_name,

			e.venue_id,
			COALESCE(v.name, '') AS venue_name,
			COALESCE(v.city, '') AS venue_city,

			e.event_type,
			e.online_url,

			e.title,
			e.description,
			e.banner_url,
			e.language,
			e.age_restriction,

			e.visibility,
			e.status,

			COALESCE(eset.seat_layout_type, '') AS seat_layout_type,

			es.event_date,
			es.start_time,
			es.end_time,
			es.is_all_day,

			CASE
				WHEN eset.seat_layout_type = 'SEATED' THEN COALESCE((
					SELECT COUNT(*)
					FROM seats s
					INNER JOIN seat_rows sr ON sr.id = s.row_id
					INNER JOIN seat_sections ss ON ss.id = sr.section_id
					INNER JOIN seat_layouts sl ON sl.id = ss.seat_layout_id
					WHERE sl.event_id = e.id AND s.status != 'DISABLED'
				), 0)
				ELSE COALESCE((
					SELECT SUM(tt.total_quantity)
					FROM ticket_types tt
					WHERE tt.event_id = e.id
				), 0)
			END AS ticket_capacity,

			CASE
				WHEN eset.seat_layout_type = 'SEATED' THEN COALESCE((
					SELECT COUNT(*)
					FROM seats s
					INNER JOIN seat_rows sr ON sr.id = s.row_id
					INNER JOIN seat_sections ss ON ss.id = sr.section_id
					INNER JOIN seat_layouts sl ON sl.id = ss.seat_layout_id
					WHERE sl.event_id = e.id AND s.status = 'AVAILABLE'
				), 0)
				ELSE COALESCE((
					SELECT SUM(tt.available_quantity)
					FROM ticket_types tt
					WHERE tt.event_id = e.id
				), 0)
			END AS tickets_available,

			CASE
				WHEN eset.seat_layout_type = 'SEATED' THEN
					COALESCE((
						SELECT COUNT(*)
						FROM seats s
						INNER JOIN seat_rows sr ON sr.id = s.row_id
						INNER JOIN seat_sections ss ON ss.id = sr.section_id
						INNER JOIN seat_layouts sl ON sl.id = ss.seat_layout_id
						WHERE sl.event_id = e.id AND s.status != 'DISABLED'
					), 0)
					-
					COALESCE((
						SELECT COUNT(*)
						FROM seats s
						INNER JOIN seat_rows sr ON sr.id = s.row_id
						INNER JOIN seat_sections ss ON ss.id = sr.section_id
						INNER JOIN seat_layouts sl ON sl.id = ss.seat_layout_id
						WHERE sl.event_id = e.id AND s.status = 'AVAILABLE'
					), 0)
				ELSE
					COALESCE((
						SELECT SUM(tt.total_quantity)
						FROM ticket_types tt
						WHERE tt.event_id = e.id
					), 0)
					-
					COALESCE((
						SELECT SUM(tt.available_quantity)
						FROM ticket_types tt
						WHERE tt.event_id = e.id
					), 0)
			END AS tickets_sold,

			e.created_at,
			e.updated_at
		`).
		Joins("INNER JOIN categories AS c ON c.id = e.category_id").
		Joins("LEFT JOIN venues AS v ON v.id = e.venue_id::uuid").
		// Joins("LEFT JOIN venues AS v ON v.id = e.venue_id").
		Joins("LEFT JOIN event_schedules AS es ON es.event_id = e.id").
		Joins("LEFT JOIN event_settings AS eset ON eset.event_id = e.id").
		Where("e.organizer_id = ?::uuid", organizerID)

	if status != "" {
		listQuery = listQuery.Where("e.status = ?", status)
	}
	if search != "" {
		listQuery = listQuery.Where(
			"LOWER(e.title) LIKE ?",
			"%"+strings.ToLower(search)+"%",
		)
	}

	if err := listQuery.
		Order("e.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&events).Error; err != nil {

		r.logger.Error(
			"event_list_fetch_failed",
			"organizer_id", organizerID,
			"page", page,
			"limit", limit,
			"status", status,
			"search", search,
			"error", err,
		)
		return nil, 0, err
	}

	r.logger.Info(
		"organizer_events_fetched",
		"organizer_id", organizerID,
		"count", len(events),
		"total", total,
		"page", page,
		"limit", limit,
	)

	return events, total, nil
}

func toEventModel(event *domain.Event) *models.EventModel {
	return &models.EventModel{
		ID:                  event.ID,
		OrganizerID:         event.OrganizerID,
		CategoryID:          event.CategoryID,
		VenueID:             event.VenueID,
		EventType:           event.EventType,
		OnlineURL:           event.OnlineURL,
		Title:               event.Title,
		Description:         event.Description,
		BannerURL:           event.BannerURL,
		Language:            event.Language,
		AgeRestriction:      event.AgeRestriction,
		Visibility:          event.Visibility,
		Highlights:          event.Highlights,
		Rules:               event.Rules,
		AttendeeInformation: event.AttendeeInformation,
		Status:              event.Status,
		CreatedAt:           event.CreatedAt,
		UpdatedAt:           event.UpdatedAt,
	}
}

func toEventDomain(
	eventModel *models.EventModel,
) *domain.Event {
	return &domain.Event{
		ID:                  eventModel.ID,
		OrganizerID:         eventModel.OrganizerID,
		CategoryID:          eventModel.CategoryID,
		VenueID:             eventModel.VenueID,
		EventType:           eventModel.EventType,
		OnlineURL:           eventModel.OnlineURL,
		Title:               eventModel.Title,
		Description:         eventModel.Description,
		BannerURL:           eventModel.BannerURL,
		Language:            eventModel.Language,
		AgeRestriction:      eventModel.AgeRestriction,
		Visibility:          eventModel.Visibility,
		Highlights:          eventModel.Highlights,
		Rules:               eventModel.Rules,
		AttendeeInformation: eventModel.AttendeeInformation,
		Status:              eventModel.Status,
		CreatedAt:           eventModel.CreatedAt,
		UpdatedAt:           eventModel.UpdatedAt,
	}
}
