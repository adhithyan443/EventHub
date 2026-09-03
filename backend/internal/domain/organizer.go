package domain

import (
	"time"

	"github.com/google/uuid"
)

type Organizer struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizerRepository interface {
	Create(organizer *Organizer) error
	FindByID(id uuid.UUID) (*Organizer, error)
	FindByUserID(userID uuid.UUID) (*Organizer, error)
}
