package repository_test

import (
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestInvoiceDomain_Initialization(t *testing.T) {
	invoiceID := uuid.New()
	bookingID := uuid.New()
	paymentID := uuid.New()
	userID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	inv := domain.Invoice{
		ID:             invoiceID,
		InvoiceNumber:  "EVH-REC-20261110-000001",
		BookingID:      bookingID,
		PaymentID:      paymentID,
		UserID:         userID,
		EventID:        eventID,
		Status:         domain.InvoiceStatusIssued,
		Currency:       "INR",
		Subtotal:       decimal.NewFromInt(1200),
		PlatformFee:    decimal.NewFromInt(40),
		TaxAmount:      decimal.NewFromInt(100),
		DiscountAmount: decimal.Zero,
		TotalAmount:    decimal.NewFromInt(1340),
		PaidAmount:     decimal.NewFromInt(1340),
		CustomerName:   "Jane Doe",
		CustomerEmail:  "jane@example.com",
		EventTitle:     "Concert 2026",
		EventDate:      "2026-11-20",
		IssuerName:     "EventHub Bookings",
		IssuedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
		Items: []domain.InvoiceItem{
			{
				ID:           uuid.New(),
				InvoiceID:    invoiceID,
				CategoryName: "VIP",
				Quantity:     2,
				UnitPrice:    decimal.NewFromInt(600),
				TotalPrice:   decimal.NewFromInt(1200),
				CreatedAt:    now,
			},
		},
	}

	if inv.Status != domain.InvoiceStatusIssued {
		t.Errorf("expected status %s, got %s", domain.InvoiceStatusIssued, inv.Status)
	}
	if len(inv.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(inv.Items))
	}
	if inv.TotalAmount.Cmp(decimal.NewFromInt(1340)) != 0 {
		t.Errorf("expected total amount 1340, got %s", inv.TotalAmount.String())
	}
}
