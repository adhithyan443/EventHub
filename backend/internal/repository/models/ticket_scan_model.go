package models

import (
	"time"

	"github.com/google/uuid"
)

type TicketScanModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TicketID  uuid.UUID `gorm:"type:uuid;not null;index"`
	ScannedBy uuid.UUID `gorm:"type:uuid;not null;index"`
	Status    string    `gorm:"type:varchar(20);not null"`
	ScannedAt time.Time `gorm:"not null"`
}

func (TicketScanModel) TableName() string {
	return "ticket_scans"
}