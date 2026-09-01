package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrganizerApplication struct {
	ID                      uuid.UUID
	UserID                  uuid.UUID
	BusinessName            string
	BusinessType            string
	Description             string
	Phone                   string
	Website                 string
	GSTNumber               string
	PANNumber               string
	BankName                string
	AccountHolderName       string
	AccountNumberEncrypted  string
	IFSCCode                string
	LogoURL                 string
	VerificationDocumentURL string
	Status                  string
	RejectionReason         string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type OrganizerApplicationRepository interface {
	Create(application *OrganizerApplication) error
	FindByUserID(userID uuid.UUID) (*OrganizerApplication, error)
	Update(application *OrganizerApplication) error
}
