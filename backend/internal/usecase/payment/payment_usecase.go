package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	bookingusecase "github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PaymentUsecase struct {
	transactionManager domain.TransactionManager
	bookingUsecase     *bookingusecase.BookingUsecase
	provider           PaymentProvider
	logger             *slog.Logger
}

func NewPaymentUsecase(
	transactionManager domain.TransactionManager,
	bookingUsecase *bookingusecase.BookingUsecase,
	provider PaymentProvider,
	logger *slog.Logger,
) *PaymentUsecase {
	return &PaymentUsecase{
		transactionManager: transactionManager,
		bookingUsecase:     bookingUsecase,
		provider:           provider,
		logger:             logger,
	}
}

type CreatePaymentRequest struct {
	CheckoutID uuid.UUID
	UserID     uuid.UUID
	Method     domain.PaymentMethod
}

func (u *PaymentUsecase) CreatePayment(
	ctx context.Context,
	req CreatePaymentRequest,
) (*domain.Payment, error) {
	if req.CheckoutID == uuid.Nil || req.UserID == uuid.Nil {
		return nil, domain.ErrInvalidPayment
	}

	if !isSupportedPaymentMethod(req.Method) {
		return nil, domain.ErrUnsupportedPaymentMethod
	}

	// Prepare the booking before initiating payment.
	booking, err := u.bookingUsecase.CreateBooking(
		ctx,
		bookingusecase.CreateBookingRequest{
			CheckoutID: req.CheckoutID,
			UserID:     req.UserID,
		},
	)
	if err != nil {
		u.logger.ErrorContext(
			ctx,
			"failed to prepare booking for payment",
			slog.String("checkout_id", req.CheckoutID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	if booking.Status == domain.BookingStatusConfirmed {
		return nil, domain.ErrPaymentAlreadyComplete
	}

	if booking.Status != domain.BookingStatusPending {
		return nil, domain.ErrReservationExpired
	}

	var (
		checkout         *domain.Checkout
		candidatePending *domain.Payment
	)

	// Phase 1 DB Transaction: Lock booking row, inspect attempts, reserve PENDING slot.
	// Never hold database transactions or locks across external gateway calls.
	err = u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			currentBooking, findBookingErr := tx.BookingRepository().
				FindByIDForUpdate(ctx, booking.ID)
			if findBookingErr != nil {
				return findBookingErr
			}

			if currentBooking.Status == domain.BookingStatusConfirmed {
				return domain.ErrPaymentAlreadyComplete
			}
			if currentBooking.Status != domain.BookingStatusPending {
				return domain.ErrReservationExpired
			}

			var validationErr error
			checkout, _, validationErr = u.validateCheckout(
				ctx,
				tx,
				req.CheckoutID,
				req.UserID,
			)
			if validationErr != nil {
				return validationErr
			}

			if checkout.TotalAmount.Cmp(booking.Total) != 0 {
				return domain.ErrPaymentAmountMismatch
			}

			existingPayments, findErr := tx.PaymentRepository().
				FindByBookingID(ctx, booking.ID.String())
			if findErr != nil {
				return findErr
			}

			for i := range existingPayments {
				existing := &existingPayments[i]
				switch existing.Status {
				case domain.PaymentStatusSuccess:
					return domain.ErrPaymentAlreadyComplete
				case domain.PaymentStatusRefundRequired:
					return domain.ErrPaymentRefundPending
				case domain.PaymentStatusPending:
					candidatePending = existing
				}
			}

			if candidatePending != nil {
				return nil
			}

			newPayment := &domain.Payment{
				BookingID: booking.ID.String(),
				Provider:  domain.PaymentProviderRazorpay,
				Method:    req.Method,
				Amount:    checkout.TotalAmount,
				Status:    domain.PaymentStatusPending,
			}

			if err := tx.PaymentRepository().Create(ctx, newPayment); err != nil {
				return err
			}

			candidatePending = newPayment
			return nil
		},
	)
	if err != nil {
		u.logger.ErrorContext(
			ctx,
			"failed to prepare payment attempt",
			slog.String("checkout_id", req.CheckoutID.String()),
			slog.String("booking_id", booking.ID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	// Phase 2: Gateway Interrogation & Order Creation (Outside DB Transaction)
	// Case A: Candidate payment has no provider_order_id. Reconcile by receipt!
	if candidatePending.ProviderOrderID == nil || *candidatePending.ProviderOrderID == "" {
		existingOrder, reconcileErr := u.provider.FindOrderByReceipt(
			ctx,
			candidatePending.ID,
			candidatePending.Amount,
			"INR",
		)
		if reconcileErr != nil {
			u.logger.WarnContext(
				ctx,
				"order reconciliation by receipt failed; failing closed to avoid duplicate orders",
				slog.String("payment_id", candidatePending.ID),
				slog.Any("error", reconcileErr),
			)
			return nil, domain.ErrPaymentInProgress
		}

		if existingOrder != nil && existingOrder.ID != "" {
			// Recovered order from gateway!
			candidatePending.ProviderOrderID = &existingOrder.ID
			if err := u.persistProviderOrderID(ctx, candidatePending.ID, existingOrder.ID); err != nil {
				u.logger.ErrorContext(
					ctx,
					"failed to persist recovered provider order ID",
					slog.String("payment_id", candidatePending.ID),
					slog.String("provider_order_id", existingOrder.ID),
					slog.Any("error", err),
				)
				return nil, err
			}
		} else {
			// Never create a new payment attempt solely because receipt lookup returned zero results.
			// Retry Order.Create with the SAME payment.ID as receipt to guarantee idempotency.
			orderID, initErr := u.provider.Initiate(
				ctx,
				candidatePending.Amount,
				"INR",
				candidatePending.ID,
			)
			if initErr != nil {
				u.logger.ErrorContext(
					ctx,
					"payment gateway order creation failed; keeping pending record for future reconciliation",
					slog.String("payment_id", candidatePending.ID),
					slog.String("booking_id", candidatePending.BookingID),
					slog.Any("error", initErr),
				)
				return nil, domain.ErrPaymentInProgress
			}

			candidatePending.ProviderOrderID = &orderID
			if err := u.persistProviderOrderID(ctx, candidatePending.ID, orderID); err != nil {
				u.logger.ErrorContext(
					ctx,
					"failed to persist provider order ID",
					slog.String("payment_id", candidatePending.ID),
					slog.String("provider_order_id", orderID),
					slog.Any("error", err),
				)
				return nil, err
			}

			u.logger.InfoContext(
				ctx,
				"payment order created",
				slog.String("payment_id", candidatePending.ID),
				slog.String("booking_id", candidatePending.BookingID),
				slog.String("provider_order_id", orderID),
			)
			return candidatePending, nil
		}
	}

	// Case B: Candidate payment has a provider_order_id. Inspect order status with gateway.
	orderStatus, checkErr := u.provider.CheckOrderStatus(
		ctx,
		*candidatePending.ProviderOrderID,
		candidatePending.Amount,
		"INR",
	)
	if checkErr != nil {
		u.logger.WarnContext(
			ctx,
			"gateway order status check failed; failing closed without altering state",
			slog.String("payment_id", candidatePending.ID),
			slog.String("provider_order_id", *candidatePending.ProviderOrderID),
			slog.Any("error", checkErr),
		)
		return nil, domain.ErrPaymentProviderUnavailable
	}

	switch orderStatus.Status {
	case OrderStatusUsable:
		// Option C Rule 1: Reuse existing pending payment and usable order.
		u.logger.InfoContext(
			ctx,
			"reusing existing pending payment order",
			slog.String("payment_id", candidatePending.ID),
			slog.String("provider_order_id", *candidatePending.ProviderOrderID),
		)
		return candidatePending, nil

	case OrderStatusInFlight:
		// Payment attempt is currently in processing (authorized / awaiting capture).
		u.logger.InfoContext(
			ctx,
			"payment attempt actively in-flight at gateway",
			slog.String("payment_id", candidatePending.ID),
			slog.String("provider_order_id", *candidatePending.ProviderOrderID),
		)
		return nil, domain.ErrPaymentInProgress

	case OrderStatusPaid:
		// Order was already paid at gateway.
		u.logger.WarnContext(
			ctx,
			"payment order already paid at gateway; blocking duplicate attempt",
			slog.String("payment_id", candidatePending.ID),
			slog.String("provider_order_id", *candidatePending.ProviderOrderID),
		)
		return nil, domain.ErrPaymentAlreadyComplete

	case OrderStatusUnusable:
		// Option C Rule 2: Order is definitively unusable (404, expired, cancelled, amount mismatch).
		// Safely retire previous attempt and create a new attempt.
		u.logger.InfoContext(
			ctx,
			"retiring unusable payment order and creating fresh attempt",
			slog.String("old_payment_id", candidatePending.ID),
			slog.String("provider_order_id", *candidatePending.ProviderOrderID),
		)

		var newPayment *domain.Payment
		newPayment, err = u.retireAndCreatePayment(
			ctx,
			candidatePending.ID,
			booking.ID,
			req.Method,
			checkout.TotalAmount,
		)
		if err != nil {
			return nil, err
		}

		newOrderID, initErr := u.provider.Initiate(
			ctx,
			newPayment.Amount,
			"INR",
			newPayment.ID,
		)
		if initErr != nil {
			u.logger.ErrorContext(
				ctx,
				"failed to initiate gateway order for replacement payment attempt",
				slog.String("payment_id", newPayment.ID),
				slog.Any("error", initErr),
			)
			return nil, domain.ErrPaymentInProgress
		}

		newPayment.ProviderOrderID = &newOrderID
		if err := u.persistProviderOrderID(ctx, newPayment.ID, newOrderID); err != nil {
			u.logger.ErrorContext(
				ctx,
				"failed to persist replacement provider order ID",
				slog.String("payment_id", newPayment.ID),
				slog.String("provider_order_id", newOrderID),
				slog.Any("error", err),
			)
			return nil, err
		}

		u.logger.InfoContext(
			ctx,
			"replacement payment order created",
			slog.String("payment_id", newPayment.ID),
			slog.String("provider_order_id", newOrderID),
		)
		return newPayment, nil

	default:
		return nil, domain.ErrPaymentProviderUnavailable
	}
}

func (u *PaymentUsecase) persistProviderOrderID(
	ctx context.Context,
	paymentID string,
	providerOrderID string,
) error {
	return u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			if err := tx.PaymentRepository().UpdateProviderOrderID(
				ctx,
				paymentID,
				providerOrderID,
			); err != nil {
				return err
			}

			// Reload and reconcile the persisted payment to confirm state
			reloaded, err := tx.PaymentRepository().FindByID(ctx, paymentID)
			if err != nil {
				return err
			}

			if reloaded.ProviderOrderID == nil || *reloaded.ProviderOrderID != providerOrderID {
				return fmt.Errorf("provider order ID reconciliation mismatch: expected %s, got %v", providerOrderID, reloaded.ProviderOrderID)
			}

			return nil
		},
	)
}

func (u *PaymentUsecase) retireAndCreatePayment(
	ctx context.Context,
	oldPaymentID string,
	bookingID uuid.UUID,
	method domain.PaymentMethod,
	amount decimal.Decimal,
) (*domain.Payment, error) {
	var newPayment *domain.Payment

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			currentBooking, findErr := tx.BookingRepository().FindByIDForUpdate(ctx, bookingID)
			if findErr != nil {
				return findErr
			}

			if currentBooking.Status == domain.BookingStatusConfirmed {
				return domain.ErrPaymentAlreadyComplete
			}
			if currentBooking.Status != domain.BookingStatusPending {
				return domain.ErrReservationExpired
			}

			if err := tx.PaymentRepository().MarkFailed(ctx, oldPaymentID); err != nil {
				return err
			}

			newPayment = &domain.Payment{
				BookingID: bookingID.String(),
				Provider:  domain.PaymentProviderRazorpay,
				Method:    method,
				Amount:    amount,
				Status:    domain.PaymentStatusPending,
			}

			return tx.PaymentRepository().Create(ctx, newPayment)
		},
	)
	if err != nil {
		return nil, err
	}

	return newPayment, nil
}

type VerifyPaymentRequest struct {
	UserID            uuid.UUID
	ProviderOrderID   string
	RazorpayPaymentID string
	RazorpaySignature string
}

type VerifyPaymentResult struct {
	Payment *domain.Payment
	Booking *domain.Booking
}

func (u *PaymentUsecase) VerifyPayment(
	ctx context.Context,
	req VerifyPaymentRequest,
) (*VerifyPaymentResult, error) {
	if req.UserID == uuid.Nil ||
		strings.TrimSpace(req.ProviderOrderID) == "" ||
		strings.TrimSpace(req.RazorpayPaymentID) == "" ||
		strings.TrimSpace(req.RazorpaySignature) == "" {
		return nil, domain.ErrInvalidPayment
	}

	var (
		payment          *domain.Payment
		confirmedBooking *domain.Booking
	)

	// Step 1: Load payment and verify booking ownership.
	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			var findErr error

			payment, findErr = tx.PaymentRepository().
				FindByProviderOrderID(ctx, req.ProviderOrderID)
			if findErr != nil {
				return findErr
			}

			if payment.Provider != domain.PaymentProviderRazorpay ||
				payment.ProviderOrderID == nil ||
				*payment.ProviderOrderID != req.ProviderOrderID {
				return domain.ErrInvalidPayment
			}

			bookingID, parseErr := uuid.Parse(payment.BookingID)
			if parseErr != nil {
				return domain.ErrInvalidPayment
			}

			booking, findErr := tx.BookingRepository().FindByID(
				ctx,
				bookingID,
			)
			if findErr != nil {
				return findErr
			}

			if booking.UserID != req.UserID {
				return domain.ErrInvalidPayment
			}

			confirmedBooking = booking
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	// Step 2: Make identical successful verification requests idempotent.
	if payment.Status == domain.PaymentStatusSuccess {
		if payment.TransactionID != nil &&
			*payment.TransactionID == req.RazorpayPaymentID {
			return &VerifyPaymentResult{
				Payment: payment,
				Booking: confirmedBooking,
			}, nil
		}

		return nil, domain.ErrPaymentAlreadyComplete
	}

	if payment.Status == domain.PaymentStatusRefundRequired {
		return nil, domain.ErrReservationExpired
	}

	if payment.Status != domain.PaymentStatusPending {
		return nil, domain.ErrInvalidPayment
	}

	// Step 3: Verify the payment with Razorpay outside the DB transaction.
	verified, err := u.provider.Verify(
		ctx,
		req.ProviderOrderID,
		req.RazorpayPaymentID,
		req.RazorpaySignature,
		payment.Amount,
		"INR",
	)
	if err != nil {
		u.logger.ErrorContext(
			ctx,
			"payment verification could not be completed",
			slog.String("payment_id", payment.ID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if !verified {
		u.logger.WarnContext(
			ctx,
			"payment verification rejected",
			slog.String("payment_id", payment.ID),
		)
		return nil, domain.ErrInvalidPayment
	}

	// Step 4: Lock the payment row and atomically finalize the booking.
	now := time.Now()
	var isRefundRequired bool

	err = u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			current, findErr := tx.PaymentRepository().
				FindByProviderOrderIDForUpdate(
					ctx,
					req.ProviderOrderID,
				)
			if findErr != nil {
				return findErr
			}

			// Another request may have completed payment while Razorpay
			// verification was in progress.
			if current.Status == domain.PaymentStatusSuccess {
				if current.TransactionID == nil ||
					*current.TransactionID != req.RazorpayPaymentID {
					return domain.ErrPaymentAlreadyComplete
				}

				bookingID, parseErr := uuid.Parse(current.BookingID)
				if parseErr != nil {
					return domain.ErrInvalidPayment
				}

				booking, bookingErr := tx.BookingRepository().FindByID(
					ctx,
					bookingID,
				)
				if bookingErr != nil {
					return bookingErr
				}

				payment = current
				confirmedBooking = booking
				return nil
			}

			if current.Status == domain.PaymentStatusRefundRequired {
				if current.TransactionID != nil &&
					*current.TransactionID == req.RazorpayPaymentID {
					isRefundRequired = true
					payment = current
					return nil
				}

				return domain.ErrReservationExpired
			}

			if current.Status != domain.PaymentStatusPending ||
				current.Amount.Cmp(payment.Amount) != 0 {
				return domain.ErrInvalidPayment
			}

			bookingID, parseErr := uuid.Parse(current.BookingID)
			if parseErr != nil {
				return domain.ErrInvalidPayment
			}

			booking, findErr := tx.BookingRepository().FindByID(
				ctx,
				bookingID,
			)
			if findErr != nil {
				return findErr
			}

			if booking.UserID != req.UserID {
				return domain.ErrInvalidPayment
			}

			if booking.Status == domain.BookingStatusConfirmed {
				// Secondary payment verified for an already confirmed booking.
				if err := tx.PaymentRepository().MarkRefundRequired(
					ctx,
					current.ID,
					req.RazorpayPaymentID,
					now,
				); err != nil {
					return err
				}
				current.Status = domain.PaymentStatusRefundRequired
				current.TransactionID = &req.RazorpayPaymentID
				current.PaidAt = &now
				payment = current
				confirmedBooking = booking
				isRefundRequired = true
				return nil
			}

			if booking.Status != domain.BookingStatusPending {
				return domain.ErrInvalidPayment
			}

			var finalizeErr error

			isRefundRequired, finalizeErr = u.finalizeBookingPayment(
				ctx,
				tx,
				current,
				booking,
				req.RazorpayPaymentID,
				now,
			)
			if finalizeErr != nil {
				return finalizeErr
			}

			if isRefundRequired {
				current.Status = domain.PaymentStatusRefundRequired
				current.TransactionID = &req.RazorpayPaymentID
				current.PaidAt = &now

				booking.Status = domain.BookingStatusCancelled
				booking.UpdatedAt = now

				payment = current
				confirmedBooking = booking

				return nil
			}

			// Update returned objects to reflect the committed state.
			current.Status = domain.PaymentStatusSuccess
			current.TransactionID = &req.RazorpayPaymentID
			current.PaidAt = &now

			booking.Status = domain.BookingStatusConfirmed
			booking.UpdatedAt = now

			payment = current
			confirmedBooking = booking

			return nil
		},
	)
	if err != nil {
		u.logger.ErrorContext(
			ctx,
			"failed to finalize verified payment",
			slog.String("payment_id", payment.ID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if isRefundRequired {
		u.logger.WarnContext(
			ctx,
			"verified payment requires refund",
			slog.String("payment_id", payment.ID),
			slog.String("booking_id", payment.BookingID),
		)
		return nil, domain.ErrReservationExpired
	}

	u.logger.InfoContext(
		ctx,
		"payment verified and booking confirmed",
		slog.String("payment_id", payment.ID),
		slog.String("booking_id", payment.BookingID),
	)

	return &VerifyPaymentResult{
		Payment: payment,
		Booking: confirmedBooking,
	}, nil
}

// finalizeBookingPayment atomically finalizes the reservation, payment,
// booking, and ticket issuance.
//
// Success path:
//   - Reservation completed.
//   - Reserved seats booked.
//   - Checkout completed.
//   - Payment successful.
//   - Booking confirmed.
//   - Tickets issued.
//
// Expired path:
//   - Payment marked REFUND_REQUIRED.
//   - Booking cancelled.
//   - No tickets issued.
//
// The caller must invoke this function inside a database transaction.
func (u *PaymentUsecase) finalizeBookingPayment(
	ctx context.Context,
	tx domain.TransactionRepositories,
	payment *domain.Payment,
	booking *domain.Booking,
	transactionID string,
	now time.Time,
) (bool, error) {
	// Re-lock and inspect the current booking status to handle duplicate captures safely.
	currentBooking, err := tx.BookingRepository().FindByIDForUpdate(ctx, booking.ID)
	if err != nil {
		return false, err
	}

	// If the booking is already confirmed, this is a duplicate captured payment!
	// Preserve the confirmed booking, preserve transaction audit information, and route to refund.
	if currentBooking.Status == domain.BookingStatusConfirmed {
		if err := tx.PaymentRepository().MarkRefundRequired(
			ctx,
			payment.ID,
			transactionID,
			now,
		); err != nil {
			return false, err
		}

		u.logger.WarnContext(
			ctx,
			"duplicate payment captured for already confirmed booking; preserving confirmed booking and flagging refund",
			slog.String("payment_id", payment.ID),
			slog.String("booking_id", booking.ID.String()),
			slog.String("transaction_id", transactionID),
		)

		return true, nil
	}

	// Step 1: Validate and complete the reservation.
	if booking.ReservationID != nil &&
		*booking.ReservationID != uuid.Nil {
		reservationID := *booking.ReservationID

		reservation, err := tx.SeatReservationRepository().
			FindByID(ctx, reservationID)
		if err != nil {
			return false, err
		}

		isExpired := reservation.Status != domain.ReservationStatusActive ||
			!now.Before(reservation.ExpiresAt)

		if isExpired {
			if err := tx.PaymentRepository().MarkRefundRequired(
				ctx,
				payment.ID,
				transactionID,
				now,
			); err != nil {
				return false, err
			}

			// Only cancel booking if it is currently pending!
			if currentBooking.Status == domain.BookingStatusPending {
				if err := tx.BookingRepository().UpdateStatus(
					ctx,
					booking.ID,
					domain.BookingStatusCancelled,
				); err != nil {
					return false, err
				}
			}

			u.logger.WarnContext(
				ctx,
				"payment captured after reservation expiry; refund required",
				slog.String("payment_id", payment.ID),
				slog.String("booking_id", booking.ID.String()),
				slog.String("reservation_id", reservationID.String()),
			)

			return true, nil
		}

		completed, err := tx.SeatReservationRepository().
			CompleteIfActive(ctx, reservationID)
		if err != nil {
			return false, err
		}

		if !completed {
			if err := tx.PaymentRepository().MarkRefundRequired(
				ctx,
				payment.ID,
				transactionID,
				now,
			); err != nil {
				return false, err
			}

			// Only cancel booking if it is currently pending!
			if currentBooking.Status == domain.BookingStatusPending {
				if err := tx.BookingRepository().UpdateStatus(
					ctx,
					booking.ID,
					domain.BookingStatusCancelled,
				); err != nil {
					return false, err
				}
			}

			u.logger.WarnContext(
				ctx,
				"reservation expired during payment finalization; refund required",
				slog.String("payment_id", payment.ID),
				slog.String("booking_id", booking.ID.String()),
				slog.String("reservation_id", reservationID.String()),
			)

			return true, nil
		}

		// Mark all seats belonging to the reservation as booked.
		reservationItems, err := tx.SeatReservationRepository().
			FindItemsByReservationID(ctx, reservationID)
		if err != nil {
			return false, err
		}

		for _, item := range reservationItems {
			if item.SeatID == nil || *item.SeatID == uuid.Nil {
				continue
			}

			if err := tx.SeatRepository().
				BookReservedSeat(ctx, *item.SeatID); err != nil {
				return false, err
			}
		}

		// Complete the checkout if one exists.
		checkout, err := tx.CheckoutRepository().
			FindByReservationID(ctx, reservationID)

		if err != nil && !errors.Is(err, domain.ErrCheckoutNotFound) {
			u.logger.ErrorContext(
				ctx,
				"failed to find checkout for reservation",
				slog.String("reservation_id", reservationID.String()),
				slog.Any("error", err),
			)
			return false, err
		}

		if err == nil && checkout != nil {
			if err := tx.CheckoutRepository().UpdateStatus(
				ctx,
				checkout.ID,
				domain.CheckoutStatusCompleted,
			); err != nil {
				u.logger.ErrorContext(
					ctx,
					"failed to complete checkout",
					slog.String("checkout_id", checkout.ID.String()),
					slog.Any("error", err),
				)
				return false, err
			}
		}
	}

	// Step 2: Mark payment successful.
	if err := tx.PaymentRepository().MarkSuccessful(
		ctx,
		payment.ID,
		transactionID,
		now,
	); err != nil {
		return false, err
	}

	// Step 3: Confirm the pending booking.
	if err := tx.BookingRepository().ConfirmPending(
		ctx,
		booking.ID,
	); err != nil {
		return false, err
	}

	// Step 4: Retrieve booking items.
	items, err := tx.BookingRepository().
		FindItemsByBookingID(ctx, booking.ID)
	if err != nil {
		u.logger.ErrorContext(
			ctx,
			"failed to retrieve booking items for ticket issuance",
			slog.String("booking_id", booking.ID.String()),
			slog.Any("error", err),
		)
		return false, err
	}

	// Step 5: Generate one ticket for each admission.
	var tickets []domain.Ticket

	for _, item := range items {
		if item.Quantity <= 0 {
			err := fmt.Errorf(
				"invalid booking item quantity for item %s",
				item.ID,
			)
			u.logger.ErrorContext(
				ctx,
				"invalid booking item quantity",
				slog.String("booking_id", booking.ID.String()),
				slog.Any("error", err),
			)
			return false, err
		}

		// Reserved seating represents exactly one seat per item.
		if item.SeatID != nil &&
			*item.SeatID != uuid.Nil &&
			item.Quantity != 1 {
			err := fmt.Errorf(
				"reserved-seat booking item %s must have quantity 1",
				item.ID,
			)
			u.logger.ErrorContext(
				ctx,
				"invalid reserved-seat booking item",
				slog.String("booking_id", booking.ID.String()),
				slog.Any("error", err),
			)
			return false, err
		}

		for admissionNumber := 1; admissionNumber <= item.Quantity; admissionNumber++ {
			tickets = append(tickets, domain.Ticket{
				TicketCode:      "EH-" + uuid.NewString(),
				BookingID:       booking.ID,
				BookingItemID:   item.ID,
				UserID:          booking.UserID,
				EventID:         booking.EventID,
				TicketTypeID:    item.TicketTypeID,
				SeatID:          item.SeatID,
				Price:           item.UnitPrice,
				Status:          domain.TicketStatusValid,
				AdmissionNumber: admissionNumber,
				IssuedAt:        now,
			})
		}
	}

	// Step 6: Reject bookings that have no ticketable items.
	if len(tickets) == 0 {
		err := fmt.Errorf(
			"booking %s has no items; cannot issue tickets",
			booking.ID,
		)

		u.logger.ErrorContext(
			ctx,
			"ticket issuance rejected",
			slog.String("booking_id", booking.ID.String()),
			slog.Any("error", err),
		)

		return false, err
	}

	// Step 7: Persist tickets in the same transaction.
	if err := tx.TicketRepository().CreateMany(ctx, tickets); err != nil {
		u.logger.ErrorContext(
			ctx,
			"failed to issue tickets",
			slog.String("booking_id", booking.ID.String()),
			slog.Any("error", err),
		)
		return false, err
	}

	u.logger.InfoContext(
		ctx,
		"tickets issued",
		slog.String("booking_id", booking.ID.String()),
		slog.String("event_id", booking.EventID.String()),
		slog.Int("ticket_count", len(tickets)),
	)

	return false, nil
}

func (u *PaymentUsecase) VerifyWebhookSignature(
	body []byte,
	signature string,
) bool {
	return u.provider.VerifyWebhookSignature(body, signature)
}

func (u *PaymentUsecase) validateCheckout(
	ctx context.Context,
	tx domain.TransactionRepositories,
	checkoutID uuid.UUID,
	userID uuid.UUID,
) (*domain.Checkout, *domain.SeatReservation, error) {
	checkout, err := tx.CheckoutRepository().FindByID(
		ctx,
		checkoutID,
	)
	if err != nil {
		return nil, nil, err
	}

	if checkout.UserID != userID ||
		checkout.Status != domain.CheckoutStatusActive ||
		!time.Now().Before(checkout.ExpiresAt) {
		return nil, nil, domain.ErrCheckoutNotPayable
	}

	reservation, err := tx.SeatReservationRepository().
		FindByIDForUpdate(ctx, checkout.ReservationID)
	if err != nil {
		return nil, nil, err
	}

	if reservation.UserID != userID ||
		reservation.Status != domain.ReservationStatusActive ||
		!time.Now().Before(reservation.ExpiresAt) {
		return nil, nil, domain.ErrCheckoutNotPayable
	}

	return checkout, reservation, nil
}

func (u *PaymentUsecase) updatePayment(
	ctx context.Context,
	payment *domain.Payment,
) error {
	return u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			return tx.PaymentRepository().Update(ctx, payment)
		},
	)
}

type razorpayWebhookPayload struct {
	Entity string `json:"entity"`
	ID     string `json:"id"`
	Event  string `json:"event"`

	Payload struct {
		Payment struct {
			Entity struct {
				ID       string `json:"id"`
				OrderID  string `json:"order_id"`
				Status   string `json:"status"`
				Amount   int64  `json:"amount"`
				Currency string `json:"currency"`
			} `json:"entity"`
		} `json:"payment"`
	} `json:"payload"`
}

func (u *PaymentUsecase) ProcessRazorpayWebhook(
	ctx context.Context,
	body []byte,
	signature string,
	eventID string,
) error {
	if !u.provider.VerifyWebhookSignature(body, signature) {
		u.logger.WarnContext(
			ctx,
			"Razorpay webhook signature verification failed",
		)
		return domain.ErrInvalidPayment
	}

	var payload razorpayWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		u.logger.WarnContext(
			ctx,
			"invalid Razorpay webhook payload",
			slog.Any("error", err),
		)
		return domain.ErrInvalidPayment
	}

	resolvedEventID := strings.TrimSpace(eventID)
	if resolvedEventID == "" &&
		payload.Entity == "event" &&
		strings.TrimSpace(payload.ID) != "" {
		resolvedEventID = strings.TrimSpace(payload.ID)
	}

	if resolvedEventID == "" ||
		strings.TrimSpace(payload.Event) == "" {
		return domain.ErrInvalidPayment
	}

	switch payload.Event {
	case "payment.captured", "payment.failed":
		entity := payload.Payload.Payment.Entity

		err := u.transactionManager.WithinTransaction(
			func(tx domain.TransactionRepositories) error {
				// Unique event IDs must be enforced by the database.
				// A duplicate insert should return ErrDuplicateWebhookEvent.
				if err := tx.RazorpayWebhookEventRepository().Create(
					ctx,
					resolvedEventID,
					payload.Event,
				); err != nil {
					return err
				}

				switch payload.Event {
				case "payment.captured":
					if strings.TrimSpace(entity.OrderID) == "" ||
						strings.TrimSpace(entity.ID) == "" {
						return domain.ErrInvalidPayment
					}

					if entity.Currency != "INR" {
						u.logger.WarnContext(
							ctx,
							"webhook payment currency mismatch",
							slog.String("currency", entity.Currency),
						)
						return domain.ErrInvalidPayment
					}

					// Lock the payment to serialize concurrent capture
					// and failure webhook processing.
					payment, err := tx.PaymentRepository().
						FindByProviderOrderIDForUpdate(
							ctx,
							entity.OrderID,
						)
					if err != nil {
						return err
					}

					if payment.Provider != domain.PaymentProviderRazorpay {
						return domain.ErrInvalidPayment
					}

					if payment.Status == domain.PaymentStatusSuccess {
						if payment.TransactionID != nil &&
							*payment.TransactionID == entity.ID {
							return nil
						}

						return domain.ErrPaymentAlreadyComplete
					}

					if payment.Status == domain.PaymentStatusRefundRequired {
						// The payment was captured after expiry.
						// Acknowledge repeated delivery without issuing tickets.
						return nil
					}

					if payment.Status != domain.PaymentStatusPending {
						u.logger.WarnContext(
							ctx,
							"payment.captured received for non-pending payment",
							slog.String("payment_id", payment.ID),
							slog.String("status", string(payment.Status)),
						)
						return domain.ErrInvalidPayment
					}

					expectedPaise, err := domain.ToPaise(payment.Amount)
					if err != nil {
						return err
					}

					if expectedPaise != entity.Amount {
						u.logger.WarnContext(
							ctx,
							"payment amount mismatch in webhook",
							slog.String("payment_id", payment.ID),
							slog.Int64("expected_paise", expectedPaise),
							slog.Int64("received_paise", entity.Amount),
						)
						return domain.ErrPaymentAmountMismatch
					}

					bookingID, err := uuid.Parse(payment.BookingID)
					if err != nil {
						return domain.ErrInvalidPayment
					}

					booking, err := tx.BookingRepository().FindByID(
						ctx,
						bookingID,
					)
					if err != nil {
						return err
					}

					if booking.Status == domain.BookingStatusConfirmed {
						// Secondary captured payment for an already confirmed booking.
						// Safely flag refund without canceling the confirmed booking or issuing duplicate tickets.
						now := time.Now()
						if err := tx.PaymentRepository().MarkRefundRequired(
							ctx,
							payment.ID,
							entity.ID,
							now,
						); err != nil {
							return err
						}

						u.logger.WarnContext(
							ctx,
							"payment captured via webhook for already confirmed booking; refund required",
							slog.String("payment_id", payment.ID),
							slog.String("booking_id", booking.ID.String()),
							slog.String("provider_order_id", entity.OrderID),
							slog.String("transaction_id", entity.ID),
						)
						return nil
					}

					if booking.Status != domain.BookingStatusPending {
						u.logger.WarnContext(
							ctx,
							"payment captured for booking that is neither pending nor confirmed",
							slog.String("payment_id", payment.ID),
							slog.String("booking_id", booking.ID.String()),
							slog.String("booking_status", string(booking.Status)),
						)
						return domain.ErrInvalidPayment
					}

					now := time.Now()

					isRefundRequired, err := u.finalizeBookingPayment(
						ctx,
						tx,
						payment,
						booking,
						entity.ID,
						now,
					)
					if err != nil {
						return err
					}

					if isRefundRequired {
						u.logger.WarnContext(
							ctx,
							"payment captured via webhook after reservation expiry; refund required",
							slog.String("payment_id", payment.ID),
							slog.String("booking_id", payment.BookingID),
							slog.String("provider_order_id", entity.OrderID),
							slog.String("transaction_id", entity.ID),
						)
						return nil
					}

					u.logger.InfoContext(
						ctx,
						"payment captured and booking confirmed via webhook",
						slog.String("payment_id", payment.ID),
						slog.String("booking_id", payment.BookingID),
						slog.String("provider_order_id", entity.OrderID),
					)

					return nil

				case "payment.failed":
					if strings.TrimSpace(entity.OrderID) == "" {
						return domain.ErrInvalidPayment
					}

					payment, err := tx.PaymentRepository().
						FindByProviderOrderIDForUpdate(
							ctx,
							entity.OrderID,
						)
					if err != nil {
						return err
					}

					if payment.Status == domain.PaymentStatusSuccess ||
						payment.Status == domain.PaymentStatusRefundRequired {
						u.logger.InfoContext(
							ctx,
							"ignoring payment.failed for already successful or refund-flagged payment",
							slog.String("payment_id", payment.ID),
							slog.String("provider_order_id", entity.OrderID),
						)
						return nil
					}

					if payment.Status != domain.PaymentStatusPending {
						return nil
					}

					if err := tx.PaymentRepository().MarkFailed(
						ctx,
						payment.ID,
					); err != nil {
						return err
					}

					u.logger.InfoContext(
						ctx,
						"payment marked as failed via webhook",
						slog.String("payment_id", payment.ID),
						slog.String("provider_order_id", entity.OrderID),
					)

					return nil
				}

				return nil
			},
		)

		if errors.Is(err, domain.ErrDuplicateWebhookEvent) {
			u.logger.InfoContext(
				ctx,
				"duplicate Razorpay webhook event received; skipping processing",
				slog.String("event_id", resolvedEventID),
				slog.String("event_type", payload.Event),
			)
			return nil
		}

		if err != nil {
			u.logger.ErrorContext(
				ctx,
				"failed to process Razorpay webhook",
				slog.String("event_id", resolvedEventID),
				slog.String("event_type", payload.Event),
				slog.Any("error", err),
			)
			return err
		}

		return nil

	default:
		u.logger.InfoContext(
			ctx,
			"unhandled Razorpay webhook event ignored",
			slog.String("event_id", resolvedEventID),
			slog.String("event_type", payload.Event),
		)
		return nil
	}
}

func isSupportedPaymentMethod(method domain.PaymentMethod) bool {
	switch method {
	case domain.PaymentMethodUPI,
		domain.PaymentMethodCard,
		domain.PaymentMethodNetBanking,
		domain.PaymentMethodWallet:
		return true
	default:
		return false
	}
}
