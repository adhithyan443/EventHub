package events

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/service/storage"
	"github.com/google/uuid"
)

var ErrEventNotEditable = errors.New("event is not editable")
var ErrEventNotDeletable = errors.New("only draft events can be deleted")
var ErrEventNotPublishable = errors.New("event is not ready to be published")

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
	if logger == nil {
		logger = slog.Default()
	}

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

	TicketTypes []TicketTypeInput
	SeatLayout  CreateSeatLayoutInput

	BannerReader      io.Reader
	BannerContentType string
	BannerSize        int64
}

type TicketTypeInput struct {
	ID          *uuid.UUID
	Name        string
	Price       float64
	Capacity    int
	Description string
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
	TicketTypes  []*domain.TicketType
	SeatLayout   *CreateSeatLayoutOutput
}

func (u *EventUsecase) CreateEvent(
	ctx context.Context,
	input CreateEventInput,
) (*CreateEventOutput, error) {
	input.EventType = strings.ToUpper(strings.TrimSpace(input.EventType))
	input.SeatLayoutType = strings.ToUpper(strings.TrimSpace(input.SeatLayoutType))
	input.BannerContentType = strings.ToLower(strings.TrimSpace(input.BannerContentType))

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
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if delErr := u.storageService.Delete(cleanupCtx, objectKey); delErr != nil {
			u.logger.Error(
				"event_s3_upload_failure_cleanup_failed",
				"event_id", eventID,
				"banner_key", objectKey,
				"error", delErr,
			)
			return nil, errors.Join(err, fmt.Errorf("clean up partially uploaded event banner: %w", delErr))
		}
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

			var ticketTypes []*domain.TicketType
			var seatLayoutOutput *CreateSeatLayoutOutput

			if input.SeatLayoutType == SeatLayoutTypeGeneral {
				for _, ticketInput := range input.TicketTypes {
					ticketType := &domain.TicketType{
						ID:                uuid.New(),
						EventID:           event.ID,
						Name:              strings.TrimSpace(ticketInput.Name),
						Price:             ticketInput.Price,
						TotalQuantity:     ticketInput.Capacity,
						AvailableQuantity: ticketInput.Capacity,
						Description:       strings.TrimSpace(ticketInput.Description),
					}

					if err := tx.TicketTypeRepository().Create(ctx, ticketType); err != nil {
						u.logger.Error(
							"event_ticket_type_create_failed",
							"event_id", event.ID,
							"ticket_type_id", ticketType.ID,
							"error", err,
						)
						return err
					}

					ticketTypes = append(ticketTypes, ticketType)
				}
			}

			if input.SeatLayoutType == SeatLayoutTypeSeated {
				seatLayoutOutput, err = createSeatLayout(
					tx,
					event.ID,
					input.SeatLayout,
				)
				if err != nil {
					u.logger.Error(
						"event_seat_layout_create_failed",
						"event_id", event.ID,
						"error", err,
					)
					return err
				}
			}

			output = CreateEventOutput{
				Event:        event,
				Schedule:     schedule,
				Setting:      setting,
				Cancellation: cancellation,
				Contact:      contact,
				TicketTypes:  ticketTypes,
				SeatLayout:   seatLayoutOutput,
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

	Contact     EventContactInput
	TicketTypes []TicketTypeInput

	BannerReader      io.Reader
	BannerContentType string
	BannerSize        int64
}

type UpdateEventOutput struct {
	Event        *domain.Event
	Schedule     *domain.EventSchedule
	Setting      *domain.EventSetting
	Cancellation *domain.EventCancellation
	Contact      *domain.EventContact
	TicketTypes  []*domain.TicketType
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
	var newObjectKey string
	var oldBannerKey string

	if input.BannerReader != nil {
		if u.storageService == nil {
			u.logger.Error(
				"event_banner_upload_storage_unavailable",
				"user_id", input.UserID,
				"event_id", input.EventID,
			)
			return nil, errors.New("storage service is unavailable")
		}

		extension := bannerExtension(input.BannerContentType)
		if extension == "" {
			u.logger.Warn(
				"event_banner_invalid_extension",
				"event_id", input.EventID,
				"user_id", input.UserID,
				"content_type", input.BannerContentType,
			)
			return nil, ErrInvalidBanner
		}

		var organizerID uuid.UUID
		err := u.transactionManager.WithinTransaction(
			func(tx domain.TransactionRepositories) error {
				organizer, err := tx.OrganizerRepository().FindByUserID(input.UserID)
				if err != nil {
					if errors.Is(err, domain.ErrOrganizerNotFound) {
						return ErrUnauthorizedOrganizer
					}
					return err
				}
				if organizer.Status != "ACTIVE" {
					return ErrUnauthorizedOrganizer
				}

				foundEvent, err := tx.EventRepository().FindByID(input.EventID)
				if err != nil {
					return err
				}

				if foundEvent.OrganizerID != organizer.ID {
					return ErrUnauthorizedOrganizer
				}

				if foundEvent.Status != domain.EventStatusDraft {
					return ErrEventNotEditable
				}

				organizerID = organizer.ID
				oldBannerKey = foundEvent.BannerURL
				return nil
			},
		)
		if err != nil {
			return nil, err
		}

		newObjectKey = fmt.Sprintf(
			"event-banners/%s/%s-%s%s",
			organizerID.String(),
			input.EventID.String(),
			uuid.New().String(),
			extension,
		)

		u.logger.Info(
			"event_banner_replacement_started",
			"event_id", input.EventID,
			"organizer_id", organizerID,
			"new_banner_key", newObjectKey,
			"content_type", input.BannerContentType,
			"file_size", input.BannerSize,
		)

		if err := u.storageService.Upload(
			ctx,
			newObjectKey,
			input.BannerReader,
			input.BannerContentType,
		); err != nil {
			u.logger.Error(
				"event_banner_replacement_upload_failed",
				"event_id", input.EventID,
				"organizer_id", organizerID,
				"new_banner_key", newObjectKey,
				"error", err,
			)
			return nil, err
		}
	}

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

			// Extension point for Week 3: Block event editing once bookings exist.
			// hasBookings, err := tx.BookingRepository().ExistsByEventID(foundEvent.ID)
			// if err != nil { return err }
			// if hasBookings { return ErrEventNotEditable }

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
			if newObjectKey != "" {
				foundEvent.BannerURL = newObjectKey
			}
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

			var ticketTypes []*domain.TicketType

			if input.SeatLayoutType == SeatLayoutTypeGeneral {
				existingTicketTypes, err := tx.TicketTypeRepository().FindByEventID(ctx, foundEvent.ID)
				if err != nil {
					u.logger.Error(
						"event_ticket_types_find_failed",
						"event_id", foundEvent.ID,
						"error", err,
					)
					return err
				}

				requestedTicketIDs := make(map[uuid.UUID]struct{}, len(input.TicketTypes))

				for _, ticketInput := range input.TicketTypes {
					if ticketInput.ID != nil {
						requestedTicketIDs[*ticketInput.ID] = struct{}{}
					}
				}

				for _, ticketInput := range input.TicketTypes {
					if ticketInput.ID == nil {
						ticketType := &domain.TicketType{
							ID:                uuid.New(),
							EventID:           foundEvent.ID,
							Name:              strings.TrimSpace(ticketInput.Name),
							Price:             ticketInput.Price,
							TotalQuantity:     ticketInput.Capacity,
							AvailableQuantity: ticketInput.Capacity,
							Description:       strings.TrimSpace(ticketInput.Description),
						}

						if err := tx.TicketTypeRepository().Create(ctx, ticketType); err != nil {
							u.logger.Error(
								"event_ticket_type_create_failed",
								"event_id", foundEvent.ID,
								"ticket_type_id", ticketType.ID,
								"error", err,
							)
							return err
						}

						ticketTypes = append(ticketTypes, ticketType)
						continue
					}

					existingTicket := findTicketTypeByID(existingTicketTypes, *ticketInput.ID)
					if existingTicket == nil {
						u.logger.Warn(
							"event_ticket_type_not_found",
							"event_id", foundEvent.ID,
							"ticket_type_id", *ticketInput.ID,
						)
						return domain.ErrTicketTypeNotFound
					}

					existingTicket.Name = strings.TrimSpace(ticketInput.Name)
					existingTicket.Price = ticketInput.Price
					existingTicket.TotalQuantity = ticketInput.Capacity
					existingTicket.AvailableQuantity = ticketInput.Capacity
					existingTicket.Description = strings.TrimSpace(ticketInput.Description)

					if err := tx.TicketTypeRepository().Update(ctx, existingTicket); err != nil {
						u.logger.Error(
							"event_ticket_type_update_failed",
							"event_id", foundEvent.ID,
							"ticket_type_id", existingTicket.ID,
							"error", err,
						)
						return err
					}

					ticketTypes = append(ticketTypes, existingTicket)
				}

				for _, existingTicket := range existingTicketTypes {
					if _, exists := requestedTicketIDs[existingTicket.ID]; exists {
						continue
					}

					if err := tx.TicketTypeRepository().Delete(ctx, existingTicket.ID); err != nil {
						u.logger.Error(
							"event_ticket_type_delete_failed",
							"event_id", foundEvent.ID,
							"ticket_type_id", existingTicket.ID,
							"error", err,
						)
						return err
					}
				}
			}

			output = UpdateEventOutput{
				Event:        foundEvent,
				Schedule:     schedule,
				Setting:      setting,
				Cancellation: cancellation,
				Contact:      contact,
				TicketTypes:  ticketTypes,
			}

			return nil
		},
	)
	if err != nil {
		if newObjectKey != "" {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if delErr := u.storageService.Delete(cleanupCtx, newObjectKey); delErr != nil {
				u.logger.Error(
					"event_banner_replacement_compensation_delete_failed",
					"event_id", input.EventID,
					"banner_key", newObjectKey,
					"error", delErr,
				)
			}
		}

		return nil, err
	}

	if newObjectKey != "" {
		u.logger.Info(
			"event_banner_replacement_succeeded",
			"event_id", output.Event.ID,
			"new_banner_key", newObjectKey,
		)

		if oldBannerKey != "" && oldBannerKey != newObjectKey {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if delErr := u.storageService.Delete(cleanupCtx, oldBannerKey); delErr != nil {
				u.logger.Warn(
					"event_old_banner_delete_failed",
					"event_id", output.Event.ID,
					"old_banner_key", oldBannerKey,
					"error", delErr,
				)
			} else {
				u.logger.Info(
					"event_old_banner_deleted",
					"event_id", output.Event.ID,
					"old_banner_key", oldBannerKey,
				)
			}
		}
	}

	u.logger.Info(
		"event_updated_successfully",
		"event_id", output.Event.ID,
		"user_id", input.UserID,
		"status", output.Event.Status,
	)

	return &output, nil
}

type GetMyEventsOutput struct {
	Events []*domain.OrganizerEvent
	Page   int
	Limit  int
	Total  int64
	// BannerImageURL string
}

type EventDetailsOutput struct {
	Event        *EventDetailsEvent
	Schedule     *EventDetailsSchedule
	Venue        *EventDetailsVenue
	Setting      *EventDetailsSetting
	Cancellation *EventDetailsCancellation
	Contact      *EventDetailsContact
	TicketTypes  []EventDetailsTicketType
	SeatLayout   *EventDetailsSeatLayout
	Statistics   EventDetailsStatistics
}

type EventDetailsEvent struct {
	ID                  uuid.UUID
	OrganizerID         uuid.UUID
	CategoryID          uuid.UUID
	CategoryName        string
	VenueID             *uuid.UUID
	EventType           string
	OnlineURL           string
	Title               string
	Description         string
	BannerURL           string
	BannerImageURL      string
	Language            string
	AgeRestriction      int
	Visibility          string
	Highlights          string
	Rules               string
	AttendeeInformation string
	Status              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type EventDetailsSchedule struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	EventDate time.Time
	StartTime *string
	EndTime   *string
	IsAllDay  bool
}

type EventDetailsVenue struct {
	ID            uuid.UUID
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

type EventDetailsSetting struct {
	ID                  uuid.UUID
	EventID             uuid.UUID
	SeatLayoutType      string
	BookingLimitPerUser int
	SalesStartDate      *time.Time
	SalesEndDate        *time.Time
}

type EventDetailsCancellation struct {
	ID                        uuid.UUID
	EventID                   uuid.UUID
	CancellationAllowed       bool
	CancellationDeadlineHours int
	RefundPolicy              string
	RefundPercentage          int
}

type EventDetailsContact struct {
	ID      uuid.UUID
	EventID uuid.UUID
	Name    string
	Phone   string
	Email   string
}

type EventDetailsTicketType struct {
	ID                uuid.UUID
	EventID           uuid.UUID
	Name              string
	Price             float64
	TotalQuantity     int
	AvailableQuantity int
	SoldQuantity      int
	Description       string
	Status            string
}

type EventDetailsSeatLayout struct {
	ID         uuid.UUID
	EventID    uuid.UUID
	LayoutName string
	Sections   []EventDetailsSeatSection
}

type EventDetailsSeatSection struct {
	ID    uuid.UUID
	Name  string
	Price float64
	Rows  []EventDetailsSeatRow
}

type EventDetailsSeatRow struct {
	ID      uuid.UUID
	RowName string
	Seats   []EventDetailsSeat
}

type EventDetailsSeat struct {
	ID         uuid.UUID
	SeatNumber int
	Status     string
}

type EventDetailsStatistics struct {
	TicketsSold    int
	TicketCapacity int
	PercentageSold int
	Revenue        float64
	IsMock         bool
}

func (u *EventUsecase) GetMyEvents(
	ctx context.Context,
	userID uuid.UUID,
	page int,
	limit int,
	status string,
	search string,
) (*GetMyEventsOutput, error) {

	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 10
	}

	if limit > 100 {
		limit = 100
	}

	var organizer *domain.Organizer
	var events []*domain.OrganizerEvent
	var total int64

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			var err error

			organizer, err = tx.OrganizerRepository().FindByUserID(userID)
			if err != nil {
				if errors.Is(err, domain.ErrOrganizerNotFound) {
					u.logger.Warn(
						"event_list_organizer_not_found",
						"user_id", userID,
					)

					return ErrUnauthorizedOrganizer
				}

				u.logger.Error(
					"event_list_organizer_lookup_failed",
					"user_id", userID,
					"error", err,
				)

				return err
			}

			if organizer.Status != "ACTIVE" {
				u.logger.Warn(
					"event_list_organizer_inactive",
					"user_id", userID,
					"organizer_id", organizer.ID,
					"status", organizer.Status,
				)

				return ErrUnauthorizedOrganizer
			}

			events, total, err = tx.EventRepository().FindByOrganizerID(
				organizer.ID,
				page,
				limit,
				status,
				search,
			)
			if err != nil {
				u.logger.Error(
					"event_list_failed",
					"user_id", userID,
					"organizer_id", organizer.ID,
					"error", err,
				)

				return err
			}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	for _, event := range events {
		if strings.TrimSpace(event.BannerURL) == "" {
			continue
		}

		bannerImageURL, err := u.storageService.GetPresignedURL(
			ctx,
			event.BannerURL,
		)
		if err != nil {
			u.logger.Error(
				"event_banner_presign_failed",
				"user_id", userID,
				"organizer_id", organizer.ID,
				"event_id", event.ID,
				"banner_key", event.BannerURL,
				"error", err,
			)

			return nil, err
		}

		event.BannerImageURL = bannerImageURL
	}

	u.logger.Info(
		"organizer_events_fetched",
		"user_id", userID,
		"organizer_id", organizer.ID,
		"count", len(events),
		"total", total,
		"page", page,
		"limit", limit,
	)

	return &GetMyEventsOutput{
		Events: events,
		Page:   page,
		Limit:  limit,
		Total:  total,
	}, nil
}

func isSupportedBannerContentType(contentType string) bool {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func (u *EventUsecase) GetEventDetails(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
) (*EventDetailsOutput, error) {
	var output EventDetailsOutput

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			organizer, err := tx.OrganizerRepository().FindByUserID(userID)
			if err != nil {
				if errors.Is(err, domain.ErrOrganizerNotFound) {
					u.logger.Warn(
						"event_details_organizer_not_found",
						"user_id", userID,
					)

					return ErrUnauthorizedOrganizer
				}

				u.logger.Error(
					"event_details_organizer_lookup_failed",
					"user_id", userID,
					"error", err,
				)

				return err
			}

			if organizer.Status != "ACTIVE" {
				u.logger.Warn(
					"event_details_organizer_inactive",
					"user_id", userID,
					"organizer_id", organizer.ID,
					"status", organizer.Status,
				)

				return ErrUnauthorizedOrganizer
			}

			event, err := tx.EventRepository().FindByID(eventID)
			if err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					return domain.ErrEventNotFound
				}

				u.logger.Error(
					"event_details_lookup_failed",
					"user_id", userID,
					"event_id", eventID,
					"error", err,
				)

				return err
			}

			if event.OrganizerID != organizer.ID {
				u.logger.Warn(
					"event_details_ownership_denied",
					"user_id", userID,
					"organizer_id", organizer.ID,
					"event_id", eventID,
					"event_organizer_id", event.OrganizerID,
				)

				return ErrUnauthorizedOrganizer
			}

			category, err := tx.CategoryRepository().FindByID(event.CategoryID)
			if err != nil {
				return err
			}

			schedule, err := tx.EventScheduleRepository().FindByEventID(eventID)
			if err != nil {
				return err
			}

			setting, err := tx.EventSettingRepository().FindByEventID(eventID)
			if err != nil {
				return err
			}

			cancellation, err := tx.EventCancellationRepository().FindByEventID(eventID)
			if err != nil {
				return err
			}

			contact, err := tx.EventContactRepository().FindByEventID(eventID)
			if err != nil {
				return err
			}

			var venue *domain.Venue

			if event.VenueID != nil {
				venue, err = tx.VenueRepository().FindByID(*event.VenueID)
				if err != nil {
					return err
				}
			}

			var seatLayout *domain.SeatLayout

			if layout, layoutErr := tx.SeatLayoutRepository().FindByEventID(eventID); layoutErr == nil {
				seatLayout = layout
			} else if !errors.Is(layoutErr, domain.ErrSeatLayoutNotFound) {
				return layoutErr
			}

			bannerImageURL := ""

			if strings.TrimSpace(event.BannerURL) != "" {
				bannerImageURL, err = u.storageService.GetPresignedURL(
					ctx,
					event.BannerURL,
				)
				if err != nil {
					u.logger.Error(
						"event_details_banner_presign_failed",
						"user_id", userID,
						"event_id", eventID,
						"banner_key", event.BannerURL,
						"error", err,
					)

					return err
				}
			}

			output.Event = &EventDetailsEvent{
				ID:                  event.ID,
				OrganizerID:         event.OrganizerID,
				CategoryID:          event.CategoryID,
				CategoryName:        category.Name,
				VenueID:             event.VenueID,
				EventType:           event.EventType,
				OnlineURL:           event.OnlineURL,
				Title:               event.Title,
				Description:         event.Description,
				BannerURL:           event.BannerURL,
				BannerImageURL:      bannerImageURL,
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

			output.Schedule = &EventDetailsSchedule{
				ID:        schedule.ID,
				EventID:   schedule.EventID,
				EventDate: schedule.EventDate,
				StartTime: formatEventTime(schedule.StartTime),
				EndTime:   formatEventTime(schedule.EndTime),
				IsAllDay:  schedule.IsAllDay,
			}

			if venue != nil {
				output.Venue = &EventDetailsVenue{
					ID:            venue.ID,
					GooglePlaceID: venue.GooglePlaceID,
					Name:          venue.Name,
					Address:       venue.Address,
					City:          venue.City,
					State:         venue.State,
					Country:       venue.Country,
					PostalCode:    venue.PostalCode,
					Latitude:      venue.Latitude,
					Longitude:     venue.Longitude,
				}
			}

			output.Setting = &EventDetailsSetting{
				ID:                  setting.ID,
				EventID:             setting.EventID,
				SeatLayoutType:      setting.SeatLayoutType,
				BookingLimitPerUser: setting.BookingLimitPerUser,
				SalesStartDate:      setting.SalesStartDate,
				SalesEndDate:        setting.SalesEndDate,
			}

			output.Cancellation = &EventDetailsCancellation{
				ID:                        cancellation.ID,
				EventID:                   cancellation.EventID,
				CancellationAllowed:       cancellation.CancellationAllowed,
				CancellationDeadlineHours: cancellation.CancellationDeadlineHours,
				RefundPolicy:              cancellation.RefundPolicy,
				RefundPercentage:          cancellation.RefundPercentage,
			}

			output.Contact = &EventDetailsContact{
				ID:      contact.ID,
				EventID: contact.EventID,
				Name:    contact.Name,
				Phone:   contact.Phone,
				Email:   contact.Email,
			}

			var totalCapacity int
			var totalSold int

			if setting.SeatLayoutType == SeatLayoutTypeSeated {
				if seatLayout != nil {
					sectionStats, err := tx.SeatSectionRepository().FindWithStatsBySeatLayoutID(seatLayout.ID)
					if err != nil {
						return err
					}

					output.TicketTypes = make([]EventDetailsTicketType, 0, len(sectionStats))

					for _, stat := range sectionStats {
						status := "AVAILABLE"
						if stat.AvailableQuantity == 0 {
							status = "SOLD_OUT"
						} else if stat.TotalCapacity > 0 && stat.AvailableQuantity <= stat.TotalCapacity/5 {
							status = "SELLING_FAST"
						}

						output.TicketTypes = append(
							output.TicketTypes,
							EventDetailsTicketType{
								ID:                stat.ID,
								EventID:           eventID,
								Name:              stat.Name,
								Price:             stat.Price,
								TotalQuantity:     stat.TotalCapacity,
								AvailableQuantity: stat.AvailableQuantity,
								SoldQuantity:      stat.SoldQuantity,
								Description:       stat.Name,
								Status:            status,
							},
						)

						totalCapacity += stat.TotalCapacity
						totalSold += stat.SoldQuantity
					}
				} else {
					output.TicketTypes = []EventDetailsTicketType{}
				}
			} else {
				ticketTypes, err := tx.TicketTypeRepository().FindByEventID(
					ctx,
					eventID,
				)
				if err != nil {
					return err
				}

				output.TicketTypes = make(
					[]EventDetailsTicketType,
					0,
					len(ticketTypes),
				)

				for _, ticketType := range ticketTypes {
					sold := ticketType.TotalQuantity - ticketType.AvailableQuantity

					if sold < 0 {
						sold = 0
					}

					status := "AVAILABLE"

					if ticketType.AvailableQuantity == 0 {
						status = "SOLD_OUT"
					} else if ticketType.AvailableQuantity <= ticketType.TotalQuantity/5 {
						status = "SELLING_FAST"
					}

					output.TicketTypes = append(
						output.TicketTypes,
						EventDetailsTicketType{
							ID:                ticketType.ID,
							EventID:           ticketType.EventID,
							Name:              ticketType.Name,
							Price:             ticketType.Price,
							TotalQuantity:     ticketType.TotalQuantity,
							AvailableQuantity: ticketType.AvailableQuantity,
							SoldQuantity:      sold,
							Description:       ticketType.Description,
							Status:            status,
						},
					)

					totalCapacity += ticketType.TotalQuantity
					totalSold += sold
				}
			}

			percentageSold := 0

			if totalCapacity > 0 {
				percentageSold = int(
					math.Round(
						float64(totalSold) /
							float64(totalCapacity) *
							100,
					),
				)
			}

			// Booking/payment functionality is not implemented yet.
			// Revenue is intentionally mocked until the payment module exists.
			const mockRevenue = 15000.00

			output.Statistics = EventDetailsStatistics{
				TicketsSold:    totalSold,
				TicketCapacity: totalCapacity,
				PercentageSold: percentageSold,
				Revenue:        mockRevenue,
				IsMock:         true,
			}

			if setting.SeatLayoutType == SeatLayoutTypeSeated && seatLayout != nil {
				sections, err := tx.SeatSectionRepository().FindBySeatLayoutID(seatLayout.ID)
				if err != nil {
					return err
				}

				detailsSections := make([]EventDetailsSeatSection, 0, len(sections))
				for _, section := range sections {
					rows, err := tx.SeatRowRepository().FindBySectionID(section.ID)
					if err != nil {
						return err
					}

					detailsRows := make([]EventDetailsSeatRow, 0, len(rows))
					for _, row := range rows {
						seats, err := tx.SeatRepository().FindByRowID(row.ID)
						if err != nil {
							return err
						}

						detailsSeats := make([]EventDetailsSeat, 0, len(seats))
						for _, seat := range seats {
							detailsSeats = append(detailsSeats, EventDetailsSeat{
								ID:         seat.ID,
								SeatNumber: seat.SeatNumber,
								Status:     seat.Status,
							})
						}

						detailsRows = append(detailsRows, EventDetailsSeatRow{
							ID:      row.ID,
							RowName: row.RowName,
							Seats:   detailsSeats,
						})
					}

					detailsSections = append(detailsSections, EventDetailsSeatSection{
						ID:    section.ID,
						Name:  section.Name,
						Price: section.Price,
						Rows:  detailsRows,
					})
				}

				output.SeatLayout = &EventDetailsSeatLayout{
					ID:         seatLayout.ID,
					EventID:    seatLayout.EventID,
					LayoutName: seatLayout.LayoutName,
					Sections:   detailsSections,
				}
			}

			return nil
		},
	)

	if err != nil {
		u.logger.Error(
			"event_details_fetch_failed",
			"user_id", userID,
			"event_id", eventID,
			"error", err,
		)

		return nil, err
	}

	u.logger.Info(
		"event_details_fetched",
		"user_id", userID,
		"event_id", eventID,
		"ticket_type_count", len(output.TicketTypes),
		"tickets_sold", output.Statistics.TicketsSold,
	)

	return &output, nil
}

func (u *EventUsecase) DeleteEvent(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
) error {
	u.logger.Info(
		"event_delete_started",
		"event_id", eventID,
		"user_id", userID,
	)

	var bannerKey string

	txErr := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			organizer, err := tx.OrganizerRepository().FindByUserID(userID)
			if err != nil {
				if errors.Is(err, domain.ErrOrganizerNotFound) {
					u.logger.Warn(
						"event_delete_organizer_not_found",
						"user_id", userID,
					)
					return ErrUnauthorizedOrganizer
				}
				u.logger.Error(
					"event_delete_organizer_lookup_failed",
					"user_id", userID,
					"error", err,
				)
				return err
			}

			if organizer.Status != "ACTIVE" {
				u.logger.Warn(
					"event_delete_organizer_inactive",
					"user_id", userID,
					"organizer_id", organizer.ID,
					"status", organizer.Status,
				)
				return ErrUnauthorizedOrganizer
			}

			foundEvent, err := tx.EventRepository().FindByID(eventID)
			if err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					u.logger.Warn(
						"event_delete_not_found",
						"event_id", eventID,
					)
					return domain.ErrEventNotFound
				}
				u.logger.Error(
					"event_delete_find_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			if foundEvent.OrganizerID != organizer.ID {
				u.logger.Warn(
					"event_delete_forbidden",
					"event_id", eventID,
					"organizer_id", organizer.ID,
					"event_organizer_id", foundEvent.OrganizerID,
				)
				return ErrUnauthorizedOrganizer
			}

			if foundEvent.Status != domain.EventStatusDraft {
				u.logger.Warn(
					"event_delete_not_draft",
					"event_id", eventID,
					"status", foundEvent.Status,
				)
				return ErrEventNotDeletable
			}

			bannerKey = foundEvent.BannerURL

			if err := tx.TicketTypeRepository().DeleteByEventID(ctx, eventID); err != nil {
				u.logger.Error(
					"event_delete_ticket_types_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			if err := tx.SeatLayoutRepository().DeleteByEventID(eventID); err != nil {
				u.logger.Error(
					"event_delete_seat_layout_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			if err := tx.EventScheduleRepository().DeleteByEventID(eventID); err != nil {
				u.logger.Error(
					"event_delete_schedule_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			if err := tx.EventSettingRepository().DeleteByEventID(eventID); err != nil {
				u.logger.Error(
					"event_delete_setting_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			if err := tx.EventCancellationRepository().DeleteByEventID(eventID); err != nil {
				u.logger.Error(
					"event_delete_cancellation_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			if err := tx.EventContactRepository().DeleteByEventID(eventID); err != nil {
				u.logger.Error(
					"event_delete_contact_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			if err := tx.EventRepository().Delete(eventID); err != nil {
				u.logger.Error(
					"event_delete_record_failed",
					"event_id", eventID,
					"error", err,
				)
				return err
			}

			return nil
		},
	)

	if txErr != nil {
		u.logger.Error(
			"event_delete_transaction_failed",
			"event_id", eventID,
			"user_id", userID,
			"error", txErr,
		)
		return txErr
	}

	if bannerKey = strings.TrimSpace(bannerKey); bannerKey != "" && u.storageService != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		if delErr := u.storageService.Delete(cleanupCtx, bannerKey); delErr != nil {
			u.logger.Warn(
				"event_delete_banner_cleanup_failed",
				"event_id", eventID,
				"banner_key", bannerKey,
				"error", delErr,
			)
		} else {
			u.logger.Info(
				"event_delete_banner_cleanup_succeeded",
				"event_id", eventID,
				"banner_key", bannerKey,
			)
		}
	}

	u.logger.Info(
		"event_deleted_successfully",
		"event_id", eventID,
		"user_id", userID,
	)

	return nil
}

func formatEventTime(value time.Time) *string {
	if value.IsZero() {
		return nil
	}

	formatted := value.Format("15:04:05")
	return &formatted
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

	if input.SeatLayoutType == SeatLayoutTypeGeneral {
		if err := validateTicketTypes(input.TicketTypes); err != nil {
			return ErrInvalidEventSettings
		}
		for _, ticket := range input.TicketTypes {
			// IDs are assigned by the use case during creation; accepting client
			// IDs could accidentally turn a create request into a cross-event update.
			if ticket.ID != nil {
				return ErrInvalidEventSettings
			}
		}
		if len(input.SeatLayout.Sections) > 0 || strings.TrimSpace(input.SeatLayout.LayoutName) != "" {
			return ErrInvalidEventSettings
		}
	}

	if input.SeatLayoutType == SeatLayoutTypeSeated {
		if len(input.TicketTypes) != 0 {
			return ErrInvalidEventSettings
		}
		if err := validateSeatLayoutStructure(input.SeatLayout.LayoutName, input.SeatLayout.Sections); err != nil {
			return err
		}
	}

	if input.BookingLimitPerUser <= 0 {
		return ErrInvalidEventSettings
	}

	if input.CancellationAllowed &&
		input.CancellationDeadlineHours <= 0 {
		return ErrInvalidCancellation
	}
	if input.RefundPercentage < 0 || input.RefundPercentage > 100 {
		return ErrInvalidCancellation
	}
	if input.CancellationAllowed && strings.TrimSpace(input.RefundPolicy) == "" {
		return ErrInvalidCancellation
	}

	if !input.CancellationAllowed &&
		(input.CancellationDeadlineHours != 0 || input.RefundPercentage != 0) {
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

	if math.IsNaN(input.Latitude) || math.IsInf(input.Latitude, 0) || input.Latitude < -90 || input.Latitude > 90 {
		return errors.New("invalid venue latitude")
	}

	if math.IsNaN(input.Longitude) || math.IsInf(input.Longitude, 0) || input.Longitude < -180 || input.Longitude > 180 {
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

	if input.SeatLayoutType == SeatLayoutTypeGeneral {
		if err := validateTicketTypes(input.TicketTypes); err != nil {
			return ErrInvalidEventSettings
		}
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

func validateTicketTypes(ticketTypes []TicketTypeInput) error {
	if len(ticketTypes) == 0 {
		return errors.New("at least one ticket type is required")
	}

	seenNames := make(map[string]struct{}, len(ticketTypes))

	for _, ticket := range ticketTypes {
		name := strings.TrimSpace(ticket.Name)

		if name == "" {
			return errors.New("ticket type name is required")
		}

		if len(name) > 100 {
			return errors.New("ticket type name must not exceed 100 characters")
		}

		if ticket.Price < 0 {
			return errors.New("ticket type price cannot be negative")
		}

		if ticket.Capacity <= 0 {
			return errors.New("ticket type capacity must be greater than zero")
		}

		key := strings.ToLower(name)

		if _, exists := seenNames[key]; exists {
			return errors.New("duplicate ticket type name")
		}

		seenNames[key] = struct{}{}
	}

	return nil
}

func findTicketTypeByID(
	ticketTypes []*domain.TicketType,
	id uuid.UUID,
) *domain.TicketType {
	for _, ticketType := range ticketTypes {
		if ticketType.ID == id {
			return ticketType
		}
	}

	return nil
}

func (u *EventUsecase) PublishEvent(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
) error {

	u.logger.Info(
		"event_publish_started",
		"user_id", userID,
		"event_id", eventID,
	)

	err := u.transactionManager.WithinTransaction(

		func(tx domain.TransactionRepositories) error {
			
			// 1. Verify organizer.
			organizer, err := tx.OrganizerRepository().FindByUserID(userID)

			if err != nil {

				if errors.Is(err, domain.ErrOrganizerNotFound) {

					u.logger.Warn(
						"event_publish_organizer_not_found",
						"user_id", userID,
						"event_id", eventID,
					)
					return ErrUnauthorizedOrganizer
				}

				u.logger.Error(
					"event_publish_organizer_lookup_failed",
					"user_id", userID,
					"event_id", eventID,
					"error", err,
				)

				return err
			}

			if organizer.Status != "ACTIVE" {
				u.logger.Warn(
					"event_publish_organizer_inactive",
					"user_id", userID,
					"event_id", eventID,
					"organizer_id", organizer.ID,
					"status", organizer.Status,
				)

				return ErrUnauthorizedOrganizer
			}

			// 2. Find event.
			event, err := tx.EventRepository().FindByID(eventID)
			if err != nil {
				return err
			}

			// 3. Ownership check.
			if event.OrganizerID != organizer.ID {
				u.logger.Warn(
					"event_publish_ownership_denied",
					"user_id", userID,
					"event_id", eventID,
					"event_organizer_id", event.OrganizerID,
					"user_organizer_id", organizer.ID,
				)

				return ErrUnauthorizedOrganizer
			}

			// 4. Only drafts can be published.
			if event.Status != domain.EventStatusDraft {
				u.logger.Warn(
					"event_publish_status_denied",
					"user_id", userID,
					"event_id", eventID,
					"status", event.Status,
				)

				return ErrEventNotPublishable
			}

			// 5. Category must still be active.
			category, err := tx.CategoryRepository().FindByID(event.CategoryID)
			if err != nil {
				return err
			}

			if category.Status != domain.CategoryStatusActive {
				return ErrEventNotPublishable
			}

			// 6. Required event information.
			if strings.TrimSpace(event.Title) == "" ||
				strings.TrimSpace(event.Description) == "" ||
				strings.TrimSpace(event.BannerURL) == "" {
				return ErrEventNotPublishable
			}

			// 7. Event type validation.
			switch event.EventType {
			case domain.EventTypeOnline:
				if strings.TrimSpace(event.OnlineURL) == "" {
					return ErrEventNotPublishable
				}

				if event.VenueID != nil {
					return ErrEventNotPublishable
				}

			case domain.EventTypePhysical:
				if event.VenueID == nil ||
					strings.TrimSpace(event.OnlineURL) != "" {
					return ErrEventNotPublishable
				}

			case domain.EventTypeHybrid:
				if event.VenueID == nil ||
					strings.TrimSpace(event.OnlineURL) == "" {
					return ErrEventNotPublishable
				}

			default:
				return ErrEventNotPublishable
			}

			// 8. Schedule is mandatory.
			schedule, err := tx.EventScheduleRepository().FindByEventID(eventID)
			if err != nil {
				return err
			}

			if schedule == nil || schedule.EventDate.IsZero() {
				return ErrEventNotPublishable
			}

			if !schedule.IsAllDay {
				if schedule.StartTime.IsZero() ||
					schedule.EndTime.IsZero() ||
					!schedule.EndTime.After(schedule.StartTime) {
					return ErrEventNotPublishable
				}
			}

			// 9. Settings are mandatory.
			setting, err := tx.EventSettingRepository().FindByEventID(eventID)
			if err != nil {
				return err
			}

			if setting == nil ||
				setting.BookingLimitPerUser <= 0 {
				return ErrEventNotPublishable
			}

			if setting.SeatLayoutType != SeatLayoutTypeGeneral &&
				setting.SeatLayoutType != SeatLayoutTypeSeated {
				return ErrEventNotPublishable
			}

			// Online events can only use General Admission.
			if event.EventType == domain.EventTypeOnline &&
				setting.SeatLayoutType == SeatLayoutTypeSeated {
				return ErrEventNotPublishable
			}

			// 10. Sales window validation.
			if err := validateSalesDates(
				setting.SalesStartDate,
				setting.SalesEndDate,
				schedule.EventDate,
			); err != nil {
				u.logger.Warn(
					"event_publish_sales_window_invalid",
					"user_id", userID,
					"event_id", eventID,
					"error", err,
				)

				return ErrEventNotPublishable
			}

			// 11. Physical / hybrid events require a venue.
			if event.EventType == domain.EventTypePhysical ||
				event.EventType == domain.EventTypeHybrid {

				if event.VenueID == nil {
					return ErrEventNotPublishable
				}

				venue, err := tx.VenueRepository().FindByID(*event.VenueID)
				if err != nil {
					return err
				}

				if strings.TrimSpace(venue.GooglePlaceID) == "" ||
					strings.TrimSpace(venue.Name) == "" ||
					strings.TrimSpace(venue.Address) == "" ||
					strings.TrimSpace(venue.City) == "" ||
					strings.TrimSpace(venue.State) == "" ||
					strings.TrimSpace(venue.Country) == "" {
					return ErrEventNotPublishable
				}
			}

			// 12. Cancellation settings.
			cancellation, err :=
				tx.EventCancellationRepository().FindByEventID(eventID)
			if err != nil {
				return err
			}

			if cancellation == nil {
				return ErrEventNotPublishable
			}

			if cancellation.CancellationAllowed {
				if cancellation.CancellationDeadlineHours <= 0 ||
					strings.TrimSpace(cancellation.RefundPolicy) == "" ||
					cancellation.RefundPercentage < 0 ||
					cancellation.RefundPercentage > 100 {
					return ErrEventNotPublishable
				}
			}

			// 13. Ticket / seat inventory.
			if setting.SeatLayoutType == SeatLayoutTypeGeneral {
				ticketTypes, err :=
					tx.TicketTypeRepository().FindByEventID(ctx, eventID)
				if err != nil {
					return err
				}

				if len(ticketTypes) == 0 {
					return ErrEventNotPublishable
				}

				for _, ticket := range ticketTypes {
					if strings.TrimSpace(ticket.Name) == "" ||
						ticket.Price < 0 ||
						ticket.TotalQuantity <= 0 ||
						ticket.AvailableQuantity < 0 ||
						ticket.AvailableQuantity > ticket.TotalQuantity {
						return ErrEventNotPublishable
					}
				}
			}

			if setting.SeatLayoutType == SeatLayoutTypeSeated {
				layout, err :=
					tx.SeatLayoutRepository().FindByEventID(eventID)

				if err != nil {
					if errors.Is(err, domain.ErrSeatLayoutNotFound) {
						return ErrEventNotPublishable
					}

					return err
				}

				if layout == nil ||
					strings.TrimSpace(layout.LayoutName) == "" {
					return ErrEventNotPublishable
				}

				sections, err :=
					tx.SeatSectionRepository().FindWithStatsBySeatLayoutID(layout.ID)
				if err != nil {
					return err
				}

				if len(sections) == 0 {
					return ErrEventNotPublishable
				}

				for _, section := range sections {
					if strings.TrimSpace(section.Name) == "" ||
						section.Price < 0 ||
						section.TotalCapacity <= 0 {
						return ErrEventNotPublishable
					}
				}
			}

			// 14. Publish.
			now := time.Now()

			event.Status = domain.EventStatusPublished
			event.UpdatedAt = now

			if err := tx.EventRepository().Update(event); err != nil {
				u.logger.Error(
					"event_publish_update_failed",
					"user_id", userID,
					"event_id", eventID,
					"error", err,
				)

				return err
			}

			u.logger.Info(
				"event_published",
				"user_id", userID,
				"organizer_id", organizer.ID,
				"event_id", eventID,
				"status", domain.EventStatusPublished,
			)

			return nil
		},
	)

	if err != nil {
		u.logger.Warn(
			"event_publish_failed",
			"user_id", userID,
			"event_id", eventID,
			"error", err,
		)

		return err
	}

	return nil
}
