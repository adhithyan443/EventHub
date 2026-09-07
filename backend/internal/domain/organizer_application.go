package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrOrganizerApplicationNotFound = errors.New("organizer application not found")

type ApplicationStatus string

const (
	ApplicationPending  ApplicationStatus = "PENDING"
	ApplicationApproved ApplicationStatus = "APPROVED"
	ApplicationRejected ApplicationStatus = "REJECTED"
)

type OrganizerApplication struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ApplicantName string

	BusinessName string
	BusinessType string
	Description  string
	Phone        string
	Website      string

	GSTNumber string
	PANNumber string

	BankName               string
	AccountHolderName      string
	AccountNumberEncrypted string
	IFSCCode               string

	AddressLine string
	City        string
	State       string
	Country     string
	PostalCode  string

	LogoURL                 string
	VerificationDocumentURL string

	Status          ApplicationStatus
	RejectionReason string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizerApplicationList struct {
	Applications []*OrganizerApplication
	Total        int64
	Page         int
	Limit        int
}

type OrganizerApplicationRepository interface {
	Create(application *OrganizerApplication) error
	FindByUserID(userID uuid.UUID) (*OrganizerApplication, error)
	Update(application *OrganizerApplication) error
	FindByID(id uuid.UUID) (*OrganizerApplication, error)
	Delete(id uuid.UUID) error
	List(page int, limit int, status ApplicationStatus) (*OrganizerApplicationList, error)
}
