package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrganizerEvent struct {
	ID          uuid.UUID
	OrganizerID uuid.UUID

	CategoryID   uuid.UUID
	CategoryName string

	VenueID   *uuid.UUID
	VenueName string
	VenueCity string

	EventType string
	OnlineURL string

	Title          string
	Description    string
	BannerURL      string
	Language       string
	AgeRestriction int

	Visibility string
	Status     string

	SeatLayoutType string

	EventDate time.Time
	StartTime *string
	EndTime   *string
	IsAllDay  bool

	TicketCapacity   int64
	TicketsAvailable int64
	TicketsSold      int64

	CreatedAt time.Time
	UpdatedAt time.Time
}
