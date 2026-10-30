package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AdminEventListItem struct {
	ID                    uuid.UUID `json:"id"`
	OrganizerID           uuid.UUID `json:"organizer_id"`
	OrganizerBusinessName string    `json:"organizer_business_name"`
	OrganizerEmail        string    `json:"organizer_email"`

	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`

	VenueID   *uuid.UUID `json:"venue_id,omitempty"`
	VenueName string     `json:"venue_name,omitempty"`
	VenueCity string     `json:"venue_city,omitempty"`

	EventType string `json:"event_type"`
	OnlineURL string `json:"online_url,omitempty"`

	Title          string `json:"title"`
	Description    string `json:"description"`
	BannerURL      string `json:"banner_url"`
	BannerImageURL string `json:"banner_image_url,omitempty"`
	Language       string `json:"language"`
	AgeRestriction int    `json:"age_restriction"`

	Visibility string `json:"visibility"`
	Status     string `json:"status"`

	SeatLayoutType string `json:"seat_layout_type"`

	EventDate time.Time `json:"event_date"`
	StartTime *string   `json:"start_time,omitempty"`
	EndTime   *string   `json:"end_time,omitempty"`
	IsAllDay  bool      `json:"is_all_day"`

	TicketCapacity   int64 `json:"ticket_capacity"`
	TicketsAvailable int64 `json:"tickets_available"`
	TicketsSold      int64 `json:"tickets_sold"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminEventFilter struct {
	Page        int        `json:"page"`
	Limit       int        `json:"limit"`
	Search      string     `json:"search"`
	Status      string     `json:"status"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
	EventType   string     `json:"event_type"`
	OrganizerID *uuid.UUID `json:"organizer_id,omitempty"`
}

type AdminEventStatusCounts struct {
	Total     int64 `json:"total"`
	Published int64 `json:"published"`
	Draft     int64 `json:"draft"`
	Cancelled int64 `json:"cancelled"`
}

type AdminEventList struct {
	Events       []*AdminEventListItem  `json:"events"`
	Total        int64                  `json:"total"`
	Page         int                    `json:"page"`
	Limit        int                    `json:"limit"`
	StatusCounts AdminEventStatusCounts `json:"status_counts"`
}

type AdminEventBookingStats struct {
	TotalBookings     int64           `json:"total_bookings"`
	ConfirmedBookings int64           `json:"confirmed_bookings"`
	CancelledBookings int64           `json:"cancelled_bookings"`
	TicketsSold       int64           `json:"tickets_sold"`
	TotalRevenue      decimal.Decimal `json:"total_revenue"`
}

type AdminEventOrganizerDetails struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	BusinessName string    `json:"business_name"`
	BusinessType string    `json:"business_type"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Status       string    `json:"status"`
	Website      string    `json:"website,omitempty"`
	LogoURL      string    `json:"logo_url,omitempty"`
	GSTNumber    string    `json:"gst_number,omitempty"`
	PANNumber    string    `json:"pan_number,omitempty"`
}

type AdminEventDetails struct {
	Event               *Event                      `json:"event"`
	Organizer           *AdminEventOrganizerDetails `json:"organizer"`
	Category            *Category                   `json:"category"`
	Schedule            *EventSchedule              `json:"schedule"`
	Venue               *Venue                      `json:"venue,omitempty"`
	Setting             *EventSetting               `json:"setting"`
	Cancellation        *EventCancellation          `json:"cancellation"`
	Contact             *EventContact               `json:"contact,omitempty"`
	TicketTypes         []*TicketType               `json:"ticket_types,omitempty"`
	SeatLayout          *SeatLayout                 `json:"seat_layout,omitempty"`
	SeatSections        []SeatSectionWithStats      `json:"seat_sections,omitempty"`
	BookingStats        AdminEventBookingStats      `json:"booking_stats"`
	BannerImageURL      string                      `json:"banner_image_url,omitempty"`
	ActiveReservations  int64                       `json:"active_reservations"`
}

type AdminUpdateEventInput struct {
	EventID             uuid.UUID
	Title               string
	Description         string
	BannerURL           string
	Language            string
	AgeRestriction      *int
	Highlights          *string
	Rules               *string
	AttendeeInformation *string

	// Contact info
	ContactName  *string
	ContactPhone *string
	ContactEmail *string

	// Draft-only editable fields
	CategoryID          *uuid.UUID
	EventType           string
	OnlineURL           string
	VenueID             *uuid.UUID
	EventDate           *time.Time
	StartTime           *time.Time
	EndTime             *time.Time
	IsAllDay            *bool
	SeatLayoutType      string
	BookingLimitPerUser *int
	SalesStartDate      *time.Time
	SalesEndDate        *time.Time

	// Cancellation settings (draft-only)
	CancellationAllowed       *bool
	CancellationDeadlineHours *int
	RefundPolicy              string
	RefundPercentage          *int
}
