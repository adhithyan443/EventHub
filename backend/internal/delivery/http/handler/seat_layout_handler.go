package handler

import (
	"errors"
	"log/slog"
	"net/http"

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
