package checkout

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type CheckoutUsecase struct {
	transactionManager domain.TransactionManager
	logger             *slog.Logger
}

func NewCheckoutUsecase(
	transactionManager domain.TransactionManager,
	logger *slog.Logger,
) *CheckoutUsecase {
	return &CheckoutUsecase{
		transactionManager: transactionManager,
		logger:             logger,
	}
}

type CreateCheckoutRequest struct {
	ReservationID uuid.UUID
	UserID        uuid.UUID
}

func (u *CheckoutUsecase) CreateCheckout(
	ctx context.Context,
	req CreateCheckoutRequest,
) (*domain.Checkout, error) {
	if req.ReservationID == uuid.Nil || req.UserID == uuid.Nil {
		u.logger.WarnContext(ctx, "invalid checkout request",
			slog.String("reservation_id", req.ReservationID.String()),
			slog.String("user_id", req.UserID.String()),
		)
		return nil, domain.ErrInvalidReservationRequest
	}

	var checkout *domain.Checkout

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			now := time.Now()

			// Lock the reservation while validating and creating checkout.
			reservation, err := tx.SeatReservationRepository().
				FindByIDForUpdate(ctx, req.ReservationID)
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

			// Return the existing checkout if one already exists.
			existing, err := tx.CheckoutRepository().
				FindByReservationID(ctx, reservation.ID)
			if err == nil {
				if existing.UserID != req.UserID {
					return domain.ErrSeatReservationNotFound
				}

				if existing.Status != domain.CheckoutStatusActive ||
					!existing.ExpiresAt.After(now) {
					return domain.ErrInvalidReservationRequest
				}

				checkout = existing
				return nil
			}

			if !errors.Is(err, domain.ErrCheckoutNotFound) {
				return err
			}

			items, err := tx.SeatReservationRepository().
				FindItemsByReservationID(ctx, reservation.ID)
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return domain.ErrInvalidReservationRequest
			}

			pricingConfig, err := tx.PricingTaxConfigurationRepository().FindActiveAt(ctx, now)
			if err != nil {
				u.logger.ErrorContext(ctx, "failed to resolve active pricing configuration for checkout",
					slog.String("reservation_id", reservation.ID.String()),
					slog.String("error", err.Error()),
				)
				return domain.ErrPricingConfigurationUnavailable
			}

			pricingBreakdown, err := domain.CalculatePricing(items, pricingConfig)
			if err != nil {
				return err
			}

			checkout = &domain.Checkout{
				ID:             uuid.New(),
				ReservationID:  reservation.ID,
				UserID:         req.UserID,
				Subtotal:       pricingBreakdown.Subtotal,
				PlatformFee:    pricingBreakdown.PlatformFee,
				TaxAmount:      pricingBreakdown.TaxAmount,
				DiscountAmount: pricingBreakdown.DiscountAmount,
				TotalAmount:    pricingBreakdown.TotalAmount,
				Status:         domain.CheckoutStatusActive,
				ExpiresAt:      reservation.ExpiresAt,
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			if err := tx.CheckoutRepository().Create(ctx, checkout); err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		u.logger.ErrorContext(ctx, "checkout creation failed",
			slog.String("reservation_id", req.ReservationID.String()),
			slog.String("user_id", req.UserID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "checkout created or retrieved",
		slog.String("checkout_id", checkout.ID.String()),
		slog.String("reservation_id", checkout.ReservationID.String()),
		slog.String("user_id", checkout.UserID.String()),
		slog.String("subtotal", checkout.Subtotal.String()),
		slog.String("total_amount", checkout.TotalAmount.String()),
		slog.String("status", checkout.Status),
	)

	return checkout, nil
}
