package invoice_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/invoice"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestReceiptPDFService_GenerateReceiptPDF_Success(t *testing.T) {
	svc := invoice.NewReceiptPDFService()

	bookingID := uuid.New()
	invoiceID := uuid.New()
	venueName := "Grand Arena"
	venueAddress := "123 Event Street, Bangalore, Karnataka 560001"
	eventTime := "06:00 PM – 09:00 PM"
	custPhone := "+91 9876543210"
	orgName := "Apex Live Experiences"
	txID := "pay_test123456789"
	seatLabel := "VIP - Row A - Seat 10"

	dto := domain.InvoiceSummaryDTO{
		ID:            invoiceID,
		ReceiptNumber: "EVH-REC-20261110-000001",
		DocumentType:  "Booking Receipt",
		Status:        domain.InvoiceStatusIssued,
		IssuedAt:      time.Date(2026, 11, 10, 18, 30, 0, 0, time.UTC),
		Booking: struct {
			ID            uuid.UUID `json:"id"`
			BookingNumber string    `json:"booking_number"`
		}{
			ID:            bookingID,
			BookingNumber: "BK-20261110-001",
		},
		Customer: struct {
			Name  string  `json:"name"`
			Email string  `json:"email"`
			Phone *string `json:"phone,omitempty"`
		}{
			Name:  "Jane Doe",
			Email: "jane.doe@example.com",
			Phone: &custPhone,
		},
		Event: struct {
			Title         string  `json:"title"`
			Date          string  `json:"date"`
			Time          *string `json:"time,omitempty"`
			VenueName     *string `json:"venue_name,omitempty"`
			VenueAddress  *string `json:"venue_address,omitempty"`
			IsOnline      bool    `json:"is_online"`
			OnlineDetails *string `json:"online_details,omitempty"`
		}{
			Title:        "Tech Innovators Summit 2026",
			Date:         "2026-11-15",
			Time:         &eventTime,
			VenueName:    &venueName,
			VenueAddress: &venueAddress,
			IsOnline:     false,
		},
		Issuer: struct {
			IssuerName    string  `json:"issuer_name"`
			OrganizerName *string `json:"organizer_name,omitempty"`
			Disclaimer    string  `json:"disclaimer"`
		}{
			IssuerName:    "EventHub Bookings",
			OrganizerName: &orgName,
			Disclaimer:    "This document is an official booking and payment receipt. Not a GST tax invoice.",
		},
		Items: []domain.InvoiceItem{
			{
				ID:           uuid.New(),
				InvoiceID:    invoiceID,
				CategoryName: "VIP Pass",
				SeatLabel:    &seatLabel,
				Quantity:     2,
				UnitPrice:    decimal.NewFromInt(1500),
				TotalPrice:   decimal.NewFromInt(3000),
				CreatedAt:    time.Now(),
			},
		},
		Pricing: struct {
			Subtotal       decimal.Decimal `json:"subtotal"`
			PlatformFee    decimal.Decimal `json:"platform_fee"`
			TaxAmount      decimal.Decimal `json:"tax_amount"`
			DiscountAmount decimal.Decimal `json:"discount_amount"`
			TotalAmount    decimal.Decimal `json:"total_amount"`
			PaidAmount     decimal.Decimal `json:"paid_amount"`
			Currency       string          `json:"currency"`
		}{
			Subtotal:       decimal.NewFromInt(3000),
			PlatformFee:    decimal.NewFromInt(100),
			TaxAmount:      decimal.NewFromInt(558),
			DiscountAmount: decimal.Zero,
			TotalAmount:    decimal.NewFromInt(3658),
			PaidAmount:     decimal.NewFromInt(3658),
			Currency:       "INR",
		},
		Payment: struct {
			Method          string    `json:"method"`
			TransactionID   *string   `json:"transaction_id,omitempty"`
			ProviderOrderID *string   `json:"provider_order_id,omitempty"`
			PaidAt          time.Time `json:"paid_at"`
		}{
			Method:        "UPI",
			TransactionID: &txID,
			PaidAt:        time.Date(2026, 11, 10, 18, 30, 0, 0, time.UTC),
		},
	}

	pdfBytes, err := svc.GenerateReceiptPDF(dto)
	if err != nil {
		t.Fatalf("expected no error generating PDF, got: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("expected non-empty PDF bytes")
	}

	// Verify PDF magic header
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Errorf("expected PDF to start with '%%PDF-', got prefix: %q", string(pdfBytes[:8]))
	}
}

func TestReceiptPDFService_GenerateReceiptPDF_VoidStatus(t *testing.T) {
	svc := invoice.NewReceiptPDFService()

	dto := domain.InvoiceSummaryDTO{
		ID:            uuid.New(),
		ReceiptNumber: "EVH-REC-20261110-000002",
		DocumentType:  "Booking Receipt",
		Status:        domain.InvoiceStatusVoid,
		IssuedAt:      time.Now(),
	}
	dto.Pricing.Currency = "INR"

	pdfBytes, err := svc.GenerateReceiptPDF(dto)
	if err != nil {
		t.Fatalf("expected no error generating PDF for VOID status, got: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("expected non-empty PDF bytes")
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Errorf("expected PDF to start with '%%PDF-'")
	}
}

func TestReceiptPDFService_GenerateReceiptPDF_LongText(t *testing.T) {
	svc := invoice.NewReceiptPDFService()

	bookingID := uuid.New()
	invoiceID := uuid.New()
	venueName := "The Ultra Grand International Convention & Exhibition Complex - Mega Auditorium Hall 5"
	venueAddress := "Plot 101/A-B, Financial Technology Innovation Corridor, Cyber City Extension, Near South Bypass Junction, Bangalore, Karnataka, PIN 560100, India"
	eventTime := "08:00 AM – 11:30 PM (IST) Extended Midnight Keynote & Networking"
	custPhone := "+91 9876543210 (Direct Extension 4432)"
	orgName := "Global Association of Advanced Software Engineers & International Consortium for Digital Transformation"
	txID := "razorpay_payment_transaction_id_long_string_example_9876543210_abcdef"
	seatLabel := "Zone Diamond - Tier 1 Box Seating - Row Alpha Beta - Seat Number 42 - Balcony Left"

	dto := domain.InvoiceSummaryDTO{
		ID:            invoiceID,
		ReceiptNumber: "EVH-REC-20261110-000003",
		DocumentType:  "Booking Receipt",
		Status:        domain.InvoiceStatusIssued,
		IssuedAt:      time.Now(),
		Booking: struct {
			ID            uuid.UUID `json:"id"`
			BookingNumber string    `json:"booking_number"`
		}{
			ID:            bookingID,
			BookingNumber: "BK-20261110-LONGREF-99999",
		},
		Customer: struct {
			Name  string  `json:"name"`
			Email string  `json:"email"`
			Phone *string `json:"phone,omitempty"`
		}{
			Name:  "Dr. Alexander Bartholomew Montgomery-Smith III",
			Email: "alexander.bartholomew.montgomery-smith.the.third@very-long-corporate-domain-name-testing.example.org",
			Phone: &custPhone,
		},
		Event: struct {
			Title         string  `json:"title"`
			Date          string  `json:"date"`
			Time          *string `json:"time,omitempty"`
			VenueName     *string `json:"venue_name,omitempty"`
			VenueAddress  *string `json:"venue_address,omitempty"`
			IsOnline      bool    `json:"is_online"`
			OnlineDetails *string `json:"online_details,omitempty"`
		}{
			Title:        "The Annual Worldwide International Colloquium on Distributed Autonomous Microservices, Cloud Native Systems & Next-Generation Artificial Intelligence Architectures (2026 Edition)",
			Date:         "2026-12-31",
			Time:         &eventTime,
			VenueName:    &venueName,
			VenueAddress: &venueAddress,
			IsOnline:     false,
		},
		Issuer: struct {
			IssuerName    string  `json:"issuer_name"`
			OrganizerName *string `json:"organizer_name,omitempty"`
			Disclaimer    string  `json:"disclaimer"`
		}{
			IssuerName:    "EventHub Global Ticketing & Booking Solutions Private Limited",
			OrganizerName: &orgName,
			Disclaimer:    "This official booking receipt confirms successful electronic settlement and admission reservation. It does not constitute a formal tax invoice or VAT credit voucher.",
		},
		Items: []domain.InvoiceItem{
			{
				ID:           uuid.New(),
				InvoiceID:    invoiceID,
				CategoryName: "All-Access Platinum Super VIP Early Bird Delegate Full-Day Pass with Exclusive Networking Lounge, Speaker Gala Dinner, and Lifetime Workshop Access",
				SeatLabel:    &seatLabel,
				Quantity:     3,
				UnitPrice:    decimal.NewFromInt(12500),
				TotalPrice:   decimal.NewFromInt(37500),
				CreatedAt:    time.Now(),
			},
		},
		Pricing: struct {
			Subtotal       decimal.Decimal `json:"subtotal"`
			PlatformFee    decimal.Decimal `json:"platform_fee"`
			TaxAmount      decimal.Decimal `json:"tax_amount"`
			DiscountAmount decimal.Decimal `json:"discount_amount"`
			TotalAmount    decimal.Decimal `json:"total_amount"`
			PaidAmount     decimal.Decimal `json:"paid_amount"`
			Currency       string          `json:"currency"`
		}{
			Subtotal:       decimal.NewFromInt(37500),
			PlatformFee:    decimal.NewFromInt(250),
			TaxAmount:      decimal.NewFromInt(6795),
			DiscountAmount: decimal.NewFromInt(1000),
			TotalAmount:    decimal.NewFromInt(43545),
			PaidAmount:     decimal.NewFromInt(43545),
			Currency:       "INR",
		},
		Payment: struct {
			Method          string    `json:"method"`
			TransactionID   *string   `json:"transaction_id,omitempty"`
			ProviderOrderID *string   `json:"provider_order_id,omitempty"`
			PaidAt          time.Time `json:"paid_at"`
		}{
			Method:        "Credit Card (MasterCard Commercial Executive)",
			TransactionID: &txID,
			PaidAt:        time.Now(),
		},
	}

	pdfBytes, err := svc.GenerateReceiptPDF(dto)
	if err != nil {
		t.Fatalf("expected no error generating PDF for long texts, got: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("expected non-empty PDF bytes")
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Errorf("expected PDF to start with '%%PDF-', got prefix: %q", string(pdfBytes[:8]))
	}
}
