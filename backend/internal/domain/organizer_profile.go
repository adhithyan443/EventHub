package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrOrganizerProfileNotFound = errors.New("organizer profile not found")

type OrganizerProfile struct {
	ID          uuid.UUID
	OrganizerID uuid.UUID

	BusinessName string
	BusinessType string
	Description  string
	Phone        string
	Email        string
	Website      string
	LogoURL      string
	GSTNumber    string
	PANNumber    string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizerProfileRepository interface {
	Create(profile *OrganizerProfile) error
	FindByOrganizerID(organizerID uuid.UUID) (*OrganizerProfile, error)
}
