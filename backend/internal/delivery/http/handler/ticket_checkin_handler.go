package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TicketCheckInHandler struct {
	checkInUseCase *ticket.CheckInUseCase
	logger         *slog.Logger
}

func NewTicketCheckInHandler(
	checkInUseCase *ticket.CheckInUseCase,
	logger *slog.Logger,
) *TicketCheckInHandler {
	return &TicketCheckInHandler{
		checkInUseCase: checkInUseCase,
		logger:         logger,
	}
}

type TicketCheckInRequest struct {
	QRPayload string `json:"qr_payload" binding:"required"`
}

type TicketCheckInResponse struct {
	TicketID   uuid.UUID `json:"ticket_id"`
	TicketCode string    `json:"ticket_code"`
	Status     string    `json:"status"`
	UsedAt     string    `json:"used_at"`
}

func (h *TicketCheckInHandler) CheckIn(ctx *gin.Context) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	eventID, err := uuid.Parse(ctx.Param("eventId"))
	if err != nil || eventID == uuid.Nil {
		ctx.Error(appErrors.NewValidationError("invalid event id"))
		return
	}

	var req TicketCheckInRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(appErrors.NewValidationError("QR payload is required"))
		return
	}

	if strings.TrimSpace(req.QRPayload) == "" {
		ctx.Error(appErrors.NewValidationError("QR payload is required"))
		return
	}

	checkedInTicket, err := h.checkInUseCase.CheckIn(
		ctx.Request.Context(),
		userID,
		eventID,
		req.QRPayload,
	)
	if err != nil {
		switch {
		case errors.Is(err, ticket.ErrInvalidCheckInRequest),
			errors.Is(err, ticket.ErrInvalidQRPayload):
			ctx.Error(appErrors.NewValidationError("invalid QR payload"))

		case errors.Is(err, ticket.ErrOrganizerNotAuthorized),
			errors.Is(err, domain.ErrOrganizerNotFound):
			ctx.Error(appErrors.NewForbiddenError(
				"not authorized to check in tickets for this event",
			))

		case errors.Is(err, domain.ErrEventNotFound),
			errors.Is(err, domain.ErrTicketNotFound):
			ctx.Error(appErrors.NewNotFoundError(
				"event or ticket not found",
			))

		case errors.Is(err, domain.ErrTicketNotValid):
			ctx.Error(appErrors.NewConflictError(
				"ticket is invalid or has already been used",
			))

		default:
			h.logger.Error(
				"ticket_check_in_handler_failed",
				"user_id", userID,
				"event_id", eventID,
				"error", err,
			)
			ctx.Error(err)
		}
		return
	}

	response := TicketCheckInResponse{
		TicketID:   checkedInTicket.ID,
		TicketCode: checkedInTicket.TicketCode,
		Status:     checkedInTicket.Status,
		UsedAt:     checkedInTicket.UsedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "ticket checked in successfully",
		"data":    response,
	})
}
