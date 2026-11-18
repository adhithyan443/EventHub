package finance

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EntryType defines whether an entry is a DEBIT or a CREDIT.
type EntryType string

const (
	EntryTypeDebit  EntryType = "DEBIT"
	EntryTypeCredit EntryType = "CREDIT"
)

// LedgerEntry represents an immutable single leg of a double-entry journal transaction.
// All fields are encapsulated to prevent external mutation of domain records.
type LedgerEntry struct {
	id          uuid.UUID
	journalID   uuid.UUID
	accountID   uuid.UUID
	entryType   EntryType
	amount      Money
	currency    string
	organizerID *uuid.UUID
	eventID     *uuid.UUID
	createdAt   time.Time
}

// cloneUUIDPtr safely creates an independent heap copy of a UUID pointer.
func cloneUUIDPtr(id *uuid.UUID) *uuid.UUID {
	if id == nil {
		return nil
	}
	copied := *id
	return &copied
}

// NewLedgerEntry constructs and validates a single double-entry line item.
// Optional subledger pointers (organizerID, eventID) are cloned to preserve encapsulation.
func NewLedgerEntry(
	id uuid.UUID,
	journalID uuid.UUID,
	accountID uuid.UUID,
	entryType EntryType,
	amount Money,
	organizerID *uuid.UUID,
	eventID *uuid.UUID,
	createdAt time.Time,
) (LedgerEntry, error) {
	cleanType := EntryType(strings.ToUpper(strings.TrimSpace(string(entryType))))

	entry := LedgerEntry{
		id:          id,
		journalID:   journalID,
		accountID:   accountID,
		entryType:   cleanType,
		amount:      amount,
		currency:    amount.Currency(),
		organizerID: cloneUUIDPtr(organizerID),
		eventID:     cloneUUIDPtr(eventID),
		createdAt:   createdAt,
	}

	if err := entry.Validate(); err != nil {
		return LedgerEntry{}, err
	}

	return entry, nil
}

// NewDebitEntry constructs a new validated debit ledger entry.
func NewDebitEntry(
	id uuid.UUID,
	accountID uuid.UUID,
	amount Money,
	organizerID *uuid.UUID,
	eventID *uuid.UUID,
	createdAt time.Time,
) (LedgerEntry, error) {
	return NewLedgerEntry(id, uuid.Nil, accountID, EntryTypeDebit, amount, organizerID, eventID, createdAt)
}

// NewCreditEntry constructs a new validated credit ledger entry.
func NewCreditEntry(
	id uuid.UUID,
	accountID uuid.UUID,
	amount Money,
	organizerID *uuid.UUID,
	eventID *uuid.UUID,
	createdAt time.Time,
) (LedgerEntry, error) {
	return NewLedgerEntry(id, uuid.Nil, accountID, EntryTypeCredit, amount, organizerID, eventID, createdAt)
}

// ID returns the unique UUID of the ledger entry.
func (e LedgerEntry) ID() uuid.UUID {
	return e.id
}

// JournalID returns the UUID of the parent journal transaction.
func (e LedgerEntry) JournalID() uuid.UUID {
	return e.journalID
}

// AccountID returns the UUID of the financial account affected.
func (e LedgerEntry) AccountID() uuid.UUID {
	return e.accountID
}

// EntryType returns DEBIT or CREDIT.
func (e LedgerEntry) EntryType() EntryType {
	return e.entryType
}

// Amount returns the Money value object.
func (e LedgerEntry) Amount() Money {
	return e.amount
}

// Currency returns the currency of the entry (INR).
func (e LedgerEntry) Currency() string {
	return e.currency
}

// OrganizerID returns a safe copy of the organizer UUID pointer for subledger reporting.
func (e LedgerEntry) OrganizerID() *uuid.UUID {
	return cloneUUIDPtr(e.organizerID)
}

// EventID returns a safe copy of the event UUID pointer for event-level reporting.
func (e LedgerEntry) EventID() *uuid.UUID {
	return cloneUUIDPtr(e.eventID)
}

// CreatedAt returns the timestamp when the entry was constructed.
func (e LedgerEntry) CreatedAt() time.Time {
	return e.createdAt
}

// Clone returns an exact, deep-cloned copy of the ledger entry.
func (e LedgerEntry) Clone() LedgerEntry {
	return LedgerEntry{
		id:          e.id,
		journalID:   e.journalID,
		accountID:   e.accountID,
		entryType:   e.entryType,
		amount:      e.amount,
		currency:    e.currency,
		organizerID: cloneUUIDPtr(e.organizerID),
		eventID:     cloneUUIDPtr(e.eventID),
		createdAt:   e.createdAt,
	}
}

// Validate verifies domain rules for an individual ledger entry.
func (e LedgerEntry) Validate() error {
	if e.id == uuid.Nil {
		return ErrInvalidEntryID
	}
	if e.accountID == uuid.Nil {
		return ErrInvalidAccountID
	}
	if !IsValidEntryType(e.entryType) {
		return ErrInvalidEntryType
	}
	if !e.amount.IsPositive() {
		return ErrNonPositiveMoney
	}
	if e.amount.Currency() != CurrencyINR || e.currency != CurrencyINR {
		return ErrUnsupportedCurrency
	}
	if e.organizerID != nil && *e.organizerID == uuid.Nil {
		return ErrInvalidOrganizerID
	}
	if e.eventID != nil && *e.eventID == uuid.Nil {
		return ErrInvalidEventID
	}
	if e.createdAt.IsZero() {
		return ErrInvalidEntryTimestamp
	}

	return nil
}

// IsValidEntryType checks whether an entry type is supported.
func IsValidEntryType(t EntryType) bool {
	switch t {
	case EntryTypeDebit, EntryTypeCredit:
		return true
	default:
		return false
	}
}

// OppositeEntryType returns CREDIT for DEBIT and DEBIT for CREDIT.
func OppositeEntryType(t EntryType) EntryType {
	if t == EntryTypeDebit {
		return EntryTypeCredit
	}
	return EntryTypeDebit
}

// WithJournalID returns a cloned copy of the entry assigned to a specific journal ID.
func (e LedgerEntry) WithJournalID(journalID uuid.UUID) LedgerEntry {
	cloned := e.Clone()
	cloned.journalID = journalID
	return cloned
}

// Reverse creates an exact opposite entry for reversal transactions.
func (e LedgerEntry) Reverse(newEntryID uuid.UUID, newJournalID uuid.UUID, now time.Time) (LedgerEntry, error) {
	return NewLedgerEntry(
		newEntryID,
		newJournalID,
		e.accountID,
		OppositeEntryType(e.entryType),
		e.amount,
		e.organizerID,
		e.eventID,
		now,
	)
}

// MarshalJSON serializes LedgerEntry into JSON with consistent public properties.
func (e LedgerEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          uuid.UUID  `json:"id"`
		JournalID   uuid.UUID  `json:"journal_id"`
		AccountID   uuid.UUID  `json:"account_id"`
		EntryType   EntryType  `json:"entry_type"`
		Amount      Money      `json:"amount"`
		Currency    string     `json:"currency"`
		OrganizerID *uuid.UUID `json:"organizer_id,omitempty"`
		EventID     *uuid.UUID `json:"event_id,omitempty"`
		CreatedAt   time.Time  `json:"created_at"`
	}{
		ID:          e.id,
		JournalID:   e.journalID,
		AccountID:   e.accountID,
		EntryType:   e.entryType,
		Amount:      e.amount,
		Currency:    e.currency,
		OrganizerID: cloneUUIDPtr(e.organizerID),
		EventID:     cloneUUIDPtr(e.eventID),
		CreatedAt:   e.createdAt,
	})
}

// UnmarshalJSON deserializes LedgerEntry from JSON.
func (e *LedgerEntry) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID          uuid.UUID  `json:"id"`
		JournalID   uuid.UUID  `json:"journal_id"`
		AccountID   uuid.UUID  `json:"account_id"`
		EntryType   EntryType  `json:"entry_type"`
		Amount      Money      `json:"amount"`
		Currency    string     `json:"currency"`
		OrganizerID *uuid.UUID `json:"organizer_id,omitempty"`
		EventID     *uuid.UUID `json:"event_id,omitempty"`
		CreatedAt   time.Time  `json:"created_at"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	entry, err := NewLedgerEntry(
		aux.ID,
		aux.JournalID,
		aux.AccountID,
		aux.EntryType,
		aux.Amount,
		aux.OrganizerID,
		aux.EventID,
		aux.CreatedAt,
	)
	if err != nil {
		return err
	}

	*e = entry
	return nil
}
