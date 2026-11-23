package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrEventNotFound          = errors.New("event not found")
	ErrEventNotPublic         = errors.New("event is not publicly available")
	ErrEventAlreadyCancelled  = errors.New("event is already cancelled")
	ErrEventCannotBeCancelled = errors.New("event cannot be cancelled")

	// Specific cancellation lifecycle errors
	ErrDraftEventCannotBeCancelled = fmt.Errorf("%w: draft events cannot be cancelled; delete the draft instead", ErrEventCannotBeCancelled)
	ErrPastEventCannotBeCancelled  = fmt.Errorf("%w: event has already concluded", ErrEventCannotBeCancelled)
	ErrCancellationReasonRequired  = errors.New("cancellation reason is required")
	ErrCancellationReasonTooLong   = errors.New("cancellation reason exceeds maximum length of 1000 characters")

	// Specific cancellation settlement errors
	ErrEventCannotBeCancelledWithCompletedSettlement = fmt.Errorf("%w: event has completed settlements with disbursed organizer funds", ErrEventCannotBeCancelled)
	ErrEventCannotBeCancelledWithActiveSettlement    = fmt.Errorf("%w: event has approved or in-flight settlement payouts currently processing", ErrEventCannotBeCancelled)
)

const (
	EventStatusDraft     = "DRAFT"
	EventStatusPublished = "PUBLISHED"
	EventStatusCancelled = "CANCELLED"
)

const (
	EventTypePhysical = "PHYSICAL"
	EventTypeOnline   = "ONLINE"
	EventTypeHybrid   = "HYBRID"
)

const (
	RefundPolicyFull     = "FULL"
	RefundPolicyPartial  = "PARTIAL"
	RefundPolicyNoRefund = "NO_REFUND"
)

type Event struct {
	ID                  uuid.UUID
	OrganizerID         uuid.UUID
	CategoryID          uuid.UUID
	VenueID             *uuid.UUID
	EventType           string
	OnlineURL           string
	Title               string
	Description         string
	BannerURL           string
	Language            string
	AgeRestriction      int
	Visibility          string
	Highlights          string
	Rules               string
	AttendeeInformation string
	Status              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type EventSchedule struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	EventDate time.Time
	StartTime time.Time
	IsAllDay  bool
	EndTime   time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EventSetting struct {
	ID                  uuid.UUID
	EventID             uuid.UUID
	SeatLayoutType      string
	BookingLimitPerUser int
	SalesStartDate      *time.Time
	SalesEndDate        *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type EventCancellation struct {
	ID                        uuid.UUID
	EventID                   uuid.UUID
	CancellationAllowed       bool
	CancellationDeadlineHours int
	RefundPolicy              string
	RefundPercentage          int
	CancelledAt               *time.Time
	CancellationReason        *string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type EventCancellationSummary struct {
	EventID                   uuid.UUID       `json:"event_id"`
	EventTitle                string          `json:"event_title"`
	EventStatus               string          `json:"event_status"`
	CancelledAt               *time.Time      `json:"cancelled_at,omitempty"`
	CancellationReason        string          `json:"cancellation_reason,omitempty"`
	TotalBookingsAffected     int             `json:"total_bookings_affected"`
	TotalTicketsAffected      int             `json:"total_tickets_affected"`
	TotalEligibleRefundAmount decimal.Decimal `json:"total_eligible_refund_amount"`
	TotalRefundedAmount       decimal.Decimal `json:"total_refunded_amount"`
	PendingRefundAmount       decimal.Decimal `json:"pending_refund_amount"`
	PendingRefundCount        int             `json:"pending_refund_count"`
	FailedRefundAmount        decimal.Decimal `json:"failed_refund_amount"`
	FailedRefundCount         int             `json:"failed_refund_count"`
	SuccessfulRefundCount     int             `json:"successful_refund_count"`
	CompletionStatus          string          `json:"completion_status"`
}

type EventRepository interface {
	Create(event *Event) error
	FindByID(id uuid.UUID) (*Event, error)
	FindByIDForUpdate(id uuid.UUID) (*Event, error)
	Update(event *Event) error
	Delete(id uuid.UUID) error

	FindByOrganizerID(
		organizerID uuid.UUID,
		page int,
		limit int,
		status string,
		search string,
	) ([]*OrganizerEvent, int64, error)

	FindPublicEvents(
		filter PublicEventFilter,
	) ([]*PublicEvent, int64, error)

	FindAllEvents(
		filter AdminEventFilter,
	) ([]*AdminEventListItem, int64, error)

	GetAdminStatusCounts() (AdminEventStatusCounts, error)
}

type EventScheduleRepository interface {
	Create(schedule *EventSchedule) error
	FindByEventID(eventID uuid.UUID) (*EventSchedule, error)
	Update(schedule *EventSchedule) error
	DeleteByEventID(eventID uuid.UUID) error
}

type EventSettingRepository interface {
	Create(setting *EventSetting) error
	FindByEventID(eventID uuid.UUID) (*EventSetting, error)
	Update(setting *EventSetting) error
	DeleteByEventID(eventID uuid.UUID) error
}

type EventCancellationRepository interface {
	Create(cancellation *EventCancellation) error
	FindByEventID(eventID uuid.UUID) (*EventCancellation, error)
	Update(cancellation *EventCancellation) error
	DeleteByEventID(eventID uuid.UUID) error
}
