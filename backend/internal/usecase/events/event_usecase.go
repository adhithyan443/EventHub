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
	IsAllDay  bool

	SeatLayoutType      string
	BookingLimitPerUser int

	CancellationAllowed       bool
	CancellationDeadlineHours int

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
				ID:             eventID,
				OrganizerID:    organizer.ID,
				CategoryID:     input.CategoryID,
				VenueID:        venueID,
				EventType:      input.EventType,
				OnlineURL:      strings.TrimSpace(input.OnlineURL),
				Title:          strings.TrimSpace(input.Title),
				Description:    strings.TrimSpace(input.Description),
				BannerURL:      objectKey,
				Language:       strings.TrimSpace(input.Language),
				AgeRestriction: input.AgeRestriction,
				Status:         domain.EventStatusDraft,
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

			output = CreateEventOutput{
				Event:        event,
				Schedule:     schedule,
				Setting:      setting,
				Cancellation: cancellation,
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
