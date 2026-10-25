package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode"

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

// authenticatedUserID retrieves the authenticated user's ID from context,
// checking both "user_id" (used by auth middleware) and "userID".
func authenticatedUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		value, exists = c.Get("userID")
	}
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

// DownloadTicketPDF handles GET /api/v1/tickets/:id/download.
func (h *TicketHandler) DownloadTicketPDF(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Authenticate customer
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	// 2. Validate ticket UUID
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil || ticketID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid ticket ID",
		})
		return
	}

	// 3. Retrieve ticket details and verify ownership via use case
	pdfData, err := h.ticketUsecase.GetTicketPDFData(ctx, userID, ticketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "ticket not found",
			})
			return
		}
		if errors.Is(err, domain.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "event not found",
			})
			return
		}

		slog.ErrorContext(ctx, "failed to retrieve ticket PDF data",
			slog.String("ticket_id", ticketID.String()),
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve ticket details",
		})
		return
	}

	// 4. Generate PDF via QRService
	pdfBytes, err := h.qrService.GeneratePDF(*pdfData)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate ticket PDF",
			slog.String("ticket_id", ticketID.String()),
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate ticket PDF",
		})
		return
	}

	// 5. Generate safe filename
	filename := sanitizeTicketFilename(pdfData.EventName, ticketID)

	// 6. Return response with download headers
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Length", strconv.Itoa(len(pdfBytes)))
	c.Header("Cache-Control", "private, no-store")

	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func sanitizeTicketFilename(eventName string, ticketID uuid.UUID) string {
	var clean strings.Builder
	clean.WriteString("ticket-")
	hasValidChar := false
	for _, r := range eventName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			clean.WriteRune(unicode.ToLower(r))
			hasValidChar = true
		} else if r == '-' || r == '_' || r == ' ' {
			if hasValidChar && clean.Len() < 50 {
				clean.WriteByte('-')
			}
		}
	}
	cleanedStr := strings.Trim(clean.String(), "-")
	if cleanedStr == "ticket" || cleanedStr == "" {
		return fmt.Sprintf("ticket-%s.pdf", ticketID.String())
	}
	return fmt.Sprintf("%s-%s.pdf", cleanedStr, ticketID.String())
}
