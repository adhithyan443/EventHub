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

type CheckInType string

const (
	CheckInTypeIndividual CheckInType = "INDIVIDUAL"
	CheckInTypeBooking    CheckInType = "BOOKING"
)

type CheckInResult struct {
	Type       CheckInType     `json:"type"`
	Ticket     *domain.Ticket  `json:"ticket,omitempty"`
	Tickets    []domain.Ticket `json:"tickets,omitempty"`
	BookingID  *uuid.UUID      `json:"booking_id,omitempty"`
	TotalCount int             `json:"total_count"`
}

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
) (*CheckInResult, error) {

	// 1. Validate request inputs.
	trimmedPayload := strings.TrimSpace(qrPayload)
	if userID == uuid.Nil || eventID == uuid.Nil || trimmedPayload == "" {
		return nil, ErrInvalidCheckInRequest
	}

	// 2. Identify QR type and verify HMAC signature.
	isBookingQR := strings.HasPrefix(trimmedPayload, bookingQRPrefix)

	var (
		ticketID  uuid.UUID
		bookingID uuid.UUID
		err       error
	)

	if isBookingQR {
		bookingID, err = uc.qrService.VerifyBookingPayload(trimmedPayload)
	} else {
		ticketID, err = uc.qrService.VerifyPayload(trimmedPayload)
	}

	if err != nil {
		uc.logger.WarnContext(ctx, "ticket check-in rejected: invalid QR",
			slog.String("user_id", userID.String()),
			slog.String("event_id", eventID.String()),
			slog.Bool("is_booking_qr", isBookingQR),
		)
		return nil, err
	}

	var result *CheckInResult

	// 3. Perform all validation and updates within a single database transaction.
	err = uc.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {

		// 3a. Resolve organizer for authenticated user.
		organizer, err := tx.OrganizerRepository().FindByUserID(userID)
		if err != nil {
			return err
		}

		// 3b. Verify event existence.
		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			return err
		}

		// 3c. Verify event ownership and organizer authorization.
		if event.OrganizerID != organizer.ID {
			return ErrOrganizerNotAuthorized
		}

		usedAt := time.Now().UTC()

		if !isBookingQR {
			// Case 1: Individual ticket check-in.
			ticket, err := tx.TicketRepository().FindByID(ctx, ticketID)
			if err != nil {
				return err
			}

			// Verify ticket belongs to requested event.
			if ticket.EventID != eventID {
				return domain.ErrTicketNotValid
			}

			// Reject already-used, cancelled, or refunded tickets.
			if ticket.Status != domain.TicketStatusValid {
				return domain.ErrTicketNotValid
			}

			// Atomically transition ticket status from VALID to USED.
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

			// Record individual ticket scan in the same transaction.
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

			ticket.Status = domain.TicketStatusUsed
			ticket.UsedAt = &usedAt

			result = &CheckInResult{
				Type:       CheckInTypeIndividual,
				Ticket:     ticket,
				Tickets:    []domain.Ticket{*ticket},
				BookingID:  &ticket.BookingID,
				TotalCount: 1,
			}
			return nil
		}

		// Case 2: Booking-level QR check-in.
		// 3d. Find booking and verify event association.
		booking, err := tx.BookingRepository().FindByID(ctx, bookingID)
		if err != nil {
			return err
		}

		if booking.EventID != eventID {
			return domain.ErrTicketNotValid
		}

		if booking.Status != domain.BookingStatusConfirmed {
			return domain.ErrTicketNotValid
		}

		// 3e. Retrieve all tickets belonging to this booking.
		tickets, err := tx.TicketRepository().FindByBookingID(ctx, bookingID)
		if err != nil {
			return err
		}

		if len(tickets) == 0 {
			return domain.ErrTicketNotFound
		}

		// 3f. Validate all tickets belonging to the booking upfront.
		// Prevent partial check-ins: all tickets must be VALID and belong to this event.
		for _, t := range tickets {
			if t.EventID != eventID || t.Status != domain.TicketStatusValid {
				return domain.ErrTicketNotValid
			}
		}

		// 3g. Atomically update all ticket statuses and insert TicketScan records.
		checkedInTickets := make([]domain.Ticket, len(tickets))
		for i, t := range tickets {
			updated, err := tx.TicketRepository().MarkAsUsedIfValid(
				ctx,
				t.ID,
				eventID,
				usedAt,
			)
			if err != nil {
				return err
			}
			if !updated {
				return domain.ErrTicketNotValid
			}

			scan := &domain.TicketScan{
				ID:        uuid.New(),
				TicketID:  t.ID,
				ScannedBy: userID,
				Status:    domain.TicketScanStatusSuccess,
				ScannedAt: usedAt,
			}
			if err := tx.TicketScanRepository().Create(ctx, scan); err != nil {
				return err
			}

			t.Status = domain.TicketStatusUsed
			t.UsedAt = &usedAt
			checkedInTickets[i] = t
		}

		result = &CheckInResult{
			Type:       CheckInTypeBooking,
			Ticket:     &checkedInTickets[0],
			Tickets:    checkedInTickets,
			BookingID:  &bookingID,
			TotalCount: len(checkedInTickets),
		}

		return nil
	})

	if err != nil {
		uc.logger.ErrorContext(ctx, "ticket check-in failed",
			slog.String("user_id", userID.String()),
			slog.String("event_id", eventID.String()),
			slog.Bool("is_booking_qr", isBookingQR),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	uc.logger.InfoContext(ctx, "ticket check-in completed",
		slog.String("user_id", userID.String()),
		slog.String("event_id", eventID.String()),
		slog.String("type", string(result.Type)),
		slog.Int("total_count", result.TotalCount),
	)

	return result, nil
}
