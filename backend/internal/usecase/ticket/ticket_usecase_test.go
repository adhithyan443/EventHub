package ticket_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Repositories ---

type mockTicketRepo struct {
	tickets map[uuid.UUID]*domain.Ticket
	err     error
}

func (m *mockTicketRepo) Create(ctx context.Context, t *domain.Ticket) error       { return m.err }
func (m *mockTicketRepo) CreateMany(ctx context.Context, ts []domain.Ticket) error { return m.err }
func (m *mockTicketRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	if m.err != nil {
		return nil, m.err
	}
	t, ok := m.tickets[id]
	if !ok {
		return nil, domain.ErrTicketNotFound
	}
	return t, nil
}
func (m *mockTicketRepo) FindByCode(ctx context.Context, code string) (*domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Ticket, error) {
	if m.err != nil {
		return nil, m.err
	}
	var res []domain.Ticket
	for _, t := range m.tickets {
		if t.UserID == userID {
			res = append(res, *t)
		}
	}
	return res, nil
}
func (m *mockTicketRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.err
}
func (m *mockTicketRepo) UpdateStatusByBookingID(ctx context.Context, bookingID uuid.UUID, status string) error {
	return m.err
}
func (m *mockTicketRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, m.err
}

type mockEventRepo struct {
	events map[uuid.UUID]*domain.Event
	err    error
}

func (m *mockEventRepo) Create(event *domain.Event) error { return m.err }
func (m *mockEventRepo) Update(event *domain.Event) error { return m.err }
func (m *mockEventRepo) Delete(id uuid.UUID) error        { return m.err }
func (m *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	ev, ok := m.events[id]
	if !ok {
		return nil, domain.ErrEventNotFound
	}
	return ev, nil
}
func (m *mockEventRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) {
	return m.FindByID(id)
}
func (m *mockEventRepo) FindByOrganizerID(organizerID uuid.UUID, page int, limit int, status string, search string) ([]*domain.OrganizerEvent, int64, error) {
	return nil, 0, nil
}
func (m *mockEventRepo) FindPublicEvents(filter domain.PublicEventFilter) ([]*domain.PublicEvent, int64, error) {
	return nil, 0, nil
}
func (m *mockEventRepo) FindAllEvents(filter domain.AdminEventFilter) ([]*domain.AdminEventListItem, int64, error) {
	return nil, 0, nil
}
func (m *mockEventRepo) GetAdminStatusCounts() (domain.AdminEventStatusCounts, error) {
	return domain.AdminEventStatusCounts{}, nil
}

type mockScheduleRepo struct {
	schedules map[uuid.UUID]*domain.EventSchedule
	err       error
}

func (m *mockScheduleRepo) Create(schedule *domain.EventSchedule) error { return m.err }
func (m *mockScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	if m.err != nil {
		return nil, m.err
	}
	sch, ok := m.schedules[eventID]
	if !ok {
		return nil, domain.ErrEventNotFound
	}
	return sch, nil
}
func (m *mockScheduleRepo) Update(schedule *domain.EventSchedule) error { return m.err }
func (m *mockScheduleRepo) DeleteByEventID(eventID uuid.UUID) error     { return m.err }

type mockUserRepo struct {
	domain.UserRepository
	users map[uuid.UUID]*domain.User
	err   error
}

func (m *mockUserRepo) Create(user *domain.User) error { return m.err }
func (m *mockUserRepo) FindByID(id uuid.UUID) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}
func (m *mockUserRepo) FindByEmail(email string) (*domain.User, error) { return nil, nil }
func (m *mockUserRepo) FindByPhone(phone string) (*domain.User, error) { return nil, nil }
func (m *mockUserRepo) Update(user *domain.User) error                 { return m.err }

type mockVenueRepo struct {
	venues      map[uuid.UUID]*domain.Venue
	err         error
	calledCount int
}

func (m *mockVenueRepo) Create(venue *domain.Venue) error { return m.err }
func (m *mockVenueRepo) FindByID(id uuid.UUID) (*domain.Venue, error) {
	m.calledCount++
	if m.err != nil {
		return nil, m.err
	}
	v, ok := m.venues[id]
	if !ok {
		return nil, domain.ErrVenueNotFound
	}
	return v, nil
}
func (m *mockVenueRepo) FindByGooglePlaceID(placeID string) (*domain.Venue, error) { return nil, nil }

type mockTicketTypeRepo struct {
	types map[uuid.UUID]*domain.TicketType
	err   error
}

func (m *mockTicketTypeRepo) Create(ctx context.Context, ticketType *domain.TicketType) error {
	return m.err
}
func (m *mockTicketTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	if m.err != nil {
		return nil, m.err
	}
	tt, ok := m.types[id]
	if !ok {
		return nil, domain.ErrTicketTypeNotFound
	}
	return tt, nil
}
func (m *mockTicketTypeRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.TicketType, error) {
	return nil, nil
}
func (m *mockTicketTypeRepo) Update(ctx context.Context, ticketType *domain.TicketType) error {
	return m.err
}
func (m *mockTicketTypeRepo) Delete(ctx context.Context, id uuid.UUID) error { return m.err }
func (m *mockTicketTypeRepo) DeleteByEventID(ctx context.Context, eventID uuid.UUID) error {
	return m.err
}
func (m *mockTicketTypeRepo) ReserveQuantityIfAvailable(ctx context.Context, ticketTypeID uuid.UUID, quantity int) error {
	return m.err
}
func (m *mockTicketTypeRepo) ReleaseReservedQuantity(ctx context.Context, ticketTypeID uuid.UUID, quantity int) error {
	return m.err
}

// --- Test Fixture ---

type testFixture struct {
	ticketRepo   *mockTicketRepo
	eventRepo    *mockEventRepo
	scheduleRepo *mockScheduleRepo
	userRepo     *mockUserRepo
	venueRepo    *mockVenueRepo
	typeRepo     *mockTicketTypeRepo
	usecase      *ticket.TicketUsecase
	userID       uuid.UUID
	ticketID     uuid.UUID
	eventID      uuid.UUID
	venueID      uuid.UUID
	ticketTypeID uuid.UUID
}

func setupFixture() *testFixture {
	userID := uuid.New()
	ticketID := uuid.New()
	eventID := uuid.New()
	venueID := uuid.New()
	ticketTypeID := uuid.New()

	ticketModel := &domain.Ticket{
		ID:              ticketID,
		TicketCode:      "EH-TCK-001",
		BookingID:       uuid.New(),
		UserID:          userID,
		EventID:         eventID,
		TicketTypeID:    &ticketTypeID,
		AdmissionNumber: 42,
		Price:           decimal.NewFromInt(150),
		Status:          domain.TicketStatusValid,
		IssuedAt:        time.Now(),
	}

	eventModel := &domain.Event{
		ID:        eventID,
		Title:     "Summer Music Festival",
		EventType: domain.EventTypePhysical,
		VenueID:   &venueID,
	}

	eventDate := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	startTime := time.Date(0, 1, 1, 18, 30, 0, 0, time.UTC)
	scheduleModel := &domain.EventSchedule{
		ID:        uuid.New(),
		EventID:   eventID,
		EventDate: eventDate,
		StartTime: startTime,
	}

	userModel := &domain.User{
		ID:       userID,
		FullName: "Jane Doe",
		Email:    "jane@example.com",
	}

	venueModel := &domain.Venue{
		ID:         venueID,
		Name:       "Grand Arena",
		Address:    "123 Stadium Way",
		City:       "Metropolis",
		State:      "NY",
		PostalCode: "10001",
	}

	ticketTypeModel := &domain.TicketType{
		ID:      ticketTypeID,
		EventID: eventID,
		Name:    "VIP Pass",
		Price:   decimal.NewFromInt(150),
	}

	ticketRepo := &mockTicketRepo{tickets: map[uuid.UUID]*domain.Ticket{ticketID: ticketModel}}
	eventRepo := &mockEventRepo{events: map[uuid.UUID]*domain.Event{eventID: eventModel}}
	scheduleRepo := &mockScheduleRepo{schedules: map[uuid.UUID]*domain.EventSchedule{eventID: scheduleModel}}
	userRepo := &mockUserRepo{users: map[uuid.UUID]*domain.User{userID: userModel}}
	venueRepo := &mockVenueRepo{venues: map[uuid.UUID]*domain.Venue{venueID: venueModel}}
	typeRepo := &mockTicketTypeRepo{types: map[uuid.UUID]*domain.TicketType{ticketTypeID: ticketTypeModel}}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := ticket.NewTicketUsecase(ticketRepo, eventRepo, scheduleRepo, userRepo, venueRepo, typeRepo, logger)

	return &testFixture{
		ticketRepo:   ticketRepo,
		eventRepo:    eventRepo,
		scheduleRepo: scheduleRepo,
		userRepo:     userRepo,
		venueRepo:    venueRepo,
		typeRepo:     typeRepo,
		usecase:      uc,
		userID:       userID,
		ticketID:     ticketID,
		eventID:      eventID,
		venueID:      venueID,
		ticketTypeID: ticketTypeID,
	}
}

// --- Test Cases ---

func TestGetTicketPDFData_Success_PhysicalEventWithVenue(t *testing.T) {
	f := setupFixture()
	ctx := context.Background()

	data, err := f.usecase.GetTicketPDFData(ctx, f.userID, f.ticketID)
	require.NoError(t, err)
	require.NotNil(t, data)

	assert.Equal(t, f.ticketID, data.TicketID)
	assert.Equal(t, "Summer Music Festival", data.EventName)
	assert.Equal(t, "Jane Doe", data.CustomerName)
	assert.Equal(t, "42", data.AdmissionNo)
	assert.Equal(t, "VIP Pass", data.TicketType)
	assert.Equal(t, "Grand Arena", data.VenueName)
	assert.Equal(t, "123 Stadium Way, Metropolis, NY, 10001", data.VenueAddress)

	// Verify combined event date + start time
	expectedDate := time.Date(2026, 8, 15, 18, 30, 0, 0, time.UTC)
	assert.Equal(t, expectedDate, data.EventDate)
	assert.Equal(t, 1, f.venueRepo.calledCount)
}

func TestGetTicketPDFData_Success_OnlineEvent_BypassesVenueLookup(t *testing.T) {
	f := setupFixture()
	ctx := context.Background()

	// Switch event to ONLINE
	f.eventRepo.events[f.eventID].EventType = domain.EventTypeOnline

	data, err := f.usecase.GetTicketPDFData(ctx, f.userID, f.ticketID)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Online events must not require physical venue and must not query venue repository
	assert.Equal(t, "", data.VenueName)
	assert.Equal(t, "", data.VenueAddress)
	assert.Equal(t, 0, f.venueRepo.calledCount)
}

func TestGetTicketPDFData_TicketNotFound(t *testing.T) {
	f := setupFixture()
	ctx := context.Background()

	_, err := f.usecase.GetTicketPDFData(ctx, f.userID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrTicketNotFound)
}

func TestGetTicketPDFData_OwnershipMismatch(t *testing.T) {
	f := setupFixture()
	ctx := context.Background()

	otherUser := uuid.New()
	_, err := f.usecase.GetTicketPDFData(ctx, otherUser, f.ticketID)
	assert.ErrorIs(t, err, domain.ErrTicketNotFound)
}

func TestGetTicketPDFData_EventNotFound(t *testing.T) {
	f := setupFixture()
	ctx := context.Background()

	// Remove event
	delete(f.eventRepo.events, f.eventID)

	_, err := f.usecase.GetTicketPDFData(ctx, f.userID, f.ticketID)
	assert.ErrorIs(t, err, domain.ErrEventNotFound)
}

func TestGetTicketPDFData_MissingOptionalData_GracefulFallback(t *testing.T) {
	f := setupFixture()
	ctx := context.Background()

	// 1. Remove schedule
	delete(f.scheduleRepo.schedules, f.eventID)
	// 2. Remove venue
	delete(f.venueRepo.venues, f.venueID)
	// 3. Remove ticket type
	delete(f.typeRepo.types, f.ticketTypeID)
	// 4. Remove user
	delete(f.userRepo.users, f.userID)
	// 5. Zero out admission number
	f.ticketRepo.tickets[f.ticketID].AdmissionNumber = 0

	data, err := f.usecase.GetTicketPDFData(ctx, f.userID, f.ticketID)
	require.NoError(t, err)
	require.NotNil(t, data)

	assert.Equal(t, "EH-TCK-001", data.AdmissionNo, "Should fallback to TicketCode when AdmissionNumber is 0")
	assert.Equal(t, "Valued Customer", data.CustomerName, "Should fallback to Valued Customer when user record missing")
	assert.Equal(t, "General Admission", data.TicketType, "Should fallback to General Admission when ticket type missing")
	assert.Equal(t, "", data.VenueName, "Should be empty when venue missing")
	assert.Equal(t, "", data.VenueAddress, "Should be empty when venue missing")
	assert.True(t, data.EventDate.IsZero(), "EventDate should be zero when schedule missing")
}

func TestGetTicketPDFData_UnexpectedDatabaseError_Propagated(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database connection failed")

	t.Run("event repo db error", func(t *testing.T) {
		f := setupFixture()
		f.eventRepo.err = dbErr
		_, err := f.usecase.GetTicketPDFData(ctx, f.userID, f.ticketID)
		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("user repo db error", func(t *testing.T) {
		f := setupFixture()
		f.userRepo.err = dbErr
		_, err := f.usecase.GetTicketPDFData(ctx, f.userID, f.ticketID)
		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("schedule repo db error", func(t *testing.T) {
		f := setupFixture()
		f.scheduleRepo.err = dbErr
		_, err := f.usecase.GetTicketPDFData(ctx, f.userID, f.ticketID)
		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("venue repo db error", func(t *testing.T) {
		f := setupFixture()
		f.venueRepo.err = dbErr
		_, err := f.usecase.GetTicketPDFData(ctx, f.userID, f.ticketID)
		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("ticket type repo db error", func(t *testing.T) {
		f := setupFixture()
		f.typeRepo.err = dbErr
		_, err := f.usecase.GetTicketPDFData(ctx, f.userID, f.ticketID)
		assert.ErrorIs(t, err, dbErr)
	})
}
