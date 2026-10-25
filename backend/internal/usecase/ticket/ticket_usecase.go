package ticket

import (
	"context"
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type TicketUsecase struct {
	ticketRepository domain.TicketRepository
	logger           *slog.Logger
}

func NewTicketUsecase(
	ticketRepository domain.TicketRepository,
	logger *slog.Logger,
) *TicketUsecase {
	return &TicketUsecase{
		ticketRepository: ticketRepository,
		logger:           logger,
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
