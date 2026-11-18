package domain

import (
	"context"
	"errors"

	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/google/uuid"
)

var (
	ErrFinanceAccountNotFound      = errors.New("finance account not found")
	ErrFinanceAccountAlreadyExists = errors.New("finance account already exists")
	ErrFinanceJournalNotFound      = errors.New("finance journal not found")
	ErrIdempotencyConflict         = errors.New("idempotency key already exists with conflicting transaction payload")
	ErrConcurrentPosting           = errors.New("journal posting conflict or concurrent modification detected")
)

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
}
