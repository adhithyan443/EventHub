package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrEventNotFound = errors.New("event not found")

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
	ID             uuid.UUID
	OrganizerID    uuid.UUID
	CategoryID     uuid.UUID
	VenueID        *uuid.UUID
	EventType      string
	OnlineURL      string
	Title          string
	Description    string
	BannerURL      string
	Language       string
	AgeRestriction int
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type EventSchedule struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	EventDate time.Time
	StartTime time.Time
	EndTime   time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EventSetting struct {
	ID                  uuid.UUID
	EventID             uuid.UUID
	SeatLayoutType      string
	BookingLimitPerUser int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type EventCancellation struct {
	ID                        uuid.UUID
	EventID                   uuid.UUID
	CancellationAllowed       bool
	CancellationDeadlineHours int
	CancelledAt               *time.Time
	CancellationReason        *string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type EventRepository interface {
	Create(event *Event) error
	FindByID(id uuid.UUID) (*Event, error)
}

type EventScheduleRepository interface {
	Create(schedule *EventSchedule) error
	FindByEventID(eventID uuid.UUID) (*EventSchedule, error)
}

type EventSettingRepository interface {
	Create(setting *EventSetting) error
	FindByEventID(eventID uuid.UUID) (*EventSetting, error)
}

type EventCancellationRepository interface {
	Create(cancellation *EventCancellation) error
	FindByEventID(eventID uuid.UUID) (*EventCancellation, error)
}
