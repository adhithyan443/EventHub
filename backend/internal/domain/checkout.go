package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	CheckoutStatusActive    = "ACTIVE"
	CheckoutStatusCompleted = "COMPLETED"
	CheckoutStatusExpired   = "EXPIRED"
)

var ErrCheckoutNotFound = errors.New("checkout not found")

// Checkout stores a customer's quoted price for a reservation.
// Monetary values are snapshots and must not change during payment retries.
type Checkout struct {
	ID             uuid.UUID
	ReservationID  uuid.UUID
	UserID         uuid.UUID
	Subtotal       decimal.Decimal
	PlatformFee    decimal.Decimal
	TaxAmount      decimal.Decimal
	DiscountAmount decimal.Decimal
	TotalAmount    decimal.Decimal
	Status         string
	ExpiresAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CheckoutRepository interface {
	Create(ctx context.Context, checkout *Checkout) error

	FindByID(ctx context.Context, id uuid.UUID) (*Checkout, error)

	FindByReservationID(
		ctx context.Context,
		reservationID uuid.UUID,
	) (*Checkout, error)

	UpdateStatus(
		ctx context.Context,
		id uuid.UUID,
		status string,
	) error
}
