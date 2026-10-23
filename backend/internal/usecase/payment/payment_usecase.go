package payment

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	bookingusecase "github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/google/uuid"
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

	// CreateBooking validates the checkout and reservation and returns
	// the existing booking when one already exists.
	booking, err := u.bookingUsecase.CreateBooking(
		ctx,
		bookingusecase.CreateBookingRequest{
			CheckoutID: req.CheckoutID,
			UserID:     req.UserID,
		},
	)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to prepare booking for payment",
			slog.String("checkout_id", req.CheckoutID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	if booking.Status != domain.BookingStatusPending {
		return nil, domain.ErrPaymentAlreadyComplete
	}

	var (
		checkout *domain.Checkout
		payment  *domain.Payment
	)

	// Validate checkout and create the pending payment in a transaction.
	err = u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
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

			var activePending *domain.Payment
			for i := range existingPayments {
				existing := &existingPayments[i]

				switch existing.Status {
				case domain.PaymentStatusSuccess:
					return domain.ErrPaymentAlreadyComplete
				case domain.PaymentStatusPending:
					if existing.ProviderOrderID != nil && *existing.ProviderOrderID != "" {
						activePending = existing
					} else {
						return domain.ErrPaymentInProgress
					}
				}
			}

			if activePending != nil {
				payment = activePending
				return nil
			}

			payment = &domain.Payment{
				BookingID: booking.ID.String(),
				Provider:  domain.PaymentProviderRazorpay,
				Method:    req.Method,
				Amount:    checkout.TotalAmount,
				Status:    domain.PaymentStatusPending,
			}

			return tx.PaymentRepository().Create(ctx, payment)
		},
	)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to prepare payment",
			slog.String("checkout_id", req.CheckoutID.String()),
			slog.String("booking_id", booking.ID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	// If an existing pending order was already initiated, reuse it directly.
	if payment.ProviderOrderID != nil && *payment.ProviderOrderID != "" {
		u.logger.InfoContext(ctx, "reusing existing pending payment order",
			slog.String("payment_id", payment.ID),
			slog.String("provider_order_id", *payment.ProviderOrderID),
		)
		return payment, nil
	}

	// Call the gateway only after the database transaction commits.
	// Do not hold database locks during external HTTP requests.
	orderID, err := u.provider.Initiate(
		ctx,
		payment.Amount,
		"INR",
		payment.ID,
	)
	if err != nil {
		u.logger.ErrorContext(ctx, "payment gateway order creation failed",
			slog.String("payment_id", payment.ID),
			slog.String("booking_id", payment.BookingID),
			slog.Any("error", err),
		)

		// A timeout may mean the gateway created the order.
		// Keep the payment pending until the outcome is reconciled.
		// Do not blindly retry or mark an inconclusive attempt failed.
		return nil, err
	}

	payment.ProviderOrderID = &orderID

	if err := u.updatePayment(ctx, payment); err != nil {
		u.logger.ErrorContext(ctx, "failed to persist provider order ID",
			slog.String("payment_id", payment.ID),
			slog.Any("error", err),
		)

		// Reconcile this payment with the gateway before allowing
		// another attempt. The gateway order may already exist.
		return nil, err
	}

	u.logger.InfoContext(ctx, "payment order created",
		slog.String("payment_id", payment.ID),
		slog.String("booking_id", payment.BookingID),
		slog.String("provider", string(payment.Provider)),
	)

	return payment, nil
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
		req.ProviderOrderID == "" ||
		req.RazorpayPaymentID == "" ||
		req.RazorpaySignature == "" {
		return nil, domain.ErrInvalidPayment
	}

	var (
		payment          *domain.Payment
		confirmedBooking *domain.Booking
	)

	// Find the local payment and validate booking ownership.
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

	// Allow an identical successful callback to be retried safely.
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

	// Verify the signature and confirm payment details with Razorpay.
	verified, err := u.provider.Verify(
		ctx,
		req.ProviderOrderID,
		req.RazorpayPaymentID,
		req.RazorpaySignature,
		payment.Amount,
		"INR",
	)
	if err != nil {
		// Network or gateway errors are inconclusive. Keep the payment
		// pending so its actual status can be reconciled.
		u.logger.ErrorContext(ctx, "payment verification could not be completed",
			slog.String("payment_id", payment.ID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if !verified {
		u.logger.WarnContext(ctx, "payment verification rejected",
			slog.String("payment_id", payment.ID),
		)
		return nil, domain.ErrInvalidPayment
	}

	// Atomically mark the payment successful and confirm its booking.
	// Both conditional repository operations must use transaction-scoped
	// repositories so either both changes commit or neither does.
	now := time.Now()
	var isRefundRequired bool

	err = u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			current, findErr := tx.PaymentRepository().
				FindByProviderOrderID(ctx, req.ProviderOrderID)
			if findErr != nil {
				return findErr
			}

			// Handle a concurrent duplicate callback.
			if current.Status == domain.PaymentStatusSuccess {
				if current.TransactionID != nil &&
					*current.TransactionID == req.RazorpayPaymentID {
					payment = current
					return nil
				}

				return domain.ErrPaymentAlreadyComplete
			}

			if current.Status == domain.PaymentStatusRefundRequired {
				if current.TransactionID != nil &&
					*current.TransactionID == req.RazorpayPaymentID {
					isRefundRequired = true
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
				payment = current
				confirmedBooking = booking
				return nil
			}

			// Keep the returned domain object consistent with the database.
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
		u.logger.ErrorContext(ctx, "failed to finalize verified payment",
			slog.String("payment_id", payment.ID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if isRefundRequired {
		return nil, domain.ErrReservationExpired
	}

	u.logger.InfoContext(ctx, "payment verified and booking confirmed",
		slog.String("payment_id", payment.ID),
		slog.String("booking_id", payment.BookingID),
	)

	return &VerifyPaymentResult{
		Payment: payment,
		Booking: confirmedBooking,
	}, nil
}

// finalizeBookingPayment processes reservation and inventory locking, marking either:
// 1. Success path: reservation COMPLETED, seats BOOKED, checkout COMPLETED, booking CONFIRMED, payment SUCCESS.
// 2. Expired path (strict rejection): payment REFUND_REQUIRED, booking CANCELLED.
// Returns (isRefundRequired bool, err error).
func (u *PaymentUsecase) finalizeBookingPayment(
	ctx context.Context,
	tx domain.TransactionRepositories,
	payment *domain.Payment,
	booking *domain.Booking,
	transactionID string,
	now time.Time,
) (bool, error) {
	// If the booking is linked to a temporary hold reservation:
	if booking.ReservationID != nil && *booking.ReservationID != uuid.Nil {
		reservationID := *booking.ReservationID

		reservation, err := tx.SeatReservationRepository().FindByID(ctx, reservationID)
		if err != nil {
			return false, err
		}

		// Check if the reservation is expired or not active.
		isExpired := reservation.Status != domain.ReservationStatusActive || now.After(reservation.ExpiresAt)

		if isExpired {
			// Option B: Strict rejection with reliable refund tracking.
			// Mark payment REFUND_REQUIRED and cancel the pending booking.
			if err := tx.PaymentRepository().MarkRefundRequired(
				ctx,
				payment.ID,
				transactionID,
				now,
			); err != nil {
				return false, err
			}

			if err := tx.BookingRepository().UpdateStatus(
				ctx,
				booking.ID,
				domain.BookingStatusCancelled,
			); err != nil {
				return false, err
			}

			u.logger.WarnContext(ctx, "payment captured after reservation expiry; marked REFUND_REQUIRED",
				slog.String("payment_id", payment.ID),
				slog.String("booking_id", booking.ID.String()),
				slog.String("reservation_id", reservationID.String()),
				slog.String("transaction_id", transactionID),
			)

			return true, nil
		}

		// Reservation is still active. Atomically complete the reservation hold
		// so the background expiry worker cannot release it.
		completed, err := tx.SeatReservationRepository().CompleteIfActive(ctx, reservationID)
		if err != nil {
			return false, err
		}

		if !completed {
			// Concurrently expired by the worker right at this millisecond!
			if err := tx.PaymentRepository().MarkRefundRequired(
				ctx,
				payment.ID,
				transactionID,
				now,
			); err != nil {
				return false, err
			}

			if err := tx.BookingRepository().UpdateStatus(
				ctx,
				booking.ID,
				domain.BookingStatusCancelled,
			); err != nil {
				return false, err
			}

			u.logger.WarnContext(ctx, "reservation expired concurrently during payment capture; marked REFUND_REQUIRED",
				slog.String("payment_id", payment.ID),
				slog.String("booking_id", booking.ID.String()),
				slog.String("reservation_id", reservationID.String()),
			)

			return true, nil
		}

		// Transition any reserved seats to BOOKED.
		items, err := tx.SeatReservationRepository().FindItemsByReservationID(ctx, reservationID)
		if err != nil {
			return false, err
		}

		for _, item := range items {
			if item.SeatID != nil && *item.SeatID != uuid.Nil {
				if err := tx.SeatRepository().BookReservedSeat(ctx, *item.SeatID); err != nil {
					return false, err
				}
			}
		}

		// Complete the persistent checkout snapshot if it exists.
		checkout, err := tx.CheckoutRepository().FindByReservationID(ctx, reservationID)
		if err != nil && !errors.Is(err, domain.ErrCheckoutNotFound) {
			u.logger.ErrorContext(ctx, "failed to find checkout by reservation ID",
				slog.String("reservation_id", reservationID.String()),
				slog.Any("error", err),
			)
			return false, err
		}

		if errors.Is(err, domain.ErrCheckoutNotFound) || checkout == nil {
			u.logger.DebugContext(ctx, "no checkout snapshot associated with reservation, skipping checkout completion",
				slog.String("reservation_id", reservationID.String()),
			)
		} else {
			if err := tx.CheckoutRepository().UpdateStatus(ctx, checkout.ID, domain.CheckoutStatusCompleted); err != nil {
				u.logger.ErrorContext(ctx, "failed to update checkout status to completed",
					slog.String("checkout_id", checkout.ID.String()),
					slog.String("reservation_id", reservationID.String()),
					slog.Any("error", err),
				)
				return false, err
			}
		}
	}

	// Conditional update: succeeds only while payment is pending.
	if err := tx.PaymentRepository().MarkSuccessful(
		ctx,
		payment.ID,
		transactionID,
		now,
	); err != nil {
		return false, err
	}

	// Conditional update: succeeds only while booking is pending.
	// Failure causes the entire transaction to roll back.
	if err := tx.BookingRepository().ConfirmPending(
		ctx,
		booking.ID,
	); err != nil {
		return false, err
	}

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
	checkout, err := tx.CheckoutRepository().FindByID(ctx, checkoutID)
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
	Entity  string `json:"entity"`
	ID      string `json:"id"`
	Event   string `json:"event"`
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
		return domain.ErrInvalidPayment
	}

	var payload razorpayWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.ErrInvalidPayment
	}

	resolvedEventID := strings.TrimSpace(eventID)
	if resolvedEventID == "" && payload.Entity == "event" && strings.TrimSpace(payload.ID) != "" {
		resolvedEventID = strings.TrimSpace(payload.ID)
	}

	if resolvedEventID == "" || strings.TrimSpace(payload.Event) == "" {
		return domain.ErrInvalidPayment
	}

	switch payload.Event {
	case "payment.captured", "payment.failed":
		entity := payload.Payload.Payment.Entity

		err := u.transactionManager.WithinTransaction(
			func(tx domain.TransactionRepositories) error {
				if err := tx.RazorpayWebhookEventRepository().Create(
					ctx,
					resolvedEventID,
					payload.Event,
				); err != nil {
					return err
				}

				switch payload.Event {
				case "payment.captured":
					if strings.TrimSpace(entity.OrderID) == "" || strings.TrimSpace(entity.ID) == "" {
						return domain.ErrInvalidPayment
					}

					if entity.Currency != "INR" {
						u.logger.WarnContext(ctx, "webhook payment currency mismatch",
							slog.String("currency", entity.Currency),
						)
						return domain.ErrInvalidPayment
					}

					payment, err := tx.PaymentRepository().FindByProviderOrderID(ctx, entity.OrderID)
					if err != nil {
						return err
					}

					if payment.Status == domain.PaymentStatusSuccess {
						if payment.TransactionID != nil && *payment.TransactionID == entity.ID {
							return nil
						}
						return domain.ErrPaymentAlreadyComplete
					}

					if payment.Status == domain.PaymentStatusRefundRequired {
						// Webhook delivery for an already refund-flagged payment is acknowledged safely
						return nil
					}

					if payment.Status != domain.PaymentStatusPending {
						u.logger.WarnContext(ctx, "payment.captured received for non-pending payment",
							slog.String("payment_id", payment.ID),
							slog.String("status", string(payment.Status)),
						)
						return domain.ErrInvalidPayment
					}

					expectedPaise, err := domain.ToPaise(payment.Amount)
					if err != nil || expectedPaise != entity.Amount {
						u.logger.WarnContext(ctx, "payment amount mismatch in webhook",
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

					booking, err := tx.BookingRepository().FindByID(ctx, bookingID)
					if err != nil {
						return err
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
						u.logger.WarnContext(ctx, "payment captured via webhook after reservation expiry; refund required",
							slog.String("payment_id", payment.ID),
							slog.String("booking_id", payment.BookingID),
							slog.String("provider_order_id", entity.OrderID),
							slog.String("transaction_id", entity.ID),
						)
						return nil
					}

					u.logger.InfoContext(ctx, "payment captured and booking confirmed via webhook",
						slog.String("payment_id", payment.ID),
						slog.String("booking_id", payment.BookingID),
						slog.String("provider_order_id", entity.OrderID),
					)
					return nil

				case "payment.failed":
					if strings.TrimSpace(entity.OrderID) == "" {
						return domain.ErrInvalidPayment
					}

					payment, err := tx.PaymentRepository().FindByProviderOrderID(ctx, entity.OrderID)
					if err != nil {
						return err
					}

					if payment.Status == domain.PaymentStatusSuccess || payment.Status == domain.PaymentStatusRefundRequired {
						u.logger.InfoContext(ctx, "ignoring payment.failed for already successful or refund-flagged payment",
							slog.String("payment_id", payment.ID),
							slog.String("provider_order_id", entity.OrderID),
						)
						return nil
					}

					if payment.Status != domain.PaymentStatusPending {
						return nil
					}

					if err := tx.PaymentRepository().MarkFailed(ctx, payment.ID); err != nil {
						return err
					}

					u.logger.InfoContext(ctx, "payment marked as failed via webhook",
						slog.String("payment_id", payment.ID),
						slog.String("provider_order_id", entity.OrderID),
					)
					return nil
				}

				return nil
			},
		)

		if errors.Is(err, domain.ErrDuplicateWebhookEvent) {
			u.logger.InfoContext(ctx, "duplicate Razorpay webhook event received, skipping processing",
				slog.String("event_id", resolvedEventID),
				slog.String("event_type", payload.Event),
			)
			return nil
		}

		if err != nil {
			return err
		}

		return nil

	default:
		u.logger.InfoContext(ctx, "unhandled Razorpay webhook event ignored",
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
