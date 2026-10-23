package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	paymentUC "github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	paymentUsecase *paymentUC.PaymentUsecase
	razorpayKeyID  string
	logger         *slog.Logger
}

func NewPaymentHandler(
	usecase *paymentUC.PaymentUsecase,
	razorpayKeyID string,
	logger *slog.Logger,
) *PaymentHandler {
	return &PaymentHandler{
		paymentUsecase: usecase,
		razorpayKeyID:  razorpayKeyID,
		logger:         logger,
	}
}

type createPaymentBody struct {
	CheckoutID string `json:"checkoutId" binding:"required"`
	Method     string `json:"method" binding:"required"`
}

type verifyPaymentBody struct {
	ProviderOrderID   string `json:"providerOrderId" binding:"required"`
	RazorpayPaymentID string `json:"razorpayPaymentId" binding:"required"`
	RazorpaySignature string `json:"razorpaySignature" binding:"required"`
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	// Obtain the user identity from JWT middleware.
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

	var body createPaymentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	checkoutID, err := uuid.Parse(body.CheckoutID)
	if err != nil || checkoutID == uuid.Nil {
		c.Error(appErrors.NewValidationError("invalid checkout ID"))
		return
	}

	method := domain.PaymentMethod(strings.ToUpper(strings.TrimSpace(body.Method)))

	switch method {
	case domain.PaymentMethodUPI,
		domain.PaymentMethodCard,
		domain.PaymentMethodNetBanking,
		domain.PaymentMethodWallet:
		// Supported method.
	default:
		c.Error(appErrors.NewValidationError("unsupported payment method"))
		return
	}

	ctx := c.Request.Context()

	h.logger.InfoContext(ctx, "creating payment",
		slog.String("user_id", userID.String()),
		slog.String("checkout_id", checkoutID.String()),
		slog.String("method", string(method)),
	)

	result, err := h.paymentUsecase.CreatePayment(
		ctx,
		paymentUC.CreatePaymentRequest{
			CheckoutID: checkoutID,
			UserID:     userID,
			Method:     method,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidPayment),
			errors.Is(err, domain.ErrUnsupportedPaymentMethod),
			errors.Is(err, domain.ErrInvalidReservationRequest):
			c.Error(appErrors.NewValidationError("invalid payment request"))

		case errors.Is(err, domain.ErrCheckoutNotPayable),
			errors.Is(err, domain.ErrPaymentAlreadyComplete),
			errors.Is(err, domain.ErrPaymentInProgress),
			errors.Is(err, domain.ErrPaymentAmountMismatch):
			c.Error(appErrors.NewValidationError(err.Error()))

		case errors.Is(err, domain.ErrCheckoutNotFound),
			errors.Is(err, domain.ErrSeatReservationNotFound),
			errors.Is(err, domain.ErrBookingNotFound),
			errors.Is(err, domain.ErrPaymentNotFound):
			c.Error(appErrors.NewNotFoundError("payment resource not found"))

		default:
			h.logger.ErrorContext(ctx, "payment creation failed",
				slog.String("user_id", userID.String()),
				slog.String("checkout_id", checkoutID.String()),
				slog.Any("error", err),
			)
			c.Error(err)
		}
		return
	}

	h.logger.InfoContext(ctx, "payment order created",
		slog.String("payment_id", result.ID),
		slog.String("user_id", userID.String()),
		slog.String("provider", string(result.Provider)),
	)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"payment": result,
			"keyId":   h.razorpayKeyID,
		},
	})
}

func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
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

	var body verifyPaymentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(appErrors.NewValidationError("invalid verification request"))
		return
	}

	body.ProviderOrderID = strings.TrimSpace(body.ProviderOrderID)
	body.RazorpayPaymentID = strings.TrimSpace(body.RazorpayPaymentID)
	body.RazorpaySignature = strings.TrimSpace(body.RazorpaySignature)

	ctx := c.Request.Context()

	result, err := h.paymentUsecase.VerifyPayment(
		ctx,
		paymentUC.VerifyPaymentRequest{
			UserID:            userID,
			ProviderOrderID:   body.ProviderOrderID,
			RazorpayPaymentID: body.RazorpayPaymentID,
			RazorpaySignature: body.RazorpaySignature,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidPayment),
			errors.Is(err, domain.ErrPaymentAmountMismatch):
			c.Error(appErrors.NewValidationError("payment verification failed"))

		case errors.Is(err, domain.ErrReservationExpired):
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "Reservation expired before payment completion. Your payment has been safely flagged for a refund.",
			})

		case errors.Is(err, domain.ErrPaymentAlreadyComplete):
			c.Error(appErrors.NewValidationError("payment already completed"))

		case errors.Is(err, domain.ErrPaymentNotFound),
			errors.Is(err, domain.ErrBookingNotFound):
			c.Error(appErrors.NewNotFoundError("payment resource not found"))

		default:
			h.logger.ErrorContext(ctx, "payment verification failed",
				slog.String("user_id", userID.String()),
				slog.Any("error", err),
			)
			c.Error(err)
		}
		return
	}

	h.logger.InfoContext(ctx, "payment verification completed",
		slog.String("payment_id", result.Payment.ID),
		slog.String("user_id", userID.String()),
		slog.String("status", string(result.Payment.Status)),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"payment": result.Payment,
			"booking": result.Booking,
		},
	})
}

func (h *PaymentHandler) RazorpayWebhook(c *gin.Context) {
	const maxWebhookBodySize = 1 << 20 // 1 MB

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		maxWebhookBodySize,
	)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "invalid Razorpay webhook body",
			slog.Any("error", err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook body"})
		return
	}

	signature := c.GetHeader("X-Razorpay-Signature")
	if !h.paymentUsecase.VerifyWebhookSignature(body, signature) {
		h.logger.WarnContext(c.Request.Context(), "Razorpay webhook signature verification failed")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	eventID := c.GetHeader("X-Razorpay-Event-Id")

	if err := h.paymentUsecase.ProcessRazorpayWebhook(
		c.Request.Context(),
		body,
		signature,
		eventID,
	); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidPayment):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook payload"})
		case errors.Is(err, domain.ErrPaymentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		case errors.Is(err, domain.ErrPaymentAmountMismatch):
			c.JSON(http.StatusBadRequest, gin.H{"error": "payment amount mismatch"})
		case errors.Is(err, domain.ErrReservationExpired):
			c.JSON(http.StatusOK, gin.H{
				"message": "payment captured after reservation expiry; refund required",
			})

		case errors.Is(err, domain.ErrPaymentAlreadyComplete):
			c.JSON(http.StatusOK, gin.H{"message": "payment already complete"})
		default:
			h.logger.ErrorContext(c.Request.Context(), "Razorpay webhook processing failed",
				slog.String("event_id", eventID),
				slog.Any("error", err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "webhook processing failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook processed"})
}
