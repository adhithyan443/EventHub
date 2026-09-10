package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	eventUsecase "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxEventBannerSize = 10 * 1024 * 1024 // 10 MB

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
		"event_create_request_received",
		"user_id", userID,
	)

	var req CreateEventRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(
			"event_create_request_validation_failed",
			"user_id", userID,
			"error", err,
		)

		ctx.Error(
			appErrors.NewValidationError(
				"invalid request data",
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

	startTime, err := time.Parse(
		"15:04",
		req.StartTime,
	)
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

	endTime, err := time.Parse(
		"15:04",
		req.EndTime,
	)
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
	}

	output, err := h.eventUsecase.CreateEvent(input)
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
		},
	})
}

func (h *EventHandler) UploadBanner(ctx *gin.Context) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		h.logger.Warn(
			"event_banner_upload_authentication_failed",
			"error", err,
		)

		ctx.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	fileHeader, err := ctx.FormFile("banner")
	if err != nil {
		h.logger.Warn(
			"event_banner_upload_file_missing",
			"user_id", userID,
			"error", err,
		)

		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Banner image is required",
		})
		return
	}

	if fileHeader.Size > maxEventBannerSize {
		h.logger.Warn(
			"event_banner_upload_file_too_large",
			"user_id", userID,
			"file_size", fileHeader.Size,
		)

		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Banner image must not exceed 10 MB",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error(
			"event_banner_upload_file_open_failed",
			"user_id", userID,
			"error", err,
		)

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Unable to process banner image",
		})
		return
	}
	defer file.Close()

	contentType, err := detectBannerContentType(file)
	if err != nil {
		h.logger.Warn(
			"event_banner_upload_content_type_detection_failed",
			"user_id", userID,
			"error", err,
		)

		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid banner image",
		})
		return
	}

	if contentType != "image/jpeg" && contentType != "image/png" {
		h.logger.Warn(
			"event_banner_upload_unsupported_content_type",
			"user_id", userID,
			"content_type", contentType,
		)

		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Only PNG and JPG images are allowed",
		})
		return
	}

	objectKey, err := h.eventUsecase.UploadBanner(
		ctx.Request.Context(),
		userID,
		file,
		contentType,
	)

	if err != nil {
		switch {
		case errors.Is(err, eventUsecase.ErrUnauthorizedOrganizer):
			h.logger.Warn(
				"event_banner_upload_forbidden",
				"user_id", userID,
			)

			ctx.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Only active organizers can upload event banners",
			})

		case errors.Is(err, eventUsecase.ErrInvalidBanner):
			h.logger.Warn(
				"event_banner_upload_invalid",
				"user_id", userID,
			)

			ctx.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Invalid banner image",
			})

		default:
			h.logger.Error(
				"event_banner_upload_request_failed",
				"user_id", userID,
				"error", err,
			)

			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to upload banner image",
			})
		}

		return
	}

	h.logger.Info(
		"event_banner_upload_request_completed",
		"user_id", userID,
		"object_key", objectKey,
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Banner uploaded successfully",
		"data": gin.H{
			"object_key": objectKey,
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
