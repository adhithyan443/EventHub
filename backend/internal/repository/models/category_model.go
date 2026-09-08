package models

import (
	"time"

	"github.com/google/uuid"
)

type CategoryModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Description string    `gorm:"type:text"`
	Status      string    `gorm:"type:varchar(20);not null;index"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"not null;autoUpdateTime"`
}

func (CategoryModel) TableName() string {
	return "categories"
}
