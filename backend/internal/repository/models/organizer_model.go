package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizerModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Status    string    `gorm:"type:varchar(30);not null;index"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (OrganizerModel) TableName() string {
	return "organizers"
}
