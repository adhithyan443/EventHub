package domain

import (
	"time"

	"github.com/google/uuid"
)

type PublicEvent struct {
	ID             uuid.UUID
	CategoryID     uuid.UUID
	CategoryName   string
	VenueID        *uuid.UUID
	VenueName      string
	VenueCity      string
	EventType      string
	Title          string
	Description    string
	BannerURL      string
	BannerImageURL string
	Language       string
	AgeRestriction int
	Visibility     string
	Status         string
	EventDate      time.Time
	StartTime      *string
	EndTime        *string
	IsAllDay       bool
	StartingPrice  float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PublicEventFilter struct {
	Page       int
	Limit      int
	Keyword    string
	CategoryID *uuid.UUID
	City       string
	EventDate  *time.Time
}

type PublicEventDetails struct {
	ID                  uuid.UUID                      `json:"id"`
	Title               string                         `json:"title"`
	Description         string                         `json:"description"`
	BannerImageURL      string                         `json:"banner_image_url"`
	CategoryID          uuid.UUID                      `json:"category_id"`
	CategoryName        string                         `json:"category_name"`
	EventType           string                         `json:"event_type"`
	OnlineURL           string                         `json:"online_url,omitempty"`
	Language            string                         `json:"language"`
	AgeRestriction      int                            `json:"age_restriction"`
	Highlights          string                         `json:"highlights,omitempty"`
	Rules               string                         `json:"rules,omitempty"`
	AttendeeInformation string                         `json:"attendee_information,omitempty"`
	Status              string                         `json:"status"`
	Visibility          string                         `json:"visibility"`
	CreatedAt           time.Time                      `json:"created_at"`
	Schedule            PublicEventDetailsSchedule     `json:"schedule"`
	Venue               *PublicEventDetailsVenue       `json:"venue,omitempty"`
	Setting             PublicEventDetailsSetting      `json:"setting"`
	Cancellation        PublicEventDetailsCancellation `json:"cancellation"`
	Contact             *PublicEventDetailsContact     `json:"contact,omitempty"`
	Organizer           PublicEventDetailsOrganizer    `json:"organizer"`
	TicketTypes         []PublicEventDetailsTicketType `json:"ticket_types"`
	StartingPrice       float64                        `json:"starting_price"`
}

type PublicEventDetailsSchedule struct {
	EventDate time.Time `json:"event_date"`
	StartTime *string   `json:"start_time,omitempty"`
	EndTime   *string   `json:"end_time,omitempty"`
	IsAllDay  bool      `json:"is_all_day"`
}

type PublicEventDetailsVenue struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	City       string    `json:"city"`
	State      string    `json:"state"`
	Country    string    `json:"country"`
	PostalCode string    `json:"postal_code,omitempty"`
	Latitude   float64   `json:"latitude,omitempty"`
	Longitude  float64   `json:"longitude,omitempty"`
}

type PublicEventDetailsSetting struct {
	SeatLayoutType      string     `json:"seat_layout_type"`
	BookingLimitPerUser int        `json:"booking_limit_per_user"`
	SalesStartDate      *time.Time `json:"sales_start_date,omitempty"`
	SalesEndDate        *time.Time `json:"sales_end_date,omitempty"`
}

type PublicEventDetailsCancellation struct {
	CancellationAllowed       bool   `json:"cancellation_allowed"`
	CancellationDeadlineHours int    `json:"cancellation_deadline_hours"`
	RefundPolicy              string `json:"refund_policy"`
	RefundPercentage          int    `json:"refund_percentage"`
}

type PublicEventDetailsContact struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type PublicEventDetailsOrganizer struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Verified  bool      `json:"verified"`
}

type PublicEventDetailsTicketType struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Price             float64   `json:"price"`
	TotalQuantity     int       `json:"total_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	Description       string    `json:"description,omitempty"`
	Status            string    `json:"status"`
}
