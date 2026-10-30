package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminEventHandler struct {
	adminEventUsecase *events.AdminEventUsecase
	logger            *slog.Logger
}

func NewAdminEventHandler(
	adminEventUsecase *events.AdminEventUsecase,
	logger *slog.Logger,
) *AdminEventHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &AdminEventHandler{
		adminEventUsecase: adminEventUsecase,
		logger:            logger,
	}
}

func (h *AdminEventHandler) ListEvents(ctx *gin.Context) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil || parsedPage < 1 {
			ctx.Error(appErrors.NewValidationError("invalid page parameter"))
			return
		}
		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			ctx.Error(appErrors.NewValidationError("invalid limit parameter, must be between 1 and 100"))
			return
		}
		limit = parsedLimit
	}

	filter := domain.AdminEventFilter{
		Page:      page,
		Limit:     limit,
		Search:    strings.TrimSpace(ctx.Query("search")),
		Status:    strings.TrimSpace(ctx.Query("status")),
		EventType: strings.TrimSpace(ctx.Query("event_type")),
	}

	if catParam := strings.TrimSpace(ctx.Query("category_id")); catParam != "" {
		catID, err := uuid.Parse(catParam)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid category_id parameter"))
			return
		}
		filter.CategoryID = &catID
	}

	if orgParam := strings.TrimSpace(ctx.Query("organizer_id")); orgParam != "" {
		orgID, err := uuid.Parse(orgParam)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid organizer_id parameter"))
			return
		}
		filter.OrganizerID = &orgID
	}

	result, err := h.adminEventUsecase.ListEvents(ctx.Request.Context(), filter)
	if err != nil {
		h.logger.ErrorContext(ctx.Request.Context(), "admin_list_events_failed", "error", err)
		mapAdminEventError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func (h *AdminEventHandler) GetEventDetails(ctx *gin.Context) {
	idParam := strings.TrimSpace(ctx.Param("id"))
	eventID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid event id"))
		return
	}

	result, err := h.adminEventUsecase.GetEventDetails(ctx.Request.Context(), eventID)
	if err != nil {
		mapAdminEventError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

type AdminUpdateEventRequest struct {
	Title               *string    `json:"title"`
	Description         *string    `json:"description"`
	BannerURL           *string    `json:"banner_url"`
	Language            *string    `json:"language"`
	AgeRestriction      *int       `json:"age_restriction"`
	Highlights          *string    `json:"highlights"`
	Rules               *string    `json:"rules"`
	AttendeeInformation *string    `json:"attendee_information"`

	ContactName  *string `json:"contact_name"`
	ContactPhone *string `json:"contact_phone"`
	ContactEmail *string `json:"contact_email"`

	CategoryID *uuid.UUID `json:"category_id"`
	EventType  *string    `json:"event_type"`
	OnlineURL  *string    `json:"online_url"`
	VenueID    *uuid.UUID `json:"venue_id"`
	EventDate  *time.Time `json:"event_date"`
	StartTime  *time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
	IsAllDay   *bool      `json:"is_all_day"`

	SeatLayoutType      *string    `json:"seat_layout_type"`
	BookingLimitPerUser *int       `json:"booking_limit_per_user"`
	SalesStartDate      *time.Time `json:"sales_start_date"`
	SalesEndDate        *time.Time `json:"sales_end_date"`

	CancellationAllowed       *bool   `json:"cancellation_allowed"`
	CancellationDeadlineHours *int    `json:"cancellation_deadline_hours"`
	RefundPolicy              *string `json:"refund_policy"`
	RefundPercentage          *int    `json:"refund_percentage"`
}

func (h *AdminEventHandler) UpdateEvent(ctx *gin.Context) {
	adminIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}
	adminID, ok := adminIDVal.(uuid.UUID)
	if !ok {
		ctx.Error(appErrors.NewUnauthorizedError("invalid user context"))
		return
	}

	idParam := strings.TrimSpace(ctx.Param("id"))
	eventID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid event id"))
		return
	}

	var req AdminUpdateEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	input := domain.AdminUpdateEventInput{
		EventID: eventID,
	}

	if req.Title != nil {
		input.Title = *req.Title
	}
	if req.Description != nil {
		input.Description = *req.Description
	}
	if req.BannerURL != nil {
		input.BannerURL = *req.BannerURL
	}
	if req.Language != nil {
		input.Language = *req.Language
	}
	input.AgeRestriction = req.AgeRestriction
	input.Highlights = req.Highlights
	input.Rules = req.Rules
	input.AttendeeInformation = req.AttendeeInformation
	input.ContactName = req.ContactName
	input.ContactPhone = req.ContactPhone
	input.ContactEmail = req.ContactEmail

	input.CategoryID = req.CategoryID
	if req.EventType != nil {
		input.EventType = *req.EventType
	}
	if req.OnlineURL != nil {
		input.OnlineURL = *req.OnlineURL
	}
	input.VenueID = req.VenueID
	input.EventDate = req.EventDate
	input.StartTime = req.StartTime
	input.EndTime = req.EndTime
	input.IsAllDay = req.IsAllDay

	if req.SeatLayoutType != nil {
		input.SeatLayoutType = *req.SeatLayoutType
	}
	input.BookingLimitPerUser = req.BookingLimitPerUser
	input.SalesStartDate = req.SalesStartDate
	input.SalesEndDate = req.SalesEndDate

	input.CancellationAllowed = req.CancellationAllowed
	input.CancellationDeadlineHours = req.CancellationDeadlineHours
	if req.RefundPolicy != nil {
		input.RefundPolicy = *req.RefundPolicy
	}
	input.RefundPercentage = req.RefundPercentage

	result, err := h.adminEventUsecase.UpdateEvent(ctx.Request.Context(), adminID, input)
	if err != nil {
		mapAdminEventError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "event updated successfully",
		"data":    result,
	})
}

func (h *AdminEventHandler) PublishEvent(ctx *gin.Context) {
	adminIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}
	adminID, ok := adminIDVal.(uuid.UUID)
	if !ok {
		ctx.Error(appErrors.NewUnauthorizedError("invalid user context"))
		return
	}

	idParam := strings.TrimSpace(ctx.Param("id"))
	eventID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid event id"))
		return
	}

	if err := h.adminEventUsecase.PublishEvent(ctx.Request.Context(), adminID, eventID); err != nil {
		mapAdminEventError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "event published successfully",
	})
}

func (h *AdminEventHandler) DeleteEvent(ctx *gin.Context) {
	adminIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}
	adminID, ok := adminIDVal.(uuid.UUID)
	if !ok {
		ctx.Error(appErrors.NewUnauthorizedError("invalid user context"))
		return
	}

	idParam := strings.TrimSpace(ctx.Param("id"))
	eventID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid event id"))
		return
	}

	if err := h.adminEventUsecase.DeleteEvent(ctx.Request.Context(), adminID, eventID); err != nil {
		mapAdminEventError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "event deleted successfully",
	})
}

func mapAdminEventError(ctx *gin.Context, err error) {
	if errors.Is(err, domain.ErrEventNotFound) {
		ctx.Error(appErrors.NewNotFoundError("event not found"))
		return
	}
	if errors.Is(err, events.ErrEventCannotBeDeletedWithBookings) {
		ctx.Error(appErrors.NewConflictError("event cannot be deleted because it has existing bookings, tickets, or active reservations"))
		return
	}
	if errors.Is(err, events.ErrPublishedEventStructuralEdit) {
		ctx.Error(appErrors.NewValidationError(err.Error()))
		return
	}
	if errors.Is(err, events.ErrCancelledEventNotEditable) {
		ctx.Error(appErrors.NewValidationError(err.Error()))
		return
	}
	if errors.Is(err, events.ErrEventNotPublishable) {
		ctx.Error(appErrors.NewValidationError(err.Error()))
		return
	}
	if errors.Is(err, events.ErrUnauthorizedOrganizer) {
		ctx.Error(appErrors.NewValidationError("event organizer is not active"))
		return
	}
	if errors.Is(err, domain.ErrCategoryNotFound) {
		ctx.Error(appErrors.NewValidationError("category not found"))
		return
	}
	ctx.Error(appErrors.NewInternalError(err.Error()))
}
