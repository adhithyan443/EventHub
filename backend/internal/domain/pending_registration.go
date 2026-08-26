package domain

import (
	"time"

	"github.com/google/uuid"
)

type PendingRegistration struct {
	ID           uuid.UUID
	FullName     string
	Email        string
	Phone        string
	PasswordHash string
	OTPHash      string
	OTPExpiresAt time.Time
	OTPAttempts  int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PendingRegistrationRepository interface {
	Create(registration *PendingRegistration) error
	FindByEmail(email string) (*PendingRegistration, error)
	Update(registration *PendingRegistration) error
	Delete(id uuid.UUID) error
}
