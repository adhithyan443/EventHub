package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventContact struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	Name      string
	Phone     string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EventContactRepository interface {
	Create(contact *EventContact) error
	FindByEventID(eventID uuid.UUID) (*EventContact, error)
}
