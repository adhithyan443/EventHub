package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrBookingNotFound             = errors.New("booking not found")
	ErrDuplicateReservationBooking = errors.New(
		"booking already exists for reservation",
	)
	ErrBookingAlreadyCancelled    = errors.New("booking is already cancelled")
	ErrBookingCannotBeCancelled   = errors.New("booking cannot be cancelled in its current state")
	ErrCancellationNotAllowed     = errors.New("cancellation is not allowed for this booking")
	ErrCancellationDeadlinePassed = errors.New("cancellation deadline has passed")
	ErrTicketAlreadyUsed          = errors.New("booking tickets have already been used and cannot be cancelled")
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

	CancellationAllowed       bool   `json:"cancellation_allowed"`
	CancellationDeadlineHours int    `json:"cancellation_deadline_hours"`
	RefundPolicy              string `json:"refund_policy"`
	RefundPercentage          int    `json:"refund_percentage"`

	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time

	Seats []SeatDetail `json:"seats,omitempty"`
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

	Seat *SeatDetail `json:"seat,omitempty"`
}

type BookingWithItems struct {
	Booking
	Items     []BookingItem `json:"items"`
	QRPayload string        `json:"qr_payload,omitempty"`
}

type BookingRepository interface {
	Create(
		ctx context.Context,
		booking *Booking,
		items []BookingItem,
	) error

	FindByID(ctx context.Context, id uuid.UUID) (*Booking, error)

	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*Booking, error)

	FindByBookingNumber(
		ctx context.Context,
		bookingNumber string,
	) (*Booking, error)

	FindByReservationID(
		ctx context.Context,
		reservationID uuid.UUID,
	) (*Booking, error)

	FindByUserID(
		ctx context.Context,
		userID uuid.UUID,
	) ([]Booking, error)

	FindWithItemsByID(
		ctx context.Context,
		id uuid.UUID,
	) (*Booking, []BookingItem, error)

	UpdateStatus(
		ctx context.Context,
		id uuid.UUID,
		status string,
	) error

	ConfirmPending(
		ctx context.Context,
		id uuid.UUID,
	) error

	FindItemsByBookingID(
		ctx context.Context,
		bookingID uuid.UUID,
	) ([]BookingItem, error)

	FindItemsByBookingIDs(
		ctx context.Context,
		bookingIDs []uuid.UUID,
	) (map[uuid.UUID][]BookingItem, error)

	CountConfirmedByEventID(
		ctx context.Context,
		eventID uuid.UUID,
	) (int64, error)

	CountByEventID(
		ctx context.Context,
		eventID uuid.UUID,
	) (int64, error)

	GetBookingStatsByEventID(
		ctx context.Context,
		eventID uuid.UUID,
	) (AdminEventBookingStats, error)
}
