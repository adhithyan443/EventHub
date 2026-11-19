package finance

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TransactionType classifies the high-level business transaction.
type TransactionType string

const (
	TransactionTypePayment    TransactionType = "PAYMENT"
	TransactionTypeRefund     TransactionType = "REFUND"
	TransactionTypePayout     TransactionType = "PAYOUT"
	TransactionTypeAdjustment TransactionType = "ADJUSTMENT"
	TransactionTypeReversal   TransactionType = "REVERSAL"
)

// FinancialEventType specifies the exact event trigger that produced the journal.
type FinancialEventType string

const (
	FinancialEventTicketSaleCaptured     FinancialEventType = "TICKET_SALE_CAPTURED"
	FinancialEventBookingCancelled       FinancialEventType = "BOOKING_CANCELLED"
	FinancialEventOrganizerPayoutSettled FinancialEventType = "ORGANIZER_PAYOUT_SETTLED"
	FinancialEventSettlementAdjustment   FinancialEventType = "SETTLEMENT_ADJUSTMENT"
	FinancialEventManualCorrection       FinancialEventType = "MANUAL_CORRECTION"
	FinancialEventJournalReversed        FinancialEventType = "JOURNAL_REVERSED"
	FinancialEventCustomerRefund          FinancialEventType = "CUSTOMER_REFUND"
	FinancialEventEventCancellationRefund FinancialEventType = "EVENT_CANCELLATION_REFUND"
	FinancialEventEscrowCapture           FinancialEventType = "ESCROW_CAPTURE"
	FinancialEventEscrowRefund            FinancialEventType = "ESCROW_REFUND"
)

// SourceType defines the origin entity type that prompted this financial transaction.
type SourceType string

const (
	SourceTypePayment    SourceType = "PAYMENT"
	SourceTypeBooking    SourceType = "BOOKING"
	SourceTypeRefund     SourceType = "REFUND"
	SourceTypeSettlement SourceType = "SETTLEMENT"
	SourceTypePayout     SourceType = "PAYOUT"
	SourceTypeManual     SourceType = "MANUAL"
)

// PostingStatus defines whether a journal is in DRAFT or has been permanently POSTED.
type PostingStatus string

const (
	PostingStatusDraft  PostingStatus = "DRAFT"
	PostingStatusPosted PostingStatus = "POSTED"
)

// JournalTransaction represents an atomic, double-entry financial journal.
// Once posted, its entries and core state are strictly immutable.
// All fields are unexported to maintain domain encapsulation.
type JournalTransaction struct {
	id                  uuid.UUID
	transactionType     TransactionType
	financialEventType  FinancialEventType
	sourceType          SourceType
	sourceID            string
	idempotencyKey      string
	currency            string
	description         string
	entries             []LedgerEntry
	reversalReferenceID *uuid.UUID
	status              PostingStatus
	createdAt           time.Time
	postedAt            *time.Time
	isReversed          bool
	reversedByJournalID *uuid.UUID
	reversedAt          *time.Time
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}

// NewJournalTransaction constructs a new journal transaction in DRAFT status.
func NewJournalTransaction(
	id uuid.UUID,
	transactionType TransactionType,
	financialEventType FinancialEventType,
	sourceType SourceType,
	sourceID string,
	idempotencyKey string,
	currency string,
	description string,
	reversalReferenceID *uuid.UUID,
	createdAt time.Time,
) (*JournalTransaction, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidJournalID
	}

	cleanCurrency := strings.ToUpper(strings.TrimSpace(currency))
	if cleanCurrency != CurrencyINR {
		return nil, ErrUnsupportedCurrency
	}

	cleanTxType := TransactionType(strings.ToUpper(strings.TrimSpace(string(transactionType))))
	if !IsValidTransactionType(cleanTxType) {
		return nil, ErrInvalidTransactionType
	}

	cleanEventType := FinancialEventType(strings.ToUpper(strings.TrimSpace(string(financialEventType))))
	if !IsValidFinancialEventType(cleanEventType) {
		return nil, ErrInvalidFinancialEventType
	}

	cleanSourceType := SourceType(strings.ToUpper(strings.TrimSpace(string(sourceType))))
	if !IsValidSourceType(cleanSourceType) {
		return nil, ErrMissingSourceType
	}

	cleanSourceID := strings.TrimSpace(sourceID)
	if cleanSourceID == "" {
		return nil, ErrMissingSourceID
	}

	cleanIdempotencyKey := strings.TrimSpace(idempotencyKey)
	if cleanIdempotencyKey == "" {
		return nil, ErrMissingIdempotencyKey
	}

	cleanDescription := strings.TrimSpace(description)
	if cleanDescription == "" {
		return nil, ErrMissingDescription
	}

	if cleanTxType == TransactionTypeReversal {
		if reversalReferenceID == nil || *reversalReferenceID == uuid.Nil {
			return nil, ErrInvalidReversalReference
		}
		if *reversalReferenceID == id {
			return nil, ErrSelfReferencingJournal
		}
		if cleanEventType != FinancialEventJournalReversed {
			return nil, ErrInvalidFinancialEventType
		}
	} else {
		if reversalReferenceID != nil {
			return nil, ErrUnexpectedReversalReference
		}
		if cleanEventType == FinancialEventJournalReversed {
			return nil, ErrInvalidFinancialEventType
		}
	}

	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return &JournalTransaction{
		id:                  id,
		transactionType:     cleanTxType,
		financialEventType:  cleanEventType,
		sourceType:          cleanSourceType,
		sourceID:            cleanSourceID,
		idempotencyKey:      cleanIdempotencyKey,
		currency:            CurrencyINR,
		description:         cleanDescription,
		entries:             make([]LedgerEntry, 0, 4),
		reversalReferenceID: cloneUUIDPtr(reversalReferenceID),
		status:              PostingStatusDraft,
		createdAt:           createdAt,
		isReversed:          false,
	}, nil
}

// ReconstituteJournalTransaction rebuilds a JournalTransaction entity from persistence storage
// and strictly validates consistent state combinations.
func ReconstituteJournalTransaction(
	id uuid.UUID,
	transactionType TransactionType,
	financialEventType FinancialEventType,
	sourceType SourceType,
	sourceID string,
	idempotencyKey string,
	currency string,
	description string,
	entries []LedgerEntry,
	reversalReferenceID *uuid.UUID,
	status PostingStatus,
	createdAt time.Time,
	postedAt *time.Time,
	isReversed bool,
	reversedByJournalID *uuid.UUID,
	reversedAt *time.Time,
) (*JournalTransaction, error) {
	clonedEntries := make([]LedgerEntry, len(entries))
	for i, e := range entries {
		clonedEntries[i] = e.Clone()
	}

	jt := &JournalTransaction{
		id:                  id,
		transactionType:     transactionType,
		financialEventType:  financialEventType,
		sourceType:          sourceType,
		sourceID:            sourceID,
		idempotencyKey:      idempotencyKey,
		currency:            currency,
		description:         description,
		entries:             clonedEntries,
		reversalReferenceID: cloneUUIDPtr(reversalReferenceID),
		status:              status,
		createdAt:           createdAt,
		postedAt:            cloneTimePtr(postedAt),
		isReversed:          isReversed,
		reversedByJournalID: cloneUUIDPtr(reversedByJournalID),
		reversedAt:          cloneTimePtr(reversedAt),
	}

	if err := jt.Validate(); err != nil {
		return nil, err
	}

	return jt, nil
}

// ID returns the journal's unique identifier.
func (j *JournalTransaction) ID() uuid.UUID {
	return j.id
}

// TransactionType returns the transaction classification.
func (j *JournalTransaction) TransactionType() TransactionType {
	return j.transactionType
}

// FinancialEventType returns the specific business trigger event.
func (j *JournalTransaction) FinancialEventType() FinancialEventType {
	return j.financialEventType
}

// SourceType returns the upstream entity source type.
func (j *JournalTransaction) SourceType() SourceType {
	return j.sourceType
}

// SourceID returns the upstream entity identifier.
func (j *JournalTransaction) SourceID() string {
	return j.sourceID
}

// IdempotencyKey returns the unique key ensuring operation idempotency.
func (j *JournalTransaction) IdempotencyKey() string {
	return j.idempotencyKey
}

// Currency returns the currency of the journal (INR).
func (j *JournalTransaction) Currency() string {
	return j.currency
}

// Description returns the narrative description of the transaction.
func (j *JournalTransaction) Description() string {
	return j.description
}

// ReversalReferenceID returns a copy of the pointer to the original reversed journal, if applicable.
func (j *JournalTransaction) ReversalReferenceID() *uuid.UUID {
	return cloneUUIDPtr(j.reversalReferenceID)
}

// Status returns the current posting status (DRAFT or POSTED).
func (j *JournalTransaction) Status() PostingStatus {
	return j.status
}

// CreatedAt returns the creation timestamp.
func (j *JournalTransaction) CreatedAt() time.Time {
	return j.createdAt
}

// PostedAt returns a copy of the posting timestamp pointer.
func (j *JournalTransaction) PostedAt() *time.Time {
	return cloneTimePtr(j.postedAt)
}

// IsReversed returns whether this journal has been negated by a reversal journal.
func (j *JournalTransaction) IsReversed() bool {
	return j.isReversed
}

// ReversedByJournalID returns a copy of the UUID pointer of the reversing journal.
func (j *JournalTransaction) ReversedByJournalID() *uuid.UUID {
	return cloneUUIDPtr(j.reversedByJournalID)
}

// ReversedAt returns a copy of the reversal timestamp pointer.
func (j *JournalTransaction) ReversedAt() *time.Time {
	return cloneTimePtr(j.reversedAt)
}

// AddEntry adds a ledger entry to a draft journal. Fails if the journal is already posted
// or if an entry with the same entry ID has already been added.
func (j *JournalTransaction) AddEntry(entry LedgerEntry) error {
	if j.status == PostingStatusPosted {
		return ErrPostedJournalImmutable
	}

	if err := entry.Validate(); err != nil {
		return err
	}

	if entry.Currency() != j.currency {
		return ErrCurrencyMismatch
	}

	// Check for duplicate entry IDs within the journal
	for _, existing := range j.entries {
		if existing.ID() == entry.ID() {
			return ErrDuplicateLedgerEntryID
		}
	}

	entryWithJournal := entry.WithJournalID(j.id)
	j.entries = append(j.entries, entryWithJournal)
	return nil
}

// Entries returns a deep clone of all ledger entries to prevent external mutation.
func (j *JournalTransaction) Entries() []LedgerEntry {
	entriesCopy := make([]LedgerEntry, len(j.entries))
	for i, entry := range j.entries {
		entriesCopy[i] = entry.Clone()
	}
	return entriesCopy
}

// GetEntries is an alias for Entries() for backward compatibility.
func (j *JournalTransaction) GetEntries() []LedgerEntry {
	return j.Entries()
}

// TotalDebits calculates the sum of all DEBIT entries in this journal.
// It returns an error if any arithmetic operation fails rather than silently ignoring errors.
func (j *JournalTransaction) TotalDebits() (Money, error) {
	total := ZeroINR()
	for _, entry := range j.entries {
		if entry.EntryType() == EntryTypeDebit {
			added, err := total.Add(entry.Amount())
			if err != nil {
				return Money{}, err
			}
			total = added
		}
	}
	return total, nil
}

// TotalCredits calculates the sum of all CREDIT entries in this journal.
// It returns an error if any arithmetic operation fails rather than silently ignoring errors.
func (j *JournalTransaction) TotalCredits() (Money, error) {
	total := ZeroINR()
	for _, entry := range j.entries {
		if entry.EntryType() == EntryTypeCredit {
			added, err := total.Add(entry.Amount())
			if err != nil {
				return Money{}, err
			}
			total = added
		}
	}
	return total, nil
}

// IsBalanced evaluates whether total debits exactly equal total credits and satisfies minimum entry rules.
// Returns an error if arithmetic evaluation encounters an error.
func (j *JournalTransaction) IsBalanced() (bool, error) {
	if len(j.entries) < 2 {
		return false, nil
	}

	hasDebit := false
	hasCredit := false
	for _, entry := range j.entries {
		if entry.EntryType() == EntryTypeDebit {
			hasDebit = true
		} else if entry.EntryType() == EntryTypeCredit {
			hasCredit = true
		}
	}

	if !hasDebit || !hasCredit {
		return false, nil
	}

	debits, err := j.TotalDebits()
	if err != nil {
		return false, err
	}

	credits, err := j.TotalCredits()
	if err != nil {
		return false, err
	}

	return debits.Equal(credits), nil
}

// Validate verifies full domain, double-entry balancing, and persisted state invariants.
func (j *JournalTransaction) Validate() error {
	if j.id == uuid.Nil {
		return ErrInvalidJournalID
	}
	if !IsValidTransactionType(j.transactionType) {
		return ErrInvalidTransactionType
	}
	if !IsValidFinancialEventType(j.financialEventType) {
		return ErrInvalidFinancialEventType
	}
	if !IsValidSourceType(j.sourceType) {
		return ErrMissingSourceType
	}
	if strings.TrimSpace(j.sourceID) == "" {
		return ErrMissingSourceID
	}
	if strings.TrimSpace(j.idempotencyKey) == "" {
		return ErrMissingIdempotencyKey
	}
	if strings.TrimSpace(j.description) == "" {
		return ErrMissingDescription
	}
	if j.currency != CurrencyINR {
		return ErrUnsupportedCurrency
	}

	// 1. Posting Status Invariants
	switch j.status {
	case PostingStatusDraft:
		if j.postedAt != nil {
			return ErrDraftCannotHavePostedAt
		}
		if j.isReversed || j.reversedByJournalID != nil || j.reversedAt != nil {
			return ErrDraftCannotBeReversed
		}
	case PostingStatusPosted:
		if j.postedAt == nil || j.postedAt.IsZero() {
			return ErrMissingPostedAt
		}
		if j.postedAt.Before(j.createdAt) {
			return ErrInvalidJournalPostingState
		}
	default:
		return ErrInvalidJournalPostingState
	}

	// 2. Reversal Transaction Type & Reference Invariants
	if j.transactionType == TransactionTypeReversal {
		if j.reversalReferenceID == nil || *j.reversalReferenceID == uuid.Nil {
			return ErrInvalidReversalReference
		}
		if *j.reversalReferenceID == j.id {
			return ErrSelfReferencingJournal
		}
		if j.financialEventType != FinancialEventJournalReversed {
			return ErrInvalidFinancialEventType
		}
		if j.isReversed {
			return ErrReversalCannotBeReversed
		}
	} else {
		if j.reversalReferenceID != nil {
			return ErrUnexpectedReversalReference
		}
		if j.financialEventType == FinancialEventJournalReversed {
			return ErrInvalidFinancialEventType
		}
	}

	// 3. Reversal Status & Metadata Invariants
	if j.isReversed {
		if j.status != PostingStatusPosted {
			return ErrDraftCannotBeReversed
		}
		if j.transactionType == TransactionTypeReversal {
			return ErrReversalCannotBeReversed
		}
		if j.reversedByJournalID == nil || *j.reversedByJournalID == uuid.Nil {
			return ErrInconsistentReversalMetadata
		}
		if *j.reversedByJournalID == j.id {
			return ErrSelfReferencingJournal
		}
		if j.reversedAt == nil || j.reversedAt.IsZero() {
			return ErrInconsistentReversalMetadata
		}
		if j.postedAt != nil && j.reversedAt.Before(*j.postedAt) {
			return ErrReversalTimestampBeforePosting
		}
	} else {
		if j.reversedByJournalID != nil || j.reversedAt != nil {
			return ErrInconsistentReversalMetadata
		}
	}

	// 4. Ledger Entries & Balancing
	if len(j.entries) < 2 {
		return ErrInsufficientLedgerEntries
	}

	seenEntryIDs := make(map[uuid.UUID]struct{}, len(j.entries))
	hasDebit := false
	hasCredit := false

	for _, entry := range j.entries {
		if err := entry.Validate(); err != nil {
			return err
		}
		if _, exists := seenEntryIDs[entry.ID()]; exists {
			return ErrDuplicateLedgerEntryID
		}
		seenEntryIDs[entry.ID()] = struct{}{}

		if entry.Currency() != j.currency {
			return ErrCurrencyMismatch
		}
		if entry.JournalID() != j.id {
			return ErrInvalidJournalID
		}
		if entry.EntryType() == EntryTypeDebit {
			hasDebit = true
		} else if entry.EntryType() == EntryTypeCredit {
			hasCredit = true
		}
	}

	if !hasDebit {
		return ErrMissingDebitEntry
	}
	if !hasCredit {
		return ErrMissingCreditEntry
	}

	debits, err := j.TotalDebits()
	if err != nil {
		return err
	}
	credits, err := j.TotalCredits()
	if err != nil {
		return err
	}

	if !debits.Equal(credits) {
		return ErrUnbalancedJournal
	}

	return nil
}

// Post balances, validates, and permanently transitions a DRAFT journal to POSTED status.
// Once posted, the journal is immutable.
func (j *JournalTransaction) Post(postedAt time.Time) error {
	if j.status == PostingStatusPosted {
		return ErrJournalAlreadyPosted
	}

	if postedAt.IsZero() {
		postedAt = time.Now()
	}

	// Temporarily set to validate posted invariant
	j.status = PostingStatusPosted
	j.postedAt = &postedAt

	if err := j.Validate(); err != nil {
		// Roll back state on validation failure
		j.status = PostingStatusDraft
		j.postedAt = nil
		return err
	}

	return nil
}

// CreateReversal generates a new reversal journal transaction in DRAFT status that negates this posted journal.
// It creates mirror opposite entries (DEBIT -> CREDIT, CREDIT -> DEBIT) for identical amounts and accounts.
//
// Lifecycle:
// Creating a reversal does NOT immediately mark this original journal as reversed.
// The reversal journal must be validated and successfully posted before the original journal
// is marked reversed via MarkReversed(reversal).
func (j *JournalTransaction) CreateReversal(
	idempotencyKey string,
	description string,
	now time.Time,
) (*JournalTransaction, error) {
	if j.status != PostingStatusPosted {
		return nil, ErrCannotReverseUnpostedJournal
	}

	if j.transactionType == TransactionTypeReversal {
		return nil, ErrCannotReverseReversalJournal
	}

	if j.isReversed {
		return nil, ErrJournalAlreadyReversed
	}

	cleanKey := strings.TrimSpace(idempotencyKey)
	if cleanKey == "" {
		return nil, ErrMissingIdempotencyKey
	}

	cleanDesc := strings.TrimSpace(description)
	if cleanDesc == "" {
		return nil, ErrMissingDescription
	}

	if now.IsZero() {
		now = time.Now()
	}

	reversalID := uuid.New()
	reversalJournal, err := NewJournalTransaction(
		reversalID,
		TransactionTypeReversal,
		FinancialEventJournalReversed,
		j.sourceType,
		j.sourceID,
		cleanKey,
		j.currency,
		cleanDesc,
		&j.id,
		now,
	)
	if err != nil {
		return nil, err
	}

	for _, originalEntry := range j.entries {
		revEntry, err := originalEntry.Reverse(uuid.New(), reversalID, now)
		if err != nil {
			return nil, err
		}
		if err := reversalJournal.AddEntry(revEntry); err != nil {
			return nil, err
		}
	}

	// NOTE: The original journal's isReversed state remains false at this stage.
	// It will be finalized via MarkReversed(reversal) once the reversal is posted.
	return reversalJournal, nil
}

// MarkReversed records that this posted journal has been finalized as reversed by a validated, posted reversal journal.
//
// Lifecycle requirements:
//  1. This original journal must be in POSTED status.
//  2. This original journal must not already be reversed.
//  3. This original journal must not be a reversal itself.
//  4. The reversal journal must not be nil, must be in POSTED status, must be TransactionTypeReversal,
//     and its ReversalReferenceID must match this journal's ID.
func (j *JournalTransaction) MarkReversed(reversal *JournalTransaction) error {
	if j.status != PostingStatusPosted {
		return ErrCannotReverseUnpostedJournal
	}

	if j.transactionType == TransactionTypeReversal {
		return ErrCannotReverseReversalJournal
	}

	if j.isReversed {
		return ErrJournalAlreadyReversed
	}

	if reversal == nil {
		return ErrNilReversalJournal
	}

	if reversal.status != PostingStatusPosted {
		return ErrReversalJournalNotPosted
	}

	if reversal.transactionType != TransactionTypeReversal {
		return ErrInvalidTransactionType
	}

	if reversal.reversalReferenceID == nil || *reversal.reversalReferenceID != j.id {
		return ErrReversalReferenceMismatch
	}

	if reversal.postedAt == nil || reversal.postedAt.IsZero() {
		return ErrMissingPostedAt
	}

	if !j.hasMatchingReversalEntries(reversal) {
		return ErrReversalEntriesMismatch
	}

	j.isReversed = true
	revID := reversal.id
	j.reversedByJournalID = &revID
	revTime := *reversal.postedAt
	j.reversedAt = &revTime
	return nil
}

// FindEntriesByAccount returns deep copies of all entries associated with a specific financial account.
func (j *JournalTransaction) FindEntriesByAccount(accountID uuid.UUID) []LedgerEntry {
	var result []LedgerEntry
	for _, entry := range j.entries {
		if entry.AccountID() == accountID {
			result = append(result, entry.Clone())
		}
	}
	return result
}

// FindEntriesByOrganizer returns deep copies of all subledger entries tagged with a specific organizer ID.
func (j *JournalTransaction) FindEntriesByOrganizer(organizerID uuid.UUID) []LedgerEntry {
	var result []LedgerEntry
	for _, entry := range j.entries {
		orgID := entry.OrganizerID()
		if orgID != nil && *orgID == organizerID {
			result = append(result, entry.Clone())
		}
	}
	return result
}

// FindEntriesByEvent returns deep copies of all subledger entries tagged with a specific event ID.
func (j *JournalTransaction) FindEntriesByEvent(eventID uuid.UUID) []LedgerEntry {
	var result []LedgerEntry
	for _, entry := range j.entries {
		evID := entry.EventID()
		if evID != nil && *evID == eventID {
			result = append(result, entry.Clone())
		}
	}
	return result
}

// IsValidTransactionType checks whether a transaction type is supported.
func IsValidTransactionType(t TransactionType) bool {
	switch t {
	case TransactionTypePayment, TransactionTypeRefund, TransactionTypePayout, TransactionTypeAdjustment, TransactionTypeReversal:
		return true
	default:
		return false
	}
}

// IsValidFinancialEventType checks whether a financial event type is supported.
func IsValidFinancialEventType(e FinancialEventType) bool {
	switch e {
	case FinancialEventTicketSaleCaptured, FinancialEventBookingCancelled, FinancialEventOrganizerPayoutSettled,
		FinancialEventSettlementAdjustment, FinancialEventManualCorrection, FinancialEventJournalReversed,
		FinancialEventCustomerRefund, FinancialEventEventCancellationRefund, FinancialEventEscrowCapture, FinancialEventEscrowRefund:
		return true
	default:
		return false
	}
}

// IsValidSourceType checks whether a source type is supported.
func IsValidSourceType(s SourceType) bool {
	switch s {
	case SourceTypePayment, SourceTypeBooking, SourceTypeRefund, SourceTypeSettlement, SourceTypePayout, SourceTypeManual:
		return true
	default:
		return false
	}
}

// MarshalJSON provides JSON serialization for JournalTransaction.
func (j *JournalTransaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID                  uuid.UUID          `json:"id"`
		TransactionType     TransactionType    `json:"transaction_type"`
		FinancialEventType  FinancialEventType `json:"financial_event_type"`
		SourceType          SourceType         `json:"source_type"`
		SourceID            string             `json:"source_id"`
		IdempotencyKey      string             `json:"idempotency_key"`
		Currency            string             `json:"currency"`
		Description         string             `json:"description"`
		Entries             []LedgerEntry      `json:"entries"`
		ReversalReferenceID *uuid.UUID         `json:"reversal_reference_id,omitempty"`
		Status              PostingStatus      `json:"status"`
		CreatedAt           time.Time          `json:"created_at"`
		PostedAt            *time.Time         `json:"posted_at,omitempty"`
		IsReversed          bool               `json:"is_reversed"`
		ReversedByJournalID *uuid.UUID         `json:"reversed_by_journal_id,omitempty"`
		ReversedAt          *time.Time         `json:"reversed_at,omitempty"`
	}{
		ID:                  j.id,
		TransactionType:     j.transactionType,
		FinancialEventType:  j.financialEventType,
		SourceType:          j.sourceType,
		SourceID:            j.sourceID,
		IdempotencyKey:      j.idempotencyKey,
		Currency:            j.currency,
		Description:         j.description,
		Entries:             j.Entries(),
		ReversalReferenceID: cloneUUIDPtr(j.reversalReferenceID),
		Status:              j.status,
		CreatedAt:           j.createdAt,
		PostedAt:            cloneTimePtr(j.postedAt),
		IsReversed:          j.isReversed,
		ReversedByJournalID: cloneUUIDPtr(j.reversedByJournalID),
		ReversedAt:          cloneTimePtr(j.reversedAt),
	})
}

// UnmarshalJSON provides JSON deserialization for JournalTransaction.
func (j *JournalTransaction) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                  uuid.UUID          `json:"id"`
		TransactionType     TransactionType    `json:"transaction_type"`
		FinancialEventType  FinancialEventType `json:"financial_event_type"`
		SourceType          SourceType         `json:"source_type"`
		SourceID            string             `json:"source_id"`
		IdempotencyKey      string             `json:"idempotency_key"`
		Currency            string             `json:"currency"`
		Description         string             `json:"description"`
		Entries             []LedgerEntry      `json:"entries"`
		ReversalReferenceID *uuid.UUID         `json:"reversal_reference_id,omitempty"`
		Status              PostingStatus      `json:"status"`
		CreatedAt           time.Time          `json:"created_at"`
		PostedAt            *time.Time         `json:"posted_at,omitempty"`
		IsReversed          bool               `json:"is_reversed"`
		ReversedByJournalID *uuid.UUID         `json:"reversed_by_journal_id,omitempty"`
		ReversedAt          *time.Time         `json:"reversed_at,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	jt, err := ReconstituteJournalTransaction(
		aux.ID,
		aux.TransactionType,
		aux.FinancialEventType,
		aux.SourceType,
		aux.SourceID,
		aux.IdempotencyKey,
		aux.Currency,
		aux.Description,
		aux.Entries,
		aux.ReversalReferenceID,
		aux.Status,
		aux.CreatedAt,
		aux.PostedAt,
		aux.IsReversed,
		aux.ReversedByJournalID,
		aux.ReversedAt,
	)
	if err != nil {
		return err
	}

	*j = *jt
	return nil
}

// sameOptionalUUID compares optional UUID values, including nil values.
func sameOptionalUUID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}

// hasMatchingReversalEntries verifies that reversal contains exactly one
// opposite entry for every original entry, preserving duplicate entries.
func (j *JournalTransaction) hasMatchingReversalEntries(
	reversal *JournalTransaction,
) bool {
	if reversal == nil || len(j.entries) != len(reversal.entries) {
		return false
	}

	matched := make([]bool, len(reversal.entries))

	for _, original := range j.entries {
		found := false

		for i, candidate := range reversal.entries {
			if matched[i] {
				continue
			}

			if candidate.accountID != original.accountID {
				continue
			}

			if candidate.entryType != OppositeEntryType(original.entryType) {
				continue
			}

			if !candidate.amount.Equal(original.amount) {
				continue
			}

			if candidate.currency != original.currency {
				continue
			}

			if !sameOptionalUUID(candidate.organizerID, original.organizerID) {
				continue
			}

			if !sameOptionalUUID(candidate.eventID, original.eventID) {
				continue
			}

			matched[i] = true
			found = true
			break
		}

		if !found {
			return false
		}
	}

	return true
}
