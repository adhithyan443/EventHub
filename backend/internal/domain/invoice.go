package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrInvoiceNotFound      = errors.New("invoice not found")
	ErrInvoiceAlreadyExists = errors.New("invoice already exists for booking")
)

const (
	InvoiceStatusIssued = "ISSUED"
	InvoiceStatusVoid   = "VOID"
)

// Invoice represents an immutable snapshot of a customer's booking receipt.
type Invoice struct {
	ID            uuid.UUID       `json:"id"`
	InvoiceNumber string          `json:"invoice_number"`
	BookingID     uuid.UUID       `json:"booking_id"`
	PaymentID     uuid.UUID       `json:"payment_id"`
	UserID        uuid.UUID       `json:"user_id"`
	EventID       uuid.UUID       `json:"event_id"`
	Status        string          `json:"status"`
	Currency      string          `json:"currency"`

	Subtotal       decimal.Decimal `json:"subtotal"`
	PlatformFee    decimal.Decimal `json:"platform_fee"`
	TaxAmount      decimal.Decimal `json:"tax_amount"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	TotalAmount    decimal.Decimal `json:"total_amount"`
	PaidAmount     decimal.Decimal `json:"paid_amount"`

	CustomerName  string  `json:"customer_name"`
	CustomerEmail string  `json:"customer_email"`
	CustomerPhone *string `json:"customer_phone,omitempty"`

	EventTitle    string  `json:"event_title"`
	EventDate     string  `json:"event_date"`     // Formatted YYYY-MM-DD
	EventTimeStr  *string `json:"event_time_str,omitempty"` // e.g. "06:00 PM – 09:00 PM"
	VenueName     *string `json:"venue_name,omitempty"`
	VenueAddress  *string `json:"venue_address,omitempty"`
	IsOnline      bool    `json:"is_online"`
	OnlineDetails *string `json:"online_details,omitempty"`

	IssuerName    string  `json:"issuer_name"`
	OrganizerName *string `json:"organizer_name,omitempty"`

	IssuedAt  time.Time `json:"issued_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Items []InvoiceItem `json:"items,omitempty"`
}

// InvoiceItem represents a ticket-level snapshot line item on an invoice.
type InvoiceItem struct {
	ID            uuid.UUID       `json:"id"`
	InvoiceID     uuid.UUID       `json:"invoice_id"`
	BookingItemID *uuid.UUID      `json:"booking_item_id,omitempty"`
	CategoryName  string          `json:"category_name"`
	SeatLabel     *string         `json:"seat_label,omitempty"`
	Quantity      int             `json:"quantity"`
	UnitPrice     decimal.Decimal `json:"unit_price"`
	TotalPrice    decimal.Decimal `json:"total_price"`
	CreatedAt     time.Time       `json:"created_at"`
}

// InvoiceSummaryDTO provides structured customer receipt details for API and UI consumers.
type InvoiceSummaryDTO struct {
	ID            uuid.UUID `json:"id"`
	ReceiptNumber string    `json:"receipt_number"`
	DocumentType  string    `json:"document_type"`
	Status        string    `json:"status"`
	IssuedAt      time.Time `json:"issued_at"`

	Booking struct {
		ID            uuid.UUID `json:"id"`
		BookingNumber string    `json:"booking_number"`
	} `json:"booking"`

	Customer struct {
		Name  string  `json:"name"`
		Email string  `json:"email"`
		Phone *string `json:"phone,omitempty"`
	} `json:"customer"`

	Event struct {
		Title         string  `json:"title"`
		Date          string  `json:"date"`
		Time          *string `json:"time,omitempty"`
		VenueName     *string `json:"venue_name,omitempty"`
		VenueAddress  *string `json:"venue_address,omitempty"`
		IsOnline      bool    `json:"is_online"`
		OnlineDetails *string `json:"online_details,omitempty"`
	} `json:"event"`

	Issuer struct {
		IssuerName    string  `json:"issuer_name"`
		OrganizerName *string `json:"organizer_name,omitempty"`
		Disclaimer    string  `json:"disclaimer"`
	} `json:"issuer"`

	Items []InvoiceItem `json:"items"`

	Pricing struct {
		Subtotal       decimal.Decimal `json:"subtotal"`
		PlatformFee    decimal.Decimal `json:"platform_fee"`
		TaxAmount      decimal.Decimal `json:"tax_amount"`
		DiscountAmount decimal.Decimal `json:"discount_amount"`
		TotalAmount    decimal.Decimal `json:"total_amount"`
		PaidAmount     decimal.Decimal `json:"paid_amount"`
		Currency       string          `json:"currency"`
	} `json:"pricing"`

	Payment struct {
		Method          string    `json:"method"`
		TransactionID   *string   `json:"transaction_id,omitempty"`
		ProviderOrderID *string   `json:"provider_order_id,omitempty"`
		PaidAt          time.Time `json:"paid_at"`
	} `json:"payment"`
}

// InvoiceRepository defines persistence operations for invoices and invoice items.
type InvoiceRepository interface {
	Create(ctx context.Context, invoice *Invoice, items []InvoiceItem) error
	FindByBookingID(ctx context.Context, bookingID uuid.UUID) (*Invoice, error)
	FindByBookingIDWithItems(ctx context.Context, bookingID uuid.UUID) (*Invoice, []InvoiceItem, error)
	FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (*Invoice, []InvoiceItem, error)
	NextReceiptNumber(ctx context.Context, t time.Time) (string, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}
