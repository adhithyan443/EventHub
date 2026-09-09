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
