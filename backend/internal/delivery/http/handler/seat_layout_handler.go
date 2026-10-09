package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	seatLayoutUsecase "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SeatLayoutHandler struct {
	seatLayoutUsecase *seatLayoutUsecase.SeatLayoutUsecase
	logger            *slog.Logger
}

func NewSeatLayoutHandler(
	seatLayoutUsecase *seatLayoutUsecase.SeatLayoutUsecase,
	logger *slog.Logger,
) *SeatLayoutHandler {
	return &SeatLayoutHandler{
		seatLayoutUsecase: seatLayoutUsecase,
		logger:            logger,
	}
}

type createSeatLayoutRequest struct {
	LayoutName string                 `json:"layoutName"`
	Sections   []createSectionRequest `json:"sections"`
}

type createSectionRequest struct {
	Name  string             `json:"name"`
	Price float64            `json:"price"`
	Rows  []createRowRequest `json:"rows"`
}

type createRowRequest struct {
	RowName string `json:"rowName"`
	Seats   int    `json:"seats"`
}

func (h *SeatLayoutHandler) CreateSeatLayout(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		h.logger.Warn("authenticated user ID missing from context")
		c.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		h.logger.Error("invalid authenticated user ID type")
		c.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.logger.Warn(
			"invalid event ID",
			slog.String("event_id", c.Param("id")),
		)
		c.Error(appErrors.NewValidationError("invalid event ID"))
		return
	}

	var req createSeatLayoutRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(
			"invalid seat layout request",
			slog.String("error", err.Error()),
		)
		c.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	h.logger.Info(
		"creating seat layout",
		slog.String("user_id", userID.String()),
		slog.String("event_id", eventID.String()),
		slog.String("layout_name", req.LayoutName),
	)

	sections := make([]seatLayoutUsecase.SeatSectionInput, 0, len(req.Sections))

	for _, section := range req.Sections {
		rows := make([]seatLayoutUsecase.SeatRowInput, 0, len(section.Rows))

		for _, row := range section.Rows {
			rows = append(rows, seatLayoutUsecase.SeatRowInput{
				RowName: row.RowName,
				Seats:   row.Seats,
			})
		}

		sections = append(sections, seatLayoutUsecase.SeatSectionInput{
			Name:  section.Name,
			Price: section.Price,
			Rows:  rows,
		})
	}

	input := seatLayoutUsecase.CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: req.LayoutName,
		Sections:   sections,
	}

	output, err := h.seatLayoutUsecase.CreateSeatLayout(input)
	if err != nil {
		switch {
		case errors.Is(err, seatLayoutUsecase.ErrInvalidSeatLayout):
			c.Error(appErrors.NewValidationError(err.Error()))

		case errors.Is(err, seatLayoutUsecase.ErrInvalidSeatSection):
			c.Error(appErrors.NewValidationError(err.Error()))

		case errors.Is(err, seatLayoutUsecase.ErrInvalidSeatRow):
			c.Error(appErrors.NewValidationError(err.Error()))

		case errors.Is(err, seatLayoutUsecase.ErrUnauthorizedOrganizer):
			c.Error(appErrors.NewForbiddenError(err.Error()))

		case errors.Is(err, domain.ErrEventNotFound):
			c.Error(appErrors.NewNotFoundError("event not found"))

		case errors.Is(err, domain.ErrSeatLayoutNotFound):
			c.Error(appErrors.NewNotFoundError("seat layout not found"))

		default:
			h.logger.Error(
				"seat layout creation failed",
				slog.String("user_id", userID.String()),
				slog.String("event_id", eventID.String()),
				slog.String("error", err.Error()),
			)
			c.Error(err)
		}

		return
	}

	h.logger.Info(
		"seat layout created successfully",
		slog.String("user_id", userID.String()),
		slog.String("event_id", eventID.String()),
		slog.String("seat_layout_id", output.Layout.ID.String()),
		slog.Int("section_count", len(output.Sections)),
		slog.Int("row_count", len(output.Rows)),
		slog.Int("seat_count", len(output.Seats)),
	)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"layout":   output.Layout,
			"sections": output.Sections,
			"rows":     output.Rows,
			"seats":    output.Seats,
		},
	})
}

type updateSeatLayoutRequest struct {
	LayoutName    string                 `json:"layout_name"`
	LayoutNameAlt string                 `json:"layoutName"`
	Sections      []updateSectionRequest `json:"sections"`
}

type updateSectionRequest struct {
	ID    *uuid.UUID         `json:"id,omitempty"`
	Name  string             `json:"name"`
	Price float64            `json:"price"`
	Rows  []updateRowRequest `json:"rows"`
}

type updateRowRequest struct {
	ID         *uuid.UUID          `json:"id,omitempty"`
	RowName    string              `json:"row_name"`
	RowNameAlt string              `json:"rowName"`
	Seats      int                 `json:"seats"`
	SeatsList  []updateSeatRequest `json:"seats_list,omitempty"`
}

type updateSeatRequest struct {
	ID         *uuid.UUID `json:"id,omitempty"`
	SeatNumber int        `json:"seat_number"`
	Status     string     `json:"status,omitempty"`
}

func (h *SeatLayoutHandler) UpdateSeatLayout(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		h.logger.Warn("authenticated user ID missing from context")
		c.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		h.logger.Error("invalid authenticated user ID type")
		c.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	rawEventID := c.Param("id")
	if rawEventID == "" {
		rawEventID = c.Param("event_id")
	}

	eventID, err := uuid.Parse(rawEventID)
	if err != nil {
		h.logger.Warn(
			"invalid event ID",
			slog.String("event_id", rawEventID),
		)
		c.Error(appErrors.NewValidationError("invalid event ID"))
		return
	}

	var req updateSeatLayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(
			"invalid seat layout update request",
			slog.String("error", err.Error()),
		)
		c.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	layoutName := strings.TrimSpace(req.LayoutName)
	if layoutName == "" {
		layoutName = strings.TrimSpace(req.LayoutNameAlt)
	}

	sections := make([]seatLayoutUsecase.UpdateSeatSectionInput, 0, len(req.Sections))
	for _, sec := range req.Sections {
		rows := make([]seatLayoutUsecase.UpdateSeatRowInput, 0, len(sec.Rows))
		for _, row := range sec.Rows {
			rowName := strings.TrimSpace(row.RowName)
			if rowName == "" {
				rowName = strings.TrimSpace(row.RowNameAlt)
			}

			seatItems := make([]seatLayoutUsecase.UpdateSeatItemInput, 0, len(row.SeatsList))
			for _, seat := range row.SeatsList {
				seatItems = append(seatItems, seatLayoutUsecase.UpdateSeatItemInput{
					ID:         seat.ID,
					SeatNumber: seat.SeatNumber,
					Status:     seat.Status,
				})
			}

			seatsCount := row.Seats
			if seatsCount <= 0 && len(seatItems) > 0 {
				seatsCount = len(seatItems)
			}

			rows = append(rows, seatLayoutUsecase.UpdateSeatRowInput{
				ID:        row.ID,
				RowName:   rowName,
				Seats:     seatsCount,
				SeatItems: seatItems,
			})
		}

		sections = append(sections, seatLayoutUsecase.UpdateSeatSectionInput{
			ID:    sec.ID,
			Name:  strings.TrimSpace(sec.Name),
			Price: sec.Price,
			Rows:  rows,
		})
	}

	input := seatLayoutUsecase.UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: layoutName,
		Sections:   sections,
	}

	output, err := h.seatLayoutUsecase.UpdateSeatLayout(c.Request.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, seatLayoutUsecase.ErrInvalidSeatLayout),
			errors.Is(err, seatLayoutUsecase.ErrInvalidSeatSection),
			errors.Is(err, seatLayoutUsecase.ErrInvalidSeatRow),
			errors.Is(err, seatLayoutUsecase.ErrSeatLayoutDraftOnly),
			errors.Is(err, seatLayoutUsecase.ErrSeatLayoutSeatedOnly),
			errors.Is(err, seatLayoutUsecase.ErrSeatLayoutStructuralChangesNotAllowed),
			errors.Is(err, seatLayoutUsecase.ErrSeatLayoutDuplicateID),
			errors.Is(err, seatLayoutUsecase.ErrSeatLayoutInvalidStatus):
			c.Error(appErrors.NewValidationError(err.Error()))

		case errors.Is(err, seatLayoutUsecase.ErrUnauthorizedOrganizer):
			c.Error(appErrors.NewForbiddenError(err.Error()))

		case errors.Is(err, domain.ErrEventNotFound):
			c.Error(appErrors.NewNotFoundError("event not found"))

		case errors.Is(err, domain.ErrSeatLayoutNotFound):
			c.Error(appErrors.NewNotFoundError("seat layout not found"))

		default:
			h.logger.Error(
				"seat layout update failed",
				slog.String("user_id", userID.String()),
				slog.String("event_id", eventID.String()),
				slog.String("error", err.Error()),
			)
			c.Error(err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"layout":   output.Layout,
			"sections": output.Sections,
			"rows":     output.Rows,
			"seats":    output.Seats,
		},
	})
}
