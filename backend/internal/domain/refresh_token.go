package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type RefreshTokenRepository interface {
	Create(token *RefreshToken) error
	FindByTokenHash(hash string) (*RefreshToken, error)
	Revoke(id uuid.UUID) error
}
