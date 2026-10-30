package ticket_test

import (
	"bytes"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
)

const testSecret = "0123456789abcdef0123456789abcdef" // 32 chars

func TestQRService_BookingPayload(t *testing.T) {
	svc, err := ticket.NewQRService(testSecret)
	require.NoError(t, err)

	bookingID := uuid.New()

	t.Run("Generate and Verify successfully", func(t *testing.T) {
		payload, err := svc.GenerateBookingPayload(bookingID)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(payload, "EVH:BK:"+bookingID.String()+"."))

		verifiedID, err := svc.VerifyBookingPayload(payload)
		require.NoError(t, err)
		assert.Equal(t, bookingID, verifiedID)
	})

	t.Run("Reject Nil UUID", func(t *testing.T) {
		_, err := svc.GenerateBookingPayload(uuid.Nil)
		assert.ErrorIs(t, err, ticket.ErrInvalidQRPayload)
	})

	t.Run("Reject Tampered Signature", func(t *testing.T) {
		payload, err := svc.GenerateBookingPayload(bookingID)
		require.NoError(t, err)

		tampered := payload[:len(payload)-4] + "ffff"
		_, err = svc.VerifyBookingPayload(tampered)
		assert.ErrorIs(t, err, ticket.ErrInvalidQRPayload)
	})

	t.Run("Reject Tampered Booking ID", func(t *testing.T) {
		payload, err := svc.GenerateBookingPayload(bookingID)
		require.NoError(t, err)

		parts := strings.Split(payload, ".")
		require.Len(t, parts, 2)

		forgedRaw := "EVH:BK:" + uuid.New().String()
		tampered := forgedRaw + "." + parts[1]

		_, err = svc.VerifyBookingPayload(tampered)
		assert.ErrorIs(t, err, ticket.ErrInvalidQRPayload)
	})

	t.Run("Reject Payload From Different Secret", func(t *testing.T) {
		svc2, err := ticket.NewQRService("fedcba9876543210fedcba9876543210")
		require.NoError(t, err)

		payload, err := svc.GenerateBookingPayload(bookingID)
		require.NoError(t, err)

		_, err = svc2.VerifyBookingPayload(payload)
		assert.ErrorIs(t, err, ticket.ErrInvalidQRPayload)
	})

	t.Run("Reject Missing Prefix", func(t *testing.T) {
		// Individual ticket payload should not be verified by VerifyBookingPayload
		ticketPayload, err := svc.GeneratePayload(bookingID)
		require.NoError(t, err)

		_, err = svc.VerifyBookingPayload(ticketPayload)
		assert.ErrorIs(t, err, ticket.ErrInvalidQRPayload)
	})
}

func TestQRService_GenerateBookingQRImage(t *testing.T) {
	svc, err := ticket.NewQRService(testSecret)
	require.NoError(t, err)

	bookingID := uuid.New()
	imgBytes, err := svc.GenerateBookingQRImage(bookingID)
	require.NoError(t, err)
	require.NotEmpty(t, imgBytes)

	// Verify image is valid PNG
	img, err := png.Decode(bytes.NewReader(imgBytes))
	require.NoError(t, err)
	assert.Equal(t, 256, img.Bounds().Dx())
	assert.Equal(t, 256, img.Bounds().Dy())
}

func TestQRService_GenerateBookingPDF(t *testing.T) {
	svc, err := ticket.NewQRService(testSecret)
	require.NoError(t, err)

	bookingID := uuid.New()

	t.Run("Reserved Seating with Multiple Seats", func(t *testing.T) {
		data := ticket.BookingPDFData{
			BookingID:     bookingID,
			BookingNumber: "EVH-TEST-MULTI-SEAT",
			EventName:     "Rock Festival 2026",
			CustomerName:  "Jane Doe",
			TicketCount:   3,
			TicketType:    "VIP Reserved",
			Seats: []string{
				"VIP Section • Row A, Seat 1",
				"VIP Section • Row A, Seat 2",
				"VIP Section • Row A, Seat 3",
			},
			EventDate:    time.Now().Add(48 * time.Hour),
			VenueName:    "Grand Arena",
			VenueAddress: "123 Stadium Road, Metro City",
		}

		pdfBytes, err := svc.GenerateBookingPDF(data)
		require.NoError(t, err)
		assert.NotEmpty(t, pdfBytes)
		assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-")))
	})

	t.Run("General Admission with No Seats", func(t *testing.T) {
		data := ticket.BookingPDFData{
			BookingID:     bookingID,
			BookingNumber: "EVH-TEST-GA",
			EventName:     "Indie Showcase",
			CustomerName:  "John Smith",
			TicketCount:   2,
			TicketType:    "General Admission",
			Seats:         nil, // GA
			EventDate:     time.Now().Add(24 * time.Hour),
			VenueName:     "The Warehouse",
			VenueAddress:  "45 Alley Way, Arts District",
		}

		pdfBytes, err := svc.GenerateBookingPDF(data)
		require.NoError(t, err)
		assert.NotEmpty(t, pdfBytes)
		assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-")))
	})

	t.Run("Nil Booking ID Fails", func(t *testing.T) {
		data := ticket.BookingPDFData{
			BookingID: uuid.Nil,
		}

		_, err := svc.GenerateBookingPDF(data)
		assert.Error(t, err)
	})
}
