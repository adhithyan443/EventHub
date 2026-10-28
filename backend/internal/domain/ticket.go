package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrTicketNotFound   = errors.New("ticket not found")
	ErrTicketCodeExists = errors.New("ticket code already exists")
	ErrTicketNotValid   = errors.New("ticket is not valid")
)

const (
	TicketStatusValid     = "VALID"
	TicketStatusUsed      = "USED"
	TicketStatusCancelled = "CANCELLED"
	TicketStatusRefunded  = "REFUNDED"
)

type Ticket struct {
	ID              uuid.UUID
	TicketCode      string
	BookingID       uuid.UUID
	BookingItemID   uuid.UUID
	UserID          uuid.UUID
	EventID         uuid.UUID
	TicketTypeID    *uuid.UUID
	AdmissionNumber int
	SeatID          *uuid.UUID
	Price           decimal.Decimal
	Status          string
	IssuedAt        time.Time
	UsedAt          *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Seat *SeatDetail `json:"seat,omitempty"`
}

type TicketRepository interface {
	Create(ctx context.Context, ticket *Ticket) error

	CreateMany(ctx context.Context, tickets []Ticket) error

	FindByID(ctx context.Context, id uuid.UUID) (*Ticket, error)

	FindByCode(ctx context.Context, code string) (*Ticket, error)

	FindByBookingID(
		ctx context.Context,
		bookingID uuid.UUID,
	) ([]Ticket, error)

	FindByUserID(
		ctx context.Context,
		userID uuid.UUID,
	) ([]Ticket, error)

	UpdateStatus(
		ctx context.Context,
		id uuid.UUID,
		status string,
	) error

	UpdateStatusByBookingID(
		ctx context.Context,
		bookingID uuid.UUID,
		status string,
	) error
}
