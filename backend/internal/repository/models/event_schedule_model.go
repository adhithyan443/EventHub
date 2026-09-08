package models

import (
	"time"

	"github.com/google/uuid"
)

type EventScheduleModel struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	EventID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	EventDate time.Time `gorm:"type:date;not null"`
	StartTime time.Time `gorm:"type:time;not null"`
	EndTime   time.Time `gorm:"type:time;not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (EventScheduleModel) TableName() string {
	return "event_schedules"
}