package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerCancellationNotice contains the information required to notify a customer
// whose bookings were affected by an event cancellation.
type CustomerCancellationNotice struct {
	CustomerID         uuid.UUID
	CustomerName       string
	CustomerEmail      string
	EventID            uuid.UUID
	EventTitle         string
	EventDate          string
	CancellationReason string
	BookingReferences  []string
	TicketsCancelled   int
	RefundAmount       decimal.Decimal
	RefundStatus       string
	RefundInformation  string
}

// OrganizerCancellationNotice contains the summary information required to notify an organizer
// when an event is cancelled by themselves or by a platform administrator.
type OrganizerCancellationNotice struct {
	OrganizerID           uuid.UUID
	OrganizerName         string
	OrganizerEmail        string
	EventID               uuid.UUID
	EventTitle            string
	EventDate             string
	CancellationReason    string
	CancelledBy           string // e.g. "Organizer" or "Platform Administrator"
	ActorRole             string // "organizer" or "admin"
	TotalBookingsAffected int
	TotalTicketsAffected  int
	TotalRefundAmount     decimal.Decimal
	SettlementImpact      string
	CancelledAt           time.Time
}

// EventCancellationNotifier defines the delivery contract for event cancellation notifications.
type EventCancellationNotifier interface {
	NotifyCustomer(ctx context.Context, notice CustomerCancellationNotice) error
	NotifyOrganizer(ctx context.Context, notice OrganizerCancellationNotice) error
}
