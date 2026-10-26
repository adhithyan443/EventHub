package ticket

import (
	"bytes"
	"fmt"
	"image/png"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
)

type TicketPDFData struct {
	TicketID     uuid.UUID
	EventName    string
	TicketType   string
	AdmissionNo  string
	Seat         string
	EventDate    time.Time
	VenueName    string
	VenueAddress string
	CustomerName string
}

type BookingPDFData struct {
	BookingID     uuid.UUID
	BookingNumber string
	EventName     string
	CustomerName  string
	TicketCount   int
	TicketType    string
	Seats         []string
	EventDate     time.Time
	VenueName     string
	VenueAddress  string
}

// GeneratePDF creates a PDF ticket containing event details and a signed QR code.
func (s *QRService) GeneratePDF(data TicketPDFData) ([]byte, error) {
	if data.TicketID == uuid.Nil {
		return nil, fmt.Errorf("ticket ID is required")
	}

	qrBytes, err := s.GenerateQRImage(data.TicketID)
	if err != nil {
		return nil, fmt.Errorf("generate ticket QR code: %w", err)
	}

	if _, err := png.Decode(bytes.NewReader(qrBytes)); err != nil {
		return nil, fmt.Errorf("decode ticket QR image: %w", err)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("EventHub Ticket", false)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(0, 12, "EventHub Ticket", "", 1, "C", false, 0, "")

	pdf.Ln(8)

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, data.EventName, "", 1, "L", false, 0, "")

	pdf.Ln(3)
	pdf.SetFont("Arial", "", 11)

	pdf.CellFormat(0, 8, "Ticket ID: "+data.TicketID.String(), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 8, "Ticket Type: "+data.TicketType, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 8, "Admission Number: "+data.AdmissionNo, "", 1, "L", false, 0, "")

	if data.Seat != "" {
		pdf.CellFormat(0, 8, "Assigned Seat: "+data.Seat, "", 1, "L", false, 0, "")
	}

	pdf.CellFormat(0, 8, "Customer: "+data.CustomerName, "", 1, "L", false, 0, "")

	if !data.EventDate.IsZero() {
		pdf.CellFormat(
			0,
			8,
			"Event Date: "+data.EventDate.Format("02 Jan 2006, 03:04 PM"),
			"",
			1,
			"L",
			false,
			0,
			"",
		)
	}

	if data.VenueName != "" {
		pdf.CellFormat(0, 8, "Venue: "+data.VenueName, "", 1, "L", false, 0, "")
	}

	if data.VenueAddress != "" {
		pdf.MultiCell(0, 8, "Address: "+data.VenueAddress, "", "L", false)
	}

	options := gofpdf.ImageOptions{
		ImageType: "PNG",
		ReadDpi:   true,
	}

	imageName := fmt.Sprintf("ticket-qr-%s.png", data.TicketID.String())
	pdf.RegisterImageOptionsReader(
		imageName,
		options,
		bytes.NewReader(qrBytes),
	)

	pdf.Ln(8)
	pdf.ImageOptions(imageName, 75, pdf.GetY(), 60, 60, false, options, 0, "")

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("generate ticket PDF: %w", err)
	}

	return output.Bytes(), nil
}

// GenerateBookingPDF creates a unified booking pass containing event details, all seat numbers, and the booking QR code.
func (s *QRService) GenerateBookingPDF(data BookingPDFData) ([]byte, error) {
	if data.BookingID == uuid.Nil {
		return nil, fmt.Errorf("booking ID is required")
	}

	qrBytes, err := s.GenerateBookingQRImage(data.BookingID)
	if err != nil {
		return nil, fmt.Errorf("generate booking QR code: %w", err)
	}

	if _, err := png.Decode(bytes.NewReader(qrBytes)); err != nil {
		return nil, fmt.Errorf("decode booking QR image: %w", err)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("EventHub Booking Pass", false)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(0, 12, "EventHub Booking Pass", "", 1, "C", false, 0, "")

	pdf.Ln(6)

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, data.EventName, "", 1, "L", false, 0, "")

	pdf.Ln(2)
	pdf.SetFont("Arial", "", 11)

	bookingRef := data.BookingNumber
	if bookingRef == "" {
		bookingRef = data.BookingID.String()
	}
	pdf.CellFormat(0, 7, "Booking Reference: "+bookingRef, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 7, "Customer: "+data.CustomerName, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 7, "Ticket Tier: "+data.TicketType, "", 1, "L", false, 0, "")

	count := data.TicketCount
	if count <= 0 {
		count = 1
	}
	pdf.CellFormat(0, 7, fmt.Sprintf("Total Admissions: %d pass(es)", count), "", 1, "L", false, 0, "")

	if len(data.Seats) > 0 {
		pdf.Ln(2)
		pdf.SetFont("Arial", "B", 11)
		pdf.CellFormat(0, 7, fmt.Sprintf("Assigned Seats (%d):", len(data.Seats)), "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 11)
		for _, seatStr := range data.Seats {
			pdf.CellFormat(0, 6, "   • "+seatStr, "", 1, "L", false, 0, "")
		}
	} else {
		pdf.CellFormat(0, 7, "Seating: General Admission", "", 1, "L", false, 0, "")
	}

	if !data.EventDate.IsZero() {
		pdf.CellFormat(
			0,
			7,
			"Event Date: "+data.EventDate.Format("02 Jan 2006, 03:04 PM"),
			"",
			1,
			"L",
			false,
			0,
			"",
		)
	}

	if data.VenueName != "" {
		pdf.CellFormat(0, 7, "Venue: "+data.VenueName, "", 1, "L", false, 0, "")
	}

	if data.VenueAddress != "" {
		pdf.MultiCell(0, 7, "Address: "+data.VenueAddress, "", "L", false)
	}

	options := gofpdf.ImageOptions{
		ImageType: "PNG",
		ReadDpi:   true,
	}

	imageName := fmt.Sprintf("booking-qr-%s.png", data.BookingID.String())
	pdf.RegisterImageOptionsReader(
		imageName,
		options,
		bytes.NewReader(qrBytes),
	)

	pdf.Ln(6)
	pdf.ImageOptions(imageName, 75, pdf.GetY(), 60, 60, false, options, 0, "")

	pdf.SetY(pdf.GetY() + 64)
	pdf.SetFont("Arial", "I", 9)
	pdf.CellFormat(0, 6, "Scan this authoritative booking QR pass at entrance for admission.", "", 1, "C", false, 0, "")

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("generate booking PDF: %w", err)
	}

	return output.Bytes(), nil
}
