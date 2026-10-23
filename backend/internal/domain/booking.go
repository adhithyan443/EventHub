package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrBookingNotFound = errors.New("booking not found")
var ErrDuplicateReservationBooking = errors.New(
	"booking already exists for reservation",
)

const (
	BookingStatusPending   = "PENDING"
	BookingStatusConfirmed = "CONFIRMED"
	BookingStatusCancelled = "CANCELLED"
)

type Booking struct {
	ID            uuid.UUID
	BookingNumber string
	UserID        uuid.UUID
	EventID       uuid.UUID
	ReservationID *uuid.UUID

	Subtotal    decimal.Decimal
	PlatformFee decimal.Decimal
	Tax         decimal.Decimal
	Discount    decimal.Decimal
	Total       decimal.Decimal

	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type BookingItem struct {
	ID           uuid.UUID
	BookingID    uuid.UUID
	SeatID       *uuid.UUID
	TicketTypeID *uuid.UUID
	Quantity     int
	UnitPrice    decimal.Decimal
	TotalPrice   decimal.Decimal
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type BookingRepository interface {
	Create(
		ctx context.Context,
		booking *Booking,
		items []BookingItem,
	) error

	FindByID(ctx context.Context, id uuid.UUID) (*Booking, error)

	FindByBookingNumber(
		ctx context.Context,
		bookingNumber string,
	) (*Booking, error)

	FindByReservationID(
		ctx context.Context,
		reservationID uuid.UUID,
	) (*Booking, error)

	UpdateStatus(
		ctx context.Context,
		id uuid.UUID,
		status string,
	) error

	ConfirmPending(
		ctx context.Context,
		id uuid.UUID,
	) error
}
