package domain

import (
	"time"

	"github.com/google/uuid"
)

type SeatSection struct {
	ID           uuid.UUID
	SeatLayoutID uuid.UUID
	Name         string
	Price        float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SeatSectionWithStats struct {
	ID                uuid.UUID
	SeatLayoutID      uuid.UUID
	Name              string
	Price             float64
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
}
