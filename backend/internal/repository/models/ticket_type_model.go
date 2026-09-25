package models

import (
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type TicketTypeModel struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	EventID           uuid.UUID `gorm:"type:uuid;not null;index"`
	Name              string    `gorm:"type:varchar(100);not null"`
	Price             float64   `gorm:"type:numeric(10,2);not null"`
	TotalQuantity     int       `gorm:"not null"`
	AvailableQuantity int       `gorm:"not null"`
	Description       string    `gorm:"type:text"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (TicketTypeModel) TableName() string {
	return "ticket_types"
}

func (m *TicketTypeModel) ToDomain() *domain.TicketType {
	return &domain.TicketType{
		ID:                m.ID,
		EventID:           m.EventID,
		Name:              m.Name,
		Price:             m.Price,
		TotalQuantity:     m.TotalQuantity,
		AvailableQuantity: m.AvailableQuantity,
		Description:       m.Description,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func TicketTypeModelFromDomain(ticketType *domain.TicketType) *TicketTypeModel {
	return &TicketTypeModel{
		ID:                ticketType.ID,
		EventID:           ticketType.EventID,
		Name:              ticketType.Name,
		Price:             ticketType.Price,
		TotalQuantity:     ticketType.TotalQuantity,
		AvailableQuantity: ticketType.AvailableQuantity,
		Description:       ticketType.Description,
		CreatedAt:         ticketType.CreatedAt,
		UpdatedAt:         ticketType.UpdatedAt,
	}
}
