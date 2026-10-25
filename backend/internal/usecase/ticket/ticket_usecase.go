package ticket

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type TicketUsecase struct {
	ticketRepository        domain.TicketRepository
	eventRepository         domain.EventRepository
	eventScheduleRepository domain.EventScheduleRepository
	userRepository          domain.UserRepository
	venueRepository         domain.VenueRepository
	ticketTypeRepository    domain.TicketTypeRepository
	logger                  *slog.Logger
}

func NewTicketUsecase(
	ticketRepository domain.TicketRepository,
	eventRepository domain.EventRepository,
	eventScheduleRepository domain.EventScheduleRepository,
	userRepository domain.UserRepository,
	venueRepository domain.VenueRepository,
	ticketTypeRepository domain.TicketTypeRepository,
	logger *slog.Logger,
) *TicketUsecase {
	return &TicketUsecase{
		ticketRepository:        ticketRepository,
		eventRepository:         eventRepository,
		eventScheduleRepository: eventScheduleRepository,
		userRepository:          userRepository,
		venueRepository:         venueRepository,
		ticketTypeRepository:    ticketTypeRepository,
		logger:                  logger,
	}
}

// GetMyTickets returns tickets belonging to the authenticated user.
func (u *TicketUsecase) GetMyTickets(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Ticket, error) {
	if userID == uuid.Nil {
		u.logger.WarnContext(ctx, "ticket listing rejected: invalid user ID")
		return nil, domain.ErrTicketNotFound
	}

	tickets, err := u.ticketRepository.FindByUserID(ctx, userID)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to retrieve customer tickets",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "customer tickets retrieved",
		slog.String("user_id", userID.String()),
		slog.Int("ticket_count", len(tickets)),
	)

	return tickets, nil
}

// GetMyTicket returns a ticket only when it belongs to the user.
func (u *TicketUsecase) GetMyTicket(
	ctx context.Context,
	userID uuid.UUID,
	ticketID uuid.UUID,
) (*domain.Ticket, error) {
	if userID == uuid.Nil || ticketID == uuid.Nil {
		return nil, domain.ErrTicketNotFound
	}

	ticket, err := u.ticketRepository.FindByID(ctx, ticketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return nil, domain.ErrTicketNotFound
		}

		u.logger.ErrorContext(ctx, "failed to retrieve ticket",
			slog.String("ticket_id", ticketID.String()),
			slog.String("error", err.Error()),
		)

		return nil, err
	}

	// Do not reveal whether another customer's ticket exists.
	if ticket.UserID != userID {
		u.logger.WarnContext(ctx, "ticket access denied",
			slog.String("user_id", userID.String()),
			slog.String("ticket_id", ticketID.String()),
		)

		return nil, domain.ErrTicketNotFound
	}

	u.logger.InfoContext(ctx, "customer ticket retrieved",
		slog.String("ticket_id", ticketID.String()),
		slog.String("user_id", userID.String()),
	)

	return ticket, nil
}

// GetTicketPDFData verifies ticket ownership and retrieves all related event,
// schedule, customer, venue, and ticket-type details for PDF generation.
func (u *TicketUsecase) GetTicketPDFData(
	ctx context.Context,
	userID uuid.UUID,
	ticketID uuid.UUID,
) (*TicketPDFData, error) {
	if userID == uuid.Nil || ticketID == uuid.Nil {
		return nil, domain.ErrTicketNotFound
	}

	// 1. Retrieve ticket and verify ownership
	ticket, err := u.GetMyTicket(ctx, userID, ticketID)
	if err != nil {
		return nil, err
	}

	// 2. Retrieve event
	event, err := u.eventRepository.FindByID(ticket.EventID)
	if err != nil {
		if errors.Is(err, domain.ErrEventNotFound) {
			u.logger.WarnContext(ctx, "event not found for ticket",
				slog.String("ticket_id", ticket.ID.String()),
				slog.String("event_id", ticket.EventID.String()),
			)
			return nil, domain.ErrEventNotFound
		}
		u.logger.ErrorContext(ctx, "failed to retrieve event for ticket",
			slog.String("ticket_id", ticket.ID.String()),
			slog.String("event_id", ticket.EventID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	// 3. Retrieve customer / user details
	customerName := "Valued Customer"
	user, err := u.userRepository.FindByID(ticket.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			u.logger.WarnContext(ctx, "user not found for ticket",
				slog.String("ticket_id", ticket.ID.String()),
				slog.String("user_id", ticket.UserID.String()),
			)
		} else {
			u.logger.ErrorContext(ctx, "failed to retrieve user for ticket",
				slog.String("ticket_id", ticket.ID.String()),
				slog.String("user_id", ticket.UserID.String()),
				slog.Any("error", err),
			)
			return nil, err
		}
	} else if user != nil {
		if strings.TrimSpace(user.FullName) != "" {
			customerName = strings.TrimSpace(user.FullName)
		} else if strings.TrimSpace(user.Email) != "" {
			customerName = strings.TrimSpace(user.Email)
		}
	}

	// 4. Retrieve event schedule
	var eventDate time.Time
	schedule, err := u.eventScheduleRepository.FindByEventID(event.ID)
	if err != nil {
		if errors.Is(err, domain.ErrEventNotFound) {
			u.logger.InfoContext(ctx, "no schedule found for event",
				slog.String("event_id", event.ID.String()),
			)
		} else {
			u.logger.ErrorContext(ctx, "failed to retrieve event schedule",
				slog.String("event_id", event.ID.String()),
				slog.Any("error", err),
			)
			return nil, err
		}
	} else if schedule != nil && !schedule.EventDate.IsZero() {
		eventDate = schedule.EventDate
		if !schedule.StartTime.IsZero() {
			eventDate = time.Date(
				schedule.EventDate.Year(),
				schedule.EventDate.Month(),
				schedule.EventDate.Day(),
				schedule.StartTime.Hour(),
				schedule.StartTime.Minute(),
				schedule.StartTime.Second(),
				0,
				schedule.EventDate.Location(),
			)
		}
	}

	// 5. Retrieve venue details according to event rules
	var venueName string
	var venueAddress string

	if event.EventType != domain.EventTypeOnline && event.VenueID != nil && *event.VenueID != uuid.Nil {
		venue, err := u.venueRepository.FindByID(*event.VenueID)
		if err != nil {
			if errors.Is(err, domain.ErrVenueNotFound) {
				u.logger.WarnContext(ctx, "venue not found for event",
					slog.String("event_id", event.ID.String()),
					slog.String("venue_id", event.VenueID.String()),
				)
			} else {
				u.logger.ErrorContext(ctx, "failed to retrieve venue for event",
					slog.String("event_id", event.ID.String()),
					slog.String("venue_id", event.VenueID.String()),
					slog.Any("error", err),
				)
				return nil, err
			}
		} else if venue != nil {
			venueName = strings.TrimSpace(venue.Name)
			var addrParts []string
			if strings.TrimSpace(venue.Address) != "" {
				addrParts = append(addrParts, strings.TrimSpace(venue.Address))
			}
			if strings.TrimSpace(venue.City) != "" {
				addrParts = append(addrParts, strings.TrimSpace(venue.City))
			}
			if strings.TrimSpace(venue.State) != "" {
				addrParts = append(addrParts, strings.TrimSpace(venue.State))
			}
			if strings.TrimSpace(venue.PostalCode) != "" {
				addrParts = append(addrParts, strings.TrimSpace(venue.PostalCode))
			}
			venueAddress = strings.Join(addrParts, ", ")
		}
	}

	// 6. Retrieve ticket type
	ticketTypeName := "General Admission"
	if ticket.TicketTypeID != nil && *ticket.TicketTypeID != uuid.Nil {
		ticketType, err := u.ticketTypeRepository.FindByID(ctx, *ticket.TicketTypeID)
		if err != nil {
			if errors.Is(err, domain.ErrTicketTypeNotFound) {
				u.logger.WarnContext(ctx, "ticket type not found",
					slog.String("ticket_id", ticket.ID.String()),
					slog.String("ticket_type_id", ticket.TicketTypeID.String()),
				)
			} else {
				u.logger.ErrorContext(ctx, "failed to retrieve ticket type",
					slog.String("ticket_id", ticket.ID.String()),
					slog.String("ticket_type_id", ticket.TicketTypeID.String()),
					slog.Any("error", err),
				)
				return nil, err
			}
		} else if ticketType != nil && strings.TrimSpace(ticketType.Name) != "" {
			ticketTypeName = strings.TrimSpace(ticketType.Name)
		}
	}

	// 7. Format admission number
	admissionNo := "N/A"
	if ticket.AdmissionNumber > 0 {
		admissionNo = strconv.Itoa(ticket.AdmissionNumber)
	} else if strings.TrimSpace(ticket.TicketCode) != "" {
		admissionNo = strings.TrimSpace(ticket.TicketCode)
	}

	eventName := "EventHub Event"
	if strings.TrimSpace(event.Title) != "" {
		eventName = strings.TrimSpace(event.Title)
	}

	return &TicketPDFData{
		TicketID:     ticket.ID,
		EventName:    eventName,
		TicketType:   ticketTypeName,
		AdmissionNo:  admissionNo,
		EventDate:    eventDate,
		VenueName:    venueName,
		VenueAddress: venueAddress,
		CustomerName: customerName,
	}, nil
}
