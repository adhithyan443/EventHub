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
	ErrInvalidRefundAmountInput       = errors.New("refund amount must not be negative")
	ErrInvalidPlatformFeeRefundAmount = errors.New("platform fee refund amount is invalid")
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

// RecordRefundInput represents the validated parameters required to record a refund into the general ledger.
type RecordRefundInput struct {
	RefundID            uuid.UUID
	RefundAmount        decimal.Decimal
	PaymentID           uuid.UUID
	BookingID           uuid.UUID
	EventID             uuid.UUID
	OrganizerID         uuid.UUID
	IsEventCancellation bool
	IsEscrowRefund      bool
	PlatformFeeRefund   decimal.Decimal
	Currency            string
	ProcessedAt         time.Time
}

// RecordEscrowCaptureInput represents the validated parameters required to record incoming funds
// for a cancelled booking into escrow liability.
type RecordEscrowCaptureInput struct {
	PaymentID     uuid.UUID
	PaymentAmount decimal.Decimal
	BookingID     uuid.UUID
	Currency      string
	CapturedAt    time.Time
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

	RecordRefundWithinTx(
		ctx context.Context,
		txRepos TransactionRepositories,
		in RecordRefundInput,
	) (*finance.JournalTransaction, error)

	RecordRefund(
		ctx context.Context,
		in RecordRefundInput,
	) (*finance.JournalTransaction, error)

	RecordEscrowCaptureWithinTx(
		ctx context.Context,
		txRepos TransactionRepositories,
		in RecordEscrowCaptureInput,
	) (*finance.JournalTransaction, error)

	RecordEscrowCapture(
		ctx context.Context,
		in RecordEscrowCaptureInput,
	) (*finance.JournalTransaction, error)
}
