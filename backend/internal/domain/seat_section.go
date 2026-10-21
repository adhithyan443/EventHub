package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrSeatSectionNotFound = errors.New("seat section not found")

type SeatSection struct {
	ID           uuid.UUID
	SeatLayoutID uuid.UUID
	Name         string
	Price        decimal.Decimal
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SeatSectionWithStats struct {
	ID                uuid.UUID
	SeatLayoutID      uuid.UUID
	Name              string
	Price             decimal.Decimal
	TotalCapacity     int
	AvailableQuantity int
	SoldQuantity      int
}

type SeatSectionRepository interface {
	Create(section *SeatSection) error
	Update(section *SeatSection) error
	Delete(id uuid.UUID) error
	DeleteBySeatLayoutID(seatLayoutID uuid.UUID) error
	FindBySeatLayoutID(seatLayoutID uuid.UUID) ([]SeatSection, error)
	FindWithStatsBySeatLayoutID(seatLayoutID uuid.UUID) ([]SeatSectionWithStats, error)
	FindByID(id uuid.UUID) (*SeatSection, error)
}
