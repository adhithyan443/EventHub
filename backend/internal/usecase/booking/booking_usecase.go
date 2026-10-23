package booking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
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
