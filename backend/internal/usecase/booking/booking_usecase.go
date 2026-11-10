package booking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/invoice"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BookingUsecase struct {
	transactionManager domain.TransactionManager
	refundProvider     domain.PaymentRefundProvider
	receiptPDFService  *invoice.ReceiptPDFService
	logger             *slog.Logger
}

func NewBookingUsecase(
	transactionManager domain.TransactionManager,
	refundProvider domain.PaymentRefundProvider,
	logger *slog.Logger,
) *BookingUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &BookingUsecase{
		transactionManager: transactionManager,
		refundProvider:     refundProvider,
		receiptPDFService:  invoice.NewReceiptPDFService(),
		logger:             logger,
	}
}

type CreateBookingRequest struct {
	CheckoutID uuid.UUID
	UserID     uuid.UUID
}

func (u *BookingUsecase) CreateBooking(
	ctx context.Context,
	req CreateBookingRequest,
) (*domain.Booking, error) {
	if req.CheckoutID == uuid.Nil || req.UserID == uuid.Nil {
		u.logger.WarnContext(ctx, "invalid booking request",
			slog.String("checkout_id", req.CheckoutID.String()),
			slog.String("user_id", req.UserID.String()),
		)
		return nil, domain.ErrInvalidReservationRequest
	}

	var booking *domain.Booking

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			now := time.Now()

			checkout, err := tx.CheckoutRepository().FindByID(
				ctx,
				req.CheckoutID,
			)
			if err != nil {
				return err
			}

			if checkout.UserID != req.UserID {
				return domain.ErrCheckoutNotFound
			}

			// Return the existing booking if this reservation was already booked.
			existing, err := tx.BookingRepository().FindByReservationID(
				ctx,
				checkout.ReservationID,
			)
			if err == nil {
				if existing.UserID != req.UserID {
					return domain.ErrBookingNotFound
				}

				booking = existing
				return nil
			}

			if !errors.Is(err, domain.ErrBookingNotFound) {
				return err
			}

			// A checkout must still be active and unexpired.
			if checkout.Status != domain.CheckoutStatusActive ||
				!checkout.ExpiresAt.After(now) {
				return domain.ErrInvalidReservationRequest
			}

			// Lock and revalidate the reservation inside this transaction.
			reservation, err := tx.SeatReservationRepository().
				FindByIDForUpdate(ctx, checkout.ReservationID)
			if err != nil {
				return err
			}

			if reservation.UserID != req.UserID {
				return domain.ErrSeatReservationNotFound
			}

			if reservation.Status != domain.ReservationStatusActive ||
				!reservation.ExpiresAt.After(now) {
				return domain.ErrInvalidReservationRequest
			}

			reservationItems, err := tx.SeatReservationRepository().
				FindItemsByReservationID(ctx, reservation.ID)
			if err != nil {
				return err
			}

			if len(reservationItems) == 0 {
				return domain.ErrInvalidReservationRequest
			}

			// Generate a booking number. The database unique constraint
			// must remain the final protection against collisions.
			bookingID := uuid.New()
			bookingNumber := fmt.Sprintf(
				"EVH-%s",
				bookingID.String()[:12],
			)

			// Retrieve the event's cancellation policy applicable at booking time.
			cancellationAllowed := false
			cancellationDeadlineHours := 0
			cancellationRefundPolicy := domain.RefundPolicyNoRefund
			cancellationRefundPercentage := 0

			cancellation, err := tx.EventCancellationRepository().FindByEventID(reservation.EventID)
			if err != nil {
				if errors.Is(err, domain.ErrEventNotFound) {
					u.logger.WarnContext(ctx, "event cancellation policy not found, applying default no-refund snapshot",
						slog.String("event_id", reservation.EventID.String()),
						slog.String("booking_id", bookingID.String()),
					)
				} else {
					u.logger.ErrorContext(ctx, "failed to retrieve event cancellation policy for booking snapshot",
						slog.String("event_id", reservation.EventID.String()),
						slog.String("booking_id", bookingID.String()),
						slog.String("error", err.Error()),
					)
					return err
				}
			} else if cancellation != nil {
				cancellationAllowed = cancellation.CancellationAllowed
				cancellationDeadlineHours = cancellation.CancellationDeadlineHours
				cancellationRefundPolicy = cancellation.RefundPolicy
				cancellationRefundPercentage = cancellation.RefundPercentage
			}

			booking = &domain.Booking{
				ID:            bookingID,
				BookingNumber: bookingNumber,
				UserID:        req.UserID,
				EventID:       reservation.EventID,
				ReservationID: &reservation.ID,

				// Use the server-generated checkout snapshot, never
				// client-supplied amounts.
				Subtotal:    checkout.Subtotal,
				PlatformFee: checkout.PlatformFee,
				Tax:         checkout.TaxAmount,
				Discount:    checkout.DiscountAmount,
				Total:       checkout.TotalAmount,

				CancellationAllowed:       cancellationAllowed,
				CancellationDeadlineHours: cancellationDeadlineHours,
				RefundPolicy:              cancellationRefundPolicy,
				RefundPercentage:          cancellationRefundPercentage,

				Status:    domain.BookingStatusPending,
				CreatedAt: now,
				UpdatedAt: now,
			}

			u.logger.InfoContext(ctx, "booking_cancellation_policy_snapshotted",
				slog.String("booking_id", bookingID.String()),
				slog.String("event_id", reservation.EventID.String()),
				slog.Bool("cancellation_allowed", booking.CancellationAllowed),
				slog.Int("cancellation_deadline_hours", booking.CancellationDeadlineHours),
				slog.String("refund_policy", booking.RefundPolicy),
				slog.Int("refund_percentage", booking.RefundPercentage),
			)

			bookingItems := make([]domain.BookingItem, 0, len(reservationItems))

			for _, item := range reservationItems {
				if item.Quantity <= 0 || item.Price.IsNegative() {
					return domain.ErrInvalidReservationRequest
				}

				bookingItems = append(bookingItems, domain.BookingItem{
					ID:           uuid.New(),
					BookingID:    bookingID,
					SeatID:       item.SeatID,
					TicketTypeID: item.TicketTypeID,
					Quantity:     item.Quantity,
					UnitPrice:    item.Price,
					TotalPrice:   item.Total(),
					CreatedAt:    now,
					UpdatedAt:    now,
				})
			}

			if err := tx.BookingRepository().Create(
				ctx,
				booking,
				bookingItems,
			); err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		if errors.Is(err, domain.ErrDuplicateReservationBooking) {
			// The failed transaction has rolled back.
			// Fetch the existing booking outside that transaction.
			existing, findErr := u.getExistingBookingAfterConflict(
				ctx,
				req.CheckoutID,
				req.UserID,
			)
			if findErr == nil {
				u.logger.InfoContext(ctx, "returning existing booking after duplicate request",
					slog.String("booking_id", existing.ID.String()),
					slog.String("checkout_id", req.CheckoutID.String()),
					slog.String("user_id", req.UserID.String()),
				)
				return existing, nil
			}

			u.logger.WarnContext(ctx, "duplicate booking detected but existing booking could not be returned",
				slog.String("checkout_id", req.CheckoutID.String()),
				slog.String("user_id", req.UserID.String()),
				slog.String("error", findErr.Error()),
			)
		}

		u.logger.ErrorContext(ctx, "booking creation failed",
			slog.String("checkout_id", req.CheckoutID.String()),
			slog.String("user_id", req.UserID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "booking created or retrieved",
		slog.String("booking_id", booking.ID.String()),
		slog.String("booking_number", booking.BookingNumber),
		slog.String("checkout_id", req.CheckoutID.String()),
		slog.String("user_id", req.UserID.String()),
		slog.String("total_amount", booking.Total.String()),
		slog.String("status", booking.Status),
	)

	return booking, nil
}

// GetMyBookings returns all bookings belonging to the authenticated customer.
func (u *BookingUsecase) GetMyBookings(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Booking, error) {
	if userID == uuid.Nil {
		u.logger.WarnContext(ctx, "booking listing rejected: invalid user ID")
		return nil, domain.ErrBookingNotFound
	}

	var bookings []domain.Booking

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			var err error
			bookings, err = tx.BookingRepository().FindByUserID(ctx, userID)
			if err != nil {
				return err
			}

			return u.populateBookingSeats(ctx, tx, bookings)
		},
	)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to retrieve customer bookings",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	if bookings == nil {
		bookings = []domain.Booking{}
	}

	return bookings, nil
}

// GetMyBookingsPaginated returns paginated bookings belonging to the authenticated customer.
func (u *BookingUsecase) GetMyBookingsPaginated(
	ctx context.Context,
	userID uuid.UUID,
	filter domain.CustomerBookingFilter,
) (*domain.CustomerBookingList, error) {
	if userID == uuid.Nil {
		u.logger.WarnContext(ctx, "booking listing rejected: invalid user ID")
		return nil, domain.ErrBookingNotFound
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 5
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}

	var bookings []domain.Booking
	var total int64

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			var err error
			bookings, total, err = tx.BookingRepository().FindByUserIDPaginated(ctx, userID, filter)
			if err != nil {
				return err
			}

			return u.populateBookingSeats(ctx, tx, bookings)
		},
	)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to retrieve paginated customer bookings",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	if bookings == nil {
		bookings = []domain.Booking{}
	}

	totalPages := 1
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(filter.Limit)))
	}

	return &domain.CustomerBookingList{
		Bookings:   bookings,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (u *BookingUsecase) populateBookingSeats(
	ctx context.Context,
	tx domain.TransactionRepositories,
	bookings []domain.Booking,
) error {
	if len(bookings) == 0 {
		return nil
	}

	bookingIDs := make([]uuid.UUID, 0, len(bookings))
	for _, b := range bookings {
		bookingIDs = append(bookingIDs, b.ID)
	}

	itemsByBooking, err := tx.BookingRepository().FindItemsByBookingIDs(ctx, bookingIDs)
	if err != nil {
		u.logger.WarnContext(ctx, "failed to find items for customer bookings", slog.Any("error", err))
		return nil
	}

	var allSeatIDs []uuid.UUID
	for _, items := range itemsByBooking {
		for _, item := range items {
			if item.SeatID != nil && *item.SeatID != uuid.Nil {
				allSeatIDs = append(allSeatIDs, *item.SeatID)
			}
		}
	}

	seatMap := make(map[uuid.UUID]domain.SeatDetail)
	if len(allSeatIDs) > 0 {
		details, err := tx.SeatRepository().FindDetailsBySeatIDs(ctx, allSeatIDs)
		if err != nil {
			u.logger.WarnContext(ctx, "failed to resolve seats for customer bookings", slog.Any("error", err))
		} else {
			seatMap = details
		}
	}

	for i := range bookings {
		bID := bookings[i].ID
		items := itemsByBooking[bID]
		var seats []domain.SeatDetail
		for _, item := range items {
			if item.SeatID != nil {
				if d, ok := seatMap[*item.SeatID]; ok {
					seats = append(seats, d)
				}
			}
		}
		bookings[i].Seats = seats
	}

	return nil
}

// GetBookingByID returns a specific booking and its associated items only when it belongs to the authenticated user.
func (u *BookingUsecase) GetBookingByID(
	ctx context.Context,
	userID uuid.UUID,
	bookingID uuid.UUID,
) (*domain.BookingWithItems, error) {
	if userID == uuid.Nil || bookingID == uuid.Nil {
		return nil, domain.ErrBookingNotFound
	}

	var result *domain.BookingWithItems

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			booking, items, err := tx.BookingRepository().FindWithItemsByID(ctx, bookingID)
			if err != nil {
				return err
			}

			// Do not reveal whether another customer's booking exists.
			if booking.UserID != userID {
				u.logger.WarnContext(ctx, "booking access denied",
					slog.String("user_id", userID.String()),
					slog.String("booking_id", bookingID.String()),
				)
				return domain.ErrBookingNotFound
			}

			var seatIDs []uuid.UUID
			for _, item := range items {
				if item.SeatID != nil && *item.SeatID != uuid.Nil {
					seatIDs = append(seatIDs, *item.SeatID)
				}
			}

			var seats []domain.SeatDetail
			if len(seatIDs) > 0 {
				seatMap, err := tx.SeatRepository().FindDetailsBySeatIDs(ctx, seatIDs)
				if err != nil {
					u.logger.WarnContext(ctx, "failed to resolve seat details for booking",
						slog.String("booking_id", bookingID.String()),
						slog.Any("error", err),
					)
				} else {
					for i := range items {
						if items[i].SeatID != nil {
							if detail, ok := seatMap[*items[i].SeatID]; ok {
								items[i].Seat = &detail
								seats = append(seats, detail)
							}
						}
					}
				}
			}

			booking.Seats = seats
			result = &domain.BookingWithItems{
				Booking: *booking,
				Items:   items,
			}
			return nil
		},
	)
	if err != nil {
		if errors.Is(err, domain.ErrBookingNotFound) {
			return nil, domain.ErrBookingNotFound
		}

		u.logger.ErrorContext(ctx, "failed to retrieve booking",
			slog.String("booking_id", bookingID.String()),
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "customer booking retrieved",
		slog.String("booking_id", bookingID.String()),
		slog.String("user_id", userID.String()),
	)

	return result, nil
}

func (u *BookingUsecase) getExistingBookingAfterConflict(
	ctx context.Context,
	checkoutID uuid.UUID,
	userID uuid.UUID,
) (*domain.Booking, error) {
	var booking *domain.Booking

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			checkout, err := tx.CheckoutRepository().FindByID(
				ctx,
				checkoutID,
			)
			if err != nil {
				return err
			}

			if checkout.UserID != userID {
				return domain.ErrCheckoutNotFound
			}

			existing, err := tx.BookingRepository().FindByReservationID(
				ctx,
				checkout.ReservationID,
			)
			if err != nil {
				return err
			}

			if existing.UserID != userID {
				return domain.ErrBookingNotFound
			}

			booking = existing
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return booking, nil
}

// GetBookingPDFData verifies booking ownership and compiles booking, ticket, seat, and venue details for PDF generation.
func (u *BookingUsecase) GetBookingPDFData(
	ctx context.Context,
	userID uuid.UUID,
	bookingID uuid.UUID,
) (*ticket.BookingPDFData, error) {
	if userID == uuid.Nil || bookingID == uuid.Nil {
		return nil, domain.ErrBookingNotFound
	}

	var pdfData *ticket.BookingPDFData

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			booking, items, err := tx.BookingRepository().FindWithItemsByID(ctx, bookingID)
			if err != nil {
				return err
			}

			if booking.UserID != userID {
				u.logger.WarnContext(ctx, "booking access denied for PDF download",
					slog.String("user_id", userID.String()),
					slog.String("booking_id", bookingID.String()),
				)
				return domain.ErrBookingNotFound
			}

			// Retrieve customer details
			customerName := "Valued Customer"
			user, err := tx.UserRepository().FindByID(booking.UserID)
			if err == nil && user != nil {
				if strings.TrimSpace(user.FullName) != "" {
					customerName = strings.TrimSpace(user.FullName)
				} else if strings.TrimSpace(user.Email) != "" {
					customerName = strings.TrimSpace(user.Email)
				}
			}

			// Retrieve event
			event, err := tx.EventRepository().FindByID(booking.EventID)
			if err != nil {
				return err
			}
			eventName := "EventHub Event"
			if strings.TrimSpace(event.Title) != "" {
				eventName = strings.TrimSpace(event.Title)
			}

			// Retrieve schedule
			var eventDate time.Time
			schedule, err := tx.EventScheduleRepository().FindByEventID(event.ID)
			if err == nil && schedule != nil && !schedule.EventDate.IsZero() {
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

			// Retrieve venue
			var venueName string
			var venueAddress string
			if event.EventType != domain.EventTypeOnline && event.VenueID != nil && *event.VenueID != uuid.Nil {
				venue, err := tx.VenueRepository().FindByID(*event.VenueID)
				if err == nil && venue != nil {
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

			// Retrieve tickets
			tickets, _ := tx.TicketRepository().FindByBookingID(ctx, bookingID)

			// Count admissions
			admissionCount := len(tickets)
			if admissionCount == 0 {
				for _, it := range items {
					admissionCount += it.Quantity
				}
			}
			if admissionCount == 0 {
				admissionCount = 1
			}

			// Resolve seat details
			var seatIDs []uuid.UUID
			for _, item := range items {
				if item.SeatID != nil && *item.SeatID != uuid.Nil {
					seatIDs = append(seatIDs, *item.SeatID)
				}
			}
			var seatLabels []string
			if len(seatIDs) > 0 {
				seatMap, err := tx.SeatRepository().FindDetailsBySeatIDs(ctx, seatIDs)
				if err == nil {
					for _, sid := range seatIDs {
						if d, ok := seatMap[sid]; ok {
							seatLabels = append(seatLabels, d.Label)
						}
					}
				}
			}

			// Retrieve ticket tier name
			ticketTypeName := "General Admission"
			var tierID *uuid.UUID
			if len(items) > 0 && items[0].TicketTypeID != nil {
				tierID = items[0].TicketTypeID
			} else if len(tickets) > 0 && tickets[0].TicketTypeID != nil {
				tierID = tickets[0].TicketTypeID
			}
			if tierID != nil && *tierID != uuid.Nil {
				tt, err := tx.TicketTypeRepository().FindByID(ctx, *tierID)
				if err == nil && tt != nil && strings.TrimSpace(tt.Name) != "" {
					ticketTypeName = strings.TrimSpace(tt.Name)
				}
			}

			pdfData = &ticket.BookingPDFData{
				BookingID:     booking.ID,
				BookingNumber: booking.BookingNumber,
				EventName:     eventName,
				CustomerName:  customerName,
				TicketCount:   admissionCount,
				TicketType:    ticketTypeName,
				Seats:         seatLabels,
				EventDate:     eventDate,
				VenueName:     venueName,
				VenueAddress:  venueAddress,
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return pdfData, nil
}

type CancelBookingOutput struct {
	BookingID     uuid.UUID       `json:"booking_id"`
	BookingNumber string          `json:"booking_number"`
	Status        string          `json:"status"`
	RefundAmount  decimal.Decimal `json:"refund_amount"`
	RefundStatus  string          `json:"refund_status"`
	RefundID      string          `json:"refund_id,omitempty"`
	CancelledAt   time.Time       `json:"cancelled_at"`
}

func (u *BookingUsecase) CancelBooking(
	ctx context.Context,
	userID uuid.UUID,
	bookingID uuid.UUID,
) (*CancelBookingOutput, error) {
	if userID == uuid.Nil || bookingID == uuid.Nil {
		return nil, domain.ErrBookingNotFound
	}

	var (
		booking           *domain.Booking
		refundAmount      decimal.Decimal
		successfulPayment *domain.Payment
		paymentUUID       uuid.UUID
		pendingRefund     *domain.Refund
		isFreeOrNoRefund  bool
		now               = time.Now()
	)

	// Phase 1: Validate booking state, policy, deadline, tickets, and prepare refund record.
	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			var err error
			booking, err = tx.BookingRepository().FindByIDForUpdate(ctx, bookingID)
			if err != nil {
				return err
			}

			if booking.UserID != userID {
				u.logger.WarnContext(ctx, "unauthorized booking cancellation attempt",
					slog.String("booking_id", bookingID.String()),
					slog.String("user_id", userID.String()),
				)
				return domain.ErrBookingNotFound
			}

			if booking.Status == domain.BookingStatusCancelled {
				return domain.ErrBookingAlreadyCancelled
			}

			if booking.Status != domain.BookingStatusConfirmed {
				return domain.ErrBookingCannotBeCancelled
			}

			if !booking.CancellationAllowed {
				return domain.ErrCancellationNotAllowed
			}

			// Validate deadline against event start time.
			schedule, err := tx.EventScheduleRepository().FindByEventID(booking.EventID)
			if err != nil {
				return err
			}
			if schedule == nil {
				return domain.ErrEventNotFound
			}

			eventStartTime := schedule.EventDate
			if !schedule.StartTime.IsZero() {
				eventStartTime = time.Date(
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

			deadline := eventStartTime.Add(-time.Duration(booking.CancellationDeadlineHours) * time.Hour)
			if now.After(deadline) {
				u.logger.WarnContext(ctx, "booking cancellation rejected: deadline passed",
					slog.String("booking_id", bookingID.String()),
					slog.Time("event_start_time", eventStartTime),
					slog.Time("deadline", deadline),
					slog.Time("attempted_at", now),
				)
				return domain.ErrCancellationDeadlinePassed
			}

			// Check tickets - cannot cancel if any ticket is used.
			tickets, err := tx.TicketRepository().FindByBookingID(ctx, booking.ID)
			if err != nil {
				return err
			}
			for _, t := range tickets {
				if t.Status == domain.TicketStatusUsed {
					return domain.ErrTicketAlreadyUsed
				}
			}

			// Calculate eligible refund amount based on snapshotted policy.
			if booking.Total.IsZero() || !booking.Total.IsPositive() {
				refundAmount = decimal.Zero
				isFreeOrNoRefund = true
			} else {
				switch strings.ToUpper(strings.TrimSpace(booking.RefundPolicy)) {
				case domain.RefundPolicyFull:
					refundAmount = booking.Total
				case domain.RefundPolicyPartial:
					if booking.RefundPercentage <= 0 {
						refundAmount = decimal.Zero
						isFreeOrNoRefund = true
					} else {
						pct := decimal.NewFromInt(int64(booking.RefundPercentage))
						refundAmount = booking.Total.Mul(pct).Div(decimal.NewFromInt(100)).RoundBank(2)
						if refundAmount.IsZero() || !refundAmount.IsPositive() {
							refundAmount = decimal.Zero
							isFreeOrNoRefund = true
						}
					}
				case domain.RefundPolicyNoRefund:
					refundAmount = decimal.Zero
					isFreeOrNoRefund = true
				default:
					refundAmount = decimal.Zero
					isFreeOrNoRefund = true
				}
			}

			if isFreeOrNoRefund {
				// Immediate cancellation without gateway call.
				if err := tx.BookingRepository().UpdateStatus(ctx, booking.ID, domain.BookingStatusCancelled); err != nil {
					return err
				}
				if err := tx.TicketRepository().UpdateStatusByBookingID(ctx, booking.ID, domain.TicketStatusCancelled); err != nil {
					return err
				}
				return u.restoreInventory(ctx, tx, booking.ID)
			}

			// Paid booking: locate successful payment.
			payments, err := tx.PaymentRepository().FindByBookingID(ctx, booking.ID.String())
			if err != nil {
				return err
			}
			for i := range payments {
				if payments[i].Status == domain.PaymentStatusSuccess {
					successfulPayment = &payments[i]
					break
				}
			}
			if successfulPayment == nil || successfulPayment.TransactionID == nil || *successfulPayment.TransactionID == "" {
				u.logger.ErrorContext(ctx, "successful payment record missing for paid booking cancellation",
					slog.String("booking_id", bookingID.String()),
				)
				return domain.ErrPaymentNotFound
			}

			paymentUUID, err = uuid.Parse(successfulPayment.ID)
			if err != nil {
				return fmt.Errorf("invalid payment id format: %w", err)
			}

			// Check existing refund records for this booking.
			existingRefunds, err := tx.RefundRepository().FindByBookingID(ctx, booking.ID)
			if err != nil {
				return err
			}
			for i := range existingRefunds {
				r := &existingRefunds[i]
				if r.Status == domain.RefundStatusSuccess || (r.Status == domain.RefundStatusPending && r.GatewayRefundID != nil && *r.GatewayRefundID != "") {
					// Gateway refund is already recorded locally from an earlier accepted attempt,
					// but booking cancellation was interrupted. Reuse this refund to complete cancellation.
					pendingRefund = r
					break
				}
				if r.Status == domain.RefundStatusPending && time.Since(r.CreatedAt) < 2*time.Minute {
					return domain.ErrRefundInProgress
				}
			}

			if pendingRefund == nil {
				// Create PENDING refund record before invoking gateway outside transaction.
				pendingRefund = &domain.Refund{
					ID:        uuid.New(),
					BookingID: booking.ID,
					PaymentID: paymentUUID,
					Amount:    refundAmount,
					Status:    domain.RefundStatusPending,
					Reason:    "CUSTOMER_CANCELLATION",
					CreatedAt: now,
					UpdatedAt: now,
				}

				if err := tx.RefundRepository().Create(ctx, pendingRefund); err != nil {
					return err
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	if isFreeOrNoRefund {
		u.logger.InfoContext(ctx, "booking_cancelled_successfully",
			slog.String("booking_id", booking.ID.String()),
			slog.String("booking_number", booking.BookingNumber),
			slog.String("refund_policy", booking.RefundPolicy),
			slog.String("refund_amount", decimal.Zero.String()),
		)

		return &CancelBookingOutput{
			BookingID:     booking.ID,
			BookingNumber: booking.BookingNumber,
			Status:        domain.BookingStatusCancelled,
			RefundAmount:  decimal.Zero,
			RefundStatus:  "NO_REFUND",
			CancelledAt:   now,
		}, nil
	}

	// Gateway execution outside DB transaction.
	if u.refundProvider == nil {
		u.logger.ErrorContext(ctx, "refund provider not configured")
		return nil, errors.New("refund provider not configured")
	}

	gatewayPaymentID := *successfulPayment.TransactionID
	idempotencyKey := pendingRefund.ID.String()

	var refundResult *domain.RefundResult

	// 1. If pendingRefund already has a GatewayRefundID (e.g. recovered from previous attempt),
	// use it directly.
	if pendingRefund.GatewayRefundID != nil && *pendingRefund.GatewayRefundID != "" {
		refundResult = &domain.RefundResult{
			RefundID: *pendingRefund.GatewayRefundID,
			Amount:   pendingRefund.Amount,
			Status:   pendingRefund.Status,
			Receipt:  booking.ID.String(),
		}
	}

	// 2. Preflight Gateway Reconciliation: Check if a refund was already accepted by the gateway
	// for this booking before issuing a new gateway refund call.
	if refundResult == nil {
		gatewayRefunds, fetchErr := u.refundProvider.FetchRefunds(ctx, gatewayPaymentID)
		if fetchErr == nil && len(gatewayRefunds) > 0 {
			refundResult = findMatchingGatewayRefund(gatewayRefunds, booking.ID, &pendingRefund.ID, refundAmount)
			if refundResult != nil {
				u.logger.InfoContext(ctx, "reconciled_existing_gateway_refund_preflight",
					slog.String("booking_id", booking.ID.String()),
					slog.String("gateway_refund_id", refundResult.RefundID),
					slog.String("status", refundResult.Status),
				)
			}
		}
	}

	// 3. Issue Gateway Refund if no existing gateway refund was discovered.
	if refundResult == nil {
		res, gatewayErr := u.refundProvider.Refund(ctx, gatewayPaymentID, refundAmount, "INR", idempotencyKey)
		if gatewayErr == nil && res != nil && (res.Status == domain.RefundStatusSuccess || res.Status == domain.RefundStatusPending) {
			refundResult = res
		} else {
			// 4. Ambiguous Timeout Recovery / Post-failure Reconciliation:
			// If the gateway returned an error (such as network timeout / 5xx) or non-success status,
			// query FetchRefunds to discover if the gateway actually accepted and processed the refund.
			reconciledRefunds, fetchErr := u.refundProvider.FetchRefunds(ctx, gatewayPaymentID)
			if fetchErr == nil && len(reconciledRefunds) > 0 {
				matched := findMatchingGatewayRefund(reconciledRefunds, booking.ID, &pendingRefund.ID, refundAmount)
				if matched != nil {
					u.logger.InfoContext(ctx, "recovered_refund_from_gateway_reconciliation",
						slog.String("booking_id", booking.ID.String()),
						slog.String("gateway_refund_id", matched.RefundID),
						slog.String("status", matched.Status),
					)
					refundResult = matched
				}
			}

			// If still no refund found, this is a confirmed gateway failure.
			if refundResult == nil {
				_ = u.transactionManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
					pendingRefund.Status = domain.RefundStatusFailed
					pendingRefund.UpdatedAt = time.Now()
					return tx.RefundRepository().Update(ctx, pendingRefund)
				})

				u.logger.ErrorContext(ctx, "booking_cancellation_gateway_refund_failed",
					slog.String("booking_id", booking.ID.String()),
					slog.String("refund_id", pendingRefund.ID.String()),
					slog.String("gateway_payment_id", gatewayPaymentID),
					slog.Any("error", gatewayErr),
				)

				if gatewayErr != nil {
					return nil, fmt.Errorf("%w: %v", domain.ErrGatewayRefundFailed, gatewayErr)
				}
				return nil, domain.ErrGatewayRefundFailed
			}
		}
	}

	// Phase 2: Gateway confirmed refund. Mark refund, booking CANCELLED, tickets CANCELLED, release inventory.
	completionTime := time.Now()
	err = u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			b, err := tx.BookingRepository().FindByIDForUpdate(ctx, booking.ID)
			if err != nil {
				return err
			}
			if b.Status == domain.BookingStatusCancelled {
				return nil
			}

			pendingRefund.Status = refundResult.Status
			pendingRefund.GatewayRefundID = &refundResult.RefundID
			pendingRefund.ProcessedAt = &completionTime
			pendingRefund.UpdatedAt = completionTime
			if err := tx.RefundRepository().Update(ctx, pendingRefund); err != nil {
				return err
			}

			if err := tx.BookingRepository().UpdateStatus(ctx, b.ID, domain.BookingStatusCancelled); err != nil {
				return err
			}

			if err := tx.TicketRepository().UpdateStatusByBookingID(ctx, b.ID, domain.TicketStatusCancelled); err != nil {
				return err
			}

			return u.restoreInventory(ctx, tx, b.ID)
		},
	)
	if err != nil {
		u.logger.ErrorContext(ctx, "booking_cancellation_finalization_failed",
			slog.String("booking_id", booking.ID.String()),
			slog.String("refund_id", pendingRefund.ID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "booking_cancelled_with_refund_success",
		slog.String("booking_id", booking.ID.String()),
		slog.String("booking_number", booking.BookingNumber),
		slog.String("refund_id", pendingRefund.ID.String()),
		slog.String("gateway_refund_id", refundResult.RefundID),
		slog.String("refund_amount", refundAmount.String()),
		slog.String("refund_status", refundResult.Status),
		slog.String("refund_policy", booking.RefundPolicy),
	)

	return &CancelBookingOutput{
		BookingID:     booking.ID,
		BookingNumber: booking.BookingNumber,
		Status:        domain.BookingStatusCancelled,
		RefundAmount:  refundAmount,
		RefundStatus:  refundResult.Status,
		RefundID:      refundResult.RefundID,
		CancelledAt:   completionTime,
	}, nil
}

func (u *BookingUsecase) restoreInventory(
	ctx context.Context,
	tx domain.TransactionRepositories,
	bookingID uuid.UUID,
) error {
	items, err := tx.BookingRepository().FindItemsByBookingID(ctx, bookingID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.TicketTypeID != nil && *item.TicketTypeID != uuid.Nil {
			if err := tx.TicketTypeRepository().ReleaseReservedQuantity(
				ctx,
				*item.TicketTypeID,
				item.Quantity,
			); err != nil {
				u.logger.ErrorContext(ctx, "failed to release ticket quantity on cancellation",
					slog.String("booking_id", bookingID.String()),
					slog.String("ticket_type_id", item.TicketTypeID.String()),
					slog.Int("quantity", item.Quantity),
					slog.Any("error", err),
				)
				return err
			}
		}

		if item.SeatID != nil && *item.SeatID != uuid.Nil {
			if err := tx.SeatRepository().ReleaseBookedSeat(
				ctx,
				*item.SeatID,
			); err != nil {
				u.logger.ErrorContext(ctx, "failed to release booked seat on cancellation",
					slog.String("booking_id", bookingID.String()),
					slog.String("seat_id", item.SeatID.String()),
					slog.Any("error", err),
				)
				return err
			}
		}
	}

	return nil
}

func findMatchingGatewayRefund(
	refunds []domain.RefundResult,
	bookingID uuid.UUID,
	pendingRefundID *uuid.UUID,
	expectedAmount decimal.Decimal,
) *domain.RefundResult {
	bookingIDStr := bookingID.String()
	var pendingRefundIDStr string
	if pendingRefundID != nil {
		pendingRefundIDStr = pendingRefundID.String()
	}

	// First pass: exact receipt match against booking ID or pending refund ID.
	for i := range refunds {
		r := &refunds[i]
		if r.Status != domain.RefundStatusSuccess && r.Status != domain.RefundStatusPending {
			continue
		}
		if r.Receipt != "" && (r.Receipt == bookingIDStr || (pendingRefundIDStr != "" && r.Receipt == pendingRefundIDStr)) {
			copied := *r
			return &copied
		}
	}

	// Second pass: amount match when exactly one active gateway refund exists.
	var amountMatch *domain.RefundResult
	count := 0
	for i := range refunds {
		r := &refunds[i]
		if r.Status != domain.RefundStatusSuccess && r.Status != domain.RefundStatusPending {
			continue
		}
		if r.Amount.Equal(expectedAmount) {
			copied := *r
			amountMatch = &copied
			count++
		}
	}
	if count == 1 {
		return amountMatch
	}

	return nil
}

// GetBookingInvoice retrieves the authoritative booking receipt snapshot for an authorized user.
// If the booking is confirmed but the historical receipt snapshot does not exist, it self-heals
// by creating the invoice snapshot.
func (u *BookingUsecase) GetBookingInvoice(
	ctx context.Context,
	userID uuid.UUID,
	bookingID uuid.UUID,
) (*domain.InvoiceSummaryDTO, error) {
	if userID == uuid.Nil || bookingID == uuid.Nil {
		return nil, domain.ErrBookingNotFound
	}

	var dto *domain.InvoiceSummaryDTO

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			booking, _, err := tx.BookingRepository().FindWithItemsByID(ctx, bookingID)
			if err != nil {
				return err
			}

			if booking.UserID != userID {
				u.logger.WarnContext(ctx, "booking receipt access denied (user mismatch)",
					slog.String("user_id", userID.String()),
					slog.String("booking_id", bookingID.String()),
				)
				return domain.ErrBookingNotFound
			}

			invoice, invItems, err := tx.InvoiceRepository().FindByBookingIDWithItems(ctx, bookingID)
			if err != nil {
				if errors.Is(err, domain.ErrInvoiceNotFound) {
					// Check eligibility for historical self-healing
					if booking.Status != domain.BookingStatusConfirmed {
						return domain.ErrInvoiceNotFound
					}

					payments, pErr := tx.PaymentRepository().FindByBookingID(ctx, bookingID.String())
					if pErr != nil {
						return pErr
					}

					var successfulPayment *domain.Payment
					for i := range payments {
						if payments[i].Status == domain.PaymentStatusSuccess {
							successfulPayment = &payments[i]
							break
						}
					}

					if successfulPayment == nil {
						return domain.ErrInvoiceNotFound
					}

					// Self-heal snapshot
					healedInv, healErr := u.createReceiptSnapshot(ctx, tx, booking, successfulPayment, time.Now())
					if healErr != nil {
						return healErr
					}
					invoice = healedInv
					invItems = healedInv.Items
				} else {
					return err
				}
			}

			// Retrieve payment details
			payMethod := "ONLINE"
			var txID *string
			var orderID *string
			paidAt := invoice.IssuedAt

			payments, _ := tx.PaymentRepository().FindByBookingID(ctx, bookingID.String())
			for _, p := range payments {
				if p.Status == domain.PaymentStatusSuccess {
					payMethod = string(p.Method)
					txID = p.TransactionID
					orderID = p.ProviderOrderID
					if p.PaidAt != nil {
						paidAt = *p.PaidAt
					}
					break
				}
			}

			d := &domain.InvoiceSummaryDTO{
				ID:            invoice.ID,
				ReceiptNumber: invoice.InvoiceNumber,
				DocumentType:  "Booking Receipt",
				Status:        invoice.Status,
				IssuedAt:      invoice.IssuedAt,
				Items:         invItems,
			}

			d.Booking.ID = booking.ID
			d.Booking.BookingNumber = booking.BookingNumber

			d.Customer.Name = invoice.CustomerName
			d.Customer.Email = invoice.CustomerEmail
			d.Customer.Phone = invoice.CustomerPhone

			d.Event.Title = invoice.EventTitle
			d.Event.Date = invoice.EventDate
			d.Event.Time = invoice.EventTimeStr
			d.Event.VenueName = invoice.VenueName
			d.Event.VenueAddress = invoice.VenueAddress
			d.Event.IsOnline = invoice.IsOnline
			d.Event.OnlineDetails = invoice.OnlineDetails

			d.Issuer.IssuerName = invoice.IssuerName
			d.Issuer.OrganizerName = invoice.OrganizerName
			d.Issuer.Disclaimer = "This document is an official booking and payment receipt confirming your reservation and payment. Not a GST tax invoice."

			d.Pricing.Subtotal = invoice.Subtotal
			d.Pricing.PlatformFee = invoice.PlatformFee
			d.Pricing.TaxAmount = invoice.TaxAmount
			d.Pricing.DiscountAmount = invoice.DiscountAmount
			d.Pricing.TotalAmount = invoice.TotalAmount
			d.Pricing.PaidAmount = invoice.PaidAmount
			d.Pricing.Currency = invoice.Currency

			d.Payment.Method = payMethod
			d.Payment.TransactionID = txID
			d.Payment.ProviderOrderID = orderID
			d.Payment.PaidAt = paidAt

			dto = d
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// GetBookingInvoicePDF generates the PDF bytes and recommended filename for a booking receipt.
func (u *BookingUsecase) GetBookingInvoicePDF(
	ctx context.Context,
	userID uuid.UUID,
	bookingID uuid.UUID,
) ([]byte, string, error) {
	dto, err := u.GetBookingInvoice(ctx, userID, bookingID)
	if err != nil {
		return nil, "", err
	}

	pdfBytes, err := u.receiptPDFService.GenerateReceiptPDF(*dto)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to generate receipt PDF",
			slog.String("booking_id", bookingID.String()),
			slog.String("receipt_number", dto.ReceiptNumber),
			slog.Any("error", err),
		)
		return nil, "", fmt.Errorf("failed to generate receipt PDF: %w", err)
	}

	filename := fmt.Sprintf("receipt-%s.pdf", dto.ReceiptNumber)
	return pdfBytes, filename, nil
}

func (u *BookingUsecase) createReceiptSnapshot(
	ctx context.Context,
	tx domain.TransactionRepositories,
	booking *domain.Booking,
	payment *domain.Payment,
	now time.Time,
) (*domain.Invoice, error) {
	if booking == nil || payment == nil {
		return nil, errors.New("booking and payment are required to create receipt snapshot")
	}

	existing, err := tx.InvoiceRepository().FindByBookingID(ctx, booking.ID)
	if err == nil && existing != nil {
		return existing, nil
	}

	paymentUUID, err := uuid.Parse(payment.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid payment id %s: %w", payment.ID, err)
	}

	receiptNumber, err := tx.InvoiceRepository().NextReceiptNumber(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("failed to generate receipt number: %w", err)
	}

	customerName := "Customer"
	customerEmail := ""
	var customerPhone *string
	user, userErr := tx.UserRepository().FindByID(booking.UserID)
	if userErr == nil && user != nil {
		if strings.TrimSpace(user.FullName) != "" {
			customerName = strings.TrimSpace(user.FullName)
		} else if strings.TrimSpace(user.Email) != "" {
			customerName = strings.TrimSpace(user.Email)
		}
		customerEmail = strings.TrimSpace(user.Email)
		if strings.TrimSpace(user.Phone) != "" {
			phone := strings.TrimSpace(user.Phone)
			customerPhone = &phone
		}
	}

	eventTitle := "Event"
	isOnline := false
	var onlineDetails *string
	var venueName *string
	var venueAddress *string
	var organizerName *string

	event, eventErr := tx.EventRepository().FindByID(booking.EventID)
	if eventErr == nil && event != nil {
		if strings.TrimSpace(event.Title) != "" {
			eventTitle = strings.TrimSpace(event.Title)
		}
		isOnline = event.EventType == domain.EventTypeOnline || event.EventType == domain.EventTypeHybrid
		if event.OnlineURL != "" {
			onlineDetails = &event.OnlineURL
		}

		if event.EventType != domain.EventTypeOnline && event.VenueID != nil && *event.VenueID != uuid.Nil {
			venue, vErr := tx.VenueRepository().FindByID(*event.VenueID)
			if vErr == nil && venue != nil {
				vName := strings.TrimSpace(venue.Name)
				if vName != "" {
					venueName = &vName
				}
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
				if len(addrParts) > 0 {
					vAddr := strings.Join(addrParts, ", ")
					venueAddress = &vAddr
				}
			}
		}

		if event.OrganizerID != uuid.Nil {
			profile, pErr := tx.OrganizerProfileRepository().FindByOrganizerID(event.OrganizerID)
			if pErr == nil && profile != nil && strings.TrimSpace(profile.BusinessName) != "" {
				bName := strings.TrimSpace(profile.BusinessName)
				organizerName = &bName
			}
		}
	}

	eventDate := now.Format("2006-01-02")
	var eventTimeStr *string
	schedule, sErr := tx.EventScheduleRepository().FindByEventID(booking.EventID)
	if sErr == nil && schedule != nil && !schedule.EventDate.IsZero() {
		eventDate = schedule.EventDate.Format("2006-01-02")
		if !schedule.StartTime.IsZero() {
			tStr := schedule.StartTime.Format("03:04 PM")
			if !schedule.EndTime.IsZero() {
				tStr = fmt.Sprintf("%s - %s", tStr, schedule.EndTime.Format("03:04 PM"))
			}
			eventTimeStr = &tStr
		} else if schedule.IsAllDay {
			allDay := "All Day"
			eventTimeStr = &allDay
		}
	}

	bookingItems, bErr := tx.BookingRepository().FindItemsByBookingID(ctx, booking.ID)
	if bErr != nil {
		return nil, fmt.Errorf("failed to fetch booking items: %w", bErr)
	}

	var seatIDs []uuid.UUID
	for _, it := range bookingItems {
		if it.SeatID != nil && *it.SeatID != uuid.Nil {
			seatIDs = append(seatIDs, *it.SeatID)
		}
	}

	var seatDetailsMap map[uuid.UUID]domain.SeatDetail
	if len(seatIDs) > 0 {
		seatDetailsMap, _ = tx.SeatRepository().FindDetailsBySeatIDs(ctx, seatIDs)
	}

	ticketTypeCache := make(map[uuid.UUID]string)

	invoiceID := uuid.New()
	invoiceItems := make([]domain.InvoiceItem, 0, len(bookingItems))

	for _, it := range bookingItems {
		categoryName := "Admission"
		if it.TicketTypeID != nil && *it.TicketTypeID != uuid.Nil {
			if name, ok := ticketTypeCache[*it.TicketTypeID]; ok {
				categoryName = name
			} else {
				tt, ttErr := tx.TicketTypeRepository().FindByID(ctx, *it.TicketTypeID)
				if ttErr == nil && tt != nil && strings.TrimSpace(tt.Name) != "" {
					categoryName = strings.TrimSpace(tt.Name)
					ticketTypeCache[*it.TicketTypeID] = categoryName
				}
			}
		}

		var seatLabel *string
		if it.SeatID != nil && seatDetailsMap != nil {
			if sd, ok := seatDetailsMap[*it.SeatID]; ok && strings.TrimSpace(sd.Label) != "" {
				label := strings.TrimSpace(sd.Label)
				seatLabel = &label
			}
		}

		itemTotalPrice := it.UnitPrice.Mul(decimal.NewFromInt(int64(it.Quantity)))
		itemID := it.ID
		invoiceItems = append(invoiceItems, domain.InvoiceItem{
			ID:            uuid.New(),
			InvoiceID:     invoiceID,
			BookingItemID: &itemID,
			CategoryName:  categoryName,
			SeatLabel:     seatLabel,
			Quantity:      it.Quantity,
			UnitPrice:     it.UnitPrice,
			TotalPrice:    itemTotalPrice,
			CreatedAt:     now,
		})
	}

	if len(invoiceItems) == 0 {
		invoiceItems = append(invoiceItems, domain.InvoiceItem{
			ID:           uuid.New(),
			InvoiceID:    invoiceID,
			CategoryName: "General Admission",
			Quantity:     1,
			UnitPrice:    booking.Total,
			TotalPrice:   booking.Total,
			CreatedAt:    now,
		})
	}

	invoice := &domain.Invoice{
		ID:             invoiceID,
		InvoiceNumber:  receiptNumber,
		BookingID:      booking.ID,
		PaymentID:      paymentUUID,
		UserID:         booking.UserID,
		EventID:        booking.EventID,
		Status:         domain.InvoiceStatusIssued,
		Currency:       "INR",
		Subtotal:       booking.Subtotal,
		PlatformFee:    booking.PlatformFee,
		TaxAmount:      booking.Tax,
		DiscountAmount: booking.Discount,
		TotalAmount:    booking.Total,
		PaidAmount:     payment.Amount,
		CustomerName:   customerName,
		CustomerEmail:  customerEmail,
		CustomerPhone:  customerPhone,
		EventTitle:     eventTitle,
		EventDate:      eventDate,
		EventTimeStr:   eventTimeStr,
		VenueName:      venueName,
		VenueAddress:   venueAddress,
		IsOnline:       isOnline,
		OnlineDetails:  onlineDetails,
		IssuerName:     "EventHub Bookings",
		OrganizerName:  organizerName,
		IssuedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
		Items:          invoiceItems,
	}

	if err := tx.InvoiceRepository().Create(ctx, invoice, invoiceItems); err != nil {
		if errors.Is(err, domain.ErrInvoiceAlreadyExists) {
			existing, getErr := tx.InvoiceRepository().FindByBookingID(ctx, booking.ID)
			if getErr == nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("failed to save invoice: %w", err)
	}

	return invoice, nil
}
