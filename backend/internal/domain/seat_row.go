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
	FindBySectionID(sectionID uuid.UUID) ([]SeatRow, error)
}
