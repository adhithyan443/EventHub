package domain

import (
	"time"

	"github.com/google/uuid"
)

type SeatLayout struct {
	ID         uuid.UUID
	EventID    uuid.UUID
	LayoutName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
