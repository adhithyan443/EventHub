package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	ReservationStatusActive    = "ACTIVE"
	ReservationStatusExpired   = "EXPIRED"
	ReservationStatusCompleted = "COMPLETED"
	ReservationStatusCancelled = "CANCELLED"
)

var ErrSeatReservationNotFound = errors.New("seat reservation not found")

var (
	ErrInvalidReservationRequest = errors.New("invalid reservation request")
	// ErrEventNotFound             = errors.New("event not found")
	ErrTicketTypeEventMismatch = errors.New("ticket type does not belong to event")
)

// SeatReservation represents a temporary hold on tickets or seats.
type SeatReservation struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	EventID   uuid.UUID
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SeatReservationRepository interface {
	Create(ctx context.Context, reservation *SeatReservation) error

	CreateItem(ctx context.Context, item *SeatReservationItem) error

	FindByID(ctx context.Context, id uuid.UUID) (*SeatReservation, error)

	FindItemsByReservationID(
		ctx context.Context,
		reservationID uuid.UUID,
	) ([]SeatReservationItem, error)

	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error

	ExpireIfActive(ctx context.Context, id uuid.UUID, now time.Time) (bool, error)
	FindExpiredActive( ctx context.Context, now time.Time,limit int) ([]SeatReservation, error)
	
}

// SeatReservationItem represents either a specific seat or
// a quantity of tickets belonging to a reservation.
//
// Reserved seat:
//
//	SeatID is set.
//	TicketTypeID is nil.
//	Quantity is 1.
//
// General Admission:
//
//	SeatID is nil.
//	TicketTypeID is set.
//	Quantity is the number of tickets.
type SeatReservationItem struct {
	ID            uuid.UUID
	ReservationID uuid.UUID
	SeatID        *uuid.UUID
	TicketTypeID  *uuid.UUID
	Quantity      int
	Price         float64
	CreatedAt     time.Time
}
