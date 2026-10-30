package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/checkout"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CheckoutHandler struct {
	checkoutUsecase *checkout.CheckoutUsecase
	logger          *slog.Logger
}

func NewCheckoutHandler(
	usecase *checkout.CheckoutUsecase,
	logger *slog.Logger,
) *CheckoutHandler {
	return &CheckoutHandler{
		checkoutUsecase: usecase,
		logger:          logger,
	}
}

type createCheckoutBody struct {
	ReservationID string `json:"reservationId"`
}

func (h *CheckoutHandler) CreateCheckout(c *gin.Context) {
	userValue, exists := c.Get("user_id")
	if !exists {
		c.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	userID, ok := userValue.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		c.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	var body createCheckoutBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	reservationID, err := uuid.Parse(body.ReservationID)
	if err != nil || reservationID == uuid.Nil {
		c.Error(appErrors.NewValidationError("invalid reservation ID"))
		return
	}

	ctx := c.Request.Context()

	h.logger.InfoContext(ctx, "creating checkout",
		slog.String("user_id", userID.String()),
		slog.String("reservation_id", reservationID.String()),
	)

	result, err := h.checkoutUsecase.CreateCheckout(ctx, checkout.CreateCheckoutRequest{
		UserID:        userID,
		ReservationID: reservationID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidReservationRequest):
			c.Error(appErrors.NewValidationError("invalid or expired reservation"))

		case errors.Is(err, domain.ErrSeatReservationNotFound),
			errors.Is(err, domain.ErrCheckoutNotFound):
			c.Error(appErrors.NewNotFoundError("reservation not found"))

		case errors.Is(err, domain.ErrPricingConfigurationUnavailable):
			h.logger.ErrorContext(ctx, "checkout creation halted: pricing configuration unavailable",
				slog.String("user_id", userID.String()),
				slog.String("reservation_id", reservationID.String()),
			)
			c.Error(appErrors.NewInternalError("pricing configuration is temporarily unavailable"))

		default:
			h.logger.ErrorContext(ctx, "checkout creation failed",
				slog.String("user_id", userID.String()),
				slog.String("reservation_id", reservationID.String()),
				slog.String("error", err.Error()),
			)
			c.Error(err)
		}
		return
	}

	h.logger.InfoContext(ctx, "checkout created",
		slog.String("checkout_id", result.ID.String()),
		slog.String("reservation_id", result.ReservationID.String()),
		slog.String("user_id", userID.String()),
		slog.String("status", result.Status),
	)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"checkout": result,
		},
	})
}
