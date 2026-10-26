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
	"github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
)

type BookingHandler struct {
	bookingUsecase *booking.BookingUsecase
	qrService      *ticket.QRService
	logger         *slog.Logger
}

func NewBookingHandler(
	bookingUsecase *booking.BookingUsecase,
	qrService *ticket.QRService,
	logger *slog.Logger,
) *BookingHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &BookingHandler{
		bookingUsecase: bookingUsecase,
		qrService:      qrService,
		logger:         logger,
	}
}

// GetMyBookings handles GET /api/v1/bookings.
func (h *BookingHandler) GetMyBookings(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	bookings, err := h.bookingUsecase.GetMyBookings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve bookings",
		})
		return
	}

	if bookings == nil {
		bookings = []domain.Booking{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": bookings,
	})
}

// GetBookingByID handles GET /api/v1/bookings/:id.
func (h *BookingHandler) GetBookingByID(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil || bookingID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid booking ID",
		})
		return
	}

	result, err := h.bookingUsecase.GetBookingByID(
		c.Request.Context(),
		userID,
		bookingID,
	)
	if err != nil {
		if errors.Is(err, domain.ErrBookingNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "booking not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve booking",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// GetBookingQR handles GET /api/v1/bookings/:id/qr.
func (h *BookingHandler) GetBookingQR(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil || bookingID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid booking ID",
		})
		return
	}

	// Verify booking ownership
	_, err = h.bookingUsecase.GetBookingByID(c.Request.Context(), userID, bookingID)
	if err != nil {
		if errors.Is(err, domain.ErrBookingNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "booking not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve booking",
		})
		return
	}

	if h.qrService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "QR service not configured",
		})
		return
	}

	png, err := h.qrService.GenerateBookingQRImage(bookingID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "failed to generate booking QR code",
			slog.String("booking_id", bookingID.String()),
			slog.Any("error", err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate booking QR code",
		})
		return
	}

	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, "image/png", png)
}

// DownloadBookingPDF handles GET /api/v1/bookings/:id/download.
func (h *BookingHandler) DownloadBookingPDF(c *gin.Context) {
	ctx := c.Request.Context()

	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil || bookingID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid booking ID",
		})
		return
	}

	pdfData, err := h.bookingUsecase.GetBookingPDFData(ctx, userID, bookingID)
	if err != nil {
		if errors.Is(err, domain.ErrBookingNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "booking not found",
			})
			return
		}

		h.logger.ErrorContext(ctx, "failed to retrieve booking PDF data",
			slog.String("booking_id", bookingID.String()),
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve booking details",
		})
		return
	}

	if h.qrService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "QR service not configured",
		})
		return
	}

	pdfBytes, err := h.qrService.GenerateBookingPDF(*pdfData)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to generate booking PDF",
			slog.String("booking_id", bookingID.String()),
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate booking PDF",
		})
		return
	}

	filename := sanitizeBookingFilename(pdfData.EventName, bookingID)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Length", strconv.Itoa(len(pdfBytes)))
	c.Header("Cache-Control", "private, no-store")

	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func sanitizeBookingFilename(eventName string, bookingID uuid.UUID) string {
	var clean strings.Builder
	clean.WriteString("booking-pass-")
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
	if cleanedStr == "booking-pass" || cleanedStr == "" {
		return fmt.Sprintf("booking-pass-%s.pdf", bookingID.String())
	}
	return fmt.Sprintf("%s-%s.pdf", cleanedStr, bookingID.String())
}
