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

type SeatSectionRepository interface {
	Create(section *SeatSection) error
	FindBySeatLayoutID(seatLayoutID uuid.UUID) ([]SeatSection, error)
}
