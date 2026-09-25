package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrSeatLayoutNotFound = errors.New("seat layout not found")

type SeatLayout struct {
	ID         uuid.UUID
	EventID    uuid.UUID
	LayoutName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type SeatLayoutRepository interface {
	Create(layout *SeatLayout) error
	FindByEventID(eventID uuid.UUID) (*SeatLayout, error)
}
