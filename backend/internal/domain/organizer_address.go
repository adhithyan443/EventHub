package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrganizerAddress struct {
	ID          uuid.UUID
	OrganizerID uuid.UUID

	AddressLine string
	City        string
	State       string
	Country     string
	PostalCode  string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizerAddressRepository interface {
	Create(address *OrganizerAddress) error
	FindByOrganizerID(organizerID uuid.UUID) (*OrganizerAddress, error)
}
