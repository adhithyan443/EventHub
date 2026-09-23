package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	eventUsecase "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxEventBannerSize = 5 * 1024 * 1024 // 5 MB

type EventHandler struct {
	eventUsecase *eventUsecase.EventUsecase
	logger       *slog.Logger
}

func NewEventHandler(
	eventUsecase *eventUsecase.EventUsecase,
	logger *slog.Logger,
) *EventHandler {
	return &EventHandler{
		eventUsecase: eventUsecase,
		logger:       logger,
	}
}

type CreateEventRequest struct {
	CategoryID uuid.UUID `json:"category_id"`

	EventType string `json:"event_type"`
	OnlineURL string `json:"online_url"`

	Title          string `json:"title"`
	Description    string `json:"description"`
	BannerURL      string `json:"banner_url"`
	Language       string `json:"language"`
	AgeRestriction int    `json:"age_restriction"`

	EventDate string `json:"event_date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`

	SeatLayoutType      string `json:"seat_layout_type"`
	BookingLimitPerUser int    `json:"booking_limit_per_user"`

	CancellationAllowed       bool `json:"cancellation_allowed"`
	CancellationDeadlineHours int  `json:"cancellation_deadline_hours"`

	Venue VenueRequest `json:"venue"`

	Contact             EventContactRequest `json:"contact"`
	Visibility          string              `json:"visibility"`
	Highlights          string              `json:"highlights"`
	Rules               string              `json:"rules"`
	AttendeeInformation string              `json:"attendee_information"`

	IsAllDay bool `json:"is_all_day"`

	SalesStartDate string `json:"sales_start_date"`
	SalesEndDate   string `json:"sales_end_date"`

	RefundPolicy     string `json:"refund_policy"`
	RefundPercentage int    `json:"refund_percentage"`
}

type UpdateEventRequest struct {
	CategoryID          uuid.UUID    `json:"category_id"`
	VenueID             *uuid.UUID   `json:"venue_id"`
	Venue               VenueRequest `json:"venue"`
	EventType           string       `json:"event_type"`
	OnlineURL           string       `json:"online_url"`
	Title               string       `json:"title"`
	Description         string       `json:"description"`
	Language            string       `json:"language"`
	AgeRestriction      int          `json:"age_restriction"`
	Visibility          string       `json:"visibility"`
	Highlights          string       `json:"highlights"`
	Rules               string       `json:"rules"`
	AttendeeInformation string       `json:"attendee_information"`

	EventDate string `json:"event_date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	IsAllDay  bool   `json:"is_all_day"`

	SeatLayoutType      string `json:"seat_layout_type"`
	BookingLimitPerUser int    `json:"booking_limit_per_user"`
	SalesStartDate      string `json:"sales_start_date"`
	SalesEndDate        string `json:"sales_end_date"`

	CancellationAllowed       bool   `json:"cancellation_allowed"`
	CancellationDeadlineHours int    `json:"cancellation_deadline_hours"`
	RefundPolicy              string `json:"refund_policy"`
	RefundPercentage          int    `json:"refund_percentage"`

	Contact EventContactRequest `json:"contact"`
}

type EventContactRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type VenueRequest struct {
	GooglePlaceID string  `json:"google_place_id"`
	Name          string  `json:"name"`
	Address       string  `json:"address"`
	City          string  `json:"city"`
	State         string  `json:"state"`
	Country       string  `json:"country"`
	PostalCode    string  `json:"postal_code"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
}

func (h *EventHandler) CreateEvent(ctx *gin.Context) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	h.logger.Info(
		"event_multipart_request_received",
		"user_id", userID,
		"content_length", ctx.Request.ContentLength,
	)

	// 1. Read "event" JSON field from multipart form.
	eventJSON := ctx.Request.FormValue("event")
	if eventJSON == "" {
		eventJSON = ctx.PostForm("event")
	}

	if eventJSON == "" {
		h.logger.Warn(
			"event_create_missing_event_payload",
			"user_id", userID,
		)

		ctx.Error(
			appErrors.NewValidationError(
				"event data is required",
			),
		)
		return
	}

	var req CreateEventRequest
	if err := json.Unmarshal([]byte(eventJSON), &req); err != nil {
		h.logger.Warn(
			"event_create_invalid_event_json",
			"user_id", userID,
			"error", err,
		)

		ctx.Error(
			appErrors.NewValidationError(
				"invalid event data",
			),
		)
		return
	}

	// 2. Read "banner" file from multipart form.
	bannerHeader, err := ctx.FormFile("banner")
	if err != nil {
		h.logger.Warn(
			"event_banner_validation_failed",
			"user_id", userID,
			"error", err,
		)

		ctx.Error(
			appErrors.NewValidationError(
				"banner image is required",
			),
		)
		return
	}

	if bannerHeader.Size > maxEventBannerSize {
		h.logger.Warn(
			"event_banner_validation_failed",
			"user_id", userID,
			"file_size", bannerHeader.Size,
			"error", "file exceeds maximum size of 5 MB",
		)

		ctx.Error(
			appErrors.NewValidationError(
				"banner image must not exceed 5 MB",
			),
		)
		return
	}

	bannerFile, err := bannerHeader.Open()
	if err != nil {
		h.logger.Error(
			"event_banner_open_failed",
			"user_id", userID,
			"error", err,
		)

		ctx.Error(err)
		return
	}
	defer bannerFile.Close()

	contentType, err := detectBannerContentType(bannerFile)
	if err != nil {
		h.logger.Warn(
			"event_banner_validation_failed",
			"user_id", userID,
			"error", err,
		)

		ctx.Error(
			appErrors.NewValidationError(
				"invalid banner image",
			),
		)
		return
	}

	if contentType != "image/jpeg" &&
		contentType != "image/png" &&
		contentType != "image/webp" {
		h.logger.Warn(
			"event_banner_validation_failed",
			"user_id", userID,
			"content_type", contentType,
			"error", "unsupported content type",
		)

		ctx.Error(
			appErrors.NewValidationError(
				"only JPEG, PNG, and WebP images are allowed",
			),
		)
		return
	}

	eventDate, err := time.Parse(
		"2006-01-02",
		req.EventDate,
	)
	if err != nil {
		h.logger.Warn(
			"event_create_invalid_event_date",
			"user_id", userID,
			"event_date", req.EventDate,
			"error", err,
		)

		ctx.Error(
			appErrors.NewValidationError(
				"invalid event date",
			),
		)
		return
	}

	salesStartDate, err := parseOptionalDate(req.SalesStartDate)
	if err != nil {
		h.logger.Warn(
			"event_create_invalid_sales_start_date",
			"user_id", userID,
			"sales_start_date", req.SalesStartDate,
			"error", err,
		)

		ctx.Error(
			appErrors.NewValidationError(
				"invalid sales start date",
			),
		)
		return
	}

	salesEndDate, err := parseOptionalDate(req.SalesEndDate)
	if err != nil {
		h.logger.Warn(
			"event_create_invalid_sales_end_date",
			"user_id", userID,
			"sales_end_date", req.SalesEndDate,
			"error", err,
		)

		ctx.Error(
			appErrors.NewValidationError(
				"invalid sales end date",
			),
		)
		return
	}

	var startTime time.Time
	var endTime time.Time

	if !req.IsAllDay {
		startTime, err = time.Parse("15:04", req.StartTime)
		if err != nil {
			h.logger.Warn(
				"event_create_invalid_start_time",
				"user_id", userID,
				"start_time", req.StartTime,
				"error", err,
			)

			ctx.Error(
				appErrors.NewValidationError(
					"invalid start time",
				),
			)
			return
		}

		endTime, err = time.Parse("15:04", req.EndTime)
		if err != nil {
			h.logger.Warn(
				"event_create_invalid_end_time",
				"user_id", userID,
				"end_time", req.EndTime,
				"error", err,
			)

			ctx.Error(
				appErrors.NewValidationError(
					"invalid end time",
				),
			)
			return
		}
	}

	input := eventUsecase.CreateEventInput{
		UserID:     userID,
		CategoryID: req.CategoryID,

		EventType: req.EventType,
		OnlineURL: req.OnlineURL,

		Title:          req.Title,
		Description:    req.Description,
		BannerURL:      req.BannerURL,
		Language:       req.Language,
		AgeRestriction: req.AgeRestriction,

		EventDate: eventDate,
		StartTime: startTime,
		EndTime:   endTime,

		SeatLayoutType:      req.SeatLayoutType,
		BookingLimitPerUser: req.BookingLimitPerUser,

		CancellationAllowed:       req.CancellationAllowed,
		CancellationDeadlineHours: req.CancellationDeadlineHours,

		Venue: eventUsecase.VenueInput{
			GooglePlaceID: req.Venue.GooglePlaceID,
			Name:          req.Venue.Name,
			Address:       req.Venue.Address,
			City:          req.Venue.City,
			State:         req.Venue.State,
			Country:       req.Venue.Country,
			PostalCode:    req.Venue.PostalCode,
			Latitude:      req.Venue.Latitude,
			Longitude:     req.Venue.Longitude,
		},

		Contact: eventUsecase.EventContactInput{
			Name:  req.Contact.Name,
			Phone: req.Contact.Phone,
			Email: req.Contact.Email,
		},

		Visibility:          req.Visibility,
		Highlights:          req.Highlights,
		Rules:               req.Rules,
		AttendeeInformation: req.AttendeeInformation,

		IsAllDay: req.IsAllDay,

		SalesStartDate:   salesStartDate,
		SalesEndDate:     salesEndDate,
		RefundPolicy:     req.RefundPolicy,
		RefundPercentage: req.RefundPercentage,

		BannerReader:      bannerFile,
		BannerContentType: contentType,
		BannerSize:        bannerHeader.Size,
	}

	output, err := h.eventUsecase.CreateEvent(ctx.Request.Context(), input)
	if err != nil {
		h.logger.Warn(
			"event_create_request_failed",
			"user_id", userID,
			"event_type", req.EventType,
			"error", err,
		)

		switch {
		case errors.Is(
			err,
			eventUsecase.ErrInvalidEventInput,
		):
			ctx.Error(
				appErrors.NewValidationError(
					"invalid event data",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrInvalidBanner,
		):
			ctx.Error(
				appErrors.NewValidationError(
					"invalid event banner",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrInvalidEventSchedule,
		):
			ctx.Error(
				appErrors.NewValidationError(
					"invalid event schedule",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrInvalidEventSettings,
		):
			ctx.Error(
				appErrors.NewValidationError(
					"invalid event settings",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrInvalidCancellation,
		):
			ctx.Error(
				appErrors.NewValidationError(
					"invalid cancellation settings",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrUnauthorizedOrganizer,
		):
			ctx.Error(
				appErrors.NewForbiddenError(
					"only active organizers can create events",
				),
			)

		case errors.Is(
			err,
			domain.ErrCategoryNotFound,
		):
			ctx.Error(
				appErrors.NewNotFoundError(
					"category not found",
				),
			)

		default:
			ctx.Error(err)
		}

		return
	}

	h.logger.Info(
		"event_create_request_completed",
		"user_id", userID,
		"event_id", output.Event.ID,
		"event_type", output.Event.EventType,
		"status", output.Event.Status,
	)

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"event":        output.Event,
			"schedule":     output.Schedule,
			"setting":      output.Setting,
			"cancellation": output.Cancellation,
			"contact":      output.Contact,
		},
	})
}

func (h *EventHandler) UpdateEvent(c *gin.Context) {
	userID, err := getAuthenticatedUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.logger.Warn(
			"event_update_invalid_event_id",
			"event_id", c.Param("id"),
			"user_id", userID,
			"error", err,
		)

		c.Error(
			appErrors.NewValidationError(
				"invalid event id",
			),
		)
		return
	}

	var req UpdateEventRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(
			"event_update_invalid_request",
			"event_id", eventID,
			"user_id", userID,
			"error", err,
		)

		c.Error(
			appErrors.NewValidationError(
				"invalid event data",
			),
		)
		return
	}

	eventDate, err := parseDate(req.EventDate)
	if err != nil {
		h.logger.Warn(
			"event_update_invalid_event_date",
			"user_id", userID,
			"event_date", req.EventDate,
			"error", err,
		)

		c.Error(
			appErrors.NewValidationError(
				"invalid event date",
			),
		)
		return
	}

	salesStartDate, err := parseOptionalDate(req.SalesStartDate)
	if err != nil {
		h.logger.Warn(
			"event_update_invalid_sales_start_date",
			"user_id", userID,
			"sales_start_date", req.SalesStartDate,
			"error", err,
		)

		c.Error(
			appErrors.NewValidationError(
				"invalid sales start date",
			),
		)
		return
	}

	salesEndDate, err := parseOptionalDate(req.SalesEndDate)
	if err != nil {
		h.logger.Warn(
			"event_update_invalid_sales_end_date",
			"user_id", userID,
			"sales_end_date", req.SalesEndDate,
			"error", err,
		)

		c.Error(
			appErrors.NewValidationError(
				"invalid sales end date",
			),
		)
		return
	}

	var startTime time.Time
	var endTime time.Time

	if !req.IsAllDay {
		startTime, err = parseTime(req.StartTime)
		if err != nil {
			h.logger.Warn(
				"event_update_invalid_start_time",
				"user_id", userID,
				"start_time", req.StartTime,
				"error", err,
			)

			c.Error(
				appErrors.NewValidationError(
					"invalid start time",
				),
			)
			return
		}

		endTime, err = parseTime(req.EndTime)
		if err != nil {
			h.logger.Warn(
				"event_update_invalid_end_time",
				"user_id", userID,
				"end_time", req.EndTime,
				"error", err,
			)

			c.Error(
				appErrors.NewValidationError(
					"invalid end time",
				),
			)
			return
		}
	}

	output, err := h.eventUsecase.UpdateEvent(
		c.Request.Context(),
		eventUsecase.UpdateEventInput{
			UserID:     userID,
			EventID:    eventID,
			CategoryID: req.CategoryID,
			VenueID:    req.VenueID,
			Venue: eventUsecase.VenueInput{
				GooglePlaceID: req.Venue.GooglePlaceID,
				Name:          req.Venue.Name,
				Address:       req.Venue.Address,
				City:          req.Venue.City,
				State:         req.Venue.State,
				Country:       req.Venue.Country,
				PostalCode:    req.Venue.PostalCode,
				Latitude:      req.Venue.Latitude,
				Longitude:     req.Venue.Longitude,
			},
			EventType:           req.EventType,
			OnlineURL:           req.OnlineURL,
			Title:               req.Title,
			Description:         req.Description,
			Language:            req.Language,
			AgeRestriction:      req.AgeRestriction,
			Visibility:          req.Visibility,
			Highlights:          req.Highlights,
			Rules:               req.Rules,
			AttendeeInformation: req.AttendeeInformation,

			EventDate: eventDate,
			StartTime: startTime,
			EndTime:   endTime,
			IsAllDay:  req.IsAllDay,

			SeatLayoutType:      req.SeatLayoutType,
			BookingLimitPerUser: req.BookingLimitPerUser,
			SalesStartDate:      salesStartDate,
			SalesEndDate:        salesEndDate,

			CancellationAllowed:       req.CancellationAllowed,
			CancellationDeadlineHours: req.CancellationDeadlineHours,
			RefundPolicy:              req.RefundPolicy,
			RefundPercentage:          req.RefundPercentage,

			Contact: eventUsecase.EventContactInput{
				Name:  req.Contact.Name,
				Phone: req.Contact.Phone,
				Email: req.Contact.Email,
			},
		},
	)
	if err != nil {
		h.logger.Warn(
			"event_update_request_failed",
			"event_id", eventID,
			"user_id", userID,
			"error", err,
		)

		switch {
		case errors.Is(
			err,
			eventUsecase.ErrInvalidEventInput,
		):
			c.Error(
				appErrors.NewValidationError(
					"invalid event data",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrInvalidEventSchedule,
		):
			c.Error(
				appErrors.NewValidationError(
					"invalid event schedule",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrInvalidEventSettings,
		):
			c.Error(
				appErrors.NewValidationError(
					"invalid event settings",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrInvalidCancellation,
		):
			c.Error(
				appErrors.NewValidationError(
					"invalid cancellation settings",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrUnauthorizedOrganizer,
		):
			c.Error(
				appErrors.NewForbiddenError(
					"you are not authorized to update this event",
				),
			)

		case errors.Is(
			err,
			eventUsecase.ErrEventNotEditable,
		):
			c.Error(
				appErrors.NewValidationError(
					"event is not editable",
				),
			)

		case errors.Is(
			err,
			domain.ErrEventNotFound,
		):
			c.Error(
				appErrors.NewNotFoundError(
					"event not found",
				),
			)

		case errors.Is(
			err,
			domain.ErrCategoryNotFound,
		):
			c.Error(
				appErrors.NewNotFoundError(
					"category not found",
				),
			)

		case errors.Is(
			err,
			domain.ErrVenueNotFound,
		):
			c.Error(
				appErrors.NewNotFoundError(
					"venue not found",
				),
			)

		default:
			c.Error(err)
		}

		return
	}

	h.logger.Info(
		"event_update_request_completed",
		"event_id", output.Event.ID,
		"user_id", userID,
		"status", output.Event.Status,
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"event":        output.Event,
			"schedule":     output.Schedule,
			"setting":      output.Setting,
			"cancellation": output.Cancellation,
			"contact":      output.Contact,
		},
	})
}

func detectBannerContentType(file io.ReadSeeker) (string, error) {
	buffer := make([]byte, 512)

	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	if n == 0 {
		return "", errors.New("empty banner file")
	}

	contentType := http.DetectContentType(buffer[:n])

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return contentType, nil
}

func parseOptionalDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err == nil {
		return &parsed, nil
	}

	parsed, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return &parsed, nil
	}

	return nil, err
}

func parseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return time.Time{}, errors.New("date is required")
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err == nil {
		return parsed, nil
	}

	parsed, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed, nil
	}

	return time.Time{}, err
}

func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return time.Time{}, errors.New("time is required")
	}

	parsed, err := time.Parse("15:04", value)
	if err == nil {
		return parsed, nil
	}

	parsed, err = time.Parse("15:04:05", value)
	if err == nil {
		return parsed, nil
	}

	parsed, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed, nil
	}

	return time.Time{}, err
}
