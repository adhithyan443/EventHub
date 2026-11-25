package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

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

type ManualBookingCheckInRequest struct {
	BookingReference string `json:"booking_reference" binding:"required"`
}

type TicketSummaryResponse struct {
	TicketID   uuid.UUID `json:"ticket_id"`
	TicketCode string    `json:"ticket_code"`
	Status     string    `json:"status"`
	UsedAt     string    `json:"used_at"`
}

type TicketCheckInResponse struct {
	Type       string                  `json:"type"`
	TicketID   uuid.UUID               `json:"ticket_id"`
	TicketCode string                  `json:"ticket_code"`
	Status     string                  `json:"status"`
	UsedAt     string                  `json:"used_at"`
	BookingID  *uuid.UUID              `json:"booking_id,omitempty"`
	TotalCount int                     `json:"total_count"`
	Tickets    []TicketSummaryResponse `json:"tickets,omitempty"`
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

	checkInResult, err := h.checkInUseCase.CheckIn(
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
			errors.Is(err, domain.ErrTicketNotFound),
			errors.Is(err, domain.ErrBookingNotFound):
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

	ticketSummaries := make([]TicketSummaryResponse, len(checkInResult.Tickets))
	for i, t := range checkInResult.Tickets {
		usedAtStr := ""
		if t.UsedAt != nil {
			usedAtStr = t.UsedAt.UTC().Format(time.RFC3339)
		}
		ticketSummaries[i] = TicketSummaryResponse{
			TicketID:   t.ID,
			TicketCode: t.TicketCode,
			Status:     t.Status,
			UsedAt:     usedAtStr,
		}
	}

	primaryTicket := checkInResult.Ticket
	primaryUsedAt := ""
	if primaryTicket.UsedAt != nil {
		primaryUsedAt = primaryTicket.UsedAt.UTC().Format(time.RFC3339)
	}

	response := TicketCheckInResponse{
		Type:       string(checkInResult.Type),
		BookingID:  checkInResult.BookingID,
		TotalCount: checkInResult.TotalCount,
		TicketID:   primaryTicket.ID,
		TicketCode: primaryTicket.TicketCode,
		Status:     primaryTicket.Status,
		UsedAt:     primaryUsedAt,
		Tickets:    ticketSummaries,
	}

	message := "ticket checked in successfully"
	if checkInResult.Type == ticket.CheckInTypeBooking {
		if checkInResult.AlreadyUsedCount > 0 {
			message = fmt.Sprintf("booking checked in successfully: %d tickets checked in (%d ticket(s) were already checked in)", checkInResult.CheckedInCount, checkInResult.AlreadyUsedCount)
		} else {
			message = fmt.Sprintf("booking checked in successfully (%d tickets)", checkInResult.TotalCount)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    response,
	})
}

// LookupManualBooking queries a booking by human-readable reference and returns admission details.
func (h *TicketCheckInHandler) LookupManualBooking(ctx *gin.Context) {
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

	var req ManualBookingCheckInRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(appErrors.NewValidationError("booking reference is required"))
		return
	}

	if strings.TrimSpace(req.BookingReference) == "" {
		ctx.Error(appErrors.NewValidationError("booking reference is required"))
		return
	}

	lookupResult, err := h.checkInUseCase.LookupManualBooking(
		ctx.Request.Context(),
		userID,
		eventID,
		req.BookingReference,
	)
	if err != nil {
		switch {
		case errors.Is(err, ticket.ErrInvalidCheckInRequest):
			ctx.Error(appErrors.NewValidationError("invalid booking reference request"))

		case errors.Is(err, ticket.ErrOrganizerNotAuthorized),
			errors.Is(err, domain.ErrOrganizerNotFound):
			ctx.Error(appErrors.NewForbiddenError(
				"not authorized to inspect bookings for this event",
			))

		case errors.Is(err, domain.ErrEventNotFound),
			errors.Is(err, domain.ErrBookingNotFound):
			ctx.Error(appErrors.NewNotFoundError(
				"booking not found for this event",
			))

		default:
			h.logger.Error(
				"manual_booking_lookup_failed",
				"user_id", userID,
				"event_id", eventID,
				"error", err,
			)
			ctx.Error(err)
		}
		return
	}

	message := "booking reference verified"
	if !lookupResult.IsEligible {
		message = "booking reference retrieved but not eligible for check-in"
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    lookupResult,
	})
}

// ConfirmManualBooking confirms manual check-in of eligible tickets for a booking reference.
func (h *TicketCheckInHandler) ConfirmManualBooking(ctx *gin.Context) {
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

	var req ManualBookingCheckInRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(appErrors.NewValidationError("booking reference is required"))
		return
	}

	if strings.TrimSpace(req.BookingReference) == "" {
		ctx.Error(appErrors.NewValidationError("booking reference is required"))
		return
	}

	checkInResult, err := h.checkInUseCase.ConfirmManualBooking(
		ctx.Request.Context(),
		userID,
		eventID,
		req.BookingReference,
	)
	if err != nil {
		switch {
		case errors.Is(err, ticket.ErrInvalidCheckInRequest):
			ctx.Error(appErrors.NewValidationError("invalid booking reference request"))

		case errors.Is(err, ticket.ErrOrganizerNotAuthorized),
			errors.Is(err, domain.ErrOrganizerNotFound):
			ctx.Error(appErrors.NewForbiddenError(
				"not authorized to check in tickets for this event",
			))

		case errors.Is(err, domain.ErrEventNotFound),
			errors.Is(err, domain.ErrBookingNotFound):
			ctx.Error(appErrors.NewNotFoundError(
				"booking not found for this event",
			))

		case errors.Is(err, ticket.ErrBookingAlreadyCheckedIn):
			ctx.Error(appErrors.NewConflictError(
				"all tickets for this booking have already been checked in",
			))

		case errors.Is(err, ticket.ErrBookingNotEligible):
			ctx.Error(appErrors.NewConflictError(
				"booking is not eligible for check-in",
			))

		case errors.Is(err, domain.ErrTicketNotValid):
			ctx.Error(appErrors.NewConflictError(
				"tickets for this booking have already been checked in or are invalid",
			))

		default:
			h.logger.Error(
				"manual_booking_confirm_failed",
				"user_id", userID,
				"event_id", eventID,
				"error", err,
			)
			ctx.Error(err)
		}
		return
	}

	ticketSummaries := make([]TicketSummaryResponse, len(checkInResult.Tickets))
	for i, t := range checkInResult.Tickets {
		usedAtStr := ""
		if t.UsedAt != nil {
			usedAtStr = t.UsedAt.UTC().Format(time.RFC3339)
		}
		ticketSummaries[i] = TicketSummaryResponse{
			TicketID:   t.ID,
			TicketCode: t.TicketCode,
			Status:     t.Status,
			UsedAt:     usedAtStr,
		}
	}

	primaryTicket := checkInResult.Ticket
	primaryUsedAt := ""
	if primaryTicket != nil && primaryTicket.UsedAt != nil {
		primaryUsedAt = primaryTicket.UsedAt.UTC().Format(time.RFC3339)
	}

	var primaryTicketID uuid.UUID
	var primaryTicketCode, primaryStatus string
	if primaryTicket != nil {
		primaryTicketID = primaryTicket.ID
		primaryTicketCode = primaryTicket.TicketCode
		primaryStatus = primaryTicket.Status
	}

	response := TicketCheckInResponse{
		Type:       string(checkInResult.Type),
		BookingID:  checkInResult.BookingID,
		TotalCount: checkInResult.TotalCount,
		TicketID:   primaryTicketID,
		TicketCode: primaryTicketCode,
		Status:     primaryStatus,
		UsedAt:     primaryUsedAt,
		Tickets:    ticketSummaries,
	}

	message := fmt.Sprintf("booking checked in successfully (%d tickets)", checkInResult.CheckedInCount)
	if checkInResult.AlreadyUsedCount > 0 {
		message = fmt.Sprintf("booking checked in successfully: %d tickets checked in (%d ticket(s) were already checked in)", checkInResult.CheckedInCount, checkInResult.AlreadyUsedCount)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    response,
	})
}
