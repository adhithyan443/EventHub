package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSeatColumnNotFound       = errors.New("seat column not found")
	ErrSeatColumnDuplicateLabel = errors.New("duplicate column label")
	ErrSeatColumnDuplicateOrder = errors.New("duplicate column order index")
)

type SeatColumn struct {
	ID           uuid.UUID `json:"id"`
	SeatLayoutID uuid.UUID `json:"seat_layout_id"`
	Label        string    `json:"label"`
	OrderIndex   int       `json:"order_index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SeatColumnRepository interface {
	Create(column *SeatColumn) error
	CreateBatch(columns []SeatColumn) error
	Update(column *SeatColumn) error
	Delete(id uuid.UUID) error
	DeleteBySeatLayoutID(layoutID uuid.UUID) error
	FindBySeatLayoutID(layoutID uuid.UUID) ([]SeatColumn, error)
	FindByID(id uuid.UUID) (*SeatColumn, error)
}
