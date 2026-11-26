package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AdminOrganizerBrief provides minimal organizer identifying details for platform-wide listings.
type AdminOrganizerBrief struct {
	ID           uuid.UUID `json:"id"`
	BusinessName string    `json:"business_name"`
	Email        string    `json:"email,omitempty"`
}

// AdminBookingFilter defines multi-criteria filtering for platform-wide admin bookings.
type AdminBookingFilter struct {
	OrganizerID   *uuid.UUID
	EventID       *uuid.UUID
	BookingStatus string
	PaymentStatus string
	Search        string
	From          *time.Time
	To            *time.Time
	Page          int
	Limit         int
}

// AdminBookingListItem represents a single booking order row in the platform admin table.
type AdminBookingListItem struct {
	ID                   uuid.UUID                `json:"id"`
	BookingNumber        string                   `json:"booking_number"`
	CreatedAt            time.Time                `json:"created_at"`
	Customer             OrganizerCustomerSummary `json:"customer"`
	Event                OrganizerEventSummary    `json:"event"`
	Organizer            AdminOrganizerBrief      `json:"organizer"`
	TicketCount          int                      `json:"ticket_count"`
	OriginalBookingTotal decimal.Decimal          `json:"original_booking_total"`
	Currency             string                   `json:"currency"`
	BookingStatus        string                   `json:"booking_status"`
	PaymentStatus        string                   `json:"payment_status"`
	PaymentMethod        string                   `json:"payment_method,omitempty"`
	RefundStatus         string                   `json:"refund_status,omitempty"`
}

// AdminBookingListResult wraps paginated admin booking items with metadata.
type AdminBookingListResult struct {
	Bookings     []AdminBookingListItem `json:"bookings"`
	TotalRecords int64                  `json:"total_records"`
	Page         int                    `json:"page"`
	Limit        int                    `json:"limit"`
	TotalPages   int                    `json:"total_pages"`
}

// AdminBookingStats encapsulates aggregate financial and order metrics across the platform.
type AdminBookingStats struct {
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

// AdminBookingDetail provides full deep inspection for a single booking order for administrators.
type AdminBookingDetail struct {
	ID                 uuid.UUID                        `json:"id"`
	BookingNumber      string                           `json:"booking_number"`
	CreatedAt          time.Time                        `json:"created_at"`
	UpdatedAt          time.Time                        `json:"updated_at"`
	BookingStatus      string                           `json:"booking_status"`
	Customer           OrganizerCustomerSummary         `json:"customer"`
	Event              OrganizerEventSummary            `json:"event"`
	Organizer          AdminOrganizerBrief              `json:"organizer"`
	Financials         OrganizerBookingFinancials       `json:"financials"`
	Items              []OrganizerBookingItemDetail     `json:"items"`
	Tickets            []OrganizerTicketAdmissionDetail `json:"tickets"`
	Payments           []OrganizerPaymentSummary        `json:"payments"`
	Refunds            []OrganizerRefundSummary         `json:"refunds"`
	CancellationPolicy OrganizerCancellationPolicy      `json:"cancellation_policy"`
}

// AdminAttendeeFilter defines query parameters for platform-wide ticket admissions.
type AdminAttendeeFilter struct {
	OrganizerID   *uuid.UUID
	EventID       *uuid.UUID
	CheckInStatus string // ALL, VALID (PENDING), USED (CHECKED_IN), CANCELLED, REFUNDED, CANCELLED_REFUNDED
	Search        string
	Page          int
	Limit         int
}

// AdminAttendeeListItem represents an individual admission record in the platform admin attendees table.
type AdminAttendeeListItem struct {
	TicketID        uuid.UUID                `json:"ticket_id"`
	TicketCode      string                   `json:"ticket_code"`
	BookingID       uuid.UUID                `json:"booking_id"`
	BookingNumber   string                   `json:"booking_number"`
	Customer        OrganizerCustomerSummary `json:"customer"`
	Event           OrganizerEventSummary    `json:"event"`
	Organizer       AdminOrganizerBrief      `json:"organizer"`
	TicketTypeName  string                   `json:"ticket_type_name"`
	AdmissionNumber int                      `json:"admission_number"`
	SeatLabel       string                   `json:"seat_label,omitempty"`
	Price           decimal.Decimal          `json:"price"`
	Status          string                   `json:"status"` // VALID, USED, CANCELLED, REFUNDED
	IssuedAt        time.Time                `json:"issued_at"`
	UsedAt          *time.Time               `json:"used_at,omitempty"`
}

// AdminAttendeeListResult wraps paginated admin attendee records with metadata.
type AdminAttendeeListResult struct {
	Attendees    []AdminAttendeeListItem `json:"attendees"`
	TotalRecords int64                   `json:"total_records"`
	Page         int                     `json:"page"`
	Limit        int                     `json:"limit"`
	TotalPages   int                     `json:"total_pages"`
}

// AdminAttendeeStats provides platform-wide aggregate counts for gate admissions.
type AdminAttendeeStats struct {
	TotalIssuedTickets       int64           `json:"total_issued_tickets"`
	ActiveEligibleAdmissions int64           `json:"active_eligible_admissions"`
	CheckedInTickets         int64           `json:"checked_in_tickets"`
	PendingCheckInTickets    int64           `json:"pending_check_in_tickets"`
	CancelledTickets         int64           `json:"cancelled_tickets"`
	CheckInRate              decimal.Decimal `json:"check_in_rate"`
}
