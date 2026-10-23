package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	SeatStatusAvailable = "AVAILABLE"
	SeatStatusReserved  = "RESERVED"
	SeatStatusBooked    = "BOOKED"
	SeatStatusDisabled  = "DISABLED"
)

var ErrSeatNotAvailable = errors.New("seat is not available")
var ErrSeatNotFound = errors.New("seat not found")

type Seat struct {
	ID         uuid.UUID
	RowID      uuid.UUID
	SeatNumber int
	Status     string
	CreatedAt  time.Time
}

type SeatRepository interface {
	Create(seat *Seat) error
	Update(seat *Seat) error
	Delete(id uuid.UUID) error
	DeleteByRowID(rowID uuid.UUID) error
	FindByRowID(rowID uuid.UUID) ([]Seat, error)
	FindByID(seatID uuid.UUID) (*Seat, error)
	CountBookedByLayoutID(layoutID uuid.UUID) (int64, error)
	ReserveIfAvailable(ctx context.Context, seatID uuid.UUID) error
	ReleaseReservedSeat(ctx context.Context, seatID uuid.UUID) error
	BookReservedSeat(ctx context.Context, seatID uuid.UUID) error
}
