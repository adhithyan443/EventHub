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
