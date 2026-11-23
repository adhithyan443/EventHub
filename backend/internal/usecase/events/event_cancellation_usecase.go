package events

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/refund"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type EventCancellationUsecase struct {
	transactionManager domain.TransactionManager
	refundProcessor    refund.RefundProcessor
	notifier           domain.EventCancellationNotifier
	dispatchTracker    domain.DispatchTracker
	logger             *slog.Logger
}

func NewEventCancellationUsecase(
	transactionManager domain.TransactionManager,
	refundProcessor refund.RefundProcessor,
	logger *slog.Logger,
) *EventCancellationUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &EventCancellationUsecase{
		transactionManager: transactionManager,
		refundProcessor:    refundProcessor,
		logger:             logger,
	}
}

func (u *EventCancellationUsecase) SetRefundProcessor(processor refund.RefundProcessor) {
	u.refundProcessor = processor
}

func (u *EventCancellationUsecase) SetNotifier(notifier domain.EventCancellationNotifier) {
	u.notifier = notifier
}

func (u *EventCancellationUsecase) SetDispatchTracker(tracker domain.DispatchTracker) {
	u.dispatchTracker = tracker
}

// CancelEventByOrganizer allows the organizer who owns an event to cancel it and generate bulk refunds.
func (u *EventCancellationUsecase) CancelEventByOrganizer(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
	reason string,
) (*domain.EventCancellationSummary, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}
	if eventID == uuid.Nil {
		return nil, errors.New("event ID cannot be nil")
	}

	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return nil, domain.ErrCancellationReasonRequired
	}
	if len(trimmedReason) > 1000 {
		return nil, domain.ErrCancellationReasonTooLong
	}

	u.logger.InfoContext(ctx, "organizer_event_cancellation_initiated",
		slog.String("actor_id", userID.String()),
		slog.String("actor_role", "organizer"),
		slog.String("event_id", eventID.String()),
		slog.String("reason", trimmedReason),
	)

	return u.cancelEventAndGenerateRefunds(ctx, eventID, trimmedReason, true, userID)
}

// CancelEventByAdmin allows platform administrators to cancel an event and generate bulk refunds.
func (u *EventCancellationUsecase) CancelEventByAdmin(
	ctx context.Context,
	adminID uuid.UUID,
	eventID uuid.UUID,
	reason string,
) (*domain.EventCancellationSummary, error) {
	if adminID == uuid.Nil {
		return nil, errors.New("admin ID cannot be nil")
	}
	if eventID == uuid.Nil {
		return nil, errors.New("event ID cannot be nil")
	}

	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return nil, domain.ErrCancellationReasonRequired
	}
	if len(trimmedReason) > 1000 {
		return nil, domain.ErrCancellationReasonTooLong
	}

	u.logger.InfoContext(ctx, "admin_event_cancellation_initiated",
		slog.String("actor_id", adminID.String()),
		slog.String("actor_role", "admin"),
		slog.String("event_id", eventID.String()),
		slog.String("reason", trimmedReason),
	)

	return u.cancelEventAndGenerateRefunds(ctx, eventID, trimmedReason, false, adminID)
}

// GetEventCancellationStatus retrieves real-time event cancellation and refund progress.
func (u *EventCancellationUsecase) GetEventCancellationStatus(
	ctx context.Context,
	eventID uuid.UUID,
) (*domain.EventCancellationSummary, error) {
	if eventID == uuid.Nil {
		return nil, errors.New("event ID cannot be nil")
	}

	var summary *domain.EventCancellationSummary
	err := u.transactionManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			return err
		}

		s, err := u.buildSummaryInTx(ctx, tx, eventID, event.Title, event.Status)
		if err != nil {
			return err
		}
		summary = s
		return nil
	})
	if err != nil {
		return nil, err
	}

	return summary, nil
}

// GetEventCancellationStatusForOrganizer retrieves cancellation progress ensuring organizer ownership.
func (u *EventCancellationUsecase) GetEventCancellationStatusForOrganizer(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
) (*domain.EventCancellationSummary, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}
	if eventID == uuid.Nil {
		return nil, errors.New("event ID cannot be nil")
	}

	var summary *domain.EventCancellationSummary
	err := u.transactionManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		organizer, err := tx.OrganizerRepository().FindByUserID(userID)
		if err != nil {
			if errors.Is(err, domain.ErrOrganizerNotFound) {
				return ErrUnauthorizedOrganizer
			}
			return err
		}
		if organizer.Status != "ACTIVE" {
			return ErrUnauthorizedOrganizer
		}

		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			return err
		}
		if event.OrganizerID != organizer.ID {
			return ErrUnauthorizedOrganizer
		}

		s, err := u.buildSummaryInTx(ctx, tx, eventID, event.Title, event.Status)
		if err != nil {
			return err
		}
		summary = s
		return nil
	})
	if err != nil {
		return nil, err
	}

	return summary, nil
}

// GetCancellationPreview retrieves a comprehensive read-only cancellation impact preview for administrators.
func (u *EventCancellationUsecase) GetCancellationPreview(
	ctx context.Context,
	eventID uuid.UUID,
) (*domain.EventCancellationPreview, error) {
	if eventID == uuid.Nil {
		return nil, errors.New("event ID cannot be nil")
	}

	var preview *domain.EventCancellationPreview
	err := u.transactionManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			return err
		}

		p, err := u.calculateCancellationPreviewInTx(ctx, tx, event, time.Now().UTC())
		if err != nil {
			return err
		}
		preview = p
		return nil
	})
	if err != nil {
		return nil, err
	}

	u.logger.InfoContext(ctx, "cancellation_preview_generated_admin",
		slog.String("event_id", eventID.String()),
		slog.Bool("is_cancellable", preview.Eligibility.IsCancellable),
	)
	return preview, nil
}

// GetCancellationPreviewForOrganizer retrieves a comprehensive read-only cancellation impact preview for the event owner.
func (u *EventCancellationUsecase) GetCancellationPreviewForOrganizer(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
) (*domain.EventCancellationPreview, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}
	if eventID == uuid.Nil {
		return nil, errors.New("event ID cannot be nil")
	}

	var preview *domain.EventCancellationPreview
	err := u.transactionManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		organizer, err := tx.OrganizerRepository().FindByUserID(userID)
		if err != nil {
			if errors.Is(err, domain.ErrOrganizerNotFound) {
				return ErrUnauthorizedOrganizer
			}
			return err
		}
		if organizer.Status != "ACTIVE" {
			return ErrUnauthorizedOrganizer
		}

		event, err := tx.EventRepository().FindByID(eventID)
		if err != nil {
			return err
		}
		if event.OrganizerID != organizer.ID {
			return ErrUnauthorizedOrganizer
		}

		p, err := u.calculateCancellationPreviewInTx(ctx, tx, event, time.Now().UTC())
		if err != nil {
			return err
		}
		preview = p
		return nil
	})
	if err != nil {
		return nil, err
	}

	u.logger.InfoContext(ctx, "cancellation_preview_generated_organizer",
		slog.String("user_id", userID.String()),
		slog.String("event_id", eventID.String()),
		slog.Bool("is_cancellable", preview.Eligibility.IsCancellable),
	)
	return preview, nil
}

func (u *EventCancellationUsecase) cancelEventAndGenerateRefunds(
	ctx context.Context,
	eventID uuid.UUID,
	trimmedReason string,
	isOrganizer bool,
	actorID uuid.UUID,
) (*domain.EventCancellationSummary, error) {
	var createdRefundIDs []uuid.UUID
	var summary *domain.EventCancellationSummary
	var customerNotices []domain.CustomerCancellationNotice
	var organizerNotice *domain.OrganizerCancellationNotice

	actorRole := "admin"
	if isOrganizer {
		actorRole = "organizer"
	}

	err := u.transactionManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		now := time.Now().UTC()

		var event *domain.Event
		if isOrganizer {
			organizer, err := tx.OrganizerRepository().FindByUserID(actorID)
			if err != nil {
				if errors.Is(err, domain.ErrOrganizerNotFound) {
					u.logger.WarnContext(ctx, "event_cancel_organizer_not_found",
						slog.String("actor_id", actorID.String()),
						slog.String("actor_role", actorRole),
						slog.String("event_id", eventID.String()),
					)
					return ErrUnauthorizedOrganizer
				}
				u.logger.ErrorContext(ctx, "event_cancel_organizer_lookup_failed",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
					slog.Any("error", err),
				)
				return err
			}

			if organizer.Status != "ACTIVE" {
				u.logger.WarnContext(ctx, "event_cancel_organizer_inactive",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("organizer_id", organizer.ID.String()),
					slog.String("status", organizer.Status),
				)
				return ErrUnauthorizedOrganizer
			}

			lockedEvent, err := tx.EventRepository().FindByIDForUpdate(eventID)
			if err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					return domain.ErrEventNotFound
				}
				return err
			}

			if lockedEvent.OrganizerID != organizer.ID {
				u.logger.WarnContext(ctx, "event_cancel_forbidden",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
					slog.String("organizer_id", organizer.ID.String()),
					slog.String("event_organizer_id", lockedEvent.OrganizerID.String()),
				)
				return ErrUnauthorizedOrganizer
			}
			event = lockedEvent
		} else {
			lockedEvent, err := tx.EventRepository().FindByIDForUpdate(eventID)
			if err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					return domain.ErrEventNotFound
				}
				return err
			}
			event = lockedEvent
		}

		var schedule *domain.EventSchedule
		var eventDateStr string = "TBD"
		if tx.EventScheduleRepository() != nil {
			s, sErr := tx.EventScheduleRepository().FindByEventID(eventID)
			if sErr != nil && !errors.Is(sErr, domain.ErrEventNotFound) {
				u.logger.ErrorContext(ctx, "event_cancel_schedule_lookup_failed",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
					slog.Any("error", sErr),
				)
				return fmt.Errorf("failed to lookup event schedule: %w", sErr)
			}
			if s != nil {
				schedule = s
				eventDateStr = s.EventDate.Format("02 Jan 2006")
			}
		}

		var settlements []*domain.Settlement
		if tx.SettlementRepository() != nil {
			settlementResult, sErr := tx.SettlementRepository().List(ctx, domain.SettlementFilter{
				EventID: &eventID,
				Limit:   100,
			})
			if sErr != nil {
				u.logger.ErrorContext(ctx, "event_cancel_settlement_lookup_failed",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
					slog.Any("error", sErr),
				)
				return fmt.Errorf("failed to verify settlement records: %w", sErr)
			}
			if settlementResult != nil {
				settlements = settlementResult.Settlements
			}
		}

		assessment := evaluateCancellationEligibility(event, schedule, settlements, now)
		if !assessment.IsCancellable {
			switch {
			case errors.Is(assessment.FirstBlockingError, domain.ErrEventAlreadyCancelled):
				u.logger.WarnContext(ctx, "event_cancel_already_cancelled",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
				)
			case errors.Is(assessment.FirstBlockingError, domain.ErrDraftEventCannotBeCancelled):
				u.logger.WarnContext(ctx, "event_cancel_draft_rejected",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
				)
			case errors.Is(assessment.FirstBlockingError, domain.ErrPastEventCannotBeCancelled):
				var evDate time.Time
				if schedule != nil {
					evDate = schedule.EventDate
				}
				u.logger.WarnContext(ctx, "event_cancel_concluded_event_rejected",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
					slog.Time("event_date", evDate),
				)
			case errors.Is(assessment.FirstBlockingError, domain.ErrEventCannotBeCancelledWithCompletedSettlement):
				u.logger.WarnContext(ctx, "event_cancel_blocked_settlement_completed",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
				)
			case errors.Is(assessment.FirstBlockingError, domain.ErrEventCannotBeCancelledWithActiveSettlement):
				u.logger.WarnContext(ctx, "event_cancel_blocked_settlement_in_flight",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
				)
			default:
				u.logger.WarnContext(ctx, "event_cancel_invalid_status",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
					slog.String("status", event.Status),
				)
			}
			return assessment.FirstBlockingError
		}

		if tx.SettlementRepository() != nil {
			for _, s := range settlements {
				if s.Status == domain.SettlementStatusPending {
					s.Status = domain.SettlementStatusRejected
					s.RejectedAt = &now
					rejectionMsg := fmt.Sprintf("Automatically rejected due to event cancellation: %s", trimmedReason)
					s.RejectionReason = &rejectionMsg
					s.UpdatedAt = now
					if err := tx.SettlementRepository().Update(ctx, s); err != nil {
						u.logger.ErrorContext(ctx, "event_cancel_reject_settlement_failed",
							slog.String("actor_id", actorID.String()),
							slog.String("actor_role", actorRole),
							slog.String("event_id", eventID.String()),
							slog.String("settlement_id", s.ID.String()),
							slog.Any("error", err),
						)
						return fmt.Errorf("failed to reject pending settlement %s: %w", s.ID, err)
					}
					u.logger.InfoContext(ctx, "event_cancel_rejected_pending_settlement",
						slog.String("actor_id", actorID.String()),
						slog.String("actor_role", actorRole),
						slog.String("event_id", eventID.String()),
						slog.String("settlement_id", s.ID.String()),
					)
				}
			}
		}

		// 4. Update Event state
		event.Status = domain.EventStatusCancelled
		event.UpdatedAt = now
		if err := tx.EventRepository().Update(event); err != nil {
			u.logger.ErrorContext(ctx, "event_cancel_update_failed", "event_id", eventID, "error", err)
			return err
		}

		cancellation, cErr := tx.EventCancellationRepository().FindByEventID(eventID)
		if cErr == nil && cancellation != nil {
			cancellation.CancelledAt = &now
			cancellation.CancellationReason = &trimmedReason
			cancellation.UpdatedAt = now
			if err := tx.EventCancellationRepository().Update(cancellation); err != nil {
				u.logger.ErrorContext(ctx, "event_cancel_cancellation_record_update_failed", "event_id", eventID, "error", err)
				return err
			}
		} else {
			newCancellation := &domain.EventCancellation{
				ID:                 uuid.New(),
				EventID:            eventID,
				CancelledAt:        &now,
				CancellationReason: &trimmedReason,
				RefundPolicy:       domain.RefundPolicyFull,
				CreatedAt:          now,
				UpdatedAt:          now,
			}
			_ = tx.EventCancellationRepository().Create(newCancellation)
		}

		// 5. Invalidate bookings, cancel tickets, and generate refunds.
		// Lock order strictly follows Event -> Booking -> SeatReservation to prevent AB-BA deadlocks with payment operations.
		bookings, err := tx.BookingRepository().FindByEventID(ctx, eventID)
		if err != nil {
			return fmt.Errorf("failed to retrieve bookings for cancelled event: %w", err)
		}

		// Retrieve existing refunds for this event to guarantee idempotency and avoid duplicates
		existingEventRefunds, err := tx.RefundRepository().FindByEventID(ctx, eventID)
		if err != nil {
			return fmt.Errorf("failed to retrieve existing refunds for cancelled event: %w", err)
		}

		existingRefundsByBooking := make(map[uuid.UUID][]domain.Refund, len(bookings))
		for _, r := range existingEventRefunds {
			existingRefundsByBooking[r.BookingID] = append(existingRefundsByBooking[r.BookingID], r)
		}

		type customerAggregate struct {
			customerID       uuid.UUID
			bookingRefs      []string
			ticketsCancelled int
			totalRefund      decimal.Decimal
		}
		customerAggregates := make(map[uuid.UUID]*customerAggregate)

		// Process each booking
		for _, booking := range bookings {
			// Find all tickets for this booking
			tickets, tErr := tx.TicketRepository().FindByBookingID(ctx, booking.ID)
			if tErr != nil {
				return fmt.Errorf("failed to find tickets for booking %s: %w", booking.ID, tErr)
			}

			var validTickets []domain.Ticket
			var validTicketIDs []uuid.UUID
			for _, t := range tickets {
				if t.Status == domain.TicketStatusValid {
					validTickets = append(validTickets, t)
					validTicketIDs = append(validTicketIDs, t.ID)
				}
			}

			// Invalidate remaining valid tickets atomically
			if len(validTicketIDs) > 0 {
				if err := tx.TicketRepository().UpdateStatusForTicketIDs(ctx, validTicketIDs, domain.TicketStatusCancelled); err != nil {
					return fmt.Errorf("failed to invalidate tickets for booking %s: %w", booking.ID, err)
				}
			}

			// Update booking status to CANCELLED if not already cancelled
			if booking.Status != domain.BookingStatusCancelled {
				if err := tx.BookingRepository().UpdateStatus(ctx, booking.ID, domain.BookingStatusCancelled); err != nil {
					return fmt.Errorf("failed to update booking status for %s: %w", booking.ID, err)
				}
			}

			// Track customer aggregate for consolidated notification
			cAgg, exists := customerAggregates[booking.UserID]
			if !exists {
				cAgg = &customerAggregate{
					customerID: booking.UserID,
				}
				customerAggregates[booking.UserID] = cAgg
			}
			cAgg.bookingRefs = append(cAgg.bookingRefs, booking.BookingNumber)
			cAgg.ticketsCancelled += len(validTickets)

			// Check if an EVENT_CANCELLATION refund was already created for this booking
			hasEventRefund := false
			for _, r := range existingRefundsByBooking[booking.ID] {
				if r.Reason == "EVENT_CANCELLATION" {
					hasEventRefund = true
					break
				}
			}
			if hasEventRefund {
				for _, r := range existingRefundsByBooking[booking.ID] {
					if r.Reason == "EVENT_CANCELLATION" {
						cAgg.totalRefund = cAgg.totalRefund.Add(r.Amount)
					}
				}
				continue
			}

			// If no valid tickets were cancelled (e.g. customer previously cancelled all of them), skip refund creation
			if len(validTickets) == 0 {
				continue
			}

			// If booking was free, skip refund creation
			if booking.Total.IsZero() || !booking.Total.IsPositive() {
				continue
			}

			// Locate successful payment
			payments, pErr := tx.PaymentRepository().FindByBookingID(ctx, booking.ID.String())
			if pErr != nil {
				return fmt.Errorf("failed to find payments for booking %s: %w", booking.ID, pErr)
			}

			var successfulPayment *domain.Payment
			for i := range payments {
				if payments[i].Status == domain.PaymentStatusSuccess {
					successfulPayment = &payments[i]
					break
				}
			}

			if successfulPayment == nil || successfulPayment.TransactionID == nil || strings.TrimSpace(*successfulPayment.TransactionID) == "" {
				u.logger.WarnContext(ctx, "event_cancel_booking_missing_successful_payment",
					slog.String("booking_id", booking.ID.String()),
					slog.String("event_id", eventID.String()),
				)
				continue
			}

			paymentUUID, err := uuid.Parse(successfulPayment.ID)
			if err != nil {
				return fmt.Errorf("invalid payment UUID format %s: %w", successfulPayment.ID, err)
			}

			// Calculate refund amount:
			// Event cancellation refunds MUST include ticket price and applicable platform fee and tax.
			// Calculate total already refunded for this booking across previous customer cancellations.
			totalAlreadyRefunded := decimal.Zero
			for _, r := range existingRefundsByBooking[booking.ID] {
				if r.Status == domain.RefundStatusSuccess || r.Status == domain.RefundStatusProcessing || r.Status == domain.RefundStatusPending {
					totalAlreadyRefunded = totalAlreadyRefunded.Add(r.Amount)
				}
			}

			maxRefundable := successfulPayment.Amount.Sub(totalAlreadyRefunded)
			if maxRefundable.IsNegative() {
				maxRefundable = decimal.Zero
			}

			var refundAmount decimal.Decimal
			if len(validTickets) == len(tickets) {
				// No prior customer cancellations: Full booking refund (ticket price + platform fee + tax - discount)
				refundAmount = booking.Total
			} else {
				// Prior customer cancellations existed: Refund remaining balance of booking total
				refundAmount = booking.Total.Sub(totalAlreadyRefunded)
			}

			if refundAmount.GreaterThan(maxRefundable) {
				refundAmount = maxRefundable
			}

			if refundAmount.IsZero() || !refundAmount.IsPositive() {
				continue
			}

			// Persist PENDING refund record
			pendingRefund := &domain.Refund{
				ID:        uuid.New(),
				BookingID: booking.ID,
				PaymentID: paymentUUID,
				Amount:    refundAmount,
				Status:    domain.RefundStatusPending,
				Reason:    "EVENT_CANCELLATION",
				CreatedAt: now,
				UpdatedAt: now,
			}

			if err := tx.RefundRepository().Create(ctx, pendingRefund); err != nil {
				return fmt.Errorf("failed to create event cancellation refund record: %w", err)
			}

			createdRefundIDs = append(createdRefundIDs, pendingRefund.ID)
			existingRefundsByBooking[booking.ID] = append(existingRefundsByBooking[booking.ID], *pendingRefund)
			cAgg.totalRefund = cAgg.totalRefund.Add(refundAmount)
		}

		// 6. Invalidate active temporary reservations and release inventory/seat holds
		var releasedSeatsCount int
		var releasedTicketsQuantity int
		var invalidatedReservationsCount int

		if tx.SeatReservationRepository() != nil {
			activeReservations, rErr := tx.SeatReservationRepository().FindActiveByEventID(ctx, eventID)
			if rErr != nil {
				u.logger.ErrorContext(ctx, "event_cancel_active_reservations_lookup_failed",
					slog.String("actor_id", actorID.String()),
					slog.String("actor_role", actorRole),
					slog.String("event_id", eventID.String()),
					slog.Any("error", rErr),
				)
				return fmt.Errorf("failed to lookup active reservations for cancelled event: %w", rErr)
			}

			for _, ar := range activeReservations {
				lockedRes, lErr := tx.SeatReservationRepository().FindByIDForUpdate(ctx, ar.ID)
				if lErr != nil {
					return fmt.Errorf("failed to lock active reservation %s: %w", ar.ID, lErr)
				}
				if lockedRes.Status != domain.ReservationStatusActive {
					continue
				}

				// If a booking was created concurrently for this reservation before it was locked, cancel it
				booking, bErr := tx.BookingRepository().FindByReservationID(ctx, lockedRes.ID)
				if bErr == nil && booking != nil && booking.Status != domain.BookingStatusCancelled {
					if err := tx.BookingRepository().UpdateStatus(ctx, booking.ID, domain.BookingStatusCancelled); err != nil {
						return fmt.Errorf("failed to cancel concurrent booking %s for reservation %s: %w", booking.ID, lockedRes.ID, err)
					}
					u.logger.InfoContext(ctx, "event_cancel_concurrent_booking_cancelled",
						slog.String("actor_id", actorID.String()),
						slog.String("actor_role", actorRole),
						slog.String("event_id", eventID.String()),
						slog.String("booking_id", booking.ID.String()),
					)
					cAgg, exists := customerAggregates[booking.UserID]
					if !exists {
						cAgg = &customerAggregate{
							customerID: booking.UserID,
						}
						customerAggregates[booking.UserID] = cAgg
					}
					cAgg.bookingRefs = append(cAgg.bookingRefs, booking.BookingNumber)
				}

				// Expire associated active checkout
				if tx.CheckoutRepository() != nil {
					if _, err := tx.CheckoutRepository().ExpireActiveByReservationID(ctx, lockedRes.ID); err != nil {
						return fmt.Errorf("failed to expire active checkout for reservation %s: %w", lockedRes.ID, err)
					}
				}

				// Find reservation items to release seat / GA inventory holds
				items, iErr := tx.SeatReservationRepository().FindItemsByReservationID(ctx, lockedRes.ID)
				if iErr != nil {
					return fmt.Errorf("failed to find items for reservation %s: %w", lockedRes.ID, iErr)
				}

				for _, item := range items {
					if item.SeatID != nil && tx.SeatRepository() != nil {
						if err := tx.SeatRepository().ReleaseReservedSeat(ctx, *item.SeatID); err != nil {
							return fmt.Errorf("failed to release reserved seat %s: %w", item.SeatID.String(), err)
						}
						releasedSeatsCount++
					} else if item.TicketTypeID != nil && tx.TicketTypeRepository() != nil {
						if err := tx.TicketTypeRepository().ReleaseReservedQuantity(ctx, *item.TicketTypeID, item.Quantity); err != nil {
							return fmt.Errorf("failed to release reserved ticket quantity for ticket type %s: %w", item.TicketTypeID.String(), err)
						}
						releasedTicketsQuantity += item.Quantity
					}
				}

				// Transition reservation status to CANCELLED
				if err := tx.SeatReservationRepository().UpdateStatus(ctx, lockedRes.ID, domain.ReservationStatusCancelled); err != nil {
					return fmt.Errorf("failed to cancel reservation %s: %w", lockedRes.ID, err)
				}
				invalidatedReservationsCount++
			}

			u.logger.InfoContext(ctx, "event_cancel_inventory_cleanup_completed",
				slog.String("actor_id", actorID.String()),
				slog.String("actor_role", actorRole),
				slog.String("event_id", eventID.String()),
				slog.Int("invalidated_reservations", invalidatedReservationsCount),
				slog.Int("released_seats", releasedSeatsCount),
				slog.Int("released_ticket_quantity", releasedTicketsQuantity),
			)
		}

		s, err := u.buildSummaryInTx(ctx, tx, eventID, event.Title, event.Status)
		if err != nil {
			return err
		}
		summary = s

		// Build customer cancellation notices from aggregated customer data
		for _, cAgg := range customerAggregates {
			if tx.UserRepository() != nil {
				user, uErr := tx.UserRepository().FindByID(cAgg.customerID)
				if uErr == nil && user != nil && strings.TrimSpace(user.Email) != "" {
					refundStatus := "PENDING"
					if cAgg.totalRefund.IsZero() {
						refundStatus = "NOT_APPLICABLE"
					}
					customerNotices = append(customerNotices, domain.CustomerCancellationNotice{
						CustomerID:         user.ID,
						CustomerName:       user.FullName,
						CustomerEmail:      user.Email,
						EventID:            eventID,
						EventTitle:         event.Title,
						EventDate:          eventDateStr,
						CancellationReason: trimmedReason,
						BookingReferences:  cAgg.bookingRefs,
						TicketsCancelled:   cAgg.ticketsCancelled,
						RefundAmount:       cAgg.totalRefund,
						RefundStatus:       refundStatus,
						RefundInformation:  "A refund has been initiated to your original payment method in accordance with standard payment processing procedures.",
					})
				}
			}
		}

		// Build organizer cancellation notice using summary metrics
		if tx.OrganizerRepository() != nil && tx.UserRepository() != nil {
			organizer, oErr := tx.OrganizerRepository().FindByID(event.OrganizerID)
			if oErr == nil && organizer != nil {
				orgUser, uErr := tx.UserRepository().FindByID(organizer.UserID)
				if uErr == nil && orgUser != nil && strings.TrimSpace(orgUser.Email) != "" {
					cancelledBy := "Event Organizer"
					if !isOrganizer {
						cancelledBy = "Platform Administrator"
					}
					organizerNotice = &domain.OrganizerCancellationNotice{
						OrganizerID:           organizer.ID,
						OrganizerName:         orgUser.FullName,
						OrganizerEmail:        orgUser.Email,
						EventID:               eventID,
						EventTitle:            event.Title,
						EventDate:             eventDateStr,
						CancellationReason:    trimmedReason,
						CancelledBy:           cancelledBy,
						ActorRole:             actorRole,
						TotalBookingsAffected: summary.TotalBookingsAffected,
						TotalTicketsAffected:  summary.TotalTicketsAffected,
						TotalRefundAmount:     summary.TotalEligibleRefundAmount,
						SettlementImpact:      "Pending settlement requests for this event have been voided and net earnings have been revoked.",
						CancelledAt:           now,
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Asynchronous Refund Processing:
	// Dispatch processing outside the database transaction so the HTTP response returns immediately.
	if len(createdRefundIDs) > 0 && u.refundProcessor != nil {
		if u.dispatchTracker != nil {
			if !u.dispatchTracker.TryAdd() {
				u.logger.WarnContext(ctx, "background_refund_dispatch_rejected_shutdown_active",
					slog.String("event_id", eventID.String()),
					slog.Int("refund_count", len(createdRefundIDs)),
				)
				return summary, nil
			}
		}
		bgCtx := context.WithoutCancel(context.Background())
		go func(refundIDs []uuid.UUID) {
			if u.dispatchTracker != nil {
				defer u.dispatchTracker.Done()
			}
			for _, refID := range refundIDs {
				_, pErr := u.refundProcessor.ProcessPendingRefund(bgCtx, refID)
				if pErr != nil {
					u.logger.Error("async_event_cancellation_refund_failed",
						slog.String("refund_id", refID.String()),
						slog.String("event_id", eventID.String()),
						slog.Any("error", pErr),
					)
				}
			}
		}(createdRefundIDs)
	}

	// Asynchronous Cancellation Notifications:
	// Dispatched outside the database transaction so that notification delivery (or failure)
	// never impacts or rolls back the successfully committed event cancellation.
	if u.notifier != nil && (len(customerNotices) > 0 || organizerNotice != nil) {
		if u.dispatchTracker != nil {
			if !u.dispatchTracker.TryAdd() {
				u.logger.WarnContext(ctx, "background_notification_dispatch_rejected_shutdown_active",
					slog.String("event_id", eventID.String()),
					slog.Int("recipient_count", len(customerNotices)+1),
				)
			} else {
				bgCtx := context.WithoutCancel(context.Background())
				go func(customers []domain.CustomerCancellationNotice, org *domain.OrganizerCancellationNotice) {
					if u.dispatchTracker != nil {
						defer u.dispatchTracker.Done()
					}
					u.dispatchNotifications(bgCtx, eventID, customers, org)
				}(customerNotices, organizerNotice)
			}
		} else {
			bgCtx := context.WithoutCancel(context.Background())
			go func(customers []domain.CustomerCancellationNotice, org *domain.OrganizerCancellationNotice) {
				u.dispatchNotifications(bgCtx, eventID, customers, org)
			}(customerNotices, organizerNotice)
		}
	}

	u.logger.InfoContext(ctx, "event_cancellation_completed",
		slog.String("event_id", eventID.String()),
		slog.String("status", summary.EventStatus),
		slog.Int("bookings_affected", summary.TotalBookingsAffected),
		slog.Int("tickets_affected", summary.TotalTicketsAffected),
		slog.String("total_eligible_refund", summary.TotalEligibleRefundAmount.String()),
		slog.Int("created_refunds_count", len(createdRefundIDs)),
	)

	return summary, nil
}

func (u *EventCancellationUsecase) dispatchNotifications(
	bgCtx context.Context,
	eventID uuid.UUID,
	customers []domain.CustomerCancellationNotice,
	org *domain.OrganizerCancellationNotice,
) {
	batchCtx, cancel := context.WithTimeout(bgCtx, 2*time.Minute)
	defer cancel()

	u.logger.InfoContext(batchCtx, "cancellation_notifications_dispatch_started",
		slog.String("event_id", eventID.String()),
		slog.Int("customer_count", len(customers)),
		slog.Bool("has_organizer", org != nil),
	)

	var deliveredCount, failedCount int

	// Send customer notifications independently
	for _, cust := range customers {
		if err := u.notifier.NotifyCustomer(batchCtx, cust); err != nil {
			failedCount++
			u.logger.ErrorContext(batchCtx, "async_customer_cancellation_notification_failed",
				slog.String("event_id", eventID.String()),
				slog.String("customer_id", cust.CustomerID.String()),
				slog.Any("error", err),
			)
		} else {
			deliveredCount++
		}
	}

	// Send organizer notification independently
	if org != nil {
		if err := u.notifier.NotifyOrganizer(batchCtx, *org); err != nil {
			failedCount++
			u.logger.ErrorContext(batchCtx, "async_organizer_cancellation_notification_failed",
				slog.String("event_id", eventID.String()),
				slog.String("organizer_id", org.OrganizerID.String()),
				slog.Any("error", err),
			)
		} else {
			deliveredCount++
		}
	}

	u.logger.InfoContext(batchCtx, "cancellation_notifications_dispatch_completed",
		slog.String("event_id", eventID.String()),
		slog.Int("delivered_count", deliveredCount),
		slog.Int("failed_count", failedCount),
	)
}

func (u *EventCancellationUsecase) buildSummaryInTx(
	ctx context.Context,
	tx domain.TransactionRepositories,
	eventID uuid.UUID,
	eventTitle string,
	eventStatus string,
) (*domain.EventCancellationSummary, error) {
	cancellation, _ := tx.EventCancellationRepository().FindByEventID(eventID)
	refunds, err := tx.RefundRepository().FindByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	var reason string
	var cancelledAt *time.Time
	if cancellation != nil {
		cancelledAt = cancellation.CancelledAt
		if cancellation.CancellationReason != nil {
			reason = *cancellation.CancellationReason
		}
	}

	bookings, _ := tx.BookingRepository().FindByEventID(ctx, eventID)
	tickets, _ := tx.TicketRepository().FindByEventID(ctx, eventID)

	totalTicketsAffected := 0
	for _, t := range tickets {
		if t.Status == domain.TicketStatusCancelled || t.Status == domain.TicketStatusRefunded {
			totalTicketsAffected++
		}
	}

	totalEligible := decimal.Zero
	totalRefunded := decimal.Zero
	pendingAmount := decimal.Zero
	failedAmount := decimal.Zero
	pendingCount := 0
	failedCount := 0
	successCount := 0

	for _, r := range refunds {
		if r.Reason == "EVENT_CANCELLATION" {
			totalEligible = totalEligible.Add(r.Amount)
			switch r.Status {
			case domain.RefundStatusSuccess:
				totalRefunded = totalRefunded.Add(r.Amount)
				successCount++
			case domain.RefundStatusProcessing, domain.RefundStatusPending:
				pendingAmount = pendingAmount.Add(r.Amount)
				pendingCount++
			case domain.RefundStatusFailed:
				failedAmount = failedAmount.Add(r.Amount)
				failedCount++
			}
		}
	}

	completionStatus := "COMPLETED"
	if len(refunds) == 0 {
		completionStatus = "NO_REFUNDS_NEEDED"
	} else if pendingCount > 0 {
		completionStatus = "PROCESSING"
	} else if failedCount > 0 {
		if successCount > 0 {
			completionStatus = "PARTIALLY_FAILED"
		} else {
			completionStatus = "FAILED"
		}
	}

	return &domain.EventCancellationSummary{
		EventID:                   eventID,
		EventTitle:                eventTitle,
		EventStatus:               eventStatus,
		CancelledAt:               cancelledAt,
		CancellationReason:        reason,
		TotalBookingsAffected:     len(bookings),
		TotalTicketsAffected:      totalTicketsAffected,
		TotalEligibleRefundAmount: totalEligible,
		TotalRefundedAmount:       totalRefunded,
		PendingRefundAmount:       pendingAmount,
		PendingRefundCount:        pendingCount,
		FailedRefundAmount:        failedAmount,
		FailedRefundCount:         failedCount,
		SuccessfulRefundCount:     successCount,
		CompletionStatus:          completionStatus,
	}, nil
}

func isEventConcluded(schedule *domain.EventSchedule, now time.Time) bool {
	if schedule == nil {
		return false
	}
	loc := time.UTC
	if schedule.EventDate.Location() != nil {
		loc = schedule.EventDate.Location()
	}

	var conclusion time.Time
	if schedule.IsAllDay || schedule.EndTime.IsZero() {
		conclusion = time.Date(
			schedule.EventDate.Year(), schedule.EventDate.Month(), schedule.EventDate.Day(),
			23, 59, 59, 999999999, loc,
		).UTC()
	} else {
		conclusion = time.Date(
			schedule.EventDate.Year(), schedule.EventDate.Month(), schedule.EventDate.Day(),
			schedule.EndTime.Hour(), schedule.EndTime.Minute(), schedule.EndTime.Second(),
			0, loc,
		).UTC()

		if !schedule.StartTime.IsZero() && schedule.EndTime.Before(schedule.StartTime) {
			conclusion = conclusion.AddDate(0, 0, 1)
		}
	}

	return now.After(conclusion)
}

type CancellationEligibilityAssessment struct {
	IsCancellable           bool
	BlockingReasons         []string
	FirstBlockingError      error
	HasCompletedSettlement  bool
	HasInFlightSettlement   bool
	PendingSettlementsCount int
	PendingSettlementsTotal decimal.Decimal
	SettlementWarning       string
}

func evaluateCancellationEligibility(
	event *domain.Event,
	schedule *domain.EventSchedule,
	settlements []*domain.Settlement,
	now time.Time,
) CancellationEligibilityAssessment {
	var assessment CancellationEligibilityAssessment
	assessment.PendingSettlementsTotal = decimal.Zero

	// 1. Lifecycle state validation
	if event.Status == domain.EventStatusCancelled {
		assessment.BlockingReasons = append(assessment.BlockingReasons, "event is already cancelled")
		if assessment.FirstBlockingError == nil {
			assessment.FirstBlockingError = domain.ErrEventAlreadyCancelled
		}
	} else if event.Status == domain.EventStatusDraft {
		assessment.BlockingReasons = append(assessment.BlockingReasons, "draft events cannot be cancelled; delete the draft instead")
		if assessment.FirstBlockingError == nil {
			assessment.FirstBlockingError = domain.ErrDraftEventCannotBeCancelled
		}
	} else if event.Status != domain.EventStatusPublished {
		assessment.BlockingReasons = append(assessment.BlockingReasons, "event cannot be cancelled")
		if assessment.FirstBlockingError == nil {
			assessment.FirstBlockingError = domain.ErrEventCannotBeCancelled
		}
	}

	// 2. Event schedule / past conclusion check
	if schedule != nil && isEventConcluded(schedule, now) {
		assessment.BlockingReasons = append(assessment.BlockingReasons, "event has already concluded")
		if assessment.FirstBlockingError == nil {
			assessment.FirstBlockingError = domain.ErrPastEventCannotBeCancelled
		}
	}

	// 3. Settlement safety checks
	for _, s := range settlements {
		switch s.Status {
		case domain.SettlementStatusCompleted:
			assessment.HasCompletedSettlement = true
			assessment.BlockingReasons = append(assessment.BlockingReasons, "event has completed settlements with disbursed organizer funds")
			if assessment.FirstBlockingError == nil {
				assessment.FirstBlockingError = domain.ErrEventCannotBeCancelledWithCompletedSettlement
			}
		case domain.SettlementStatusApproved, domain.SettlementStatusProcessing:
			assessment.HasInFlightSettlement = true
			assessment.BlockingReasons = append(assessment.BlockingReasons, "event has approved or in-flight settlement payouts currently processing")
			if assessment.FirstBlockingError == nil {
				assessment.FirstBlockingError = domain.ErrEventCannotBeCancelledWithActiveSettlement
			}
		case domain.SettlementStatusPending:
			assessment.PendingSettlementsCount++
			assessment.PendingSettlementsTotal = assessment.PendingSettlementsTotal.Add(s.Amount)
		}
	}

	if assessment.PendingSettlementsCount > 0 {
		assessment.SettlementWarning = fmt.Sprintf(
			"%d pending settlement request(s) totaling %s will be automatically rejected upon cancellation",
			assessment.PendingSettlementsCount,
			assessment.PendingSettlementsTotal.StringFixed(2),
		)
	}

	assessment.IsCancellable = len(assessment.BlockingReasons) == 0
	return assessment
}

func (u *EventCancellationUsecase) calculateCancellationPreviewInTx(
	ctx context.Context,
	tx domain.TransactionRepositories,
	event *domain.Event,
	now time.Time,
) (*domain.EventCancellationPreview, error) {
	var schedule *domain.EventSchedule
	eventDateStr := "TBD"
	if tx.EventScheduleRepository() != nil {
		s, sErr := tx.EventScheduleRepository().FindByEventID(event.ID)
		if sErr != nil && !errors.Is(sErr, domain.ErrEventNotFound) {
			return nil, fmt.Errorf("failed to lookup event schedule: %w", sErr)
		}
		if s != nil {
			schedule = s
			eventDateStr = s.EventDate.Format("02 Jan 2006")
		}
	}

	var settlements []*domain.Settlement
	if tx.SettlementRepository() != nil {
		settlementResult, sErr := tx.SettlementRepository().List(ctx, domain.SettlementFilter{
			EventID: &event.ID,
			Limit:   100,
		})
		if sErr != nil {
			return nil, fmt.Errorf("failed to verify settlement records: %w", sErr)
		}
		if settlementResult != nil {
			settlements = settlementResult.Settlements
		}
	}

	assessment := evaluateCancellationEligibility(event, schedule, settlements, now)
	blockingReasons := assessment.BlockingReasons
	if blockingReasons == nil {
		blockingReasons = []string{}
	}

	var bookings []domain.Booking
	if tx.BookingRepository() != nil {
		b, err := tx.BookingRepository().FindByEventID(ctx, event.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve bookings: %w", err)
		}
		bookings = b
	}

	var tickets []domain.Ticket
	if tx.TicketRepository() != nil {
		t, err := tx.TicketRepository().FindByEventID(ctx, event.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve tickets: %w", err)
		}
		tickets = t
	}

	ticketsByBooking := make(map[uuid.UUID][]domain.Ticket, len(bookings))
	totalTicketsSold := 0
	affectedTickets := 0
	for _, t := range tickets {
		ticketsByBooking[t.BookingID] = append(ticketsByBooking[t.BookingID], t)
		if t.Status == domain.TicketStatusValid || t.Status == domain.TicketStatusUsed {
			totalTicketsSold++
		}
		if t.Status == domain.TicketStatusValid {
			affectedTickets++
		}
	}

	var activeReservationsReleased, reservedSeatsReleased, gaQuantityReleased, activeCheckoutsExpired int
	if tx.SeatReservationRepository() != nil {
		activeRes, rErr := tx.SeatReservationRepository().FindActiveByEventID(ctx, event.ID)
		if rErr != nil {
			return nil, fmt.Errorf("failed to retrieve active reservations: %w", rErr)
		}
		activeReservationsReleased = len(activeRes)
		for _, ar := range activeRes {
			items, iErr := tx.SeatReservationRepository().FindItemsByReservationID(ctx, ar.ID)
			if iErr != nil {
				return nil, fmt.Errorf("failed to retrieve reservation items: %w", iErr)
			}
			for _, item := range items {
				if item.SeatID != nil {
					reservedSeatsReleased++
				} else if item.TicketTypeID != nil {
					gaQuantityReleased += item.Quantity
				}
			}
			if tx.CheckoutRepository() != nil {
				checkout, cErr := tx.CheckoutRepository().FindByReservationID(ctx, ar.ID)
				if cErr == nil && checkout != nil && checkout.Status == domain.CheckoutStatusActive {
					activeCheckoutsExpired++
				}
			}
		}
	}

	var existingRefunds []domain.Refund
	if tx.RefundRepository() != nil {
		r, err := tx.RefundRepository().FindByEventID(ctx, event.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve refunds: %w", err)
		}
		existingRefunds = r
	}

	existingRefundsByBooking := make(map[uuid.UUID][]domain.Refund, len(bookings))
	alreadyRefundedAmount := decimal.Zero
	for _, r := range existingRefunds {
		existingRefundsByBooking[r.BookingID] = append(existingRefundsByBooking[r.BookingID], r)
		if r.Status == domain.RefundStatusSuccess {
			alreadyRefundedAmount = alreadyRefundedAmount.Add(r.Amount)
		}
	}

	totalBookings := len(bookings)
	affectedBookings := 0
	grossCollectedAmount := decimal.Zero
	expectedRefundAmount := decimal.Zero
	totalPlatformFeeRefund := decimal.Zero
	totalTaxRefund := decimal.Zero

	for _, booking := range bookings {
		if booking.Status == domain.BookingStatusConfirmed {
			affectedBookings++
		}

		var payments []domain.Payment
		if tx.PaymentRepository() != nil {
			p, pErr := tx.PaymentRepository().FindByBookingID(ctx, booking.ID.String())
			if pErr != nil {
				return nil, fmt.Errorf("failed to find payments for booking %s: %w", booking.ID, pErr)
			}
			payments = p
		}

		var successfulPayment *domain.Payment
		for i := range payments {
			if payments[i].Status == domain.PaymentStatusSuccess {
				grossCollectedAmount = grossCollectedAmount.Add(payments[i].Amount)
				if successfulPayment == nil && payments[i].TransactionID != nil && strings.TrimSpace(*payments[i].TransactionID) != "" {
					successfulPayment = &payments[i]
				}
			}
		}

		// Free bookings or bookings without successful payment cannot generate refunds
		if successfulPayment == nil || booking.Total.IsZero() || !booking.Total.IsPositive() {
			continue
		}

		// Check if an EVENT_CANCELLATION refund was already created
		hasEventRefund := false
		for _, r := range existingRefundsByBooking[booking.ID] {
			if r.Reason == "EVENT_CANCELLATION" {
				hasEventRefund = true
				break
			}
		}
		if hasEventRefund {
			continue
		}

		// Valid tickets check
		bookingTickets := ticketsByBooking[booking.ID]
		validCount := 0
		for _, bt := range bookingTickets {
			if bt.Status == domain.TicketStatusValid {
				validCount++
			}
		}
		if validCount == 0 {
			continue
		}

		// Calculate total already refunded for this booking across prior customer cancellations
		totalAlreadyRefundedForBooking := decimal.Zero
		priorPlatformFeeRefunded := decimal.Zero
		priorTaxRefunded := decimal.Zero

		for _, r := range existingRefundsByBooking[booking.ID] {
			if r.Status == domain.RefundStatusSuccess || r.Status == domain.RefundStatusProcessing || r.Status == domain.RefundStatusPending {
				totalAlreadyRefundedForBooking = totalAlreadyRefundedForBooking.Add(r.Amount)
			}
			if r.Status == domain.RefundStatusSuccess && booking.Total.IsPositive() {
				fee := r.Amount.Mul(booking.PlatformFee).Div(booking.Total).RoundBank(2)
				tax := r.Amount.Mul(booking.Tax).Div(booking.Total).RoundBank(2)
				priorPlatformFeeRefunded = priorPlatformFeeRefunded.Add(fee)
				priorTaxRefunded = priorTaxRefunded.Add(tax)
			}
		}

		maxRefundable := successfulPayment.Amount.Sub(totalAlreadyRefundedForBooking)
		if maxRefundable.IsNegative() {
			maxRefundable = decimal.Zero
		}

		var refundAmount decimal.Decimal
		if validCount == len(bookingTickets) {
			refundAmount = booking.Total
		} else {
			refundAmount = booking.Total.Sub(totalAlreadyRefundedForBooking)
		}

		if refundAmount.GreaterThan(maxRefundable) {
			refundAmount = maxRefundable
		}

		if refundAmount.IsZero() || !refundAmount.IsPositive() {
			continue
		}

		expectedRefundAmount = expectedRefundAmount.Add(refundAmount)

		// Platform fee and tax portions
		var platformFeeRefund decimal.Decimal
		var taxRefund decimal.Decimal
		if booking.Total.IsPositive() {
			if booking.PlatformFee.IsPositive() {
				remainingFee := booking.PlatformFee.Sub(priorPlatformFeeRefunded)
				if remainingFee.IsPositive() {
					computedFee := refundAmount.Mul(booking.PlatformFee).Div(booking.Total).RoundBank(2)
					if computedFee.GreaterThan(remainingFee) {
						platformFeeRefund = remainingFee
					} else {
						platformFeeRefund = computedFee
					}
				}
			}
			if booking.Tax.IsPositive() {
				remainingTax := booking.Tax.Sub(priorTaxRefunded)
				if remainingTax.IsPositive() {
					computedTax := refundAmount.Mul(booking.Tax).Div(booking.Total).RoundBank(2)
					if computedTax.GreaterThan(remainingTax) {
						taxRefund = remainingTax
					} else {
						taxRefund = computedTax
					}
				}
			}
			if platformFeeRefund.Add(taxRefund).GreaterThan(refundAmount) {
				if platformFeeRefund.GreaterThan(refundAmount) {
					platformFeeRefund = refundAmount
					taxRefund = decimal.Zero
				} else {
					taxRefund = refundAmount.Sub(platformFeeRefund)
				}
			}
		}

		totalPlatformFeeRefund = totalPlatformFeeRefund.Add(platformFeeRefund)
		totalTaxRefund = totalTaxRefund.Add(taxRefund)
	}

	organizerPayableRevocation := expectedRefundAmount.Sub(totalPlatformFeeRefund).Sub(totalTaxRefund)
	if organizerPayableRevocation.IsNegative() {
		organizerPayableRevocation = decimal.Zero
	}

	return &domain.EventCancellationPreview{
		Event: domain.CancellationEventInfo{
			ID:            event.ID,
			Title:         event.Title,
			Status:        event.Status,
			ScheduledDate: eventDateStr,
		},
		Eligibility: domain.CancellationEligibility{
			IsCancellable:   assessment.IsCancellable,
			BlockingReasons: blockingReasons,
		},
		OperationalImpact: domain.CancellationOperationalImpact{
			TotalBookings:                    totalBookings,
			AffectedBookings:                 affectedBookings,
			TotalTicketsSold:                 totalTicketsSold,
			AffectedTickets:                  affectedTickets,
			ActiveReservationsReleased:       activeReservationsReleased,
			ReservedSeatsReleased:            reservedSeatsReleased,
			GeneralAdmissionQuantityReleased: gaQuantityReleased,
			ActiveCheckoutsExpired:           activeCheckoutsExpired,
		},
		FinancialImpact: domain.CancellationFinancialImpact{
			Currency:                   "INR",
			GrossCollectedAmount:       grossCollectedAmount,
			AlreadyRefundedAmount:      alreadyRefundedAmount,
			ExpectedRefundAmount:       expectedRefundAmount,
			PlatformFeeRefund:          totalPlatformFeeRefund,
			TaxRefund:                  totalTaxRefund,
			OrganizerPayableRevocation: organizerPayableRevocation,
		},
		SettlementImpact: domain.CancellationSettlementImpact{
			HasCompletedSettlement:  assessment.HasCompletedSettlement,
			HasInFlightSettlement:   assessment.HasInFlightSettlement,
			PendingSettlementsCount: assessment.PendingSettlementsCount,
			PendingSettlementsTotal: assessment.PendingSettlementsTotal,
			SettlementWarning:       assessment.SettlementWarning,
		},
		GeneratedAt: now,
	}, nil
}
