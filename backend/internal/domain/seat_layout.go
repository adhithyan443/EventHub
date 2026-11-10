package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrSeatLayoutNotFound = errors.New("seat layout not found")

type SeatLayout struct {
	ID         uuid.UUID
	EventID    uuid.UUID
	LayoutName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type SeatLayoutRepository interface {
	Create(layout *SeatLayout) error
	Update(layout *SeatLayout) error
	FindByEventID(eventID uuid.UUID) (*SeatLayout, error)
	DeleteByEventID(eventID uuid.UUID) error
}

type PublicSeatLayout struct {
	ID         uuid.UUID           `json:"id"`
	EventID    uuid.UUID           `json:"event_id"`
	LayoutName string              `json:"layout_name"`
	Columns    []PublicSeatColumn  `json:"columns"`
	Sections   []PublicSeatSection `json:"sections"`
}

type PublicSeatColumn struct {
	ID         uuid.UUID `json:"id"`
	Label      string    `json:"label"`
	OrderIndex int       `json:"order_index"`
}

type PublicSeatSection struct {
	ID    uuid.UUID       `json:"id"`
	Name  string          `json:"name"`
	Price float64         `json:"price"`
	Rows  []PublicSeatRow `json:"rows"`
}

type PublicSeatRow struct {
	ID      uuid.UUID    `json:"id"`
	RowName string       `json:"row_name"`
	Seats   []PublicSeat `json:"seats"`
}

type PublicSeat struct {
	ID          uuid.UUID  `json:"id"`
	ColumnID    *uuid.UUID `json:"column_id,omitempty"`
	GridCol     int        `json:"grid_col"`
	ColumnLabel string     `json:"column_label,omitempty"`
	SeatNumber  int        `json:"seat_number"`
	Status      string     `json:"status"`
}
