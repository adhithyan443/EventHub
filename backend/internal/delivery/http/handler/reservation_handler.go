package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/reservation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReservationHandler struct {
	reservationUsecase *reservation.ReservationUsecase
	logger             *slog.Logger
}

func NewReservationHandler(
	usecase *reservation.ReservationUsecase,
	logger *slog.Logger,
) *ReservationHandler {
	return &ReservationHandler{
		reservationUsecase: usecase,
		logger:             logger,
	}
}

type createReservationBody struct {
	SeatIDs      []string `json:"seat_ids"`
	TicketTypeID *string  `json:"ticket_type_id"`
	Quantity     int      `json:"quantity"`
}

func (h *ReservationHandler) CreateReservation(c *gin.Context) {
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

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil || eventID == uuid.Nil {
		c.Error(appErrors.NewValidationError("invalid event ID"))
		return
	}

	var body createReservationBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	req := reservation.CreateReservationRequest{
		UserID:   userID,
		EventID:  eventID,
		Quantity: body.Quantity,
	}

	for _, rawID := range body.SeatIDs {
		seatID, err := uuid.Parse(rawID)
		if err != nil || seatID == uuid.Nil {
			c.Error(appErrors.NewValidationError("invalid seat ID"))
			return
		}
		req.SeatIDs = append(req.SeatIDs, seatID)
	}

	if body.TicketTypeID != nil {
		ticketTypeID, err := uuid.Parse(*body.TicketTypeID)
		if err != nil || ticketTypeID == uuid.Nil {
			c.Error(appErrors.NewValidationError("invalid ticket type ID"))
			return
		}
		req.TicketTypeID = &ticketTypeID
	}

	ctx := c.Request.Context()

	h.logger.InfoContext(ctx, "creating reservation",
		slog.String("user_id", userID.String()),
		slog.String("event_id", eventID.String()),
	)

	result, err := h.reservationUsecase.CreateReservation(ctx, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidReservationRequest):
			c.Error(appErrors.NewValidationError("invalid reservation request"))

		case errors.Is(err, domain.ErrEventNotFound),
			errors.Is(err, domain.ErrSeatNotFound),
			errors.Is(err, domain.ErrSeatLayoutNotFound),
			errors.Is(err, domain.ErrTicketTypeNotFound),
			errors.Is(err, domain.ErrEventNotPublic):
			c.Error(appErrors.NewNotFoundError("event or requested inventory not found"))

		case errors.Is(err, domain.ErrSeatNotAvailable),
			errors.Is(err, domain.ErrInsufficientTicketQuantity):
			c.Error(appErrors.NewConflictError("requested inventory is no longer available"))

		case errors.Is(err, domain.ErrTicketTypeEventMismatch):
			c.Error(appErrors.NewValidationError("ticket type does not belong to this event"))

		default:
			h.logger.ErrorContext(ctx, "reservation creation failed",
				slog.String("user_id", userID.String()),
				slog.String("event_id", eventID.String()),
				slog.String("error", err.Error()),
			)
			c.Error(err)
		}
		return
	}

	h.logger.InfoContext(ctx, "reservation created",
		slog.String("reservation_id", result.ID.String()),
		slog.String("user_id", userID.String()),
		slog.String("event_id", eventID.String()),
	)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"reservation": result,
		},
	})
}

func (h *ReservationHandler) CancelReservation(c *gin.Context) {
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

	reservationID, err := uuid.Parse(c.Param("reservationId"))
	if err != nil || reservationID == uuid.Nil {
		c.Error(appErrors.NewValidationError("invalid reservation ID"))
		return
	}

	ctx := c.Request.Context()

	h.logger.InfoContext(ctx, "cancelling reservation",
		slog.String("reservation_id", reservationID.String()),
		slog.String("user_id", userID.String()),
	)

	if err := h.reservationUsecase.CancelReservation(
		ctx,
		reservationID,
		userID,
	); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidReservationRequest):
			c.Error(appErrors.NewValidationError("invalid reservation request"))

		case errors.Is(err, domain.ErrSeatReservationNotFound):
			c.Error(appErrors.NewNotFoundError("active reservation not found"))

		default:
			h.logger.ErrorContext(ctx, "reservation cancellation failed",
				slog.String("reservation_id", reservationID.String()),
				slog.String("user_id", userID.String()),
				slog.String("error", err.Error()),
			)
			c.Error(err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "reservation cancelled successfully",
	})
}
