package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrLedgerJournalNotFound = errors.New("journal not found")
)

// AdminLedgerJournalListItem represents a journal summary item in the admin ledger list.
type AdminLedgerJournalListItem struct {
	ID                  uuid.UUID       `json:"id"`
	TransactionType     string          `json:"transaction_type"`
	FinancialEventType  string          `json:"financial_event_type"`
	SourceType          string          `json:"source_type"`
	SourceID            string          `json:"source_id"`
	IdempotencyKey      string          `json:"idempotency_key"`
	Currency            string          `json:"currency"`
	Description         string          `json:"description"`
	Status              string          `json:"status"`
	TotalDebit          decimal.Decimal `json:"total_debit"`
	TotalCredit         decimal.Decimal `json:"total_credit"`
	IsBalanced          bool            `json:"is_balanced"`
	IsReversed          bool            `json:"is_reversed"`
	ReversalReferenceID *uuid.UUID      `json:"reversal_reference_id,omitempty"`
	ReversedByJournalID *uuid.UUID      `json:"reversed_by_journal_id,omitempty"`
	ReversedAt          *time.Time      `json:"reversed_at,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	PostedAt            *time.Time      `json:"posted_at,omitempty"`
}

// AdminLedgerEntryItem represents a single ledger line item with resolved account metadata.
type AdminLedgerEntryItem struct {
	ID          uuid.UUID       `json:"id"`
	JournalID   uuid.UUID       `json:"journal_id"`
	AccountID   uuid.UUID       `json:"account_id"`
	AccountCode string          `json:"account_code"`
	AccountName string          `json:"account_name"`
	AccountType string          `json:"account_type"`
	EntryType   string          `json:"entry_type"` // DEBIT or CREDIT
	Amount      decimal.Decimal `json:"amount"`
	Currency    string          `json:"currency"`
	OrganizerID *uuid.UUID      `json:"organizer_id,omitempty"`
	EventID     *uuid.UUID      `json:"event_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// AdminLedgerJournalDetail represents full journal details including line entries.
type AdminLedgerJournalDetail struct {
	ID                  uuid.UUID              `json:"id"`
	TransactionType     string                 `json:"transaction_type"`
	FinancialEventType  string                 `json:"financial_event_type"`
	SourceType          string                 `json:"source_type"`
	SourceID            string                 `json:"source_id"`
	IdempotencyKey      string                 `json:"idempotency_key"`
	Currency            string                 `json:"currency"`
	Description         string                 `json:"description"`
	Status              string                 `json:"status"`
	TotalDebit          decimal.Decimal        `json:"total_debit"`
	TotalCredit         decimal.Decimal        `json:"total_credit"`
	IsBalanced          bool                   `json:"is_balanced"`
	IsReversed          bool                   `json:"is_reversed"`
	ReversalReferenceID *uuid.UUID             `json:"reversal_reference_id,omitempty"`
	ReversedByJournalID *uuid.UUID             `json:"reversed_by_journal_id,omitempty"`
	ReversedAt          *time.Time             `json:"reversed_at,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	PostedAt            *time.Time             `json:"posted_at,omitempty"`
	Entries             []AdminLedgerEntryItem `json:"entries"`
}

// AdminLedgerFilter contains filtering and pagination options for listing journals.
type AdminLedgerFilter struct {
	Page            int
	Limit           int
	TransactionType string
	Status          string
	Search          string
	From            *time.Time
	To              *time.Time
}

// AdminLedgerListResult represents the paginated response for journals.
type AdminLedgerListResult struct {
	Journals     []AdminLedgerJournalListItem `json:"journals"`
	Page         int                          `json:"page"`
	Limit        int                          `json:"limit"`
	TotalRecords int64                        `json:"total_records"`
	TotalPages   int                          `json:"total_pages"`
}

// AdminLedgerRepository defines repository operations for querying the financial ledger.
type AdminLedgerRepository interface {
	ListJournals(ctx context.Context, filter AdminLedgerFilter) (*AdminLedgerListResult, error)
	GetJournalByID(ctx context.Context, id uuid.UUID) (*AdminLedgerJournalDetail, error)
}

// AdminLedgerUsecase defines business logic for admin ledger viewing.
type AdminLedgerUsecase interface {
	ListJournals(ctx context.Context, filter AdminLedgerFilter) (*AdminLedgerListResult, error)
	GetJournal(ctx context.Context, id uuid.UUID) (*AdminLedgerJournalDetail, error)
}
