package models

import (
	"time"

	"github.com/google/uuid"
)

type SeatModel struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	RowID      uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_row_seat"`
	SeatNumber int       `gorm:"not null;uniqueIndex:idx_row_seat"`

	Status string `gorm:"type:varchar(20);not null;index"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
}

func (SeatModel) TableName() string {
	return "seats"
}
