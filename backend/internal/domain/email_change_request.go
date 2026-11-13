package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmailChangeRequestNotFound = errors.New("email change request not found")
	ErrEmailAlreadyInUse          = errors.New("email is already registered")
	ErrEmailReserved              = errors.New("email is already reserved by another request")
	ErrInvalidOTP                 = errors.New("invalid verification code")
	ErrOTPExpired                 = errors.New("verification code has expired")
	ErrMaxAttemptsExceeded        = errors.New("maximum verification attempts exceeded")
	ErrResendCooldownActive       = errors.New("please wait before requesting another code")
)

type EmailChangeRequest struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	NewEmail          string
	OTPHash           string
	OTPExpiresAt      time.Time
	OTPAttempts       int
	ResendAvailableAt time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type EmailChangeRequestRepository interface {
	Create(req *EmailChangeRequest) error
	Upsert(req *EmailChangeRequest) error
	FindByID(id uuid.UUID) (*EmailChangeRequest, error)
	FindByUserID(userID uuid.UUID) (*EmailChangeRequest, error)
	FindByNewEmail(email string) (*EmailChangeRequest, error)
	Update(req *EmailChangeRequest) error
	IncrementAttempts(id uuid.UUID) (int, error)
	UpdateResendCooldown(id uuid.UUID, userID uuid.UUID, newCooldown time.Time) error
	RollbackResendCooldown(id uuid.UUID, userID uuid.UUID, targetCooldown time.Time) error
	UpdateOTPAndCooldown(id uuid.UUID, oldHash string, newHash string, newExpiry time.Time, newCooldown time.Time) (bool, error)
	DeleteByIDAndUserID(id uuid.UUID, userID uuid.UUID) error
	DeleteByUserID(userID uuid.UUID) error
	DeleteByID(id uuid.UUID) error
}
