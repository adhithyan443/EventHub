package domain

import (
	"time"

	"github.com/google/uuid"
)

type SeatRow struct {
	ID        uuid.UUID
	SectionID uuid.UUID
	RowName   string
	CreatedAt time.Time
}

type SeatRowRepository interface {
	Create(row *SeatRow) error
	Update(row *SeatRow) error
	Delete(id uuid.UUID) error
	DeleteBySectionID(sectionID uuid.UUID) error
	FindBySectionID(sectionID uuid.UUID) ([]SeatRow, error)
}
