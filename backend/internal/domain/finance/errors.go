package finance

import "errors"

var (
	// Financial Account Errors
	ErrInvalidAccountID      = errors.New("invalid financial account id")
	ErrInvalidAccountCode    = errors.New("invalid financial account code")
	ErrInvalidAccountName    = errors.New("invalid financial account name")
	ErrInvalidAccountType    = errors.New("invalid financial account type")
	ErrInvalidNormalBalance  = errors.New("invalid normal balance")
	ErrNormalBalanceMismatch = errors.New("normal balance does not match standard accounting convention for account type")
	ErrAccountInactive       = errors.New("financial account is inactive")

	// Currency & Money Errors
	ErrUnsupportedCurrency = errors.New("unsupported currency: only INR is supported")
	ErrFractionalPaise     = errors.New("monetary amount cannot have fractional paise")
	ErrNonPositiveMoney    = errors.New("money amount must be strictly positive")
	ErrNegativeMoney       = errors.New("money amount cannot be negative")
	ErrCurrencyMismatch    = errors.New("currency mismatch between monetary amounts")

	// Ledger Entry Errors
	ErrInvalidEntryID         = errors.New("invalid ledger entry id")
	ErrInvalidEntryType       = errors.New("invalid ledger entry type: must be DEBIT or CREDIT")
	ErrInvalidOrganizerID     = errors.New("invalid organizer id")
	ErrInvalidEventID         = errors.New("invalid event id")
	ErrInvalidEntryTimestamp  = errors.New("invalid or missing ledger entry creation timestamp")
	ErrDuplicateLedgerEntryID = errors.New("duplicate ledger entry id within journal")

	// Journal Transaction Errors
	ErrInvalidJournalID               = errors.New("invalid journal transaction id")
	ErrInvalidTransactionType         = errors.New("invalid journal transaction type")
	ErrInvalidFinancialEventType      = errors.New("invalid financial event type")
	ErrMissingSourceType              = errors.New("missing journal source type")
	ErrMissingSourceID                = errors.New("missing journal source id")
	ErrMissingIdempotencyKey          = errors.New("missing journal idempotency key")
	ErrMissingDescription             = errors.New("missing journal description")
	ErrInsufficientLedgerEntries      = errors.New("journal must contain at least two ledger entries (at least one debit and one credit)")
	ErrMissingDebitEntry              = errors.New("journal must contain at least one debit entry")
	ErrMissingCreditEntry             = errors.New("journal must contain at least one credit entry")
	ErrUnbalancedJournal              = errors.New("journal is unbalanced: total debits must equal total credits")
	ErrPostedJournalImmutable         = errors.New("posted journal transaction is immutable and cannot be modified")
	ErrJournalAlreadyPosted           = errors.New("journal transaction is already posted")
	ErrJournalAlreadyReversed         = errors.New("journal transaction has already been reversed")
	ErrCannotReverseUnpostedJournal   = errors.New("cannot reverse an unposted journal transaction")
	ErrCannotReverseReversalJournal   = errors.New("cannot reverse a journal that is already a reversal")
	ErrInvalidReversalReference       = errors.New("invalid reversal reference id: reversal must reference a valid original journal")
	ErrInvalidJournalPostingState     = errors.New("invalid journal posting status")
	ErrMissingPostedAt                = errors.New("posted journal transaction must have a valid posted timestamp")
	ErrDraftCannotHavePostedAt        = errors.New("draft journal transaction cannot have a posted timestamp")
	ErrDraftCannotBeReversed          = errors.New("draft journal transaction cannot be marked as reversed")
	ErrInconsistentReversalMetadata   = errors.New("inconsistent reversal metadata: reversed journal must have valid reversing journal ID and reversed timestamp")
	ErrReversalTimestampBeforePosting = errors.New("journal reversal timestamp cannot be before posting timestamp")
	ErrReversalCannotBeReversed       = errors.New("a reversal journal cannot be marked as reversed")
	ErrSelfReferencingJournal         = errors.New("journal transaction cannot reference itself")
	ErrUnexpectedReversalReference    = errors.New("non-reversal journal cannot have a reversal reference ID")
	ErrReversalJournalNotPosted       = errors.New("cannot mark original journal as reversed with an unposted reversal journal")
	ErrNilReversalJournal             = errors.New("reversal journal transaction cannot be nil")
	ErrReversalReferenceMismatch      = errors.New("reversal journal reference ID does not match original journal ID")
	ErrReversalEntriesMismatch        = errors.New("reversal journal entries do not match the original journal entries")
)
