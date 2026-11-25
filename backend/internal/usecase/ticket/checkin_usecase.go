package ticket

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrInvalidCheckInRequest   = errors.New("invalid check-in request")
	ErrOrganizerNotAuthorized  = errors.New("organizer is not authorized for this event")
	ErrBookingAlreadyCheckedIn = fmt.Errorf("%w: all tickets for this booking have already been checked in", domain.ErrTicketNotValid)
	ErrBookingNotEligible      = fmt.Errorf("%w: booking is not eligible for check-in", domain.ErrTicketNotValid)
	ErrBookingEventMismatch    = errors.New("booking does not belong to this event")
)

type CheckInType string

const (
	CheckInTypeIndividual CheckInType = "INDIVIDUAL"
	CheckInTypeBooking    CheckInType = "BOOKING"
)

type CheckInResult struct {
	Type             CheckInType     `json:"type"`
	Ticket           *domain.Ticket  `json:"ticket,omitempty"`
	Tickets          []domain.Ticket `json:"tickets,omitempty"`
	BookingID        *uuid.UUID      `json:"booking_id,omitempty"`
	BookingNumber    string          `json:"booking_number,omitempty"`
	TotalCount       int             `json:"total_count"`
	AlreadyUsedCount int             `json:"already_used_count,omitempty"`
	CheckedInCount   int             `json:"checked_in_count,omitempty"`
}

type CustomerSummary struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type TicketAdmissionDetail struct {
	TicketID        uuid.UUID  `json:"ticket_id"`
	TicketCode      string     `json:"ticket_code"`
	TicketTypeName  string     `json:"ticket_type_name"`
	AdmissionNumber int        `json:"admission_number"`
	SeatLabel       string     `json:"seat_label,omitempty"`
	Status          string     `json:"status"`
	UsedAt          *time.Time `json:"used_at,omitempty"`
}

type ManualBookingLookupResult struct {
	BookingID                   uuid.UUID               `json:"booking_id"`
	BookingReference            string                  `json:"booking_reference"`
	EventID                     uuid.UUID               `json:"event_id"`
	EventTitle                  string                  `json:"event_title"`
	Customer                    CustomerSummary         `json:"customer"`
	TotalAdmissions             int                     `json:"total_admissions"`
	AlreadyUsedAdmissions       int                     `json:"already_used_admissions"`
	RemainingEligibleAdmissions int                     `json:"remaining_eligible_admissions"`
	CancelledAdmissions         int                     `json:"cancelled_admissions"`
	IsEligible                  bool                    `json:"is_eligible"`
	IneligibleReason            string                  `json:"ineligible_reason"`
	Tickets                     []TicketAdmissionDetail `json:"tickets"`
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
		// 3d. Find booking and acquire lock for update.
		booking, err := tx.BookingRepository().FindByIDForUpdate(ctx, bookingID)
		if err != nil {
			return err
		}

		res, err := uc.checkInBookingTicketsTx(
			ctx,
			tx,
			booking,
			eventID,
			userID,
			"BOOKING_QR",
			usedAt,
		)
		if err != nil {
			return err
		}

		result = res
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

	if isBookingQR {
		uc.logger.InfoContext(ctx, "booking check-in completed",
			slog.String("check_in_method", "BOOKING_QR"),
			slog.String("user_id", userID.String()),
			slog.String("event_id", eventID.String()),
			slog.String("booking_id", result.BookingID.String()),
			slog.Int("total_admissions", result.TotalCount),
			slog.Int("already_used_admissions", result.AlreadyUsedCount),
			slog.Int("checked_in_admissions", result.CheckedInCount),
		)
	} else {
		uc.logger.InfoContext(ctx, "ticket check-in completed",
			slog.String("check_in_method", "INDIVIDUAL_QR"),
			slog.String("user_id", userID.String()),
			slog.String("event_id", eventID.String()),
			slog.String("ticket_id", result.Ticket.ID.String()),
		)
	}

	return result, nil
}

// checkInBookingTicketsTx executes the core atomic check-in of eligible tickets in a booking.
func (uc *CheckInUseCase) checkInBookingTicketsTx(
	ctx context.Context,
	tx domain.TransactionRepositories,
	booking *domain.Booking,
	eventID uuid.UUID,
	organizerUserID uuid.UUID,
	method string,
	usedAt time.Time,
) (*CheckInResult, error) {
	if booking.EventID != eventID {
		return nil, domain.ErrTicketNotValid
	}

	if booking.Status != domain.BookingStatusConfirmed && booking.Status != domain.BookingStatusPartiallyCancelled {
		return nil, ErrBookingNotEligible
	}

	tickets, err := tx.TicketRepository().FindByBookingID(ctx, booking.ID)
	if err != nil {
		return nil, err
	}

	if len(tickets) == 0 {
		return nil, domain.ErrTicketNotFound
	}

	var validTickets []domain.Ticket
	var alreadyUsedCount int
	var cancelledCount int

	for _, t := range tickets {
		if t.EventID != eventID {
			return nil, domain.ErrTicketNotValid
		}
		switch t.Status {
		case domain.TicketStatusValid:
			validTickets = append(validTickets, t)
		case domain.TicketStatusUsed:
			alreadyUsedCount++
		case domain.TicketStatusCancelled, domain.TicketStatusRefunded:
			cancelledCount++
		default:
			// Treat unexpected statuses as not valid
		}
	}

	if len(validTickets) == 0 {
		if alreadyUsedCount > 0 {
			return nil, ErrBookingAlreadyCheckedIn
		}
		return nil, ErrBookingNotEligible
	}

	checkedInTickets := make([]domain.Ticket, len(validTickets))
	for i, t := range validTickets {
		updated, err := tx.TicketRepository().MarkAsUsedIfValid(
			ctx,
			t.ID,
			eventID,
			usedAt,
		)
		if err != nil {
			return nil, err
		}
		if !updated {
			// A concurrent check-in/scan modified this ticket; rollback entire transaction
			return nil, domain.ErrTicketNotValid
		}

		scan := &domain.TicketScan{
			ID:        uuid.New(),
			TicketID:  t.ID,
			ScannedBy: organizerUserID,
			Status:    domain.TicketScanStatusSuccess,
			ScannedAt: usedAt,
		}
		if err := tx.TicketScanRepository().Create(ctx, scan); err != nil {
			return nil, err
		}

		t.Status = domain.TicketStatusUsed
		t.UsedAt = &usedAt
		checkedInTickets[i] = t
	}

	return &CheckInResult{
		Type:             CheckInTypeBooking,
		Ticket:           &checkedInTickets[0],
		Tickets:          checkedInTickets,
		BookingID:        &booking.ID,
		BookingNumber:    booking.BookingNumber,
		TotalCount:       len(tickets),
		AlreadyUsedCount: alreadyUsedCount,
		CheckedInCount:   len(checkedInTickets),
	}, nil
}

// LookupManualBooking retrieves sanitized booking and ticket information by booking reference.
func (uc *CheckInUseCase) LookupManualBooking(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
	bookingReference string,
) (*ManualBookingLookupResult, error) {
	normRef := normalizeBookingReference(bookingReference)
	if userID == uuid.Nil || eventID == uuid.Nil || normRef == "" {
		return nil, ErrInvalidCheckInRequest
	}

	var result *ManualBookingLookupResult

	err := uc.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		organizer, err := tx.OrganizerRepository().FindByUserID(userID)
		if err != nil {
			return err
		}

		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			return err
		}

		if event.OrganizerID != organizer.ID {
			return ErrOrganizerNotAuthorized
		}

		booking, err := tx.BookingRepository().FindByBookingNumber(ctx, normRef)
		if err != nil {
			return err
		}

		if booking.EventID != eventID {
			return domain.ErrBookingNotFound
		}

		tickets, err := tx.TicketRepository().FindByBookingID(ctx, booking.ID)
		if err != nil {
			return err
		}

		var customerName, customerEmail, customerPhone string
		if user, uErr := tx.UserRepository().FindByID(booking.UserID); uErr == nil && user != nil {
			customerName = user.FullName
			customerEmail = user.Email
			customerPhone = user.Phone
		}

		var seatIDs []uuid.UUID
		for _, t := range tickets {
			if t.SeatID != nil && *t.SeatID != uuid.Nil {
				seatIDs = append(seatIDs, *t.SeatID)
			}
		}
		var seatDetailsMap map[uuid.UUID]domain.SeatDetail
		if len(seatIDs) > 0 {
			seatDetailsMap, _ = tx.SeatRepository().FindDetailsBySeatIDs(ctx, seatIDs)
		}

		ticketTypeCache := make(map[uuid.UUID]string)
		var alreadyUsedCount, remainingEligibleCount, cancelledCount int
		admissionDetails := make([]TicketAdmissionDetail, len(tickets))

		for i, t := range tickets {
			typeName := "Admission"
			if t.TicketTypeID != nil && *t.TicketTypeID != uuid.Nil {
				if cached, ok := ticketTypeCache[*t.TicketTypeID]; ok {
					typeName = cached
				} else {
					if tt, ttErr := tx.TicketTypeRepository().FindByID(ctx, *t.TicketTypeID); ttErr == nil && tt != nil && strings.TrimSpace(tt.Name) != "" {
						typeName = strings.TrimSpace(tt.Name)
						ticketTypeCache[*t.TicketTypeID] = typeName
					}
				}
			}

			var seatLabel string
			if t.SeatID != nil && seatDetailsMap != nil {
				if sd, ok := seatDetailsMap[*t.SeatID]; ok && strings.TrimSpace(sd.Label) != "" {
					seatLabel = strings.TrimSpace(sd.Label)
				}
			}

			switch t.Status {
			case domain.TicketStatusValid:
				remainingEligibleCount++
			case domain.TicketStatusUsed:
				alreadyUsedCount++
			case domain.TicketStatusCancelled, domain.TicketStatusRefunded:
				cancelledCount++
			}

			admissionDetails[i] = TicketAdmissionDetail{
				TicketID:        t.ID,
				TicketCode:      t.TicketCode,
				TicketTypeName:  typeName,
				AdmissionNumber: t.AdmissionNumber,
				SeatLabel:       seatLabel,
				Status:          t.Status,
				UsedAt:          t.UsedAt,
			}
		}

		isBookingStatusEligible := (booking.Status == domain.BookingStatusConfirmed || booking.Status == domain.BookingStatusPartiallyCancelled)
		isEligible := isBookingStatusEligible && remainingEligibleCount > 0
		var ineligibleReason string

		if !isBookingStatusEligible {
			ineligibleReason = "Booking is cancelled or not confirmed"
		} else if remainingEligibleCount == 0 {
			if alreadyUsedCount > 0 {
				ineligibleReason = "All tickets for this booking have already been checked in"
			} else {
				ineligibleReason = "No valid tickets remain for this booking"
			}
		}

		result = &ManualBookingLookupResult{
			BookingID:        booking.ID,
			BookingReference: booking.BookingNumber,
			EventID:          event.ID,
			EventTitle:       event.Title,
			Customer: CustomerSummary{
				FullName: customerName,
				Email:    maskEmail(customerEmail),
				Phone:    maskPhone(customerPhone),
			},
			TotalAdmissions:             len(tickets),
			AlreadyUsedAdmissions:       alreadyUsedCount,
			RemainingEligibleAdmissions: remainingEligibleCount,
			CancelledAdmissions:         cancelledCount,
			IsEligible:                  isEligible,
			IneligibleReason:            ineligibleReason,
			Tickets:                     admissionDetails,
		}

		return nil
	})

	if err != nil {
		uc.logger.WarnContext(ctx, "manual booking lookup failed",
			slog.String("user_id", userID.String()),
			slog.String("event_id", eventID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return result, nil
}

// ConfirmManualBooking atomically checks in remaining valid tickets for a booking reference.
func (uc *CheckInUseCase) ConfirmManualBooking(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
	bookingReference string,
) (*CheckInResult, error) {
	normRef := normalizeBookingReference(bookingReference)
	if userID == uuid.Nil || eventID == uuid.Nil || normRef == "" {
		return nil, ErrInvalidCheckInRequest
	}

	var result *CheckInResult

	err := uc.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		organizer, err := tx.OrganizerRepository().FindByUserID(userID)
		if err != nil {
			return err
		}

		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			return err
		}

		if event.OrganizerID != organizer.ID {
			return ErrOrganizerNotAuthorized
		}

		booking, err := tx.BookingRepository().FindByBookingNumber(ctx, normRef)
		if err != nil {
			return err
		}

		if booking.EventID != eventID {
			return domain.ErrBookingNotFound
		}

		booking, err = tx.BookingRepository().FindByIDForUpdate(ctx, booking.ID)
		if err != nil {
			return err
		}

		usedAt := time.Now().UTC()

		res, err := uc.checkInBookingTicketsTx(
			ctx,
			tx,
			booking,
			eventID,
			userID,
			"MANUAL_REFERENCE",
			usedAt,
		)
		if err != nil {
			return err
		}

		result = res
		return nil
	})

	if err != nil {
		uc.logger.WarnContext(ctx, "manual booking check-in failed",
			slog.String("user_id", userID.String()),
			slog.String("event_id", eventID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	uc.logger.InfoContext(ctx, "booking check-in completed",
		slog.String("check_in_method", "MANUAL_REFERENCE"),
		slog.String("user_id", userID.String()),
		slog.String("event_id", eventID.String()),
		slog.String("booking_id", result.BookingID.String()),
		slog.Int("total_admissions", result.TotalCount),
		slog.Int("already_used_admissions", result.AlreadyUsedCount),
		slog.Int("checked_in_admissions", result.CheckedInCount),
	)

	return result, nil
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	name, domainPart := parts[0], parts[1]
	if len(name) <= 1 {
		return "*@" + domainPart
	}
	if len(name) == 2 {
		return string(name[0]) + "*@" + domainPart
	}
	return string(name[0]) + "***" + string(name[len(name)-1]) + "@" + domainPart
}

func maskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	if len(phone) <= 4 {
		return "****"
	}
	last4 := phone[len(phone)-4:]
	return strings.Repeat("*", 6) + last4
}

func normalizeBookingReference(ref string) string {
	trimmed := strings.TrimSpace(ref)
	if len(trimmed) >= 4 && strings.EqualFold(trimmed[:4], "EVH-") {
		return "EVH-" + strings.ToLower(trimmed[4:])
	}
	return strings.ToLower(trimmed)
}
