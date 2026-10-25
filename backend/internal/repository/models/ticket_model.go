package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TicketModel struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TicketCode    string          `gorm:"type:varchar(100);uniqueIndex;not null"`
	BookingID     uuid.UUID       `gorm:"type:uuid;not null;index"`
	BookingItemID uuid.UUID       `gorm:"type:uuid;not null;index"`
	UserID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	EventID       uuid.UUID       `gorm:"type:uuid;not null;index"`
	TicketTypeID  *uuid.UUID      `gorm:"type:uuid"`
	SeatID        *uuid.UUID      `gorm:"type:uuid;index"`
	Price         decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	Status        string          `gorm:"type:varchar(20);not null;default:VALID;index"`
	IssuedAt      time.Time       `gorm:"not null"`
	UsedAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (TicketModel) TableName() string {
	return "tickets"
}

func (t *TicketModel) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}

	return nil
}
