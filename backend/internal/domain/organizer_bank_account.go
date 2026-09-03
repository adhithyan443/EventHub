package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrganizerBankAccount struct {
	ID          uuid.UUID
	OrganizerID uuid.UUID

	BankName               string
	AccountHolderName      string
	AccountNumberEncrypted string
	IFSCCode               string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizerBankAccountRepository interface {
	Create(account *OrganizerBankAccount) error
	FindByOrganizerID(organizerID uuid.UUID) (*OrganizerBankAccount, error)
}
