package models

import (
	"time"

	"github.com/google/uuid"
)

type SeatRowModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	SectionID uuid.UUID `gorm:"type:uuid;not null;index"`
	RowName   string    `gorm:"type:varchar(50);not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
}

func (SeatRowModel) TableName() string {
	return "seat_rows"
}
