package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
)

type TicketHandler struct {
	ticketUsecase *ticket.TicketUsecase
	qrService     *ticket.QRService
}

func NewTicketHandler(ticketUsecase *ticket.TicketUsecase, qrService *ticket.QRService) *TicketHandler {
	return &TicketHandler{
		ticketUsecase: ticketUsecase,
		qrService:     qrService,
	}
}

// GetMyTickets handles GET /api/v1/tickets.
func (h *TicketHandler) GetMyTickets(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	tickets, err := h.ticketUsecase.GetMyTickets(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve tickets",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tickets,
	})
}

// GetMyTicket handles GET /api/v1/tickets/:id.
func (h *TicketHandler) GetMyTicket(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil || ticketID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid ticket ID",
		})
		return
	}

	result, err := h.ticketUsecase.GetMyTicket(
		c.Request.Context(),
		userID,
		ticketID,
	)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "ticket not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve ticket",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// authenticatedUserID must match the value stored by your existing
// authentication middleware. Update the context key if necessary.
func authenticatedUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}

	switch id := value.(type) {
	case uuid.UUID:
		return id, id != uuid.Nil
	case string:
		parsed, err := uuid.Parse(id)
		return parsed, err == nil && parsed != uuid.Nil
	default:
		return uuid.Nil, false
	}
}

// GetMyTicketQR handles GET /api/v1/tickets/:id/qr.
func (h *TicketHandler) GetMyTicketQR(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil || ticketID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid ticket ID",
		})
		return
	}

	// Verify ticket ownership before generating the QR image.
	result, err := h.ticketUsecase.GetMyTicket(
		c.Request.Context(),
		userID,
		ticketID,
	)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "ticket not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve ticket",
		})
		return
	}

	// GenerateQRImage belongs to the QRService introduced earlier.
	png, err := h.qrService.GenerateQRImage(result.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate ticket QR code",
		})
		return
	}

	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, "image/png", png)
}

func (h *TicketHandler) DownloadTicketPDF(c *gin.Context) {
	ctx := c.Request.Context()

	// Get the authenticated user's ID.
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	// Parse the ticket ID.
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ticket ID",
		})
		return
	}

	// Retrieve the ticket and verify ownership.
	ticket, err := h.ticketUsecase.GetMyTicket(ctx, userID, ticketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Ticket not found",
			})
			return
		}

		slog.ErrorContext(ctx, "failed to retrieve ticket for PDF download",
			slog.String("ticket_id", ticketID.String()),
			slog.Any("error", err),
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate ticket",
		})
		return
	}

	// TODO: Map ticket and event details into the PDF data structure.
	// TODO: Generate the PDF and return it as a downloadable file.
	_ = ticket

	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "PDF generation is not implemented yet",
	})
}
