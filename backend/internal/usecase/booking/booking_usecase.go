package booking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
	"github.com/google/uuid"
)

type BookingUsecase struct {
	transactionManager domain.TransactionManager
	logger             *slog.Logger
}

func NewBookingUsecase(
	transactionManager domain.TransactionManager,
	logger *slog.Logger,
) *BookingUsecase {
	return &BookingUsecase{
		transactionManager: transactionManager,
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

				Status:    domain.BookingStatusPending,
				CreatedAt: now,
				UpdatedAt: now,
			}

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

	u.logger.InfoContext(ctx, "customer bookings retrieved",
		slog.String("user_id", userID.String()),
		slog.Int("booking_count", len(bookings)),
	)

	return bookings, nil
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
