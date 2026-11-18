package models

import (
	"time"

	"github.com/google/uuid"
)

// FinanceJournalModel represents an atomic double-entry general ledger journal header.
// Database table: finance_journals
type FinanceJournalModel struct {
	ID                     uuid.UUID                 `gorm:"type:uuid;primaryKey"`
	TransactionType        string                    `gorm:"type:varchar(50);not null"`
	FinancialEventType     string                    `gorm:"type:varchar(50);not null"`
	SourceType             string                    `gorm:"type:varchar(50);not null"`
	SourceID               string                    `gorm:"type:varchar(255);not null"`
	IdempotencyKey         string                    `gorm:"type:varchar(255);not null;uniqueIndex"`
	IdempotencyFingerprint string                    `gorm:"type:char(64);not null"`
	Currency               string                    `gorm:"type:varchar(3);not null;default:'INR'"`
	Description            string                    `gorm:"type:text;not null"`
	Status                 string                    `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	CreatedAt              time.Time                 `gorm:"type:timestamptz;not null;default:now()"`
	PostedAt               *time.Time                `gorm:"type:timestamptz"`
	IsReversed             bool                      `gorm:"not null;default:false"`
	ReversalReferenceID    *uuid.UUID                `gorm:"type:uuid"`
	ReversedByJournalID    *uuid.UUID                `gorm:"type:uuid"`
	ReversedAt             *time.Time                `gorm:"type:timestamptz"`
	Entries                []FinanceLedgerEntryModel `gorm:"foreignKey:JournalID;references:ID"`
}

func (FinanceJournalModel) TableName() string {
	return "finance_journals"
}
