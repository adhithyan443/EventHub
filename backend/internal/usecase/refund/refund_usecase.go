package refund

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const ProcessingLeaseDuration = 5 * time.Minute

type RefundProcessor interface {
	ProcessPendingRefund(ctx context.Context, refundID uuid.UUID) (*domain.Refund, error)
	ReconcileStuckRefunds(ctx context.Context, olderThan time.Duration, limit int) (int, error)
	GetRefundByID(ctx context.Context, refundID uuid.UUID) (*domain.Refund, error)
	GetRefundsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Refund, error)
}

type RefundUsecase struct {
	transactionManager domain.TransactionManager
	refundProvider     domain.PaymentRefundProvider
	financeUsecase     domain.FinanceUsecase
	logger             *slog.Logger
}

func NewRefundUsecase(
	transactionManager domain.TransactionManager,
	refundProvider domain.PaymentRefundProvider,
	logger *slog.Logger,
) *RefundUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &RefundUsecase{
		transactionManager: transactionManager,
		refundProvider:     refundProvider,
		logger:             logger,
	}
}

func (u *RefundUsecase) SetFinanceUsecase(financeUsecase domain.FinanceUsecase) {
	u.financeUsecase = financeUsecase
}

// ProcessPendingRefund processes a pending or processing refund record with the payment gateway.
// It enforces the payment refund cap by locking the parent payment row, implements a safe
// processing lease mechanism against duplicate in-flight executions, checks authoritative
// gateway status, and prevents false-positive refund matching.
func (u *RefundUsecase) ProcessPendingRefund(
	ctx context.Context,
	refundID uuid.UUID,
) (*domain.Refund, error) {
	if refundID == uuid.Nil {
		return nil, errors.New("refund ID cannot be nil")
	}

	var targetRefund *domain.Refund
	var gatewayPaymentID string
	usedGatewayIDs := make(map[string]bool)

	// Step 1: Lock refund and parent payment row, validate state, lease, and total payment cap.
	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			ref, err := tx.RefundRepository().FindByIDForUpdate(ctx, refundID)
			if err != nil {
				return err
			}

			// Idempotency: If already successful, return directly.
			if ref.Status == domain.RefundStatusSuccess {
				u.logger.InfoContext(ctx, "refund_already_success",
					slog.String("refund_id", ref.ID.String()),
					slog.String("booking_id", ref.BookingID.String()),
				)
				targetRefund = ref
				return nil
			}

			// Processing Lease / Claim Check: Prevent concurrent in-flight duplicate processing.
			if ref.Status == domain.RefundStatusProcessing && time.Since(ref.UpdatedAt) < ProcessingLeaseDuration {
				u.logger.WarnContext(ctx, "refund_processing_claim_active",
					slog.String("refund_id", ref.ID.String()),
					slog.Duration("active_since", time.Since(ref.UpdatedAt)),
				)
				return domain.ErrRefundInProgress
			}

			// Lock parent payment row with FindByIDForUpdate to strictly serialize all concurrent
			// refunds associated with the same captured payment.
			payment, err := tx.PaymentRepository().FindByIDForUpdate(ctx, ref.PaymentID.String())
			if err != nil {
				u.logger.ErrorContext(ctx, "refund_payment_not_found",
					slog.String("refund_id", ref.ID.String()),
					slog.String("payment_id", ref.PaymentID.String()),
					slog.Any("error", err),
				)
				return domain.ErrPaymentNotFound
			}

			if payment.Status != domain.PaymentStatusSuccess && payment.Status != domain.PaymentStatusRefundRequired {
				u.logger.ErrorContext(ctx, "refund_invalid_payment_status",
					slog.String("refund_id", ref.ID.String()),
					slog.String("payment_id", ref.PaymentID.String()),
					slog.String("status", string(payment.Status)),
				)
				return errors.New("payment is not eligible for refund")
			}

			if payment.TransactionID == nil || strings.TrimSpace(*payment.TransactionID) == "" {
				u.logger.ErrorContext(ctx, "refund_payment_missing_transaction_id",
					slog.String("refund_id", ref.ID.String()),
					slog.String("payment_id", ref.PaymentID.String()),
				)
				return errors.New("payment has no gateway transaction ID")
			}
			gatewayPaymentID = strings.TrimSpace(*payment.TransactionID)

			// Cap validation: Total successful, processing, and prior pending refunds must not exceed payment amount.
			allRefunds, err := tx.RefundRepository().FindByPaymentID(ctx, ref.PaymentID)
			if err != nil {
				return err
			}

			existingRefunded := decimal.Zero
			for _, r := range allRefunds {
				if r.GatewayRefundID != nil && *r.GatewayRefundID != "" && r.ID != ref.ID {
					usedGatewayIDs[*r.GatewayRefundID] = true
				}
				if r.ID != ref.ID {
					if r.Status == domain.RefundStatusSuccess || r.Status == domain.RefundStatusProcessing || (r.Status == domain.RefundStatusPending && (r.CreatedAt.Before(ref.CreatedAt) || (r.CreatedAt.Equal(ref.CreatedAt) && r.ID.String() < ref.ID.String()))) {
						existingRefunded = existingRefunded.Add(r.Amount)
					}
				}
			}

			if existingRefunded.Add(ref.Amount).GreaterThan(payment.Amount) {
				u.logger.ErrorContext(ctx, "refund_amount_exceeds_payment",
					slog.String("refund_id", ref.ID.String()),
					slog.String("payment_id", ref.PaymentID.String()),
					slog.String("payment_amount", payment.Amount.String()),
					slog.String("existing_refunded", existingRefunded.String()),
					slog.String("requested_refund", ref.Amount.String()),
				)
				return domain.ErrRefundExceedsPayment
			}

			// Claim / renew lease by transitioning status to PROCESSING with updated timestamp.
			ref.Status = domain.RefundStatusProcessing
			ref.UpdatedAt = time.Now()
			if err := tx.RefundRepository().Update(ctx, ref); err != nil {
				return err
			}

			targetRefund = ref
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	// If refund was already successful, return immediately.
	if targetRefund.Status == domain.RefundStatusSuccess {
		return targetRefund, nil
	}

	if u.refundProvider == nil {
		u.logger.ErrorContext(ctx, "refund_provider_not_configured")
		return nil, errors.New("refund provider not configured")
	}

	idempotencyKey := targetRefund.ID.String()
	var refundResult *domain.RefundResult

	// Step 2: Gateway preflight check with authoritative status fetching.
	// Query Razorpay to discover if this refund was already accepted or in-flight on the gateway.
	gatewayRefunds, fetchErr := u.refundProvider.FetchRefunds(ctx, gatewayPaymentID)
	if fetchErr == nil && len(gatewayRefunds) > 0 {
		// A. If targetRefund already has a GatewayRefundID, match it directly in fetched refunds.
		if targetRefund.GatewayRefundID != nil && strings.TrimSpace(*targetRefund.GatewayRefundID) != "" {
			for _, gr := range gatewayRefunds {
				if gr.RefundID == *targetRefund.GatewayRefundID {
					refundResult = &gr
					break
				}
			}
		}

		// B. If not found by GatewayRefundID, match by strict receipt/idempotency key excluding used IDs.
		if refundResult == nil {
			refundResult = findMatchingGatewayRefund(gatewayRefunds, targetRefund.ID, targetRefund.Amount, usedGatewayIDs)
		}

		if refundResult != nil {
			u.logger.InfoContext(ctx, "refund_preflight_reconciled",
				slog.String("refund_id", targetRefund.ID.String()),
				slog.String("gateway_refund_id", refundResult.RefundID),
				slog.String("status", refundResult.Status),
			)
		}
	} else if fetchErr != nil && targetRefund.GatewayRefundID != nil && strings.TrimSpace(*targetRefund.GatewayRefundID) != "" {
		// If gateway query failed but we already have a GatewayRefundID, do NOT issue another refund!
		u.logger.WarnContext(ctx, "gateway_fetch_failed_for_existing_refund_id",
			slog.String("refund_id", targetRefund.ID.String()),
			slog.String("gateway_refund_id", *targetRefund.GatewayRefundID),
			slog.Any("error", fetchErr),
		)
		return nil, fmt.Errorf("failed to verify gateway status of existing refund: %w", fetchErr)
	}

	// Step 3: Issue Gateway Refund only if no existing gateway refund was discovered.
	if refundResult == nil {
		res, gatewayErr := u.refundProvider.Refund(ctx, gatewayPaymentID, targetRefund.Amount, "INR", idempotencyKey)
		if gatewayErr == nil && res != nil && (res.Status == domain.RefundStatusSuccess || res.Status == domain.RefundStatusPending || res.Status == domain.RefundStatusProcessing) {
			refundResult = res
		} else {
			// Step 4: Ambiguous Timeout Recovery / Post-failure Reconciliation.
			reconciledRefunds, fetchErr := u.refundProvider.FetchRefunds(ctx, gatewayPaymentID)
			if fetchErr == nil && len(reconciledRefunds) > 0 {
				matched := findMatchingGatewayRefund(reconciledRefunds, targetRefund.ID, targetRefund.Amount, usedGatewayIDs)
				if matched != nil {
					u.logger.InfoContext(ctx, "refund_post_failure_reconciled",
						slog.String("refund_id", targetRefund.ID.String()),
						slog.String("gateway_refund_id", matched.RefundID),
						slog.String("status", matched.Status),
					)
					refundResult = matched
				}
			}

			// If still no refund found, this is a confirmed gateway failure.
			if refundResult == nil {
				_ = u.transactionManager.WithinTransaction(
					func(tx domain.TransactionRepositories) error {
						targetRefund.Status = domain.RefundStatusFailed
						targetRefund.UpdatedAt = time.Now()
						return tx.RefundRepository().Update(ctx, targetRefund)
					},
				)

				u.logger.ErrorContext(ctx, "gateway_refund_execution_failed",
					slog.String("refund_id", targetRefund.ID.String()),
					slog.String("booking_id", targetRefund.BookingID.String()),
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

	// Step 5: Update refund record in DB with confirmed gateway result.
	completionTime := time.Now()
	finalStatus := refundResult.Status
	if strings.EqualFold(finalStatus, "processed") || strings.EqualFold(finalStatus, "success") {
		finalStatus = domain.RefundStatusSuccess
	} else if strings.EqualFold(finalStatus, "pending") {
		finalStatus = domain.RefundStatusPending
	} else if strings.EqualFold(finalStatus, "processing") {
		finalStatus = domain.RefundStatusProcessing
	}

	err = u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			freshRefund, err := tx.RefundRepository().FindByIDForUpdate(ctx, targetRefund.ID)
			if err != nil {
				return err
			}
			if freshRefund.Status == domain.RefundStatusSuccess {
				u.logger.InfoContext(ctx, "refund_already_success_in_db",
					slog.String("refund_id", freshRefund.ID.String()),
				)
				targetRefund = freshRefund
				return nil
			}

			freshRefund.Status = finalStatus
			freshRefund.GatewayRefundID = &refundResult.RefundID
			if finalStatus == domain.RefundStatusSuccess {
				freshRefund.ProcessedAt = &completionTime
			}
			freshRefund.UpdatedAt = completionTime
			if err := tx.RefundRepository().Update(ctx, freshRefund); err != nil {
				return err
			}
			targetRefund = freshRefund

			if finalStatus == domain.RefundStatusSuccess {
				if err := u.recordRefundJournal(ctx, tx, freshRefund, completionTime); err != nil {
					return err
				}
			}
			return nil
		},
	)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed_to_save_confirmed_refund",
			slog.String("refund_id", targetRefund.ID.String()),
			slog.String("gateway_refund_id", refundResult.RefundID),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "refund_processed_successfully",
		slog.String("refund_id", targetRefund.ID.String()),
		slog.String("booking_id", targetRefund.BookingID.String()),
		slog.String("gateway_refund_id", refundResult.RefundID),
		slog.String("amount", targetRefund.Amount.String()),
		slog.String("status", finalStatus),
	)

	return targetRefund, nil
}

// ReconcileStuckRefunds queries refunds that remain PENDING or PROCESSING beyond olderThan,
// and reconciles them against Razorpay.
func (u *RefundUsecase) ReconcileStuckRefunds(
	ctx context.Context,
	olderThan time.Duration,
	limit int,
) (int, error) {
	if olderThan <= 0 {
		olderThan = 5 * time.Minute
	}
	cutoff := time.Now().Add(-olderThan)

	var stuckRefunds []domain.Refund
	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			refs, err := tx.RefundRepository().FindStuckRefunds(ctx, cutoff, limit)
			if err != nil {
				return err
			}
			stuckRefunds = refs
			return nil
		},
	)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed_to_fetch_stuck_refunds", slog.Any("error", err))
		return 0, err
	}

	if len(stuckRefunds) == 0 {
		return 0, nil
	}

	u.logger.InfoContext(ctx, "reconciling_stuck_refunds",
		slog.Int("count", len(stuckRefunds)),
		slog.Duration("older_than", olderThan),
	)

	reconciledCount := 0
	for _, ref := range stuckRefunds {
		if err := ctx.Err(); err != nil {
			return reconciledCount, err
		}

		reconciled, err := u.reconcileSingleRefund(ctx, ref)
		if err != nil {
			u.logger.WarnContext(ctx, "reconcile_single_refund_failed",
				slog.String("refund_id", ref.ID.String()),
				slog.Any("error", err),
			)
			continue
		}
		if reconciled {
			reconciledCount++
		}
	}

	return reconciledCount, nil
}

func (u *RefundUsecase) reconcileSingleRefund(
	ctx context.Context,
	ref domain.Refund,
) (bool, error) {
	if u.refundProvider == nil {
		return false, errors.New("refund provider not configured")
	}

	var gatewayPaymentID string
	usedGatewayIDs := make(map[string]bool)

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			payment, err := tx.PaymentRepository().FindByID(ctx, ref.PaymentID.String())
			if err != nil {
				return err
			}
			if payment.TransactionID == nil || strings.TrimSpace(*payment.TransactionID) == "" {
				return errors.New("payment missing transaction ID")
			}
			gatewayPaymentID = strings.TrimSpace(*payment.TransactionID)

			allRefunds, err := tx.RefundRepository().FindByPaymentID(ctx, ref.PaymentID)
			if err == nil {
				for _, r := range allRefunds {
					if r.GatewayRefundID != nil && *r.GatewayRefundID != "" && r.ID != ref.ID {
						usedGatewayIDs[*r.GatewayRefundID] = true
					}
				}
			}

			return nil
		},
	)
	if err != nil {
		return false, err
	}

	gatewayRefunds, err := u.refundProvider.FetchRefunds(ctx, gatewayPaymentID)
	if err != nil {
		return false, err
	}

	var matched *domain.RefundResult

	// Check by GatewayRefundID if already set on record
	if ref.GatewayRefundID != nil && strings.TrimSpace(*ref.GatewayRefundID) != "" {
		for _, gr := range gatewayRefunds {
			if gr.RefundID == *ref.GatewayRefundID {
				matched = &gr
				break
			}
		}
	}

	// Otherwise match strictly by receipt/idempotency key without using already-claimed IDs
	if matched == nil {
		matched = findMatchingGatewayRefund(gatewayRefunds, ref.ID, ref.Amount, usedGatewayIDs)
	}

	now := time.Now()

	if matched != nil {
		finalStatus := matched.Status
		if strings.EqualFold(finalStatus, "processed") || strings.EqualFold(finalStatus, "success") {
			finalStatus = domain.RefundStatusSuccess
		} else if strings.EqualFold(finalStatus, "failed") {
			finalStatus = domain.RefundStatusFailed
		} else {
			finalStatus = domain.RefundStatusProcessing
		}

		return true, u.transactionManager.WithinTransaction(
			func(tx domain.TransactionRepositories) error {
				locked, err := tx.RefundRepository().FindByIDForUpdate(ctx, ref.ID)
				if err != nil {
					return err
				}
				// If status already resolved, do nothing.
				if locked.Status == domain.RefundStatusSuccess {
					return nil
				}
				locked.Status = finalStatus
				locked.GatewayRefundID = &matched.RefundID
				if finalStatus == domain.RefundStatusSuccess {
					locked.ProcessedAt = &now
				}
				locked.UpdatedAt = now
				if err := tx.RefundRepository().Update(ctx, locked); err != nil {
					return err
				}

				if finalStatus == domain.RefundStatusSuccess {
					if err := u.recordRefundJournal(ctx, tx, locked, now); err != nil {
						return err
					}
				}
				return nil
			},
		)
	}

	// If no match on gateway and refund was never processed (status is PENDING),
	// attempt to process it with gateway now.
	if ref.Status == domain.RefundStatusPending {
		_, err := u.ProcessPendingRefund(ctx, ref.ID)
		return err == nil, err
	}

	return false, nil
}

func (u *RefundUsecase) GetRefundByID(ctx context.Context, refundID uuid.UUID) (*domain.Refund, error) {
	var refund *domain.Refund
	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			r, err := tx.RefundRepository().FindByID(ctx, refundID)
			if err != nil {
				return err
			}
			refund = r
			return nil
		},
	)
	return refund, err
}

func (u *RefundUsecase) GetRefundsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Refund, error) {
	var refunds []domain.Refund
	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			refs, err := tx.RefundRepository().FindByBookingID(ctx, bookingID)
			if err != nil {
				return err
			}
			refunds = refs
			return nil
		},
	)
	return refunds, err
}

func findMatchingGatewayRefund(
	gatewayRefunds []domain.RefundResult,
	expectedRefundID uuid.UUID,
	expectedAmount decimal.Decimal,
	usedGatewayIDs map[string]bool,
) *domain.RefundResult {
	refundIDStr := strings.ToLower(expectedRefundID.String())

	for _, gr := range gatewayRefunds {
		// Never match a gateway refund that is already claimed by another refund record
		if usedGatewayIDs[gr.RefundID] {
			continue
		}

		// Strict receipt/idempotency key match with exact amount match.
		// Amount-only matching is strictly rejected to prevent false-positive matches.
		if strings.ToLower(strings.TrimSpace(gr.Receipt)) == refundIDStr && gr.Amount.Equal(expectedAmount) {
			copied := gr
			return &copied
		}
	}

	return nil
}

func (u *RefundUsecase) recordRefundJournal(
	ctx context.Context,
	tx domain.TransactionRepositories,
	targetRefund *domain.Refund,
	processedAt time.Time,
) error {
	if u.financeUsecase == nil {
		return nil
	}

	if targetRefund == nil || tx == nil {
		return nil
	}

	// Fetch parent payment
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

	// Fetch booking
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

	// Check if this is an escrow refund (captured payment on cancelled booking)
	isEscrowRefund := payment.Status == domain.PaymentStatusRefundRequired

	var eventID uuid.UUID
	var organizerID uuid.UUID
	var platformFeeRefund decimal.Decimal
	var taxRefund decimal.Decimal
	isEventCancellation := false

	if isEscrowRefund {
		// In escrow refund, funds flow from 2020 Customer Refund Payable to 1010 Clearing.
		// Organizer and platform fee accounts are untouched.
		eventID = booking.EventID
		if tx.EventRepository() != nil {
			if event, eventErr := tx.EventRepository().FindByID(booking.EventID); eventErr == nil && event != nil {
				organizerID = event.OrganizerID
			}
		}
	} else {
		// Event or customer cancellation refund
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
		Currency:            "INR",
		ProcessedAt:         processedAt,
	}

	if _, err := u.financeUsecase.RecordRefundWithinTx(ctx, tx, refundInput); err != nil {
		u.logger.ErrorContext(ctx, "failed to record refund journal entry",
			slog.String("refund_id", targetRefund.ID.String()),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to record refund journal: %w", err)
	}

	return nil
}

var _ RefundProcessor = (*RefundUsecase)(nil)
