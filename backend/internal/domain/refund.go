package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrRefundNotFound        = errors.New("refund not found")
	ErrRefundAlreadyProcessed = errors.New("refund has already been processed")
	ErrGatewayRefundFailed   = errors.New("payment gateway failed to process refund")
	ErrRefundInProgress      = errors.New("refund is already in progress")
)

type RefundStatus = string

const (
	RefundStatusPending RefundStatus = "PENDING"
	RefundStatusSuccess RefundStatus = "SUCCESS"
	RefundStatusFailed  RefundStatus = "FAILED"
)

type Refund struct {
	ID              uuid.UUID
	BookingID       uuid.UUID
	PaymentID       uuid.UUID
	GatewayRefundID *string
	Amount          decimal.Decimal
	Status          string
	Reason          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ProcessedAt     *time.Time
}

type RefundResult struct {
	RefundID string
	Amount   decimal.Decimal
	Status   string
	Receipt  string
}

type PaymentRefundProvider interface {
	Refund(
		ctx context.Context,
		gatewayPaymentID string,
		amount decimal.Decimal,
		currency string,
		idempotencyKey string,
	) (*RefundResult, error)
	FetchRefunds(
		ctx context.Context,
		gatewayPaymentID string,
	) ([]RefundResult, error)
}

type RefundRepository interface {
	Create(ctx context.Context, refund *Refund) error
	FindByID(ctx context.Context, id uuid.UUID) (*Refund, error)
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*Refund, error)
	FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]Refund, error)
	Update(ctx context.Context, refund *Refund) error
}
