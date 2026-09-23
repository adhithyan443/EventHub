package events

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/service/storage"
	"github.com/google/uuid"
)

var ErrEventNotEditable = errors.New("event is not editable")

var (
	ErrUnauthorizedOrganizer = errors.New("user is not an active organizer")
	ErrInvalidEventInput     = errors.New("invalid event input")
	ErrInvalidEventSchedule  = errors.New("invalid event schedule")
	ErrInvalidEventSettings  = errors.New("invalid event settings")
	ErrInvalidCancellation   = errors.New("invalid cancellation settings")
	ErrInvalidBanner         = errors.New("invalid event banner")
)

const (
	SeatLayoutTypeSeated  = "SEATED"
	SeatLayoutTypeGeneral = "GENERAL"
)

type EventUsecase struct {
	transactionManager domain.TransactionManager
	storageService     *storage.S3Service
	logger             *slog.Logger
}

func NewEventUsecase(
	transactionManager domain.TransactionManager,
	storageService *storage.S3Service,
	logger *slog.Logger,
) *EventUsecase {
	return &EventUsecase{
		transactionManager: transactionManager,
		storageService:     storageService,
		logger:             logger,
	}
}

type EventContactInput struct {
	Name  string
	Phone string
	Email string
}

type CreateEventInput struct {
	UserID     uuid.UUID
	CategoryID uuid.UUID
	Contact    EventContactInput

	EventType string
	OnlineURL string

	Title          string
	Description    string
	BannerURL      string
	Language       string
	AgeRestriction int

	Visibility          string
	Highlights          string
	Rules               string
	AttendeeInformation string

	EventDate time.Time
	StartTime time.Time
	EndTime   time.Time
	IsAllDay  bool

	SeatLayoutType      string
	BookingLimitPerUser int
	SalesStartDate      *time.Time
	SalesEndDate        *time.Time

	CancellationAllowed       bool
	CancellationDeadlineHours int
	RefundPolicy              string
	RefundPercentage          int

	Venue VenueInput

	BannerReader      io.Reader
	BannerContentType string
	BannerSize        int64
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
	Contact      *domain.EventContact
}

func (u *EventUsecase) CreateEvent(
	ctx context.Context,
	input CreateEventInput,
) (*CreateEventOutput, error) {
	if err := validateCreateEventInput(input); err != nil {
		u.logger.Warn(
			"event_create_validation_failed",
			"user_id", input.UserID,
			"event_type", input.EventType,
			"error", err,
		)

		return nil, err
	}

	if u.storageService == nil {
		u.logger.Error(
			"event_banner_upload_storage_unavailable",
			"user_id", input.UserID,
		)

		return nil, errors.New("storage service is unavailable")
	}

	// 1. Pre-verify that the user is an active organizer and category exists.
	var organizer *domain.Organizer

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			org, err := tx.OrganizerRepository().FindByUserID(input.UserID)
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

			if org.Status != "ACTIVE" {
				u.logger.Warn(
					"event_create_organizer_inactive",
					"user_id", input.UserID,
					"organizer_id", org.ID,
					"status", org.Status,
				)
				return ErrUnauthorizedOrganizer
			}

			organizer = org

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

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	// 2. Generate Event ID before S3 upload.
	eventID := uuid.New()

	u.logger.Info(
		"event_id_generated",
		"event_id", eventID,
		"organizer_id", organizer.ID,
	)

	// 3. Build S3 key.
	extension := bannerExtension(input.BannerContentType)
	if extension == "" {
		u.logger.Warn(
			"event_banner_invalid_extension",
			"event_id", eventID,
			"organizer_id", organizer.ID,
			"content_type", input.BannerContentType,
		)
		return nil, ErrInvalidBanner
	}

	objectKey := fmt.Sprintf(
		"event-banners/%s/%s%s",
		organizer.ID.String(),
		eventID.String(),
		extension,
	)

	// 4. Upload banner to S3.
	u.logger.Info(
		"event_banner_upload_started",
		"event_id", eventID,
		"organizer_id", organizer.ID,
		"banner_key", objectKey,
		"content_type", input.BannerContentType,
		"file_size", input.BannerSize,
	)

	if err := u.storageService.Upload(
		ctx,
		objectKey,
		input.BannerReader,
		input.BannerContentType,
	); err != nil {
		u.logger.Error(
			"event_banner_upload_failed",
			"event_id", eventID,
			"organizer_id", organizer.ID,
			"banner_key", objectKey,
			"error", err,
		)
		return nil, err
	}

	u.logger.Info(
		"event_banner_upload_succeeded",
		"event_id", eventID,
		"organizer_id", organizer.ID,
		"banner_key", objectKey,
	)

	// 5. Begin PostgreSQL transaction.
	u.logger.Info(
		"event_db_transaction_started",
		"event_id", eventID,
		"organizer_id", organizer.ID,
	)

	var output CreateEventOutput

	txErr := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			// Resolve venue only for physical/hybrid events.
			var venue *domain.Venue

			if input.EventType == domain.EventTypePhysical ||
				input.EventType == domain.EventTypeHybrid {

				v, err := tx.VenueRepository().
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
					venue = v
					u.logger.Info(
						"event_create_venue_reused",
						"venue_id", venue.ID,
						"google_place_id", venue.GooglePlaceID,
					)
				}
			}

			// Prepare nullable venue ID.
			var venueID *uuid.UUID
			if venue != nil {
				venueID = &venue.ID
			}

			// Create Event with pre-generated eventID and objectKey in BannerURL.
			event := &domain.Event{
				ID:                  eventID,
				OrganizerID:         organizer.ID,
				CategoryID:          input.CategoryID,
				VenueID:             venueID,
				EventType:           input.EventType,
				OnlineURL:           strings.TrimSpace(input.OnlineURL),
				Title:               strings.TrimSpace(input.Title),
				Description:         strings.TrimSpace(input.Description),
				BannerURL:           objectKey,
				Language:            strings.TrimSpace(input.Language),
				AgeRestriction:      input.AgeRestriction,
				Visibility:          strings.TrimSpace(input.Visibility),
				Highlights:          strings.TrimSpace(input.Highlights),
				Rules:               strings.TrimSpace(input.Rules),
				AttendeeInformation: strings.TrimSpace(input.AttendeeInformation),
				Status:              domain.EventStatusDraft,
			}

			if err := tx.EventRepository().Create(event); err != nil {
				u.logger.Error(
					"event_create_failed",
					"user_id", input.UserID,
					"organizer_id", organizer.ID,
					"event_id", eventID,
					"event_type", event.EventType,
					"venue_id", venueID,
					"error", err,
				)
				return err
			}

			// Create Event Schedule.
			schedule := &domain.EventSchedule{
				ID:        uuid.New(),
				EventID:   eventID,
				EventDate: input.EventDate,
				StartTime: input.StartTime,
				EndTime:   input.EndTime,
				IsAllDay:  input.IsAllDay,
			}

			if err := tx.EventScheduleRepository().Create(schedule); err != nil {
				u.logger.Error(
					"event_schedule_create_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			// Create Event Settings.
			setting := &domain.EventSetting{
				ID:                  uuid.New(),
				EventID:             eventID,
				SeatLayoutType:      input.SeatLayoutType,
				BookingLimitPerUser: input.BookingLimitPerUser,
				SalesStartDate:      input.SalesStartDate,
				SalesEndDate:        input.SalesEndDate,
			}

			if err := tx.EventSettingRepository().Create(setting); err != nil {
				u.logger.Error(
					"event_setting_create_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			// Create Event Cancellation Settings.
			cancellation := &domain.EventCancellation{
				ID:                        uuid.New(),
				EventID:                   eventID,
				RefundPolicy:              input.RefundPolicy,
				RefundPercentage:          input.RefundPercentage,
				CancellationAllowed:       input.CancellationAllowed,
				CancellationDeadlineHours: input.CancellationDeadlineHours,
			}

			if err := tx.EventCancellationRepository().Create(cancellation); err != nil {
				u.logger.Error(
					"event_cancellation_create_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			// Contact information is optional. Only create a database row
			// when at least one contact field is supplied.
			var contact *domain.EventContact

			contactName := strings.TrimSpace(input.Contact.Name)
			contactPhone := strings.TrimSpace(input.Contact.Phone)
			contactEmail := strings.TrimSpace(input.Contact.Email)

			if contactName != "" || contactPhone != "" || contactEmail != "" {
				contact = &domain.EventContact{
					ID:      uuid.New(),
					EventID: eventID,
					Name:    contactName,
					Phone:   contactPhone,
					Email:   contactEmail,
				}

				if err := tx.EventContactRepository().Create(contact); err != nil {
					u.logger.Error(
						"event_contact_create_failed",
						"event_id", eventID,
						"error", err,
					)
					return err
				}

				u.logger.Info(
					"event_contact_created",
					"event_id", eventID,
				)
			}

			output = CreateEventOutput{
				Event:        event,
				Schedule:     schedule,
				Setting:      setting,
				Cancellation: cancellation,
				Contact:      contact,
			}

			return nil
		},
	)

	// 6. Handle failure vs success.
	if txErr != nil {
		u.logger.Error(
			"event_db_transaction_failed",
			"event_id", eventID,
			"organizer_id", organizer.ID,
			"error", txErr,
		)

		cleanupCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			10*time.Second,
		)
		defer cancel()

		u.logger.Info(
			"event_s3_compensation_delete_started",
			"event_id", eventID,
			"organizer_id", organizer.ID,
			"banner_key", objectKey,
		)

		if delErr := u.storageService.Delete(cleanupCtx, objectKey); delErr != nil {
			u.logger.Error(
				"event_s3_compensation_delete_failed",
				"event_id", eventID,
				"organizer_id", organizer.ID,
				"banner_key", objectKey,
				"error", delErr,
			)
		} else {
			u.logger.Info(
				"event_s3_compensation_delete_succeeded",
				"event_id", eventID,
				"organizer_id", organizer.ID,
				"banner_key", objectKey,
			)
		}

		return nil, txErr
	}

	u.logger.Info(
		"event_db_transaction_committed",
		"event_id", eventID,
		"organizer_id", organizer.ID,
	)

	u.logger.Info(
		"event_created_successfully",
		"event_id", output.Event.ID,
		"organizer_id", output.Event.OrganizerID,
		"event_type", output.Event.EventType,
		"venue_id", output.Event.VenueID,
		"status", output.Event.Status,
	)

	return &output, nil
}

type UpdateEventInput struct {
	UserID     uuid.UUID
	EventID    uuid.UUID
	CategoryID uuid.UUID
	VenueID    *uuid.UUID
	Venue      VenueInput

	EventType string
	OnlineURL string

	Title          string
	Description    string
	Language       string
	AgeRestriction int

	Visibility          string
	Highlights          string
	Rules               string
	AttendeeInformation string

	EventDate time.Time
	StartTime time.Time
	EndTime   time.Time
	IsAllDay  bool

	SeatLayoutType      string
	BookingLimitPerUser int
	SalesStartDate      *time.Time
	SalesEndDate        *time.Time

	CancellationAllowed       bool
	CancellationDeadlineHours int
	RefundPolicy              string
	RefundPercentage          int

	Contact EventContactInput
}

type UpdateEventOutput struct {
	Event        *domain.Event
	Schedule     *domain.EventSchedule
	Setting      *domain.EventSetting
	Cancellation *domain.EventCancellation
	Contact      *domain.EventContact
}

func (u *EventUsecase) UpdateEvent(
	ctx context.Context,
	input UpdateEventInput,
) (*UpdateEventOutput, error) {
	u.logger.Info(
		"event_update_started",
		"event_id", input.EventID,
		"user_id", input.UserID,
	)

	if err := validateUpdateEventInput(input); err != nil {
		u.logger.Warn(
			"event_update_validation_failed",
			"event_id", input.EventID,
			"user_id", input.UserID,
			"error", err,
		)

		return nil, err
	}

	var output UpdateEventOutput

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			// Find the event.
			foundEvent, err := tx.EventRepository().FindByID(input.EventID)
			if err != nil {
				u.logger.Warn(
					"event_update_lookup_failed",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"error", err,
				)

				return err
			}

			// Ensure the authenticated organizer owns the event.
			organizer, err := tx.OrganizerRepository().FindByUserID(input.UserID)
			if err != nil {
				if errors.Is(err, domain.ErrOrganizerNotFound) {
					u.logger.Warn(
						"event_update_organizer_not_found",
						"event_id", input.EventID,
						"user_id", input.UserID,
					)
					return ErrUnauthorizedOrganizer
				}

				u.logger.Error(
					"event_update_organizer_lookup_failed",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"error", err,
				)
				return err
			}

			if organizer.Status != "ACTIVE" {
				u.logger.Warn(
					"event_update_organizer_inactive",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"organizer_id", organizer.ID,
					"status", organizer.Status,
				)
				return ErrUnauthorizedOrganizer
			}

			if foundEvent.OrganizerID != organizer.ID {
				u.logger.Warn(
					"event_update_ownership_denied",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"event_organizer_id", foundEvent.OrganizerID,
					"user_organizer_id", organizer.ID,
				)
				return ErrUnauthorizedOrganizer
			}

			// Only draft events can currently be edited.
			if foundEvent.Status != domain.EventStatusDraft {
				u.logger.Warn(
					"event_update_status_denied",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"status", foundEvent.Status,
				)

				return ErrEventNotEditable
			}

			// Verify category exists.
			categories, err := tx.CategoryRepository().FindActive()
			if err != nil {
				u.logger.Error(
					"event_update_category_lookup_failed",
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
					"event_update_category_not_found",
					"user_id", input.UserID,
					"category_id", input.CategoryID,
				)
				return domain.ErrCategoryNotFound
			}

			// Resolve venue only for physical/hybrid events.
			var venueID *uuid.UUID

			if input.EventType == domain.EventTypePhysical ||
				input.EventType == domain.EventTypeHybrid {

				if input.VenueID != nil && *input.VenueID != uuid.Nil {
					v, err := tx.VenueRepository().FindByID(*input.VenueID)
					if err != nil {
						if errors.Is(err, domain.ErrVenueNotFound) {
							u.logger.Warn(
								"event_update_venue_not_found",
								"venue_id", *input.VenueID,
							)
							return domain.ErrVenueNotFound
						}

						u.logger.Error(
							"event_update_venue_lookup_failed",
							"venue_id", *input.VenueID,
							"error", err,
						)
						return err
					}

					venueID = &v.ID
				} else if hasVenueInput(input.Venue) {
					v, err := tx.VenueRepository().
						FindByGooglePlaceID(input.Venue.GooglePlaceID)

					if err != nil && !errors.Is(err, domain.ErrVenueNotFound) {
						u.logger.Error(
							"event_update_venue_lookup_failed",
							"google_place_id", input.Venue.GooglePlaceID,
							"error", err,
						)
						return err
					}

					if errors.Is(err, domain.ErrVenueNotFound) {
						venue := &domain.Venue{
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
								"event_update_venue_creation_failed",
								"google_place_id", input.Venue.GooglePlaceID,
								"error", err,
							)
							return err
						}

						venueID = &venue.ID
						u.logger.Info(
							"event_update_venue_created",
							"venue_id", venue.ID,
							"google_place_id", venue.GooglePlaceID,
						)
					} else {
						venueID = &v.ID
						u.logger.Info(
							"event_update_venue_reused",
							"venue_id", v.ID,
							"google_place_id", v.GooglePlaceID,
						)
					}
				}
			}

			// Update editable event fields.
			foundEvent.CategoryID = input.CategoryID
			foundEvent.VenueID = venueID
			foundEvent.EventType = strings.TrimSpace(input.EventType)
			foundEvent.OnlineURL = strings.TrimSpace(input.OnlineURL)
			foundEvent.Title = strings.TrimSpace(input.Title)
			foundEvent.Description = strings.TrimSpace(input.Description)
			foundEvent.Language = strings.TrimSpace(input.Language)
			foundEvent.AgeRestriction = input.AgeRestriction
			foundEvent.Visibility = strings.TrimSpace(input.Visibility)
			foundEvent.Highlights = strings.TrimSpace(input.Highlights)
			foundEvent.Rules = strings.TrimSpace(input.Rules)
			foundEvent.AttendeeInformation = strings.TrimSpace(input.AttendeeInformation)
			foundEvent.UpdatedAt = time.Now()

			if err := tx.EventRepository().Update(foundEvent); err != nil {
				u.logger.Error(
					"event_update_failed",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"error", err,
				)

				return err
			}

			// Update Event Schedule.
			var schedule *domain.EventSchedule
			existingSchedule, err := tx.EventScheduleRepository().FindByEventID(foundEvent.ID)
			if err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					schedule = &domain.EventSchedule{
						ID:        uuid.New(),
						EventID:   foundEvent.ID,
						EventDate: input.EventDate,
						StartTime: input.StartTime,
						EndTime:   input.EndTime,
						IsAllDay:  input.IsAllDay,
					}
					if err := tx.EventScheduleRepository().Create(schedule); err != nil {
						u.logger.Error(
							"event_schedule_create_failed",
							"event_id", foundEvent.ID,
							"error", err,
						)
						return err
					}
				} else {
					u.logger.Error(
						"event_schedule_lookup_failed",
						"event_id", foundEvent.ID,
						"error", err,
					)
					return err
				}
			} else {
				existingSchedule.EventDate = input.EventDate
				existingSchedule.StartTime = input.StartTime
				existingSchedule.EndTime = input.EndTime
				existingSchedule.IsAllDay = input.IsAllDay
				existingSchedule.UpdatedAt = time.Now()
				if err := tx.EventScheduleRepository().Update(existingSchedule); err != nil {
					u.logger.Error(
						"event_schedule_update_failed",
						"event_id", foundEvent.ID,
						"error", err,
					)
					return err
				}
				schedule = existingSchedule
			}

			// Update Event Settings.
			var setting *domain.EventSetting
			existingSetting, err := tx.EventSettingRepository().FindByEventID(foundEvent.ID)
			if err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					setting = &domain.EventSetting{
						ID:                  uuid.New(),
						EventID:             foundEvent.ID,
						SeatLayoutType:      input.SeatLayoutType,
						BookingLimitPerUser: input.BookingLimitPerUser,
						SalesStartDate:      input.SalesStartDate,
						SalesEndDate:        input.SalesEndDate,
					}
					if err := tx.EventSettingRepository().Create(setting); err != nil {
						u.logger.Error(
							"event_setting_create_failed",
							"event_id", foundEvent.ID,
							"error", err,
						)
						return err
					}
				} else {
					u.logger.Error(
						"event_setting_lookup_failed",
						"event_id", foundEvent.ID,
						"error", err,
					)
					return err
				}
			} else {
				existingSetting.SeatLayoutType = input.SeatLayoutType
				existingSetting.BookingLimitPerUser = input.BookingLimitPerUser
				existingSetting.SalesStartDate = input.SalesStartDate
				existingSetting.SalesEndDate = input.SalesEndDate
				existingSetting.UpdatedAt = time.Now()
				if err := tx.EventSettingRepository().Update(existingSetting); err != nil {
					u.logger.Error(
						"event_setting_update_failed",
						"event_id", foundEvent.ID,
						"error", err,
					)
					return err
				}
				setting = existingSetting
			}

			// Update Event Cancellation.
			var cancellation *domain.EventCancellation
			existingCancellation, err := tx.EventCancellationRepository().FindByEventID(foundEvent.ID)
			if err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					cancellation = &domain.EventCancellation{
						ID:                        uuid.New(),
						EventID:                   foundEvent.ID,
						CancellationAllowed:       input.CancellationAllowed,
						CancellationDeadlineHours: input.CancellationDeadlineHours,
						RefundPolicy:              input.RefundPolicy,
						RefundPercentage:          input.RefundPercentage,
					}
					if err := tx.EventCancellationRepository().Create(cancellation); err != nil {
						u.logger.Error(
							"event_cancellation_create_failed",
							"event_id", foundEvent.ID,
							"error", err,
						)
						return err
					}
				} else {
					u.logger.Error(
						"event_cancellation_lookup_failed",
						"event_id", foundEvent.ID,
						"error", err,
					)
					return err
				}
			} else {
				existingCancellation.CancellationAllowed = input.CancellationAllowed
				existingCancellation.CancellationDeadlineHours = input.CancellationDeadlineHours
				existingCancellation.RefundPolicy = input.RefundPolicy
				existingCancellation.RefundPercentage = input.RefundPercentage
				existingCancellation.UpdatedAt = time.Now()
				if err := tx.EventCancellationRepository().Update(existingCancellation); err != nil {
					u.logger.Error(
						"event_cancellation_update_failed",
						"event_id", foundEvent.ID,
						"error", err,
					)
					return err
				}
				cancellation = existingCancellation
			}

			// Update Event Contact.
			var contact *domain.EventContact
			contactName := strings.TrimSpace(input.Contact.Name)
			contactPhone := strings.TrimSpace(input.Contact.Phone)
			contactEmail := strings.TrimSpace(input.Contact.Email)

			if contactName != "" || contactPhone != "" || contactEmail != "" {
				existingContact, err := tx.EventContactRepository().FindByEventID(foundEvent.ID)
				if err != nil {
					if errors.Is(err, domain.ErrEventNotFound) {
						contact = &domain.EventContact{
							ID:      uuid.New(),
							EventID: foundEvent.ID,
							Name:    contactName,
							Phone:   contactPhone,
							Email:   contactEmail,
						}
						if err := tx.EventContactRepository().Create(contact); err != nil {
							u.logger.Error(
								"event_contact_create_failed",
								"event_id", foundEvent.ID,
								"error", err,
							)
							return err
						}
					} else {
						u.logger.Error(
							"event_contact_lookup_failed",
							"event_id", foundEvent.ID,
							"error", err,
						)
						return err
					}
				} else {
					existingContact.Name = contactName
					existingContact.Phone = contactPhone
					existingContact.Email = contactEmail
					existingContact.UpdatedAt = time.Now()
					if err := tx.EventContactRepository().Update(existingContact); err != nil {
						u.logger.Error(
							"event_contact_update_failed",
							"event_id", foundEvent.ID,
							"error", err,
						)
						return err
					}
					contact = existingContact
				}
			} else {
				if err := tx.EventContactRepository().DeleteByEventID(foundEvent.ID); err != nil {
					u.logger.Error(
						"event_contact_delete_failed",
						"event_id", foundEvent.ID,
						"error", err,
					)
					return err
				}
			}

			output = UpdateEventOutput{
				Event:        foundEvent,
				Schedule:     schedule,
				Setting:      setting,
				Cancellation: cancellation,
				Contact:      contact,
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	u.logger.Info(
		"event_updated_successfully",
		"event_id", output.Event.ID,
		"user_id", input.UserID,
		"status", output.Event.Status,
	)

	return &output, nil
}
func isSupportedBannerContentType(contentType string) bool {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func bannerExtension(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
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

	if input.BannerReader == nil {
		return ErrInvalidBanner
	}

	if !isSupportedBannerContentType(input.BannerContentType) {
		return ErrInvalidBanner
	}

	if input.BannerSize <= 0 || input.BannerSize > 5*1024*1024 {
		return ErrInvalidBanner
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

	if input.IsAllDay {
		if !input.StartTime.IsZero() || !input.EndTime.IsZero() {
			return ErrInvalidEventSchedule
		}
	} else {
		if input.StartTime.IsZero() {
			return ErrInvalidEventSchedule
		}

		if input.EndTime.IsZero() {
			return ErrInvalidEventSchedule
		}

		if !input.EndTime.After(input.StartTime) {
			return ErrInvalidEventSchedule
		}
	}

	if err := validateSalesWindow(input); err != nil {
		return ErrInvalidEventSettings
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

	if err := validateEventContact(input.Contact); err != nil {
		return ErrInvalidEventInput
	}

	return nil
}

func validateEventContact(input EventContactInput) error {
	name := strings.TrimSpace(input.Name)
	phone := strings.TrimSpace(input.Phone)
	email := strings.TrimSpace(input.Email)

	// Contact information is optional, but must be complete when provided.
	if name == "" && phone == "" && email == "" {
		return nil
	}

	if name == "" || phone == "" || email == "" {
		return errors.New("event contact information must be complete")
	}

	if len(name) > 100 {
		return errors.New("event contact name is too long")
	}

	if len(phone) > 20 {
		return errors.New("event contact phone is too long")
	}

	if len(email) > 255 {
		return errors.New("event contact email is too long")
	}

	if !strings.Contains(email, "@") {
		return errors.New("invalid event contact email")
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

func validateSalesDates(salesStartDate, salesEndDate *time.Time, eventDate time.Time) error {
	if salesStartDate == nil && salesEndDate == nil {
		return nil
	}

	if salesStartDate == nil || salesEndDate == nil {
		return errors.New("sales start and end dates must both be provided")
	}

	if salesStartDate.After(*salesEndDate) {
		return errors.New("sales start date cannot be after sales end date")
	}

	truncEventDate := eventDate.Truncate(24 * time.Hour)

	if salesStartDate.After(truncEventDate) {
		return errors.New("sales start date cannot be after event date")
	}

	if salesEndDate.After(truncEventDate) {
		return errors.New("sales end date cannot be after event date")
	}

	return nil
}

func validateSalesWindow(input CreateEventInput) error {
	return validateSalesDates(input.SalesStartDate, input.SalesEndDate, input.EventDate)
}

func validateUpdateEventInput(input UpdateEventInput) error {
	if input.UserID == uuid.Nil {
		return ErrInvalidEventInput
	}

	if input.EventID == uuid.Nil {
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

	if input.AgeRestriction < 0 {
		return ErrInvalidEventInput
	}

	if err := validateUpdateEventType(input); err != nil {
		return ErrInvalidEventInput
	}

	if input.EventDate.IsZero() {
		return ErrInvalidEventSchedule
	}

	if input.IsAllDay {
		if !input.StartTime.IsZero() || !input.EndTime.IsZero() {
			return ErrInvalidEventSchedule
		}
	} else {
		if input.StartTime.IsZero() {
			return ErrInvalidEventSchedule
		}

		if input.EndTime.IsZero() {
			return ErrInvalidEventSchedule
		}

		if !input.EndTime.After(input.StartTime) {
			return ErrInvalidEventSchedule
		}
	}

	if err := validateSalesDates(input.SalesStartDate, input.SalesEndDate, input.EventDate); err != nil {
		return ErrInvalidEventSettings
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

	if err := validateEventContact(input.Contact); err != nil {
		return ErrInvalidEventInput
	}

	return nil
}

func validateUpdateEventType(input UpdateEventInput) error {
	switch input.EventType {
	case domain.EventTypePhysical:
		if strings.TrimSpace(input.OnlineURL) != "" {
			return errors.New("physical event cannot have online url")
		}

		if (input.VenueID == nil || *input.VenueID == uuid.Nil) && !hasVenueInput(input.Venue) {
			return errors.New("physical event requires a venue")
		}

		if hasVenueInput(input.Venue) {
			return validateVenueInput(input.Venue)
		}

		return nil

	case domain.EventTypeOnline:
		if err := validateOnlineURL(input.OnlineURL); err != nil {
			return err
		}

		if (input.VenueID != nil && *input.VenueID != uuid.Nil) || hasVenueInput(input.Venue) {
			return errors.New("online event cannot have venue")
		}

		return nil

	case domain.EventTypeHybrid:
		if (input.VenueID == nil || *input.VenueID == uuid.Nil) && !hasVenueInput(input.Venue) {
			return errors.New("hybrid event requires a venue")
		}

		if hasVenueInput(input.Venue) {
			if err := validateVenueInput(input.Venue); err != nil {
				return err
			}
		}

		return validateOnlineURL(input.OnlineURL)

	default:
		return errors.New("invalid event type")
	}
}
