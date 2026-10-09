package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	SeatStatusAvailable = "AVAILABLE"
	SeatStatusReserved  = "RESERVED"
	SeatStatusBooked    = "BOOKED"
	SeatStatusDisabled  = "DISABLED"
)

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
	CountBookedByLayoutID(layoutID uuid.UUID) (int64, error)
}
