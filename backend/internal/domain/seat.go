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
	ColumnID   *uuid.UUID
	GridCol    int
	SeatNumber int
	Status     string
	CreatedAt  time.Time
}

type SeatDetail struct {
	SeatID      uuid.UUID  `json:"seat_id"`
	SeatNumber  int        `json:"seat_number"`
	ColumnID    *uuid.UUID `json:"column_id,omitempty"`
	GridCol     int        `json:"grid_col"`
	ColumnLabel string     `json:"column_label,omitempty"`
	RowName     string     `json:"row_name"`
	SectionName string     `json:"section_name"`
	Label       string     `json:"label"`
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
	CountBookedOrReservedByColumnID(ctx context.Context, columnID uuid.UUID) (int64, error)
	CountBookedOrReservedBySeatIDs(ctx context.Context, seatIDs []uuid.UUID) (int64, error)
	FindByColumnID(ctx context.Context, columnID uuid.UUID) ([]Seat, error)
	ReserveIfAvailable(ctx context.Context, seatID uuid.UUID) error
	ReleaseReservedSeat(ctx context.Context, seatID uuid.UUID) error
	BookReservedSeat(ctx context.Context, seatID uuid.UUID) error
	ReleaseBookedSeat(ctx context.Context, seatID uuid.UUID) error
}
