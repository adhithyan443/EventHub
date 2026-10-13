package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrTicketTypeNotFound = errors.New("ticket type not found")
var ErrInsufficientTicketQuantity = errors.New("insufficient ticket quantity available")

type TicketType struct {
	ID                uuid.UUID
	EventID           uuid.UUID
	Name              string
	Price             float64
	TotalQuantity     int
	AvailableQuantity int
	Description       string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type TicketTypeRepository interface {
	Create(ctx context.Context, ticketType *TicketType) error
	FindByID(ctx context.Context, id uuid.UUID) (*TicketType, error)
	FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*TicketType, error)
	Update(ctx context.Context, ticketType *TicketType) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByEventID(ctx context.Context, eventID uuid.UUID) error
	ReserveQuantityIfAvailable(ctx context.Context, ticketTypeID uuid.UUID, quantity int) error
}
