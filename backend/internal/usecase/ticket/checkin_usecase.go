package ticket

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrInvalidCheckInRequest  = errors.New("invalid check-in request")
	ErrOrganizerNotAuthorized = errors.New("organizer is not authorized for this event")
)

type CheckInUseCase struct {
	txManager domain.TransactionManager
	qrService *QRService
	logger    *slog.Logger
}

func NewCheckInUseCase(
	txManager domain.TransactionManager,
	qrService *QRService,
	logger *slog.Logger,
) *CheckInUseCase {
	return &CheckInUseCase{
		txManager: txManager,
		qrService: qrService,
		logger:    logger,
	}
}

func (uc *CheckInUseCase) CheckIn(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
	qrPayload string,
) (*domain.Ticket, error) {

	// 1. Validate request inputs.
	if userID == uuid.Nil ||
		eventID == uuid.Nil ||
		strings.TrimSpace(qrPayload) == "" {
		return nil, ErrInvalidCheckInRequest
	}

	// 2. Verify the signed QR payload.
	ticketID, err := uc.qrService.VerifyPayload(qrPayload)
	if err != nil {
		uc.logger.WarnContext(ctx, "ticket check-in rejected: invalid QR",
			slog.String("user_id", userID.String()),
			slog.String("event_id", eventID.String()),
		)
		return nil, err
	}

	var checkedInTicket *domain.Ticket

	// 3. Perform all database operations in one transaction.
	err = uc.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {

		// Resolve the organizer using the authenticated user.
		organizer, err := tx.OrganizerRepository().FindByUserID(userID)
		if err != nil {
			return err
		}

		// Find the event.
		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			return err
		}

		// Verify event ownership.
		if event.OrganizerID != organizer.ID {
			return ErrOrganizerNotAuthorized
		}

		// Find the ticket.
		ticket, err := tx.TicketRepository().FindByID(ctx, ticketID)
		if err != nil {
			return err
		}

		// Ensure the ticket belongs to the requested event.
		if ticket.EventID != eventID {
			return domain.ErrTicketNotValid
		}

		usedAt := time.Now().UTC()

		// Atomically mark the ticket as used.
		updated, err := tx.TicketRepository().MarkAsUsedIfValid(
			ctx,
			ticketID,
			eventID,
			usedAt,
		)
		if err != nil {
			return err
		}

		if !updated {
			return domain.ErrTicketNotValid
		}

		// Record the successful scan in the same transaction.
		scan := &domain.TicketScan{
			ID:        uuid.New(),
			TicketID:  ticketID,
			ScannedBy: userID,
			Status:    domain.TicketScanStatusSuccess,
			ScannedAt: usedAt,
		}

		if err := tx.TicketScanRepository().Create(ctx, scan); err != nil {
			return err
		}

		// Prepare the response.
		ticket.Status = domain.TicketStatusUsed
		ticket.UsedAt = &usedAt
		checkedInTicket = ticket

		return nil
	})

	if err != nil {
		uc.logger.ErrorContext(ctx, "ticket check-in failed",
			slog.String("user_id", userID.String()),
			slog.String("event_id", eventID.String()),
			slog.String("ticket_id", ticketID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	uc.logger.InfoContext(ctx, "ticket check-in completed",
		slog.String("user_id", userID.String()),
		slog.String("event_id", eventID.String()),
		slog.String("ticket_id", ticketID.String()),
	)

	return checkedInTicket, nil
}
