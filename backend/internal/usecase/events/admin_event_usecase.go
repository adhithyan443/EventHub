package events

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/service/storage"
	"github.com/google/uuid"
)

var (
	ErrEventCannotBeDeletedWithBookings = errors.New("event cannot be deleted because it has existing bookings, tickets, or active reservations")
	ErrPublishedEventStructuralEdit     = errors.New("structural fields (dates, venue, layout, tickets) cannot be modified on published events")
	ErrCancelledEventNotEditable        = errors.New("cancelled events cannot be edited")
)

type AdminEventUsecase struct {
	txManager      domain.TransactionManager
	eventRepo      domain.EventRepository
	storageService *storage.S3Service
	logger         *slog.Logger
}

func NewAdminEventUsecase(
	txManager domain.TransactionManager,
	eventRepo domain.EventRepository,
	storageService *storage.S3Service,
	logger *slog.Logger,
) *AdminEventUsecase {
	if logger == nil {
		logger = slog.Default()
	}

	return &AdminEventUsecase{
		txManager:      txManager,
		eventRepo:      eventRepo,
		storageService: storageService,
		logger:         logger,
	}
}

func (u *AdminEventUsecase) ListEvents(
	ctx context.Context,
	filter domain.AdminEventFilter,
) (*domain.AdminEventList, error) {
	u.logger.InfoContext(ctx, "admin_event_list_requested",
		"page", filter.Page,
		"limit", filter.Limit,
		"status", filter.Status,
		"category_id", filter.CategoryID,
		"event_type", filter.EventType,
		"organizer_id", filter.OrganizerID,
		"search", filter.Search,
	)

	events, total, err := u.eventRepo.FindAllEvents(filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "admin_event_list_failed", "error", err)
		return nil, err
	}

	statusCounts, err := u.eventRepo.GetAdminStatusCounts()
	if err != nil {
		u.logger.WarnContext(ctx, "admin_event_status_counts_failed", "error", err)
	}

	// Presign banner URLs if storage service is available
	if u.storageService != nil {
		for _, ev := range events {
			if strings.TrimSpace(ev.BannerURL) != "" {
				if presigned, presignErr := u.storageService.GetPresignedURL(ctx, ev.BannerURL); presignErr == nil {
					ev.BannerImageURL = presigned
				}
			}
		}
	}

	return &domain.AdminEventList{
		Events:       events,
		Total:        total,
		Page:         filter.Page,
		Limit:        filter.Limit,
		StatusCounts: statusCounts,
	}, nil
}

func (u *AdminEventUsecase) GetEventDetails(
	ctx context.Context,
	eventID uuid.UUID,
) (*domain.AdminEventDetails, error) {
	u.logger.InfoContext(ctx, "admin_event_details_requested", "event_id", eventID)

	var details domain.AdminEventDetails

	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		// 1. Event
		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			if errors.Is(err, domain.ErrEventNotFound) {
				return domain.ErrEventNotFound
			}
			u.logger.ErrorContext(ctx, "admin_event_details_event_failed", "event_id", eventID, "error", err)
			return err
		}
		details.Event = event

		// 2. Organizer details
		organizer, err := tx.OrganizerRepository().FindByID(event.OrganizerID)
		if err == nil && organizer != nil {
			orgDetails := &domain.AdminEventOrganizerDetails{
				ID:     organizer.ID,
				UserID: organizer.UserID,
				Status: organizer.Status,
			}

			if profile, pErr := tx.OrganizerProfileRepository().FindByOrganizerID(organizer.ID); pErr == nil && profile != nil {
				orgDetails.BusinessName = profile.BusinessName
				orgDetails.BusinessType = profile.BusinessType
				orgDetails.Email = profile.Email
				orgDetails.Phone = profile.Phone
				orgDetails.Website = profile.Website
				orgDetails.LogoURL = profile.LogoURL
				orgDetails.GSTNumber = profile.GSTNumber
				orgDetails.PANNumber = profile.PANNumber
			}

			if user, uErr := tx.UserRepository().FindByID(organizer.UserID); uErr == nil && user != nil {
				if orgDetails.Email == "" {
					orgDetails.Email = user.Email
				}
				if orgDetails.Phone == "" {
					orgDetails.Phone = user.Phone
				}
			}

			details.Organizer = orgDetails
		}

		// 3. Category
		category, err := tx.CategoryRepository().FindByID(event.CategoryID)
		if err == nil {
			details.Category = category
		}

		// 4. Schedule
		schedule, err := tx.EventScheduleRepository().FindByEventID(eventID)
		if err == nil {
			details.Schedule = schedule
		}

		// 5. Venue
		if event.VenueID != nil {
			venue, err := tx.VenueRepository().FindByID(*event.VenueID)
			if err == nil {
				details.Venue = venue
			}
		}

		// 6. Setting
		setting, err := tx.EventSettingRepository().FindByEventID(eventID)
		if err == nil {
			details.Setting = setting
		}

		// 7. Cancellation
		cancellation, err := tx.EventCancellationRepository().FindByEventID(eventID)
		if err == nil {
			details.Cancellation = cancellation
		}

		// 8. Contact
		contact, err := tx.EventContactRepository().FindByEventID(eventID)
		if err == nil {
			details.Contact = contact
		}

		// 9. Ticket Types & Seat Layout
		if setting != nil && setting.SeatLayoutType == SeatLayoutTypeSeated {
			if layout, lErr := tx.SeatLayoutRepository().FindByEventID(eventID); lErr == nil && layout != nil {
				details.SeatLayout = layout
				if sections, sErr := tx.SeatSectionRepository().FindWithStatsBySeatLayoutID(layout.ID); sErr == nil {
					details.SeatSections = sections
				}
			}
		} else {
			if ticketTypes, ttErr := tx.TicketTypeRepository().FindByEventID(ctx, eventID); ttErr == nil {
				details.TicketTypes = ticketTypes
			}
		}

		// 10. Booking stats
		bookingStats, err := tx.BookingRepository().GetBookingStatsByEventID(ctx, eventID)
		if err == nil {
			details.BookingStats = bookingStats
		}

		// 11. Active reservations
		activeReservations, err := tx.SeatReservationRepository().CountActiveByEventID(ctx, eventID, time.Now())
		if err == nil {
			details.ActiveReservations = activeReservations
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 12. Presign banner URL
	if details.Event != nil && strings.TrimSpace(details.Event.BannerURL) != "" && u.storageService != nil {
		if presigned, presignErr := u.storageService.GetPresignedURL(ctx, details.Event.BannerURL); presignErr == nil {
			details.BannerImageURL = presigned
		}
	}

	return &details, nil
}

func (u *AdminEventUsecase) UpdateEvent(
	ctx context.Context,
	adminID uuid.UUID,
	input domain.AdminUpdateEventInput,
) (*domain.AdminEventDetails, error) {
	u.logger.InfoContext(ctx, "admin_event_update_started",
		"admin_id", adminID,
		"event_id", input.EventID,
	)

	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		foundEvent, err := tx.EventRepository().FindByID(input.EventID)
		if err != nil {
			if errors.Is(err, domain.ErrEventNotFound) {
				return domain.ErrEventNotFound
			}
			return err
		}

		// Cancelled events cannot be edited
		if foundEvent.Status == domain.EventStatusCancelled {
			u.logger.WarnContext(ctx, "admin_event_update_cancelled_rejected",
				"admin_id", adminID,
				"event_id", input.EventID,
			)
			return ErrCancelledEventNotEditable
		}

		// Published events: restrict to descriptive metadata only
		if foundEvent.Status == domain.EventStatusPublished {
			if (input.CategoryID != nil && *input.CategoryID != foundEvent.CategoryID) ||
				(input.EventType != "" && input.EventType != foundEvent.EventType) ||
				(input.VenueID != nil && (foundEvent.VenueID == nil || *input.VenueID != *foundEvent.VenueID)) ||
				input.EventDate != nil || input.StartTime != nil || input.EndTime != nil || input.IsAllDay != nil ||
				input.SeatLayoutType != "" || input.BookingLimitPerUser != nil ||
				input.SalesStartDate != nil || input.SalesEndDate != nil ||
				input.CancellationAllowed != nil || input.CancellationDeadlineHours != nil ||
				input.RefundPolicy != "" || input.RefundPercentage != nil {
				u.logger.WarnContext(ctx, "admin_event_update_published_structural_rejected",
					"admin_id", adminID,
					"event_id", input.EventID,
				)
				return ErrPublishedEventStructuralEdit
			}

			// Update allowed descriptive metadata
			if strings.TrimSpace(input.Title) != "" {
				foundEvent.Title = strings.TrimSpace(input.Title)
			}
			if strings.TrimSpace(input.Description) != "" {
				foundEvent.Description = strings.TrimSpace(input.Description)
			}
			if strings.TrimSpace(input.BannerURL) != "" {
				foundEvent.BannerURL = strings.TrimSpace(input.BannerURL)
			}
			if strings.TrimSpace(input.Language) != "" {
				foundEvent.Language = strings.TrimSpace(input.Language)
			}
			if input.AgeRestriction != nil {
				if *input.AgeRestriction < 0 {
					return ErrInvalidEventInput
				}
				foundEvent.AgeRestriction = *input.AgeRestriction
			}
			if input.Highlights != nil {
				foundEvent.Highlights = strings.TrimSpace(*input.Highlights)
			}
			if input.Rules != nil {
				foundEvent.Rules = strings.TrimSpace(*input.Rules)
			}
			if input.AttendeeInformation != nil {
				foundEvent.AttendeeInformation = strings.TrimSpace(*input.AttendeeInformation)
			}

			foundEvent.UpdatedAt = time.Now()
			if err := tx.EventRepository().Update(foundEvent); err != nil {
				return err
			}

			// Update contact if provided (preserving unspecified contact fields)
			if input.ContactName != nil || input.ContactPhone != nil || input.ContactEmail != nil {
				contact, cErr := tx.EventContactRepository().FindByEventID(foundEvent.ID)
				if cErr != nil && !errors.Is(cErr, domain.ErrEventNotFound) {
					return cErr
				}
				if contact != nil {
					if input.ContactName != nil {
						contact.Name = strings.TrimSpace(*input.ContactName)
					}
					if input.ContactPhone != nil {
						contact.Phone = strings.TrimSpace(*input.ContactPhone)
					}
					if input.ContactEmail != nil {
						contact.Email = strings.TrimSpace(*input.ContactEmail)
					}
					contact.UpdatedAt = time.Now()
					if err := tx.EventContactRepository().Update(contact); err != nil {
						return err
					}
				} else {
					name := ""
					if input.ContactName != nil {
						name = strings.TrimSpace(*input.ContactName)
					}
					phone := ""
					if input.ContactPhone != nil {
						phone = strings.TrimSpace(*input.ContactPhone)
					}
					email := ""
					if input.ContactEmail != nil {
						email = strings.TrimSpace(*input.ContactEmail)
					}
					if err := tx.EventContactRepository().Create(&domain.EventContact{
						ID:        uuid.New(),
						EventID:   foundEvent.ID,
						Name:      name,
						Phone:     phone,
						Email:     email,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}); err != nil {
						return err
					}
				}
			}

			return nil
		}

		// Draft events: full editing allowed
		if strings.TrimSpace(input.Title) != "" {
			foundEvent.Title = strings.TrimSpace(input.Title)
		}
		if strings.TrimSpace(input.Description) != "" {
			foundEvent.Description = strings.TrimSpace(input.Description)
		}
		if strings.TrimSpace(input.BannerURL) != "" {
			foundEvent.BannerURL = strings.TrimSpace(input.BannerURL)
		}
		if strings.TrimSpace(input.Language) != "" {
			foundEvent.Language = strings.TrimSpace(input.Language)
		}
		if input.AgeRestriction != nil {
			if *input.AgeRestriction < 0 {
				return ErrInvalidEventInput
			}
			foundEvent.AgeRestriction = *input.AgeRestriction
		}
		if input.Highlights != nil {
			foundEvent.Highlights = strings.TrimSpace(*input.Highlights)
		}
		if input.Rules != nil {
			foundEvent.Rules = strings.TrimSpace(*input.Rules)
		}
		if input.AttendeeInformation != nil {
			foundEvent.AttendeeInformation = strings.TrimSpace(*input.AttendeeInformation)
		}

		if input.CategoryID != nil && *input.CategoryID != uuid.Nil {
			cat, catErr := tx.CategoryRepository().FindByID(*input.CategoryID)
			if catErr != nil || cat == nil {
				return domain.ErrCategoryNotFound
			}
			foundEvent.CategoryID = *input.CategoryID
		}

		if input.EventType != "" {
			foundEvent.EventType = input.EventType
		}
		if input.OnlineURL != "" {
			foundEvent.OnlineURL = input.OnlineURL
		}
		if input.VenueID != nil {
			foundEvent.VenueID = input.VenueID
		}

		foundEvent.UpdatedAt = time.Now()
		if err := tx.EventRepository().Update(foundEvent); err != nil {
			return err
		}

		// Update or create schedule
		if input.EventDate != nil {
			schedule, sErr := tx.EventScheduleRepository().FindByEventID(foundEvent.ID)
			if sErr == nil && schedule != nil {
				schedule.EventDate = *input.EventDate
				if input.StartTime != nil {
					schedule.StartTime = *input.StartTime
				}
				if input.EndTime != nil {
					schedule.EndTime = *input.EndTime
				}
				if input.IsAllDay != nil {
					schedule.IsAllDay = *input.IsAllDay
				}
				schedule.UpdatedAt = time.Now()
				_ = tx.EventScheduleRepository().Update(schedule)
			} else {
				startTime := time.Time{}
				if input.StartTime != nil {
					startTime = *input.StartTime
				}
				endTime := time.Time{}
				if input.EndTime != nil {
					endTime = *input.EndTime
				}
				isAllDay := false
				if input.IsAllDay != nil {
					isAllDay = *input.IsAllDay
				}
				_ = tx.EventScheduleRepository().Create(&domain.EventSchedule{
					ID:        uuid.New(),
					EventID:   foundEvent.ID,
					EventDate: *input.EventDate,
					StartTime: startTime,
					EndTime:   endTime,
					IsAllDay:  isAllDay,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			}
		}

		// Update or create settings
		if input.SeatLayoutType != "" || input.BookingLimitPerUser != nil || input.SalesStartDate != nil || input.SalesEndDate != nil {
			setting, setErr := tx.EventSettingRepository().FindByEventID(foundEvent.ID)
			if setErr == nil && setting != nil {
				if input.SeatLayoutType != "" {
					setting.SeatLayoutType = input.SeatLayoutType
				}
				if input.BookingLimitPerUser != nil {
					setting.BookingLimitPerUser = *input.BookingLimitPerUser
				}
				if input.SalesStartDate != nil {
					setting.SalesStartDate = input.SalesStartDate
				}
				if input.SalesEndDate != nil {
					setting.SalesEndDate = input.SalesEndDate
				}
				setting.UpdatedAt = time.Now()
				_ = tx.EventSettingRepository().Update(setting)
			}
		}

		// Update or create cancellation
		if input.CancellationAllowed != nil || input.CancellationDeadlineHours != nil || input.RefundPolicy != "" || input.RefundPercentage != nil {
			cancellation, cErr := tx.EventCancellationRepository().FindByEventID(foundEvent.ID)
			if cErr == nil && cancellation != nil {
				if input.CancellationAllowed != nil {
					cancellation.CancellationAllowed = *input.CancellationAllowed
				}
				if input.CancellationDeadlineHours != nil {
					cancellation.CancellationDeadlineHours = *input.CancellationDeadlineHours
				}
				if input.RefundPolicy != "" {
					cancellation.RefundPolicy = input.RefundPolicy
				}
				if input.RefundPercentage != nil {
					cancellation.RefundPercentage = *input.RefundPercentage
				}
				cancellation.UpdatedAt = time.Now()
				_ = tx.EventCancellationRepository().Update(cancellation)
			}
		}

		// Update or create contact (preserving unspecified contact fields)
		if input.ContactName != nil || input.ContactPhone != nil || input.ContactEmail != nil {
			contact, cErr := tx.EventContactRepository().FindByEventID(foundEvent.ID)
			if cErr != nil && !errors.Is(cErr, domain.ErrEventNotFound) {
				return cErr
			}
			if contact != nil {
				if input.ContactName != nil {
					contact.Name = strings.TrimSpace(*input.ContactName)
				}
				if input.ContactPhone != nil {
					contact.Phone = strings.TrimSpace(*input.ContactPhone)
				}
				if input.ContactEmail != nil {
					contact.Email = strings.TrimSpace(*input.ContactEmail)
				}
				contact.UpdatedAt = time.Now()
				if err := tx.EventContactRepository().Update(contact); err != nil {
					return err
				}
			} else {
				name := ""
				if input.ContactName != nil {
					name = strings.TrimSpace(*input.ContactName)
				}
				phone := ""
				if input.ContactPhone != nil {
					phone = strings.TrimSpace(*input.ContactPhone)
				}
				email := ""
				if input.ContactEmail != nil {
					email = strings.TrimSpace(*input.ContactEmail)
				}
				if err := tx.EventContactRepository().Create(&domain.EventContact{
					ID:        uuid.New(),
					EventID:   foundEvent.ID,
					Name:      name,
					Phone:     phone,
					Email:     email,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return u.GetEventDetails(ctx, input.EventID)
}

func (u *AdminEventUsecase) PublishEvent(
	ctx context.Context,
	adminID uuid.UUID,
	eventID uuid.UUID,
) error {
	u.logger.InfoContext(ctx, "admin_event_publish_started",
		"admin_id", adminID,
		"event_id", eventID,
	)

	return u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			if errors.Is(err, domain.ErrEventNotFound) {
				return domain.ErrEventNotFound
			}
			return err
		}

		// 1. Only drafts can be published
		if event.Status != domain.EventStatusDraft {
			u.logger.WarnContext(ctx, "admin_event_publish_not_draft",
				"event_id", eventID,
				"status", event.Status,
			)
			return ErrEventNotPublishable
		}

		// 2. Organizer must be active
		organizer, err := tx.OrganizerRepository().FindByID(event.OrganizerID)
		if err != nil || organizer == nil || organizer.Status != "ACTIVE" {
			u.logger.WarnContext(ctx, "admin_event_publish_organizer_inactive",
				"event_id", eventID,
				"organizer_id", event.OrganizerID,
			)
			return ErrUnauthorizedOrganizer
		}

		// 3. Category must be active
		category, err := tx.CategoryRepository().FindByID(event.CategoryID)
		if err != nil || category == nil || category.Status != domain.CategoryStatusActive {
			u.logger.WarnContext(ctx, "admin_event_publish_category_inactive",
				"event_id", eventID,
				"category_id", event.CategoryID,
			)
			return ErrEventNotPublishable
		}

		// 4. Required fields
		if strings.TrimSpace(event.Title) == "" ||
			strings.TrimSpace(event.Description) == "" ||
			strings.TrimSpace(event.BannerURL) == "" {
			return ErrEventNotPublishable
		}

		// 5. Event type validation
		switch event.EventType {
		case domain.EventTypeOnline:
			if strings.TrimSpace(event.OnlineURL) == "" || event.VenueID != nil {
				return ErrEventNotPublishable
			}
		case domain.EventTypePhysical:
			if event.VenueID == nil || strings.TrimSpace(event.OnlineURL) != "" {
				return ErrEventNotPublishable
			}
		case domain.EventTypeHybrid:
			if event.VenueID == nil || strings.TrimSpace(event.OnlineURL) == "" {
				return ErrEventNotPublishable
			}
		default:
			return ErrEventNotPublishable
		}

		// 6. Schedule is mandatory
		schedule, err := tx.EventScheduleRepository().FindByEventID(eventID)
		if err != nil || schedule == nil || schedule.EventDate.IsZero() {
			return ErrEventNotPublishable
		}

		if !schedule.IsAllDay {
			if schedule.StartTime.IsZero() || schedule.EndTime.IsZero() || !schedule.EndTime.After(schedule.StartTime) {
				return ErrEventNotPublishable
			}
		}

		// 7. Settings are mandatory
		setting, err := tx.EventSettingRepository().FindByEventID(eventID)
		if err != nil || setting == nil || setting.BookingLimitPerUser <= 0 {
			return ErrEventNotPublishable
		}

		if setting.SeatLayoutType != SeatLayoutTypeGeneral && setting.SeatLayoutType != SeatLayoutTypeSeated {
			return ErrEventNotPublishable
		}

		if event.EventType == domain.EventTypeOnline && setting.SeatLayoutType == SeatLayoutTypeSeated {
			return ErrEventNotPublishable
		}

		// 8. Sales window validation
		if err := validateSalesDates(setting.SalesStartDate, setting.SalesEndDate, schedule.EventDate); err != nil {
			return ErrEventNotPublishable
		}

		// 9. Venue validation for physical/hybrid
		if event.EventType == domain.EventTypePhysical || event.EventType == domain.EventTypeHybrid {
			if event.VenueID == nil {
				return ErrEventNotPublishable
			}
			venue, err := tx.VenueRepository().FindByID(*event.VenueID)
			if err != nil || venue == nil {
				return ErrEventNotPublishable
			}
			if strings.TrimSpace(venue.Name) == "" || strings.TrimSpace(venue.Address) == "" {
				return ErrEventNotPublishable
			}
		}

		// 10. Cancellation settings
		cancellation, err := tx.EventCancellationRepository().FindByEventID(eventID)
		if err != nil || cancellation == nil {
			return ErrEventNotPublishable
		}

		// 11. Ticket/seat inventory
		if setting.SeatLayoutType == SeatLayoutTypeGeneral {
			ticketTypes, err := tx.TicketTypeRepository().FindByEventID(ctx, eventID)
			if err != nil || len(ticketTypes) == 0 {
				return ErrEventNotPublishable
			}
			for _, tt := range ticketTypes {
				if strings.TrimSpace(tt.Name) == "" || tt.Price.IsNegative() || tt.TotalQuantity <= 0 {
					return ErrEventNotPublishable
				}
			}
		} else if setting.SeatLayoutType == SeatLayoutTypeSeated {
			layout, err := tx.SeatLayoutRepository().FindByEventID(eventID)
			if err != nil || layout == nil || strings.TrimSpace(layout.LayoutName) == "" {
				return ErrEventNotPublishable
			}
			sections, err := tx.SeatSectionRepository().FindWithStatsBySeatLayoutID(layout.ID)
			if err != nil || len(sections) == 0 {
				return ErrEventNotPublishable
			}
			for _, section := range sections {
				if strings.TrimSpace(section.Name) == "" || section.Price.IsNegative() || section.TotalCapacity <= 0 {
					return ErrEventNotPublishable
				}
			}
		}

		// 12. Publish
		now := time.Now()
		event.Status = domain.EventStatusPublished
		event.UpdatedAt = now

		if err := tx.EventRepository().Update(event); err != nil {
			u.logger.ErrorContext(ctx, "admin_event_publish_failed", "event_id", eventID, "error", err)
			return err
		}

		u.logger.InfoContext(ctx, "admin_event_published",
			"admin_id", adminID,
			"event_id", eventID,
		)

		return nil
	})
}

func (u *AdminEventUsecase) DeleteEvent(
	ctx context.Context,
	adminID uuid.UUID,
	eventID uuid.UUID,
) error {
	u.logger.InfoContext(ctx, "admin_event_delete_started",
		"admin_id", adminID,
		"event_id", eventID,
	)

	var bannerKey string

	txErr := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		foundEvent, err := tx.EventRepository().FindByIDForUpdate(eventID)
		if err != nil {
			if errors.Is(err, domain.ErrEventNotFound) {
				return domain.ErrEventNotFound
			}
			return err
		}

		// Historical bookings check: Prevent deletion if ANY booking records exist
		// (confirmed, cancelled, pending, etc.) to protect foreign key constraints and audit history.
		totalBookings, err := tx.BookingRepository().CountByEventID(ctx, eventID)
		if err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_booking_count_failed", "event_id", eventID, "error", err)
			return err
		}
		if totalBookings > 0 {
			u.logger.WarnContext(ctx, "admin_event_delete_has_bookings",
				"event_id", eventID,
				"total_bookings", totalBookings,
			)
			return ErrEventCannotBeDeletedWithBookings
		}

		// Historical tickets check: Prevent deletion if any tickets were ever issued.
		totalTickets, err := tx.TicketRepository().CountByEventID(ctx, eventID)
		if err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_ticket_count_failed", "event_id", eventID, "error", err)
			return err
		}
		if totalTickets > 0 {
			u.logger.WarnContext(ctx, "admin_event_delete_has_tickets",
				"event_id", eventID,
				"total_tickets", totalTickets,
			)
			return ErrEventCannotBeDeletedWithBookings
		}

		activeResCount, err := tx.SeatReservationRepository().CountActiveByEventID(ctx, eventID, time.Now())
		if err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_active_res_count_failed", "event_id", eventID, "error", err)
			return err
		}
		if activeResCount > 0 {
			u.logger.WarnContext(ctx, "admin_event_delete_has_active_reservations",
				"event_id", eventID,
				"active_res_count", activeResCount,
			)
			return ErrEventCannotBeDeletedWithBookings
		}

		bannerKey = foundEvent.BannerURL

		// Cascade child configuration deletions inside transaction
		if err := tx.TicketTypeRepository().DeleteByEventID(ctx, eventID); err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_ticket_types_failed", "event_id", eventID, "error", err)
			return err
		}

		if err := tx.SeatLayoutRepository().DeleteByEventID(eventID); err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_seat_layout_failed", "event_id", eventID, "error", err)
			return err
		}

		if err := tx.EventScheduleRepository().DeleteByEventID(eventID); err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_schedule_failed", "event_id", eventID, "error", err)
			return err
		}

		if err := tx.EventSettingRepository().DeleteByEventID(eventID); err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_setting_failed", "event_id", eventID, "error", err)
			return err
		}

		if err := tx.EventCancellationRepository().DeleteByEventID(eventID); err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_cancellation_failed", "event_id", eventID, "error", err)
			return err
		}

		if err := tx.EventContactRepository().DeleteByEventID(eventID); err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_contact_failed", "event_id", eventID, "error", err)
			return err
		}

		if err := tx.EventRepository().Delete(eventID); err != nil {
			u.logger.ErrorContext(ctx, "admin_event_delete_event_failed", "event_id", eventID, "error", err)
			return err
		}

		u.logger.InfoContext(ctx, "admin_event_deleted",
			"admin_id", adminID,
			"event_id", eventID,
		)

		return nil
	})

	if txErr != nil {
		return txErr
	}

	// Outside transaction: cleanup banner if present
	if bannerKey = strings.TrimSpace(bannerKey); bannerKey != "" && u.storageService != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		if delErr := u.storageService.Delete(cleanupCtx, bannerKey); delErr != nil {
			u.logger.WarnContext(ctx, "admin_event_delete_banner_cleanup_failed",
				slog.String("event_id", eventID.String()),
				slog.String("banner_key", bannerKey),
				slog.String("error", delErr.Error()),
			)
		}
	}

	return nil
}
