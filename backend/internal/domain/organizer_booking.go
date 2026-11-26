package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrganizerCustomerSummary captures the ticket purchaser / booking contact details.
// Purchaser information is distinct from individual ticket admission passes.
type OrganizerCustomerSummary struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

// OrganizerEventSummary provides minimal event context for bookings and attendees.
type OrganizerEventSummary struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	EventType string    `json:"event_type,omitempty"`
	BannerURL string    `json:"banner_url,omitempty"`
	VenueName string    `json:"venue_name,omitempty"`
}

// OrganizerBookingFilter defines multi-criteria filtering for organizer bookings.
type OrganizerBookingFilter struct {
	OrganizerID   uuid.UUID
	EventID       *uuid.UUID
	BookingStatus string
	PaymentStatus string
	Search        string
	From          *time.Time
	To            *time.Time
	Page          int
	Limit         int
}

// OrganizerBookingListItem represents a single booking order row in the organizer table.
type OrganizerBookingListItem struct {
	ID                   uuid.UUID                `json:"id"`
	BookingNumber        string                   `json:"booking_number"`
	CreatedAt            time.Time                `json:"created_at"`
	Customer             OrganizerCustomerSummary `json:"customer"`
	Event                OrganizerEventSummary    `json:"event"`
	TicketCount          int                      `json:"ticket_count"`
	OriginalBookingTotal decimal.Decimal          `json:"original_booking_total"`
	Currency             string                   `json:"currency"`
	BookingStatus        string                   `json:"booking_status"`
	// PaymentStatus represents the authoritative payment status for the booking.
	// Note: A booking may have multiple payment attempts (e.g. failed attempts followed by success).
	// The repository layer determines the authoritative payment status and financial aggregation in Phase 2.
	PaymentStatus string `json:"payment_status"`
	PaymentMethod string `json:"payment_method,omitempty"`
	RefundStatus  string `json:"refund_status,omitempty"`
}

// OrganizerBookingListResult wraps paginated booking items with metadata.
type OrganizerBookingListResult struct {
	Bookings     []OrganizerBookingListItem `json:"bookings"`
	TotalRecords int64                      `json:"total_records"`
	Page         int                        `json:"page"`
	Limit        int                        `json:"limit"`
	TotalPages   int                        `json:"total_pages"`
}

// OrganizerBookingStats encapsulates aggregate financial and order metrics for an organizer.
// Note: Captured amounts represent actual gateway payments and must not be confused with
// organizer earnings or payout balances. Settlement and payout calculations remain isolated
// within the finance/settlement domain.
type OrganizerBookingStats struct {
	TotalBookings              int64           `json:"total_bookings"`
	ConfirmedBookings          int64           `json:"confirmed_bookings"`
	CancelledBookings          int64           `json:"cancelled_bookings"`
	PartiallyCancelledBookings int64           `json:"partially_cancelled_bookings"`
	TotalTicketsSold           int64           `json:"total_tickets_sold"`
	GrossCapturedAmount        decimal.Decimal `json:"gross_captured_amount"`
	TotalRefundedAmount        decimal.Decimal `json:"total_refunded_amount"`
	NetCapturedAmount          decimal.Decimal `json:"net_captured_amount"`
	Currency                   string          `json:"currency"`
}

// OrganizerBookingFinancials contains the full monetary reconciliation of an order.
// Captured amounts represent actual payment gateway receipts and do not reflect
// organizer payout earnings or ledger allocations.
type OrganizerBookingFinancials struct {
	Subtotal                 decimal.Decimal `json:"subtotal"`
	PlatformFee              decimal.Decimal `json:"platform_fee"`
	Tax                      decimal.Decimal `json:"tax"`
	Discount                 decimal.Decimal `json:"discount"`
	OriginalBookingTotal     decimal.Decimal `json:"original_booking_total"`
	SuccessfulCapturedAmount decimal.Decimal `json:"successful_captured_amount"`
	TotalRefundedAmount      decimal.Decimal `json:"total_refunded_amount"`
	NetCapturedAfterRefunds  decimal.Decimal `json:"net_captured_after_refunds"`
	Currency                 string          `json:"currency"`
}

// OrganizerBookingItemDetail represents an order line item (ticket type or reserved seat).
type OrganizerBookingItemDetail struct {
	ID             uuid.UUID       `json:"id"`
	TicketTypeID   *uuid.UUID      `json:"ticket_type_id,omitempty"`
	TicketTypeName string          `json:"ticket_type_name"`
	Quantity       int             `json:"quantity"`
	UnitPrice      decimal.Decimal `json:"unit_price"`
	TotalPrice     decimal.Decimal `json:"total_price"`
	SeatID         *uuid.UUID      `json:"seat_id,omitempty"`
	SeatLabel      string          `json:"seat_label,omitempty"`
}

// OrganizerTicketAdmissionDetail represents an individual digital ticket admission pass.
type OrganizerTicketAdmissionDetail struct {
	ID              uuid.UUID       `json:"id"`
	TicketCode      string          `json:"ticket_code"`
	AdmissionNumber int             `json:"admission_number"`
	TicketTypeName  string          `json:"ticket_type_name"`
	SeatLabel       string          `json:"seat_label,omitempty"`
	Price           decimal.Decimal `json:"price"`
	Status          string          `json:"status"`
	IssuedAt        time.Time       `json:"issued_at"`
	UsedAt          *time.Time      `json:"used_at,omitempty"`
}

// OrganizerPaymentSummary provides sanitized gateway transaction details for a single payment attempt.
// Note: A booking may have multiple payment attempts across its lifecycle (e.g. initial failure followed by success).
// The repository layer will determine the authoritative payment status and transaction associations in Phase 2.
type OrganizerPaymentSummary struct {
	ID              string          `json:"id"`
	TransactionID   *string         `json:"transaction_id,omitempty"`
	Provider        string          `json:"provider"`
	ProviderOrderID *string         `json:"provider_order_id,omitempty"`
	PaymentMethod   string          `json:"payment_method"`
	Amount          decimal.Decimal `json:"amount"`
	Status          string          `json:"status"`
	PaidAt          *time.Time      `json:"paid_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// OrganizerRefundSummary captures gateway refund lifecycle information.
type OrganizerRefundSummary struct {
	ID              uuid.UUID       `json:"id"`
	PaymentID       uuid.UUID       `json:"payment_id"`
	GatewayRefundID *string         `json:"gateway_refund_id,omitempty"`
	Amount          decimal.Decimal `json:"amount"`
	Status          string          `json:"status"`
	Reason          string          `json:"reason"`
	CreatedAt       time.Time       `json:"created_at"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty"`
}

// OrganizerCancellationPolicy stores the policy snapshot active at booking time.
type OrganizerCancellationPolicy struct {
	CancellationAllowed       bool   `json:"cancellation_allowed"`
	CancellationDeadlineHours int    `json:"cancellation_deadline_hours"`
	RefundPolicy              string `json:"refund_policy"`
	RefundPercentage          int    `json:"refund_percentage"`
}

// OrganizerBookingDetail provides full deep inspection for a single organizer booking order.
type OrganizerBookingDetail struct {
	ID                 uuid.UUID                        `json:"id"`
	BookingNumber      string                           `json:"booking_number"`
	CreatedAt          time.Time                        `json:"created_at"`
	UpdatedAt          time.Time                        `json:"updated_at"`
	BookingStatus      string                           `json:"booking_status"`
	Customer           OrganizerCustomerSummary         `json:"customer"`
	Event              OrganizerEventSummary            `json:"event"`
	Financials         OrganizerBookingFinancials       `json:"financials"`
	Items              []OrganizerBookingItemDetail     `json:"items"`
	Tickets            []OrganizerTicketAdmissionDetail `json:"tickets"`
	Payments           []OrganizerPaymentSummary        `json:"payments"` // Contains all payment attempts associated with this booking
	Refunds            []OrganizerRefundSummary         `json:"refunds"`
	CancellationPolicy OrganizerCancellationPolicy      `json:"cancellation_policy"`
}

// OrganizerAttendeeFilter defines query parameters for the ticket admission attendee table.
type OrganizerAttendeeFilter struct {
	OrganizerID   uuid.UUID
	EventID       *uuid.UUID
	CheckInStatus string // ALL, VALID (PENDING), USED (CHECKED_IN), CANCELLED, REFUNDED, CANCELLED_REFUNDED
	Search        string
	Page          int
	Limit         int
}

// OrganizerAttendeeListItem represents an individual admission record in the attendees table.
type OrganizerAttendeeListItem struct {
	TicketID        uuid.UUID                `json:"ticket_id"`
	TicketCode      string                   `json:"ticket_code"`
	BookingID       uuid.UUID                `json:"booking_id"`
	BookingNumber   string                   `json:"booking_number"`
	Customer        OrganizerCustomerSummary `json:"customer"`
	Event           OrganizerEventSummary    `json:"event"`
	TicketTypeName  string                   `json:"ticket_type_name"`
	AdmissionNumber int                      `json:"admission_number"`
	SeatLabel       string                   `json:"seat_label,omitempty"`
	Price           decimal.Decimal          `json:"price"`
	Status          string                   `json:"status"` // VALID, USED, CANCELLED, REFUNDED
	IssuedAt        time.Time                `json:"issued_at"`
	UsedAt          *time.Time               `json:"used_at,omitempty"`
}

// OrganizerAttendeeListResult wraps paginated attendee records with metadata.
type OrganizerAttendeeListResult struct {
	Attendees    []OrganizerAttendeeListItem `json:"attendees"`
	TotalRecords int64                       `json:"total_records"`
	Page         int                         `json:"page"`
	Limit        int                         `json:"limit"`
	TotalPages   int                         `json:"total_pages"`
}

// OrganizerAttendeeStats provides aggregate counts for gate admissions.
type OrganizerAttendeeStats struct {
	TotalIssuedTickets       int64           `json:"total_issued_tickets"`
	ActiveEligibleAdmissions int64           `json:"active_eligible_admissions"`
	CheckedInTickets         int64           `json:"checked_in_tickets"`
	PendingCheckInTickets    int64           `json:"pending_check_in_tickets"`
	CancelledTickets         int64           `json:"cancelled_tickets"`
	CheckInRate              decimal.Decimal `json:"check_in_rate"`
}
