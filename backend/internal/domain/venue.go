package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrVenueNotFound = errors.New("venue not found")

type Venue struct {
	ID            uuid.UUID
	GooglePlaceID string
	Name          string
	Address       string
	City          string
	State         string
	Country       string
	PostalCode    string
	Latitude      float64
	Longitude     float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type VenueRepository interface {
	Create(venue *Venue) error
	FindByID(id uuid.UUID) (*Venue, error)
	FindByGooglePlaceID(placeID string) (*Venue, error)
}
