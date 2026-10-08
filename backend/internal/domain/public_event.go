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
