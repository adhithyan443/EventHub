package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FinanceLedgerEntryModel represents a single balanced debit or credit line item in the general ledger.
// Database table: finance_ledger_entries
type FinanceLedgerEntryModel struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey"`
	JournalID   uuid.UUID       `gorm:"type:uuid;not null;index"`
	AccountID   uuid.UUID       `gorm:"type:uuid;not null;index"`
	EntryType   string          `gorm:"type:varchar(10);not null"`
	Amount      decimal.Decimal `gorm:"type:numeric(20,2);not null"`
	Currency    string          `gorm:"type:varchar(3);not null;default:'INR'"`
	OrganizerID *uuid.UUID      `gorm:"type:uuid;index"`
	EventID     *uuid.UUID      `gorm:"type:uuid;index"`
	CreatedAt   time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (FinanceLedgerEntryModel) TableName() string {
	return "finance_ledger_entries"
}
