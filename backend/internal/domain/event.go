package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEventNotFound  = errors.New("event not found")
	ErrEventNotPublic = errors.New("event is not publicly available")
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

type EventRepository interface {
	Create(event *Event) error
	FindByID(id uuid.UUID) (*Event, error)
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
