package invoice

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/jung-kurt/gofpdf"
)

type ReceiptPDFService struct{}

func NewReceiptPDFService() *ReceiptPDFService {
	return &ReceiptPDFService{}
}

// GenerateReceiptPDF generates an authoritative, clean A4 PDF booking receipt.
func (s *ReceiptPDFService) GenerateReceiptPDF(data domain.InvoiceSummaryDTO) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetTitle("Booking Receipt - "+data.ReceiptNumber, false)
	pdf.AddPage()

	// Header: Platform branding & Document Title
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(0, 121, 109) // Brand Teal (#00796d)
	pdf.CellFormat(100, 10, "EventHub", "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(30, 41, 59) // Slate Ink
	pdf.CellFormat(80, 10, "BOOKING RECEIPT", "", 1, "R", false, 0, "")

	pdf.Ln(2)

	// Divider line
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(4)

	// Status Watermark if VOID
	if strings.ToUpper(data.Status) == domain.InvoiceStatusVoid {
		pdf.SetFont("Arial", "B", 11)
		pdf.SetTextColor(220, 38, 38) // Red
		pdf.CellFormat(0, 7, "*** VOID / CANCELLED RECEIPT ***", "", 1, "C", false, 0, "")
		pdf.Ln(2)
	}

	// Receipt Metadata Box (Two-column layout)
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 116, 139) // Slate gray

	// Helper for safe single-line cells that scale font down if text is too wide
	fitCell := func(w, h float64, text string, borderStr string, ln int, alignStr string, defaultSize float64) {
		pdf.SetFont("Arial", "", defaultSize)
		size := defaultSize
		for pdf.GetStringWidth(text) > (w-1.0) && size > 6.0 {
			size -= 0.5
			pdf.SetFont("Arial", "", size)
		}
		pdf.CellFormat(w, h, text, borderStr, ln, alignStr, false, 0, "")
		pdf.SetFont("Arial", "", defaultSize)
	}

	topY := pdf.GetY()

	// Left column: Receipt info
	pdf.SetXY(15, topY)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(30, 41, 59)
	pdf.CellFormat(35, 5, "Receipt Number:", "", 0, "L", false, 0, "")
	fitCell(55, 5, data.ReceiptNumber, "", 1, "L", 9)

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(35, 5, "Receipt Date:", "", 0, "L", false, 0, "")
	fitCell(55, 5, data.IssuedAt.Format("02 Jan 2006, 03:04 PM"), "", 1, "L", 9)

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(35, 5, "Booking Reference:", "", 0, "L", false, 0, "")
	fitCell(55, 5, data.Booking.BookingNumber, "", 1, "L", 9)

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(35, 5, "Payment ID:", "", 0, "L", false, 0, "")
	txID := "Completed"
	if data.Payment.TransactionID != nil && *data.Payment.TransactionID != "" {
		txID = *data.Payment.TransactionID
	}
	fitCell(55, 5, txID, "", 1, "L", 9)

	// Right column: Customer & Issuer info
	pdf.SetXY(110, topY)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(30, 41, 59)
	pdf.CellFormat(25, 5, "Billed To:", "", 0, "L", false, 0, "")
	fitCell(60, 5, data.Customer.Name, "", 1, "L", 9)

	pdf.SetX(110)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(25, 5, "Email:", "", 0, "L", false, 0, "")
	fitCell(60, 5, data.Customer.Email, "", 1, "L", 9)

	if data.Customer.Phone != nil && *data.Customer.Phone != "" {
		pdf.SetX(110)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(25, 5, "Phone:", "", 0, "L", false, 0, "")
		fitCell(60, 5, *data.Customer.Phone, "", 1, "L", 9)
	}

	pdf.SetX(110)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(25, 5, "Issuer:", "", 0, "L", false, 0, "")
	issuer := data.Issuer.IssuerName
	if issuer == "" {
		issuer = "EventHub Bookings"
	}
	fitCell(60, 5, issuer, "", 1, "L", 9)

	pdf.Ln(4)

	// Section 2: Event Details Block (Dynamic Height & Multi-line wrapping)
	cardX := 15.0
	cardW := 180.0
	innerX := cardX + 3.0
	innerW := cardW - 6.0 // 174.0 mm

	pdf.SetFont("Arial", "B", 10)
	titleLines := pdf.SplitLines([]byte(data.Event.Title), innerW)
	titleLineH := 4.8
	titleBlockH := float64(len(titleLines)) * titleLineH

	pdf.SetFont("Arial", "", 8.5)
	eventTimeStr := "Time TBA"
	if data.Event.Time != nil && *data.Event.Time != "" {
		eventTimeStr = *data.Event.Time
	}
	dateDisplay := fmt.Sprintf("Date & Time: %s | %s", data.Event.Date, eventTimeStr)
	dateLines := pdf.SplitLines([]byte(dateDisplay), innerW)
	dateLineH := 4.2
	dateBlockH := float64(len(dateLines)) * dateLineH

	venueDisplay := "Venue: Venue TBA"
	if data.Event.IsOnline {
		venueDisplay = "Event Type: Online Event"
		if data.Event.OnlineDetails != nil && *data.Event.OnlineDetails != "" {
			venueDisplay += " (" + *data.Event.OnlineDetails + ")"
		}
	} else if data.Event.VenueName != nil && *data.Event.VenueName != "" {
		venueDisplay = "Venue: " + *data.Event.VenueName
		if data.Event.VenueAddress != nil && *data.Event.VenueAddress != "" {
			venueDisplay += ", " + *data.Event.VenueAddress
		}
	}
	venueLines := pdf.SplitLines([]byte(venueDisplay), innerW)
	venueLineH := 4.2
	venueBlockH := float64(len(venueLines)) * venueLineH

	verticalPadding := 3.0
	cardHeight := verticalPadding + titleBlockH + 1.5 + dateBlockH + 1.5 + venueBlockH + verticalPadding

	cardStartY := pdf.GetY()
	pdf.SetFillColor(248, 250, 252) // Slate 50
	pdf.SetDrawColor(226, 232, 240)
	pdf.Rect(cardX, cardStartY, cardW, cardHeight, "FD")

	curY := cardStartY + verticalPadding
	// Title
	pdf.SetXY(innerX, curY)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(15, 23, 42)
	pdf.MultiCell(innerW, titleLineH, data.Event.Title, "", "L", false)
	curY = pdf.GetY() + 1.5

	// Date
	pdf.SetXY(innerX, curY)
	pdf.SetFont("Arial", "", 8.5)
	pdf.SetTextColor(71, 85, 105)
	pdf.MultiCell(innerW, dateLineH, dateDisplay, "", "L", false)
	curY = pdf.GetY() + 1.5

	// Venue
	pdf.SetXY(innerX, curY)
	pdf.MultiCell(innerW, venueLineH, venueDisplay, "", "L", false)

	pdf.SetY(cardStartY + cardHeight + 4)

	// Section 3: Itemized Ticket Table (Dynamic Heights & Multi-line Wrapping)
	col1W := 95.0
	col2W := 25.0
	col3W := 30.0
	col4W := 30.0

	pdf.SetFillColor(241, 245, 249) // Slate 100
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetTextColor(30, 41, 59)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(col1W, 7, " Item / Ticket Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(col2W, 7, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(col3W, 7, "Unit Price", "1", 0, "R", true, 0, "")
	pdf.CellFormat(col4W, 7, "Line Total ", "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 8.5)
	pdf.SetTextColor(51, 65, 85)

	curr := data.Pricing.Currency
	if curr == "" {
		curr = "INR"
	}

	for _, item := range data.Items {
		desc := item.CategoryName
		if item.SeatLabel != nil && *item.SeatLabel != "" {
			desc += " (" + *item.SeatLabel + ")"
		}

		pdf.SetFont("Arial", "", 8.5)
		descLines := pdf.SplitLines([]byte(desc), col1W-4.0)
		lineH := 4.2
		rowHeight := float64(len(descLines)) * lineH
		if rowHeight < 6.5 {
			rowHeight = 6.5
		}

		if pdf.GetY()+rowHeight > 270 {
			pdf.AddPage()
		}

		rowY := pdf.GetY()

		// Cell borders
		pdf.SetDrawColor(226, 232, 240)
		pdf.Rect(15, rowY, col1W, rowHeight, "D")
		pdf.Rect(15+col1W, rowY, col2W, rowHeight, "D")
		pdf.Rect(15+col1W+col2W, rowY, col3W, rowHeight, "D")
		pdf.Rect(15+col1W+col2W+col3W, rowY, col4W, rowHeight, "D")

		// Col 1: Description
		pdf.SetXY(17, rowY+(rowHeight-float64(len(descLines))*lineH)/2)
		pdf.SetTextColor(51, 65, 85)
		pdf.MultiCell(col1W-4.0, lineH, desc, "", "L", false)

		// Col 2: Qty
		pdf.SetXY(15+col1W, rowY+(rowHeight-lineH)/2)
		pdf.CellFormat(col2W, lineH, fmt.Sprintf("%d", item.Quantity), "", 0, "C", false, 0, "")

		// Col 3: Unit Price
		pdf.SetXY(15+col1W+col2W, rowY+(rowHeight-lineH)/2)
		pdf.CellFormat(col3W-2, lineH, fmt.Sprintf("%s %s", curr, item.UnitPrice.StringFixed(2)), "", 0, "R", false, 0, "")

		// Col 4: Line Total
		pdf.SetXY(15+col1W+col2W+col3W, rowY+(rowHeight-lineH)/2)
		pdf.CellFormat(col4W-2, lineH, fmt.Sprintf("%s %s ", curr, item.TotalPrice.StringFixed(2)), "", 0, "R", false, 0, "")

		pdf.SetY(rowY + rowHeight)
	}

	pdf.Ln(2)

	// Section 4: Financial Summary Block
	if pdf.GetY()+50 > 275 {
		pdf.AddPage()
	}
	summaryY := pdf.GetY()
	pdf.SetXY(110, summaryY)

	pdf.SetFont("Arial", "", 8.5)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(50, 5, "Ticket Subtotal:", "", 0, "R", false, 0, "")
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(35, 5, fmt.Sprintf("%s %s", curr, data.Pricing.Subtotal.StringFixed(2)), "", 1, "R", false, 0, "")

	pdf.SetX(110)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(50, 5, "Convenience Fee:", "", 0, "R", false, 0, "")
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(35, 5, fmt.Sprintf("%s %s", curr, data.Pricing.PlatformFee.StringFixed(2)), "", 1, "R", false, 0, "")

	pdf.SetX(110)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(50, 5, "Applicable Taxes:", "", 0, "R", false, 0, "")
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(35, 5, fmt.Sprintf("%s %s", curr, data.Pricing.TaxAmount.StringFixed(2)), "", 1, "R", false, 0, "")

	if data.Pricing.DiscountAmount.IsPositive() {
		pdf.SetX(110)
		pdf.SetTextColor(71, 85, 105)
		pdf.CellFormat(50, 5, "Discount:", "", 0, "R", false, 0, "")
		pdf.SetTextColor(22, 101, 52) // Green
		pdf.CellFormat(35, 5, fmt.Sprintf("-%s %s", curr, data.Pricing.DiscountAmount.StringFixed(2)), "", 1, "R", false, 0, "")
	}

	pdf.SetDrawColor(203, 213, 225)
	pdf.Line(135, pdf.GetY()+1, 195, pdf.GetY()+1)
	pdf.Ln(2)

	// Total Paid Row
	pdf.SetX(110)
	pdf.SetFont("Arial", "B", 10.5)
	pdf.SetTextColor(0, 121, 109) // Brand Teal
	pdf.CellFormat(50, 7, "Total Paid:", "", 0, "R", false, 0, "")
	pdf.CellFormat(35, 7, fmt.Sprintf("%s %s", curr, data.Pricing.PaidAmount.StringFixed(2)), "", 1, "R", false, 0, "")

	pdf.Ln(6)

	// Payment method note
	pdf.SetFont("Arial", "I", 8.5)
	pdf.SetTextColor(100, 116, 139)
	payMethod := data.Payment.Method
	if payMethod == "" {
		payMethod = "Online"
	}
	pdf.CellFormat(0, 5, fmt.Sprintf("Payment Method: %s | Status: Paid / Confirmed", payMethod), "", 1, "L", false, 0, "")

	// Section 5: Notice & Disclaimers Footer
	pdf.Ln(6)
	pdf.SetDrawColor(226, 232, 240)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(4)

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(148, 163, 184) // Slate 400
	disclaimer := data.Issuer.Disclaimer
	if disclaimer == "" {
		disclaimer = "This document is an official payment confirmation and booking receipt."
	}
	pdf.MultiCell(180, 4, disclaimer+"\nNote: This receipt confirms payment only. For admission to the event, please download and present your EventHub Ticket / Booking Pass with QR code.", "", "C", false)

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("failed to generate receipt PDF: %w", err)
	}

	return output.Bytes(), nil
}
