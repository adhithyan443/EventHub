package models

import (
	"time"

	"github.com/google/uuid"
)

// FinanceIdempotencyKeyModel ensures operation idempotency and payload consistency.
// Database table: finance_idempotency_keys
type FinanceIdempotencyKeyModel struct {
	IdempotencyKey string    `gorm:"type:varchar(255);primaryKey"`
	Fingerprint    string    `gorm:"type:char(64);not null"`
	JournalID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	CreatedAt      time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (FinanceIdempotencyKeyModel) TableName() string {
	return "finance_idempotency_keys"
}
