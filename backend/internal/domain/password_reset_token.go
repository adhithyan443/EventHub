package domain

import (
	"time"

	"github.com/google/uuid"
)

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type PasswordResetTokenRepository interface {
	Create(token *PasswordResetToken) error
	FindByTokenHash(hash string) (*PasswordResetToken, error)
	MarkUsed(id uuid.UUID) error
}
