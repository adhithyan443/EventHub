package domain

import (
	"context"
	"errors"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrInconsistentBookingAmounts     = errors.New("booking amount breakdown does not match total or payment amount")
	ErrTaxAccountNotConfigured         = errors.New("tax account not configured in chart of accounts")
	ErrUnsupportedDiscountAllocation   = errors.New("unsupported discount allocation in financial ledger")
	ErrFinanceAccountInactive         = errors.New("finance account is inactive")
)

// RecordPaymentCaptureInput represents the validated financial parameters required to record
// a captured customer ticket booking payment into the general ledger.
type RecordPaymentCaptureInput struct {
	PaymentID     uuid.UUID
	PaymentAmount decimal.Decimal
	Currency      string

	BookingID   uuid.UUID
	Subtotal    decimal.Decimal
	PlatformFee decimal.Decimal
	Tax         decimal.Decimal
	Discount    decimal.Decimal
	Total       decimal.Decimal

	EventID     uuid.UUID
	OrganizerID uuid.UUID
	UserID      uuid.UUID

	CapturedAt time.Time
}

// FinanceUsecase defines double-entry financial accounting use cases for EventHub.
type FinanceUsecase interface {
	RecordPaymentCaptureWithinTx(
		ctx context.Context,
		txRepos TransactionRepositories,
		in RecordPaymentCaptureInput,
	) (*finance.JournalTransaction, error)

	RecordPaymentCapture(
		ctx context.Context,
		in RecordPaymentCaptureInput,
	) (*finance.JournalTransaction, error)
}
