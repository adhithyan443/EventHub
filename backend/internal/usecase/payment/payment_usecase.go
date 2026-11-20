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
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	bookingusecase "github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/refund"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PaymentUsecase struct {
	transactionManager domain.TransactionManager
	bookingUsecase     *bookingusecase.BookingUsecase
	financeUsecase     domain.FinanceUsecase
	refundProcessor    refund.RefundProcessor
	dispatchTracker    domain.DispatchTracker
	provider           PaymentProvider
	logger             *slog.Logger
}

func NewPaymentUsecase(
	transactionManager domain.TransactionManager,
	bookingUsecase *bookingusecase.BookingUsecase,
	financeUsecase domain.FinanceUsecase,
	provider PaymentProvider,
	logger *slog.Logger,
) (*PaymentUsecase, error) {
	if transactionManager == nil {
		return nil, errors.New("transactionManager cannot be nil")
	}
	if bookingUsecase == nil {
		return nil, errors.New("bookingUsecase cannot be nil")
	}
	if financeUsecase == nil {
		return nil, errors.New("financeUsecase cannot be nil")
	}
	if provider == nil {
		return nil, errors.New("provider cannot be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &PaymentUsecase{
		transactionManager: transactionManager,
		bookingUsecase:     bookingUsecase,
		financeUsecase:     financeUsecase,
		provider:           provider,
		logger:             logger,
	}, nil
}

func (u *PaymentUsecase) SetRefundProcessor(processor refund.RefundProcessor) {
	u.refundProcessor = processor
}

func (u *PaymentUsecase) SetDispatchTracker(tracker domain.DispatchTracker) {
	u.dispatchTracker = tracker
}

func (u *PaymentUsecase) dispatchBackgroundRefund(refundID uuid.UUID) {
	if refundID == uuid.Nil || u.refundProcessor == nil {
		return
	}
	if u.dispatchTracker != nil {
		if !u.dispatchTracker.TryAdd() {
			u.logger.Warn("background_refund_dispatch_rejected_shutdown_active",
				slog.String("refund_id", refundID.String()),
			)
			return
		}
	}
	bgCtx := context.WithoutCancel(context.Background())
	go func(refID uuid.UUID) {
		if u.dispatchTracker != nil {
			defer u.dispatchTracker.Done()
		}
		_, err := u.refundProcessor.ProcessPendingRefund(bgCtx, refID)
		if err != nil {
			u.logger.Error("async_payment_cancellation_refund_failed",
				slog.String("refund_id", refID.String()),
				slog.Any("error", err),
			)
		}
	}(refundID)
}

func (u *PaymentUsecase) handleCancelledBookingPayment(
	ctx context.Context,
	tx domain.TransactionRepositories,
	payment *domain.Payment,
	booking *domain.Booking,
	transactionID string,
	reason string,
	now time.Time,
) (uuid.UUID, error) {
	if reason == "" {
		reason = "EVENT_CANCELLATION"
	}

	if payment.Status == domain.PaymentStatusPending {
		if err := tx.PaymentRepository().MarkRefundRequired(
			ctx,
			payment.ID,
			transactionID,
			now,
		); err != nil {
			return uuid.Nil, err
		}
		payment.Status = domain.PaymentStatusRefundRequired
		payment.TransactionID = &transactionID
		payment.PaidAt = &now
	}

	paymentUUID, err := uuid.Parse(payment.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid payment UUID format %s: %w", payment.ID, err)
	}

	if u.financeUsecase != nil {
		escrowCaptureInput := domain.RecordEscrowCaptureInput{
			PaymentID:     paymentUUID,
			PaymentAmount: payment.Amount,
			BookingID:     booking.ID,
			Currency:      domainfinance.CurrencyINR,
			CapturedAt:    now,
		}
		if _, err := u.financeUsecase.RecordEscrowCaptureWithinTx(ctx, tx, escrowCaptureInput); err != nil {
			u.logger.ErrorContext(ctx, "failed to record escrow capture in financial ledger",
				slog.String("payment_id", payment.ID),
				slog.String("booking_id", booking.ID.String()),
				slog.Any("error", err),
			)
			return uuid.Nil, fmt.Errorf("failed to record escrow capture: %w", err)
		}
	}

	existingRefunds, err := tx.RefundRepository().FindByPaymentID(ctx, paymentUUID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to query existing refunds for payment %s: %w", payment.ID, err)
	}

	for _, r := range existingRefunds {
		if r.Status == domain.RefundStatusSuccess ||
			r.Status == domain.RefundStatusProcessing ||
			r.Status == domain.RefundStatusPending {
			u.logger.InfoContext(ctx, "refund_already_exists_for_cancelled_booking",
				slog.String("payment_id", payment.ID),
				slog.String("refund_id", r.ID.String()),
				slog.String("status", string(r.Status)),
			)
			return uuid.Nil, nil
		}
	}

	refundRecord := &domain.Refund{
		ID:        uuid.New(),
		BookingID: booking.ID,
		PaymentID: paymentUUID,
		Amount:    payment.Amount,
		Status:    domain.RefundStatusPending,
		Reason:    reason,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := tx.RefundRepository().Create(ctx, refundRecord); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create cancellation refund: %w", err)
	}

	if booking.Status != domain.BookingStatusCancelled {
		_ = tx.BookingRepository().UpdateStatus(ctx, booking.ID, domain.BookingStatusCancelled)
		booking.Status = domain.BookingStatusCancelled
	}

	if booking.ReservationID != nil && *booking.ReservationID != uuid.Nil {
		reservationID := *booking.ReservationID
		_, _ = tx.SeatReservationRepository().CancelIfActive(ctx, reservationID, booking.UserID)
		_, _ = tx.CheckoutRepository().ExpireActiveByReservationID(ctx, reservationID)

		// Release reserved items
		items, err := tx.SeatReservationRepository().FindItemsByReservationID(ctx, reservationID)
		if err == nil {
			for _, item := range items {
				if item.SeatID != nil && *item.SeatID != uuid.Nil {
					_ = tx.SeatRepository().ReleaseReservedSeat(ctx, *item.SeatID)
				}
				if item.TicketTypeID != nil && *item.TicketTypeID != uuid.Nil && item.Quantity > 0 {
					_ = tx.TicketTypeRepository().ReleaseReservedQuantity(ctx, *item.TicketTypeID, item.Quantity)
				}
			}
		}
	}

	u.logger.InfoContext(ctx, "created_refund_for_cancelled_booking_payment",
		slog.String("payment_id", payment.ID),
		slog.String("booking_id", booking.ID.String()),
		slog.String("refund_id", refundRecord.ID.String()),
		slog.String("reason", reason),
		slog.String("amount", refundRecord.Amount.String()),
	)

	return refundRecord.ID, nil
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
				return domain.ErrPaymentRefundRequired
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
				return domain.ErrPaymentRefundRequired
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
	Invoice *domain.Invoice
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
		invoice          *domain.Invoice
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
			bookingID, parseErr := uuid.Parse(payment.BookingID)
			if parseErr != nil {
				return nil, domain.ErrInvalidPayment
			}

			err = u.transactionManager.WithinTransaction(
				func(tx domain.TransactionRepositories) error {
					inv, findErr := tx.InvoiceRepository().FindByBookingID(ctx, bookingID)
					if findErr == nil {
						invoice = inv
						return nil
					}
					if !errors.Is(findErr, domain.ErrInvoiceNotFound) {
						u.logger.ErrorContext(ctx, "failed to query existing invoice on idempotent payment verification",
							slog.String("booking_id", bookingID.String()),
							slog.Any("error", findErr),
						)
						return findErr
					}

					// Self-heal: generate receipt snapshot
					inv, healErr := u.createReceiptSnapshot(ctx, tx, confirmedBooking, payment, time.Now())
					if healErr != nil {
						u.logger.ErrorContext(ctx, "failed to self-heal missing receipt snapshot on idempotent verification",
							slog.String("booking_id", bookingID.String()),
							slog.Any("error", healErr),
						)
						return healErr
					}
					invoice = inv
					return nil
				},
			)
			if err != nil {
				return nil, err
			}

			return &VerifyPaymentResult{
				Payment: payment,
				Booking: confirmedBooking,
				Invoice: invoice,
			}, nil
		}

		return nil, domain.ErrPaymentAlreadyComplete
	}

	if payment.Status == domain.PaymentStatusRefundRequired {
		return nil, domain.ErrPaymentRefundRequired
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
	var newRefundID uuid.UUID

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

				inv, findErr := tx.InvoiceRepository().FindByBookingID(ctx, bookingID)
				if findErr == nil {
					invoice = inv
				} else if errors.Is(findErr, domain.ErrInvoiceNotFound) {
					inv, healErr := u.createReceiptSnapshot(ctx, tx, booking, payment, now)
					if healErr != nil {
						return healErr
					}
					invoice = inv
				} else {
					return findErr
				}

				return nil
			}

			if current.Status == domain.PaymentStatusRefundRequired {
				if current.TransactionID != nil &&
					*current.TransactionID == req.RazorpayPaymentID {
					isRefundRequired = true
					payment = current
					return nil
				}

				return domain.ErrPaymentRefundRequired
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

			// Check if booking is cancelled or event is cancelled (e.g. event cancellation happened while payment was in-flight).
			var isEventCancelled bool
			event, eventErr := tx.EventRepository().FindByID(booking.EventID)
			if eventErr == nil && event != nil && event.Status == domain.EventStatusCancelled {
				isEventCancelled = true
			}

			if booking.Status == domain.BookingStatusCancelled || isEventCancelled {
				refID, err := u.handleCancelledBookingPayment(
					ctx,
					tx,
					current,
					booking,
					req.RazorpayPaymentID,
					"EVENT_CANCELLATION",
					now,
				)
				if err != nil {
					return err
				}
				newRefundID = refID
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
				&newRefundID,
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

			// Load the newly generated invoice
			inv, findErr := tx.InvoiceRepository().FindByBookingID(ctx, booking.ID)
			if findErr != nil {
				return findErr
			}
			invoice = inv

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
		if newRefundID != uuid.Nil {
			u.dispatchBackgroundRefund(newRefundID)
		}
		u.logger.WarnContext(
			ctx,
			"verified payment requires refund",
			slog.String("payment_id", payment.ID),
			slog.String("booking_id", payment.BookingID),
		)
		return nil, domain.ErrPaymentRefundRequired
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
		Invoice: invoice,
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
	refundIDOut *uuid.UUID,
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

	var isEventCancelled bool
	event, eventErr := tx.EventRepository().FindByID(booking.EventID)
	if eventErr == nil && event != nil && event.Status == domain.EventStatusCancelled {
		isEventCancelled = true
	}

	if currentBooking.Status == domain.BookingStatusCancelled || isEventCancelled {
		refID, err := u.handleCancelledBookingPayment(
			ctx,
			tx,
			payment,
			currentBooking,
			transactionID,
			"EVENT_CANCELLATION",
			now,
		)
		if err != nil {
			return false, err
		}
		if refundIDOut != nil && refID != uuid.Nil {
			*refundIDOut = refID
		}
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
			refID, err := u.handleCancelledBookingPayment(
				ctx,
				tx,
				payment,
				currentBooking,
				transactionID,
				"RESERVATION_EXPIRED",
				now,
			)
			if err != nil {
				return false, err
			}
			if refundIDOut != nil && refID != uuid.Nil {
				*refundIDOut = refID
			}

			u.logger.WarnContext(
				ctx,
				"payment captured after reservation expiry; refund required and scheduled",
				slog.String("payment_id", payment.ID),
				slog.String("booking_id", booking.ID.String()),
				slog.String("reservation_id", reservationID.String()),
				slog.String("refund_id", refID.String()),
			)

			return true, nil
		}

		completed, err := tx.SeatReservationRepository().
			CompleteIfActive(ctx, reservationID)
		if err != nil {
			return false, err
		}

		if !completed {
			refID, err := u.handleCancelledBookingPayment(
				ctx,
				tx,
				payment,
				currentBooking,
				transactionID,
				"RESERVATION_EXPIRED",
				now,
			)
			if err != nil {
				return false, err
			}
			if refundIDOut != nil && refID != uuid.Nil {
				*refundIDOut = refID
			}

			u.logger.WarnContext(
				ctx,
				"reservation expired during payment finalization; refund required and scheduled",
				slog.String("payment_id", payment.ID),
				slog.String("booking_id", booking.ID.String()),
				slog.String("reservation_id", reservationID.String()),
				slog.String("refund_id", refID.String()),
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

	// Finance Ledger Recording: Atomically post double-entry financial journal for payment capture.
	event, err = tx.EventRepository().FindByID(booking.EventID)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to find event for payment capture finance journal",
			slog.String("event_id", booking.EventID.String()),
			slog.Any("error", err),
		)
		return false, err
	}

	paymentUUID, err := uuid.Parse(payment.ID)
	if err != nil {
		u.logger.ErrorContext(ctx, "invalid payment ID UUID for finance journal",
			slog.String("payment_id", payment.ID),
			slog.Any("error", err),
		)
		return false, fmt.Errorf("invalid payment UUID %s: %w", payment.ID, err)
	}

	financeInput := domain.RecordPaymentCaptureInput{
		PaymentID:     paymentUUID,
		PaymentAmount: payment.Amount,
		Currency:      domainfinance.CurrencyINR,
		BookingID:     booking.ID,
		Subtotal:      booking.Subtotal,
		PlatformFee:   booking.PlatformFee,
		Tax:           booking.Tax,
		Discount:      booking.Discount,
		Total:         booking.Total,
		EventID:       booking.EventID,
		OrganizerID:   event.OrganizerID,
		UserID:        booking.UserID,
		CapturedAt:    now,
	}

	if _, err := u.financeUsecase.RecordPaymentCaptureWithinTx(ctx, tx, financeInput); err != nil {
		u.logger.ErrorContext(ctx, "failed to record payment capture in financial ledger",
			slog.String("payment_id", payment.ID),
			slog.String("booking_id", booking.ID.String()),
			slog.Any("error", err),
		)
		return false, err
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

	// Step 8: Generate booking receipt snapshot in the same atomic transaction.
	if _, err := u.createReceiptSnapshot(ctx, tx, booking, payment, now); err != nil {
		u.logger.ErrorContext(
			ctx,
			"failed to create booking receipt snapshot",
			slog.String("booking_id", booking.ID.String()),
			slog.Any("error", err),
		)
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
		Refund struct {
			Entity struct {
				ID        string                 `json:"id"`
				PaymentID string                 `json:"payment_id"`
				Amount    int64                  `json:"amount"`
				Currency  string                 `json:"currency"`
				Status    string                 `json:"status"`
				Receipt   string                 `json:"receipt"`
				Notes     map[string]interface{} `json:"notes"`
			} `json:"entity"`
		} `json:"refund"`
	} `json:"payload"`
}

func (u *PaymentUsecase) ProcessRazorpayWebhook(
	ctx context.Context,
	body []byte,
	signature string,
	eventID string,
) error {
	var newRefundID uuid.UUID
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
							bookingID, parseErr := uuid.Parse(payment.BookingID)
							if parseErr != nil {
								return domain.ErrInvalidPayment
							}
							_, findErr := tx.InvoiceRepository().FindByBookingID(ctx, bookingID)
							if findErr != nil {
								if errors.Is(findErr, domain.ErrInvoiceNotFound) {
									booking, bErr := tx.BookingRepository().FindByID(ctx, bookingID)
									if bErr != nil {
										return bErr
									}
									if _, healErr := u.createReceiptSnapshot(ctx, tx, booking, payment, time.Now()); healErr != nil {
										return healErr
									}
								} else {
									return findErr
								}
							}
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

					var isEventCancelled bool
					event, eventErr := tx.EventRepository().FindByID(booking.EventID)
					if eventErr == nil && event != nil && event.Status == domain.EventStatusCancelled {
						isEventCancelled = true
					}

					if booking.Status == domain.BookingStatusCancelled || isEventCancelled {
						now := time.Now()
						refID, err := u.handleCancelledBookingPayment(
							ctx,
							tx,
							payment,
							booking,
							entity.ID,
							"EVENT_CANCELLATION",
							now,
						)
						if err != nil {
							return err
						}
						newRefundID = refID
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
						&newRefundID,
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

		if newRefundID != uuid.Nil {
			u.dispatchBackgroundRefund(newRefundID)
		}

		return nil

	case "refund.processed", "refund.failed", "refund.created":
		refundEntity := payload.Payload.Refund.Entity
		err := u.transactionManager.WithinTransaction(
			func(tx domain.TransactionRepositories) error {
				if err := tx.RazorpayWebhookEventRepository().Create(
					ctx,
					resolvedEventID,
					payload.Event,
				); err != nil {
					return err
				}

				if strings.TrimSpace(refundEntity.ID) == "" {
					return domain.ErrInvalidPayment
				}

				// Look up refund record
				var refund *domain.Refund
				var err error

				refund, err = tx.RefundRepository().FindByGatewayRefundIDForUpdate(ctx, refundEntity.ID)
				if err != nil && errors.Is(err, domain.ErrRefundNotFound) {
					var refundUUID uuid.UUID
					if parsed, pErr := uuid.Parse(strings.TrimSpace(refundEntity.Receipt)); pErr == nil && parsed != uuid.Nil {
						refundUUID = parsed
					} else if idKey, ok := refundEntity.Notes["idempotency_key"].(string); ok {
						if parsedKey, pErr := uuid.Parse(strings.TrimSpace(idKey)); pErr == nil && parsedKey != uuid.Nil {
							refundUUID = parsedKey
						}
					}

					if refundUUID != uuid.Nil {
						refund, err = tx.RefundRepository().FindByIDForUpdate(ctx, refundUUID)
					}
				}

				if err != nil {
					if errors.Is(err, domain.ErrRefundNotFound) {
						u.logger.WarnContext(ctx, "refund webhook received for untracked refund",
							slog.String("event_id", resolvedEventID),
							slog.String("event_type", payload.Event),
							slog.String("gateway_refund_id", refundEntity.ID),
						)
						return nil
					}
					return err
				}

				if refund == nil {
					return nil
				}

				if refundEntity.Amount <= 0 {
					u.logger.WarnContext(ctx, "invalid zero or negative refund amount in webhook",
						slog.String("refund_id", refund.ID.String()),
						slog.Int64("received_paise", refundEntity.Amount),
					)
					return domain.ErrInvalidPayment
				}

				expectedPaise, err := domain.ToPaise(refund.Amount)
				if err != nil {
					return err
				}
				if expectedPaise != refundEntity.Amount {
					u.logger.WarnContext(ctx, "refund amount mismatch in webhook",
						slog.String("refund_id", refund.ID.String()),
						slog.Int64("expected_paise", expectedPaise),
						slog.Int64("received_paise", refundEntity.Amount),
					)
					return domain.ErrPaymentAmountMismatch
				}
				if refundEntity.Currency != "INR" {
					return domain.ErrInvalidPayment
				}

				now := time.Now()

				switch payload.Event {
				case "refund.processed":
					if refund.Status == domain.RefundStatusSuccess {
						u.logger.InfoContext(ctx, "ignoring refund.processed for already successful refund",
							slog.String("refund_id", refund.ID.String()),
							slog.String("gateway_refund_id", refundEntity.ID),
						)
						return nil
					}

					refund.Status = domain.RefundStatusSuccess
					refund.GatewayRefundID = &refundEntity.ID
					refund.ProcessedAt = &now
					refund.UpdatedAt = now
					if err := tx.RefundRepository().Update(ctx, refund); err != nil {
						return err
					}

					if err := u.recordRefundJournal(ctx, tx, refund, now); err != nil {
						return err
					}

					u.logger.InfoContext(ctx, "refund marked as success via webhook",
						slog.String("refund_id", refund.ID.String()),
						slog.String("gateway_refund_id", refundEntity.ID),
					)

				case "refund.failed":
					if refund.Status == domain.RefundStatusSuccess {
						u.logger.WarnContext(ctx, "ignoring refund.failed for already successful refund",
							slog.String("refund_id", refund.ID.String()),
							slog.String("gateway_refund_id", refundEntity.ID),
						)
						return nil
					}

					refund.Status = domain.RefundStatusFailed
					refund.GatewayRefundID = &refundEntity.ID
					refund.UpdatedAt = now
					if err := tx.RefundRepository().Update(ctx, refund); err != nil {
						return err
					}

					u.logger.InfoContext(ctx, "refund marked as failed via webhook",
						slog.String("refund_id", refund.ID.String()),
						slog.String("gateway_refund_id", refundEntity.ID),
					)

				case "refund.created":
					if refund.Status == domain.RefundStatusSuccess || refund.Status == domain.RefundStatusFailed {
						return nil
					}
					refund.Status = domain.RefundStatusProcessing
					refund.GatewayRefundID = &refundEntity.ID
					refund.UpdatedAt = now
					if err := tx.RefundRepository().Update(ctx, refund); err != nil {
						return err
					}

					u.logger.InfoContext(ctx, "refund marked as processing via webhook",
						slog.String("refund_id", refund.ID.String()),
						slog.String("gateway_refund_id", refundEntity.ID),
					)
				}

				return nil
			},
		)

		if errors.Is(err, domain.ErrDuplicateWebhookEvent) {
			u.logger.InfoContext(ctx, "duplicate Razorpay webhook event received; skipping processing",
				slog.String("event_id", resolvedEventID),
				slog.String("event_type", payload.Event),
			)
			return nil
		}

		if err != nil {
			u.logger.ErrorContext(ctx, "failed to process Razorpay refund webhook",
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

// createReceiptSnapshot creates an immutable booking receipt and line items snapshot
// within an active database transaction. It is safe for concurrent calls and idempotent.
func (u *PaymentUsecase) createReceiptSnapshot(
	ctx context.Context,
	tx domain.TransactionRepositories,
	booking *domain.Booking,
	payment *domain.Payment,
	now time.Time,
) (*domain.Invoice, error) {
	if booking == nil || payment == nil {
		return nil, errors.New("booking and payment are required to create receipt snapshot")
	}

	// Idempotency: if invoice already exists for this booking, return it immediately
	existing, err := tx.InvoiceRepository().FindByBookingID(ctx, booking.ID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, domain.ErrInvoiceNotFound) {
		return nil, fmt.Errorf("failed to query existing invoice: %w", err)
	}

	paymentUUID, err := uuid.Parse(payment.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid payment id %s: %w", payment.ID, err)
	}

	receiptNumber, err := tx.InvoiceRepository().NextReceiptNumber(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("failed to generate receipt number: %w", err)
	}

	// 1. Customer details snapshot
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

	// 2. Event details snapshot
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

	// 3. Schedule details snapshot
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

	// 4. Line items snapshot
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
			return nil, fmt.Errorf("failed to retrieve existing invoice after duplicate error: %w", getErr)
		}
		return nil, fmt.Errorf("failed to save invoice: %w", err)
	}

	return invoice, nil
}

func (u *PaymentUsecase) recordRefundJournal(
	ctx context.Context,
	tx domain.TransactionRepositories,
	targetRefund *domain.Refund,
	processedAt time.Time,
) error {
	if u.financeUsecase == nil || targetRefund == nil || tx == nil {
		return nil
	}

	if tx.PaymentRepository() == nil {
		return errors.New("PaymentRepository cannot be nil for refund journaling")
	}
	payment, err := tx.PaymentRepository().FindByID(ctx, targetRefund.PaymentID.String())
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			u.logger.WarnContext(ctx, "payment not found for refund journaling; skipping ledger entry",
				slog.String("refund_id", targetRefund.ID.String()),
				slog.String("payment_id", targetRefund.PaymentID.String()),
			)
			return nil
		}
		u.logger.ErrorContext(ctx, "failed to find payment for refund journaling",
			slog.String("refund_id", targetRefund.ID.String()),
			slog.String("payment_id", targetRefund.PaymentID.String()),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to find payment %s: %w", targetRefund.PaymentID, err)
	}

	if tx.BookingRepository() == nil {
		return errors.New("BookingRepository cannot be nil for refund journaling")
	}
	booking, err := tx.BookingRepository().FindByID(ctx, targetRefund.BookingID)
	if err != nil {
		if errors.Is(err, domain.ErrBookingNotFound) {
			u.logger.WarnContext(ctx, "booking not found for refund journaling; skipping ledger entry",
				slog.String("refund_id", targetRefund.ID.String()),
				slog.String("booking_id", targetRefund.BookingID.String()),
			)
			return nil
		}
		u.logger.ErrorContext(ctx, "failed to find booking for refund journaling",
			slog.String("refund_id", targetRefund.ID.String()),
			slog.String("booking_id", targetRefund.BookingID.String()),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to find booking %s: %w", targetRefund.BookingID, err)
	}

	isEscrowRefund := payment.Status == domain.PaymentStatusRefundRequired

	var eventID uuid.UUID
	var organizerID uuid.UUID
	var platformFeeRefund decimal.Decimal
	var taxRefund decimal.Decimal
	isEventCancellation := false

	if isEscrowRefund {
		eventID = booking.EventID
		if tx.EventRepository() != nil {
			if event, eventErr := tx.EventRepository().FindByID(booking.EventID); eventErr == nil && event != nil {
				organizerID = event.OrganizerID
			}
		}
	} else {
		eventID = booking.EventID
		if tx.EventRepository() != nil {
			event, eventErr := tx.EventRepository().FindByID(booking.EventID)
			if eventErr != nil {
				return fmt.Errorf("failed to find event %s for refund journaling: %w", booking.EventID, eventErr)
			}
			organizerID = event.OrganizerID
		}

		if targetRefund.Reason == "EVENT_CANCELLATION" {
			isEventCancellation = true
		}

		var priorPlatformFeeRefunded decimal.Decimal
		var priorTaxRefunded decimal.Decimal
		if tx.RefundRepository() != nil {
			if allRefunds, rErr := tx.RefundRepository().FindByBookingID(ctx, booking.ID); rErr == nil {
				for _, r := range allRefunds {
					if r.ID != targetRefund.ID && r.Status == domain.RefundStatusSuccess {
						if booking.Total.IsPositive() {
							priorFee := r.Amount.Mul(booking.PlatformFee).Div(booking.Total).RoundBank(2)
							priorTax := r.Amount.Mul(booking.Tax).Div(booking.Total).RoundBank(2)
							priorPlatformFeeRefunded = priorPlatformFeeRefunded.Add(priorFee)
							priorTaxRefunded = priorTaxRefunded.Add(priorTax)
						}
					}
				}
			}
		}

		if booking.Total.IsPositive() {
			if booking.PlatformFee.IsPositive() {
				remainingFee := booking.PlatformFee.Sub(priorPlatformFeeRefunded)
				if remainingFee.IsPositive() {
					computedFee := targetRefund.Amount.Mul(booking.PlatformFee).Div(booking.Total).RoundBank(2)
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
					computedTax := targetRefund.Amount.Mul(booking.Tax).Div(booking.Total).RoundBank(2)
					if computedTax.GreaterThan(remainingTax) {
						taxRefund = remainingTax
					} else {
						taxRefund = computedTax
					}
				}
			}
			if platformFeeRefund.Add(taxRefund).GreaterThan(targetRefund.Amount) {
				if platformFeeRefund.GreaterThan(targetRefund.Amount) {
					platformFeeRefund = targetRefund.Amount
					taxRefund = decimal.Zero
				} else {
					taxRefund = targetRefund.Amount.Sub(platformFeeRefund)
				}
			}
		}
	}

	refundInput := domain.RecordRefundInput{
		RefundID:            targetRefund.ID,
		RefundAmount:        targetRefund.Amount,
		PaymentID:           targetRefund.PaymentID,
		BookingID:           targetRefund.BookingID,
		EventID:             eventID,
		OrganizerID:         organizerID,
		IsEventCancellation: isEventCancellation,
		IsEscrowRefund:      isEscrowRefund,
		PlatformFeeRefund:   platformFeeRefund,
		TaxRefund:           taxRefund,
		Currency:            domainfinance.CurrencyINR,
		ProcessedAt:         processedAt,
	}

	if _, err := u.financeUsecase.RecordRefundWithinTx(ctx, tx, refundInput); err != nil {
		u.logger.ErrorContext(ctx, "failed to record refund journal entry via webhook",
			slog.String("refund_id", targetRefund.ID.String()),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to record refund journal: %w", err)
	}

	return nil
}
