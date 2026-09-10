package models

import (
	"time"

	"github.com/google/uuid"
)

type EventCancellationModel struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	EventID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	CancellationAllowed       bool `gorm:"not null;default:false"`
	CancellationDeadlineHours int  `gorm:"not null;default:0"`

	CancelledAt        *time.Time `gorm:"type:timestamp"`
	CancellationReason *string    `gorm:"type:text"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (EventCancellationModel) TableName() string {
	return "event_cancellations"
}
