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
	EventDate    time.Time
	VenueName    string
	VenueAddress string
	CustomerName string
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

	imageName := "ticket-qr.png"
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
