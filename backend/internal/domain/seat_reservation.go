package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ReservationStatusActive    = "ACTIVE"
	ReservationStatusExpired   = "EXPIRED"
	ReservationStatusCompleted = "COMPLETED"
	ReservationStatusCancelled = "CANCELLED"
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
