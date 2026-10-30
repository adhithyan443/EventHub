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

type SeatDetail struct {
	SeatID      uuid.UUID `json:"seat_id"`
	SeatNumber  int       `json:"seat_number"`
	RowName     string    `json:"row_name"`
	SectionName string    `json:"section_name"`
	Label       string    `json:"label"`
}

type SeatRepository interface {
	Create(seat *Seat) error
	Update(seat *Seat) error
	Delete(id uuid.UUID) error
	DeleteByRowID(rowID uuid.UUID) error
	FindByRowID(rowID uuid.UUID) ([]Seat, error)
	FindByID(seatID uuid.UUID) (*Seat, error)
	FindDetailsBySeatIDs(ctx context.Context, seatIDs []uuid.UUID) (map[uuid.UUID]SeatDetail, error)
	CountBookedByLayoutID(layoutID uuid.UUID) (int64, error)
	ReserveIfAvailable(ctx context.Context, seatID uuid.UUID) error
	ReleaseReservedSeat(ctx context.Context, seatID uuid.UUID) error
	BookReservedSeat(ctx context.Context, seatID uuid.UUID) error
	ReleaseBookedSeat(ctx context.Context, seatID uuid.UUID) error
}
