package domain

import (
	"context"
	"errors"

	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrFinanceAccountNotFound      = errors.New("finance account not found")
	ErrFinanceAccountAlreadyExists = errors.New("finance account already exists")
	ErrFinanceJournalNotFound      = errors.New("finance journal not found")
	ErrIdempotencyConflict         = errors.New("idempotency key already exists with conflicting transaction payload")
	ErrConcurrentPosting           = errors.New("journal posting conflict or concurrent modification detected")
)

// OrganizerSubledgerBalance represents the aggregate posted financial ledger balance
// for an organizer on Account 2010 (Organizer Payable).
type OrganizerSubledgerBalance struct {
	OrganizerID     uuid.UUID
	GrossEarnings   decimal.Decimal // Total credits from TICKET_SALE_CAPTURED
	RefundsDeducted decimal.Decimal // Total debits from CUSTOMER_REFUND and EVENT_CANCELLATION_REFUND
	NetEarnings     decimal.Decimal // GrossEarnings - RefundsDeducted
	SettledAmount   decimal.Decimal // Total debits from ORGANIZER_PAYOUT_SETTLED
	TotalCredits    decimal.Decimal // All credits to Account 2010
	TotalDebits     decimal.Decimal // All debits from Account 2010
	NetPayable      decimal.Decimal // TotalCredits - TotalDebits
	Currency        string
}

// EventSubledgerBalance represents the aggregate posted financial ledger balance
// for an individual event on Account 2010 (Organizer Payable).
type EventSubledgerBalance struct {
	EventID         uuid.UUID
	OrganizerID     uuid.UUID
	GrossEarnings   decimal.Decimal // Event credits from TICKET_SALE_CAPTURED
	RefundsDeducted decimal.Decimal // Event debits from CUSTOMER_REFUND and EVENT_CANCELLATION_REFUND
	NetEarnings     decimal.Decimal // GrossEarnings - RefundsDeducted
	SettledAmount   decimal.Decimal // Event debits from ORGANIZER_PAYOUT_SETTLED
	TotalCredits    decimal.Decimal // All credits for this event on Account 2010
	TotalDebits     decimal.Decimal // All debits for this event on Account 2010
	NetPayable      decimal.Decimal // TotalCredits - TotalDebits
	Currency        string
}

type FinanceAccountRepository interface {
	Create(ctx context.Context, account *finance.Account) error
	FindByID(ctx context.Context, id uuid.UUID) (*finance.Account, error)
	FindByCode(ctx context.Context, code string) (*finance.Account, error)
}

type FinanceJournalRepository interface {
	RecordJournal(
		ctx context.Context,
		journal *finance.JournalTransaction,
	) (*finance.JournalTransaction, error)

	FindByID(
		ctx context.Context,
		id uuid.UUID,
	) (*finance.JournalTransaction, error)

	FindByIdempotencyKey(
		ctx context.Context,
		key string,
	) (*finance.JournalTransaction, error)

	// GetOrganizerBalance computes the aggregate posted Account 2010 balance for an organizer.
	GetOrganizerBalance(
		ctx context.Context,
		organizerID uuid.UUID,
	) (*OrganizerSubledgerBalance, error)

	// GetOrganizerEventBalances computes the posted Account 2010 balances for each event owned by an organizer.
	GetOrganizerEventBalances(
		ctx context.Context,
		organizerID uuid.UUID,
	) ([]EventSubledgerBalance, error)

	// GetEventBalance computes the posted Account 2010 balance for a single event owned by an organizer.
	GetEventBalance(
		ctx context.Context,
		organizerID uuid.UUID,
		eventID uuid.UUID,
	) (*EventSubledgerBalance, error)
}
