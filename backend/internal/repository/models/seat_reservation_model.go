package models

import (
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SeatReservationModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	EventID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Status    string    `gorm:"type:varchar(20);not null;index"`
	ExpiresAt time.Time `gorm:"not null;index"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (SeatReservationModel) TableName() string {
	return "seat_reservations"
}

func (m *SeatReservationModel) ToDomain() *domain.SeatReservation {
	return &domain.SeatReservation{
		ID:        m.ID,
		UserID:    m.UserID,
		EventID:   m.EventID,
		Status:    m.Status,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func SeatReservationModelFromDomain(
	reservation *domain.SeatReservation,
) *SeatReservationModel {
	return &SeatReservationModel{
		ID:        reservation.ID,
		UserID:    reservation.UserID,
		EventID:   reservation.EventID,
		Status:    reservation.Status,
		ExpiresAt: reservation.ExpiresAt,
		CreatedAt: reservation.CreatedAt,
		UpdatedAt: reservation.UpdatedAt,
	}
}

// SeatReservationItemModel represents one reserved seat or a
// quantity of General Admission tickets.
type SeatReservationItemModel struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey"`
	ReservationID uuid.UUID       `gorm:"type:uuid;not null;index"`
	SeatID        *uuid.UUID      `gorm:"type:uuid;index"`
	TicketTypeID  *uuid.UUID      `gorm:"type:uuid;index"`
	Quantity      int             `gorm:"not null;default:1"`
	Price         decimal.Decimal `gorm:"type:numeric(10,2);not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
}

func (SeatReservationItemModel) TableName() string {
	return "seat_reservation_items"
}

func (m *SeatReservationItemModel) ToDomain() *domain.SeatReservationItem {
	return &domain.SeatReservationItem{
		ID:            m.ID,
		ReservationID: m.ReservationID,
		SeatID:        m.SeatID,
		TicketTypeID:  m.TicketTypeID,
		Quantity:      m.Quantity,
		Price:         m.Price,
		CreatedAt:     m.CreatedAt,
	}
}

func SeatReservationItemModelFromDomain(
	item *domain.SeatReservationItem,
) *SeatReservationItemModel {
	return &SeatReservationItemModel{
		ID:            item.ID,
		ReservationID: item.ReservationID,
		SeatID:        item.SeatID,
		TicketTypeID:  item.TicketTypeID,
		Quantity:      item.Quantity,
		Price:         item.Price,
		CreatedAt:     item.CreatedAt,
	}
}
