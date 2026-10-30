package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrOrganizerNotFound = errors.New("organizer not found")

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
	List(filter AdminOrganizerListFilter) (*AdminOrganizerList, error)
}

type AdminOrganizerItem struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	UserName            string
	UserEmail           string
	UserPhone           string
	BusinessName        string
	BusinessType        string
	BusinessDescription string
	BusinessPhone       string
	BusinessEmail       string
	Website             string
	LogoURL             string
	GSTNumber           string
	PANNumber           string
	Status              string
	CreatedAt           time.Time
}

type AdminOrganizerListFilter struct {
	Page   int
	Limit  int
	Search string
	Status string
}

type AdminOrganizerList struct {
	Organizers []*AdminOrganizerItem
	Total      int64
	Page       int
	Limit      int
}
