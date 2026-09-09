package events

import (
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrUnauthorizedOrganizer = errors.New("user is not an active organizer")
	ErrInvalidEventInput     = errors.New("invalid event input")
	ErrInvalidEventSchedule  = errors.New("invalid event schedule")
	ErrInvalidEventSettings  = errors.New("invalid event settings")
	ErrInvalidCancellation   = errors.New("invalid cancellation settings")
)

const (
	SeatLayoutTypeSeated  = "SEATED"
	SeatLayoutTypeGeneral = "GENERAL"
)

type EventUsecase struct {
	transactionManager domain.TransactionManager
	logger             *slog.Logger
}

func NewEventUsecase(
	transactionManager domain.TransactionManager,
	logger *slog.Logger,
) *EventUsecase {
	return &EventUsecase{
		transactionManager: transactionManager,
		logger:             logger,
	}
}

type CreateEventInput struct {
	UserID     uuid.UUID
	CategoryID uuid.UUID

	EventType string
	OnlineURL string

	Title          string
	Description    string
	BannerURL      string
	Language       string
	AgeRestriction int

	EventDate time.Time
	StartTime time.Time
	EndTime   time.Time

	SeatLayoutType      string
	BookingLimitPerUser int

	CancellationAllowed       bool
	CancellationDeadlineHours int

	Venue VenueInput
}

type VenueInput struct {
	GooglePlaceID string
	Name          string
	Address       string
	City          string
	State         string
	Country       string
	PostalCode    string
	Latitude      float64
	Longitude     float64
}

type CreateEventOutput struct {
	Event        *domain.Event
	Schedule     *domain.EventSchedule
	Setting      *domain.EventSetting
	Cancellation *domain.EventCancellation
}

func (u *EventUsecase) CreateEvent(
	input CreateEventInput,
) (*CreateEventOutput, error) {
	if err := validateCreateEventInput(input); err != nil {
		u.logger.Error(
			"event_create_validation_failed",
			"user_id", input.UserID,
			"event_type", input.EventType,
			"error", err,
		)

		return nil, err
	}

	var output CreateEventOutput

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			// 1. Verify organizer
			organizer, err := tx.OrganizerRepository().
				FindByUserID(input.UserID)

			if err != nil {
				if errors.Is(err, domain.ErrOrganizerNotFound) {
					u.logger.Warn(
						"event_create_organizer_not_found",
						"user_id", input.UserID,
					)

					return ErrUnauthorizedOrganizer
				}

				u.logger.Error(
					"event_create_organizer_lookup_failed",
					"user_id", input.UserID,
					"error", err,
				)

				return err
			}

			if organizer.Status != "ACTIVE" {
				u.logger.Warn(
					"event_create_organizer_inactive",
					"user_id", input.UserID,
					"organizer_id", organizer.ID,
					"status", organizer.Status,
				)

				return ErrUnauthorizedOrganizer
			}

			// 2. Verify active category
			categories, err := tx.CategoryRepository().FindActive()
			if err != nil {
				u.logger.Error(
					"event_create_category_lookup_failed",
					"user_id", input.UserID,
					"category_id", input.CategoryID,
					"error", err,
				)

				return err
			}

			categoryFound := false

			for _, category := range categories {
				if category.ID == input.CategoryID {
					categoryFound = true
					break
				}
			}

			if !categoryFound {
				u.logger.Warn(
					"event_create_category_not_found",
					"user_id", input.UserID,
					"category_id", input.CategoryID,
				)

				return domain.ErrCategoryNotFound
			}

			// 3. Resolve venue only for physical/hybrid events.
			var venue *domain.Venue

			if input.EventType == domain.EventTypePhysical ||
				input.EventType == domain.EventTypeHybrid {

				venue, err = tx.VenueRepository().
					FindByGooglePlaceID(input.Venue.GooglePlaceID)

				if err != nil && !errors.Is(err, domain.ErrVenueNotFound) {
					u.logger.Error(
						"event_create_venue_lookup_failed",
						"google_place_id", input.Venue.GooglePlaceID,
						"error", err,
					)

					return err
				}

				if errors.Is(err, domain.ErrVenueNotFound) {
					venue = &domain.Venue{
						ID:            uuid.New(),
						GooglePlaceID: strings.TrimSpace(input.Venue.GooglePlaceID),
						Name:          strings.TrimSpace(input.Venue.Name),
						Address:       strings.TrimSpace(input.Venue.Address),
						City:          strings.TrimSpace(input.Venue.City),
						State:         strings.TrimSpace(input.Venue.State),
						Country:       strings.TrimSpace(input.Venue.Country),
						PostalCode:    strings.TrimSpace(input.Venue.PostalCode),
						Latitude:      input.Venue.Latitude,
						Longitude:     input.Venue.Longitude,
					}

					if err := tx.VenueRepository().Create(venue); err != nil {
						u.logger.Error(
							"event_create_venue_creation_failed",
							"google_place_id", input.Venue.GooglePlaceID,
							"error", err,
						)

						return err
					}

					u.logger.Info(
						"event_create_venue_created",
						"venue_id", venue.ID,
						"google_place_id", venue.GooglePlaceID,
					)
				} else {
					u.logger.Info(
						"event_create_venue_reused",
						"venue_id", venue.ID,
						"google_place_id", venue.GooglePlaceID,
					)
				}
			}

			// 4. Prepare nullable venue ID.
			var venueID *uuid.UUID

			if venue != nil {
				venueID = &venue.ID
			}

			// 5. Create Event
			event := &domain.Event{
				ID:             uuid.New(),
				OrganizerID:    organizer.ID,
				CategoryID:     input.CategoryID,
				VenueID:        venueID,
				EventType:      input.EventType,
				OnlineURL:      strings.TrimSpace(input.OnlineURL),
				Title:          strings.TrimSpace(input.Title),
				Description:    strings.TrimSpace(input.Description),
				BannerURL:      strings.TrimSpace(input.BannerURL),
				Language:       strings.TrimSpace(input.Language),
				AgeRestriction: input.AgeRestriction,
				Status:         domain.EventStatusDraft,
			}

			if err := tx.EventRepository().Create(event); err != nil {
				u.logger.Error(
					"event_create_failed",
					"user_id", input.UserID,
					"organizer_id", organizer.ID,
					"event_id", event.ID,
					"event_type", event.EventType,
					"venue_id", venueID,
					"error", err,
				)

				return err
			}

			// 6. Create Event Schedule
			schedule := &domain.EventSchedule{
				ID:        uuid.New(),
				EventID:   event.ID,
				EventDate: input.EventDate,
				StartTime: input.StartTime,
				EndTime:   input.EndTime,
			}

			if err := tx.EventScheduleRepository().Create(schedule); err != nil {
				u.logger.Error(
					"event_schedule_create_failed",
					"event_id", event.ID,
					"error", err,
				)

				return err
			}

			// 7. Create Event Settings
			setting := &domain.EventSetting{
				ID:                  uuid.New(),
				EventID:             event.ID,
				SeatLayoutType:      input.SeatLayoutType,
				BookingLimitPerUser: input.BookingLimitPerUser,
			}

			if err := tx.EventSettingRepository().Create(setting); err != nil {
				u.logger.Error(
					"event_setting_create_failed",
					"event_id", event.ID,
					"error", err,
				)

				return err
			}

			// 8. Create Event Cancellation Settings
			cancellation := &domain.EventCancellation{
				ID:                        uuid.New(),
				EventID:                   event.ID,
				CancellationAllowed:       input.CancellationAllowed,
				CancellationDeadlineHours: input.CancellationDeadlineHours,
			}

			if err := tx.EventCancellationRepository().Create(cancellation); err != nil {
				u.logger.Error(
					"event_cancellation_create_failed",
					"event_id", event.ID,
					"error", err,
				)

				return err
			}

			output = CreateEventOutput{
				Event:        event,
				Schedule:     schedule,
				Setting:      setting,
				Cancellation: cancellation,
			}

			return nil
		},
	)

	if err != nil {
		u.logger.Error(
			"event_create_transaction_failed",
			"user_id", input.UserID,
			"event_type", input.EventType,
			"error", err,
		)

		return nil, err
	}

	u.logger.Info(
		"event_created",
		"event_id", output.Event.ID,
		"organizer_id", output.Event.OrganizerID,
		"event_type", output.Event.EventType,
		"venue_id", output.Event.VenueID,
		"status", output.Event.Status,
	)

	return &output, nil
}

func validateCreateEventInput(input CreateEventInput) error {
	if input.UserID == uuid.Nil {
		return ErrInvalidEventInput
	}

	if input.CategoryID == uuid.Nil {
		return ErrInvalidEventInput
	}

	if strings.TrimSpace(input.Title) == "" {
		return ErrInvalidEventInput
	}

	if strings.TrimSpace(input.Description) == "" {
		return ErrInvalidEventInput
	}

	if strings.TrimSpace(input.BannerURL) == "" {
		return ErrInvalidEventInput
	}

	if input.AgeRestriction < 0 {
		return ErrInvalidEventInput
	}

	if err := validateEventType(input); err != nil {
		return ErrInvalidEventInput
	}

	if input.EventDate.IsZero() {
		return ErrInvalidEventSchedule
	}

	if input.StartTime.IsZero() {
		return ErrInvalidEventSchedule
	}

	if input.EndTime.IsZero() {
		return ErrInvalidEventSchedule
	}

	if !input.EndTime.After(input.StartTime) {
		return ErrInvalidEventSchedule
	}

	if input.SeatLayoutType != SeatLayoutTypeSeated &&
		input.SeatLayoutType != SeatLayoutTypeGeneral {
		return ErrInvalidEventSettings
	}

	// Online events cannot use reserved seating.
	if input.EventType == domain.EventTypeOnline &&
		input.SeatLayoutType == SeatLayoutTypeSeated {
		return ErrInvalidEventSettings
	}

	if input.BookingLimitPerUser <= 0 {
		return ErrInvalidEventSettings
	}

	if input.CancellationAllowed &&
		input.CancellationDeadlineHours <= 0 {
		return ErrInvalidCancellation
	}

	if !input.CancellationAllowed &&
		input.CancellationDeadlineHours != 0 {
		return ErrInvalidCancellation
	}

	return nil
}

func validateEventType(input CreateEventInput) error {
	switch input.EventType {
	case domain.EventTypePhysical:
		if strings.TrimSpace(input.OnlineURL) != "" {
			return errors.New("physical event cannot have online url")
		}

		return validateVenueInput(input.Venue)

	case domain.EventTypeOnline:
		if err := validateOnlineURL(input.OnlineURL); err != nil {
			return err
		}

		if hasVenueInput(input.Venue) {
			return errors.New("online event cannot have venue")
		}

		return nil

	case domain.EventTypeHybrid:
		if err := validateVenueInput(input.Venue); err != nil {
			return err
		}

		return validateOnlineURL(input.OnlineURL)

	default:
		return errors.New("invalid event type")
	}
}

func validateOnlineURL(value string) error {
	value = strings.TrimSpace(value)

	if value == "" {
		return errors.New("online url is required")
	}

	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return errors.New("invalid online url")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("online url must use http or https")
	}

	if parsedURL.Host == "" {
		return errors.New("online url must contain a host")
	}

	return nil
}

func hasVenueInput(input VenueInput) bool {
	return strings.TrimSpace(input.GooglePlaceID) != "" ||
		strings.TrimSpace(input.Name) != "" ||
		strings.TrimSpace(input.Address) != "" ||
		strings.TrimSpace(input.City) != "" ||
		strings.TrimSpace(input.State) != "" ||
		strings.TrimSpace(input.Country) != "" ||
		strings.TrimSpace(input.PostalCode) != "" ||
		input.Latitude != 0 ||
		input.Longitude != 0
}

func validateVenueInput(input VenueInput) error {
	if strings.TrimSpace(input.GooglePlaceID) == "" {
		return errors.New("google place id is required")
	}

	if strings.TrimSpace(input.Name) == "" {
		return errors.New("venue name is required")
	}

	if strings.TrimSpace(input.Address) == "" {
		return errors.New("venue address is required")
	}

	if strings.TrimSpace(input.City) == "" {
		return errors.New("venue city is required")
	}

	if strings.TrimSpace(input.State) == "" {
		return errors.New("venue state is required")
	}

	if strings.TrimSpace(input.Country) == "" {
		return errors.New("venue country is required")
	}

	if input.Latitude < -90 || input.Latitude > 90 {
		return errors.New("invalid venue latitude")
	}

	if input.Longitude < -180 || input.Longitude > 180 {
		return errors.New("invalid venue longitude")
	}

	return nil
}
