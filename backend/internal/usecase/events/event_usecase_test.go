package events

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type mockTxManager struct {
	repos domain.TransactionRepositories
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

type mockTxRepos struct {
	userRepo              domain.UserRepository
	pendingRegRepo        domain.PendingRegistrationRepository
	passwordResetRepo     domain.PasswordResetTokenRepository
	organizerAppRepo      domain.OrganizerApplicationRepository
	organizerRepo         domain.OrganizerRepository
	organizerProfileRepo  domain.OrganizerProfileRepository
	organizerAddressRepo  domain.OrganizerAddressRepository
	organizerBankRepo     domain.OrganizerBankAccountRepository
	categoryRepo          domain.CategoryRepository
	venueRepo             domain.VenueRepository
	eventRepo             domain.EventRepository
	eventScheduleRepo     domain.EventScheduleRepository
	eventSettingRepo      domain.EventSettingRepository
	eventCancellationRepo domain.EventCancellationRepository
	eventContactRepo      domain.EventContactRepository
	seatLayoutRepo        domain.SeatLayoutRepository
	seatSectionRepo       domain.SeatSectionRepository
	seatRowRepo           domain.SeatRowRepository
	seatRepo              domain.SeatRepository
	ticketTypeRepo        domain.TicketTypeRepository
}

func (m *mockTxRepos) UserRepository() domain.UserRepository { return m.userRepo }
func (m *mockTxRepos) PendingRegistrationRepository() domain.PendingRegistrationRepository {
	return m.pendingRegRepo
}
func (m *mockTxRepos) PasswordResetTokenRepository() domain.PasswordResetTokenRepository {
	return m.passwordResetRepo
}
func (m *mockTxRepos) OrganizerApplicationRepository() domain.OrganizerApplicationRepository {
	return m.organizerAppRepo
}
func (m *mockTxRepos) OrganizerRepository() domain.OrganizerRepository { return m.organizerRepo }
func (m *mockTxRepos) OrganizerProfileRepository() domain.OrganizerProfileRepository {
	return m.organizerProfileRepo
}
func (m *mockTxRepos) OrganizerAddressRepository() domain.OrganizerAddressRepository {
	return m.organizerAddressRepo
}
func (m *mockTxRepos) OrganizerBankAccountRepository() domain.OrganizerBankAccountRepository {
	return m.organizerBankRepo
}
func (m *mockTxRepos) CategoryRepository() domain.CategoryRepository { return m.categoryRepo }
func (m *mockTxRepos) VenueRepository() domain.VenueRepository       { return m.venueRepo }
func (m *mockTxRepos) EventRepository() domain.EventRepository       { return m.eventRepo }
func (m *mockTxRepos) EventScheduleRepository() domain.EventScheduleRepository {
	return m.eventScheduleRepo
}
func (m *mockTxRepos) EventSettingRepository() domain.EventSettingRepository {
	return m.eventSettingRepo
}
func (m *mockTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return m.eventCancellationRepo
}
func (m *mockTxRepos) EventContactRepository() domain.EventContactRepository {
	return m.eventContactRepo
}
func (m *mockTxRepos) SeatLayoutRepository() domain.SeatLayoutRepository   { return m.seatLayoutRepo }
func (m *mockTxRepos) SeatSectionRepository() domain.SeatSectionRepository { return m.seatSectionRepo }
func (m *mockTxRepos) SeatRowRepository() domain.SeatRowRepository         { return m.seatRowRepo }
func (m *mockTxRepos) SeatRepository() domain.SeatRepository               { return m.seatRepo }
func (m *mockTxRepos) TicketTypeRepository() domain.TicketTypeRepository   { return m.ticketTypeRepo }

type mockOrganizerRepo struct {
	organizer *domain.Organizer
	err       error
}

func (m *mockOrganizerRepo) Create(organizer *domain.Organizer) error { return nil }
func (m *mockOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	return m.organizer, m.err
}
func (m *mockOrganizerRepo) Update(organizer *domain.Organizer) error { return nil }
func (m *mockOrganizerRepo) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	return m.organizer, m.err
}

type mockEventRepo struct {
	event *domain.Event
	err   error
}

func (m *mockEventRepo) Create(event *domain.Event) error             { return nil }
func (m *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) { return m.event, m.err }
func (m *mockEventRepo) Update(event *domain.Event) error {
	m.event = event
	return nil
}
func (m *mockEventRepo) Delete(id uuid.UUID) error { return nil }
func (m *mockEventRepo) FindByOrganizerID(organizerID uuid.UUID, page, limit int, status, search string) ([]*domain.OrganizerEvent, int64, error) {
	return nil, 0, nil
}

type mockEventSettingRepo struct {
	setting *domain.EventSetting
	err     error
}

func (m *mockEventSettingRepo) Create(setting *domain.EventSetting) error { return nil }
func (m *mockEventSettingRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSetting, error) {
	return m.setting, m.err
}
func (m *mockEventSettingRepo) Update(setting *domain.EventSetting) error {
	m.setting = setting
	return nil
}
func (m *mockEventSettingRepo) DeleteByEventID(eventID uuid.UUID) error { return nil }

type mockCategoryRepo struct {
	categories []*domain.Category
}

func (m *mockCategoryRepo) Create(category *domain.Category) error { return nil }
func (m *mockCategoryRepo) FindByID(id uuid.UUID) (*domain.Category, error) {
	for _, c := range m.categories {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}
func (m *mockCategoryRepo) FindActive() ([]*domain.Category, error) { return m.categories, nil }
func (m *mockCategoryRepo) Update(category *domain.Category) error  { return nil }
func (m *mockCategoryRepo) Delete(id uuid.UUID) error               { return nil }

type mockVenueRepo struct {
	venue *domain.Venue
}

func (m *mockVenueRepo) Create(venue *domain.Venue) error { return nil }
func (m *mockVenueRepo) FindByID(id uuid.UUID) (*domain.Venue, error) {
	if m.venue != nil && m.venue.ID == id {
		return m.venue, nil
	}
	return nil, domain.ErrVenueNotFound
}
func (m *mockVenueRepo) FindByGooglePlaceID(placeID string) (*domain.Venue, error) {
	if m.venue != nil && m.venue.GooglePlaceID == placeID {
		return m.venue, nil
	}
	return nil, domain.ErrVenueNotFound
}

type mockEventScheduleRepo struct {
	schedule *domain.EventSchedule
}

func (m *mockEventScheduleRepo) Create(schedule *domain.EventSchedule) error { return nil }
func (m *mockEventScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	return m.schedule, nil
}
func (m *mockEventScheduleRepo) Update(schedule *domain.EventSchedule) error {
	m.schedule = schedule
	return nil
}
func (m *mockEventScheduleRepo) DeleteByEventID(eventID uuid.UUID) error { return nil }

type mockEventCancellationRepo struct {
	cancellation *domain.EventCancellation
}

func (m *mockEventCancellationRepo) Create(c *domain.EventCancellation) error { return nil }
func (m *mockEventCancellationRepo) FindByEventID(eventID uuid.UUID) (*domain.EventCancellation, error) {
	return m.cancellation, nil
}
func (m *mockEventCancellationRepo) Update(c *domain.EventCancellation) error {
	m.cancellation = c
	return nil
}
func (m *mockEventCancellationRepo) DeleteByEventID(eventID uuid.UUID) error { return nil }

type mockEventContactRepo struct {
	contact *domain.EventContact
}

func (m *mockEventContactRepo) Create(c *domain.EventContact) error { return nil }
func (m *mockEventContactRepo) FindByEventID(eventID uuid.UUID) (*domain.EventContact, error) {
	return m.contact, nil
}
func (m *mockEventContactRepo) Update(c *domain.EventContact) error {
	m.contact = c
	return nil
}
func (m *mockEventContactRepo) DeleteByEventID(eventID uuid.UUID) error { return nil }

type mockTicketTypeRepo struct {
	ticketTypes []*domain.TicketType
}

func (m *mockTicketTypeRepo) Create(ctx context.Context, t *domain.TicketType) error {
	m.ticketTypes = append(m.ticketTypes, t)
	return nil
}
func (m *mockTicketTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	for _, t := range m.ticketTypes {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, domain.ErrTicketTypeNotFound
}
func (m *mockTicketTypeRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.TicketType, error) {
	return m.ticketTypes, nil
}
func (m *mockTicketTypeRepo) Update(ctx context.Context, t *domain.TicketType) error { return nil }
func (m *mockTicketTypeRepo) Delete(ctx context.Context, id uuid.UUID) error         { return nil }
func (m *mockTicketTypeRepo) DeleteByEventID(ctx context.Context, eventID uuid.UUID) error {
	return nil
}

func setupTestUpdateEnvironment(existingSeatLayoutType string) (*EventUsecase, UpdateEventInput, *mockEventSettingRepo) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	eventDate := time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(24 * time.Hour)
	salesStart := eventDate.Add(-5 * 24 * time.Hour)
	salesEnd := eventDate.Add(-2 * 24 * time.Hour)

	mockOrg := &mockOrganizerRepo{
		organizer: &domain.Organizer{
			ID:     organizerID,
			UserID: userID,
			Status: "ACTIVE",
		},
	}

	mockEvt := &mockEventRepo{
		event: &domain.Event{
			ID:          eventID,
			OrganizerID: organizerID,
			CategoryID:  categoryID,
			VenueID:     &venueID,
			EventType:   domain.EventTypePhysical,
			Title:       "Sample Event",
			Description: "Description",
			Status:      domain.EventStatusDraft,
		},
	}

	mockSetting := &mockEventSettingRepo{
		setting: &domain.EventSetting{
			ID:                  uuid.New(),
			EventID:             eventID,
			SeatLayoutType:      existingSeatLayoutType,
			BookingLimitPerUser: 5,
			SalesStartDate:      &salesStart,
			SalesEndDate:        &salesEnd,
		},
	}

	mockCat := &mockCategoryRepo{
		categories: []*domain.Category{
			{ID: categoryID, Name: "Music", Status: domain.CategoryStatusActive},
		},
	}

	mockVen := &mockVenueRepo{
		venue: &domain.Venue{
			ID:            venueID,
			GooglePlaceID: "place_123",
			Name:          "Venue Center",
			Address:       "123 Main St",
			City:          "Metro",
			Country:       "Country",
		},
	}

	mockSched := &mockEventScheduleRepo{
		schedule: &domain.EventSchedule{
			ID:        uuid.New(),
			EventID:   eventID,
			EventDate: eventDate,
			StartTime: eventDate.Add(10 * time.Hour),
			EndTime:   eventDate.Add(12 * time.Hour),
		},
	}

	mockCancel := &mockEventCancellationRepo{
		cancellation: &domain.EventCancellation{
			ID:                  uuid.New(),
			EventID:             eventID,
			CancellationAllowed: false,
		},
	}

	mockContact := &mockEventContactRepo{
		contact: &domain.EventContact{
			ID:      uuid.New(),
			EventID: eventID,
			Name:    "Test Contact",
			Phone:   "+1234567890",
			Email:   "test@contact.com",
		},
	}

	mockTicket := &mockTicketTypeRepo{
		ticketTypes: []*domain.TicketType{},
	}

	repos := &mockTxRepos{
		organizerRepo:         mockOrg,
		eventRepo:             mockEvt,
		eventSettingRepo:      mockSetting,
		categoryRepo:          mockCat,
		venueRepo:             mockVen,
		eventScheduleRepo:     mockSched,
		eventCancellationRepo: mockCancel,
		eventContactRepo:      mockContact,
		ticketTypeRepo:        mockTicket,
	}

	txManager := &mockTxManager{repos: repos}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	usecase := NewEventUsecase(txManager, nil, logger)

	input := UpdateEventInput{
		UserID:              userID,
		EventID:             eventID,
		CategoryID:          categoryID,
		VenueID:             &venueID,
		EventType:           domain.EventTypePhysical,
		Title:               "Updated Event Title",
		Description:         "Updated Description",
		EventDate:           eventDate,
		StartTime:           eventDate.Add(10 * time.Hour),
		EndTime:             eventDate.Add(12 * time.Hour),
		SeatLayoutType:      existingSeatLayoutType,
		BookingLimitPerUser: 5,
		SalesStartDate:      &salesStart,
		SalesEndDate:        &salesEnd,
		Contact: EventContactInput{
			Name:  "Test Contact",
			Phone: "+1234567890",
			Email: "test@contact.com",
		},
		TicketTypes: []TicketTypeInput{
			{Name: "General Admission", Price: 100, Capacity: 50},
		},
	}

	return usecase, input, mockSetting
}

func TestUpdateEvent_RejectModeChange_GeneralToSeated(t *testing.T) {
	usecase, input, _ := setupTestUpdateEnvironment(SeatLayoutTypeGeneral)
	input.SeatLayoutType = SeatLayoutTypeSeated

	output, err := usecase.UpdateEvent(context.Background(), input)

	if !errors.Is(err, ErrTicketingModeImmutable) {
		t.Fatalf("expected ErrTicketingModeImmutable, got: %v", err)
	}
	if output != nil {
		t.Fatalf("expected nil output on error, got: %v", output)
	}
}

func TestUpdateEvent_RejectModeChange_SeatedToGeneral(t *testing.T) {
	usecase, input, _ := setupTestUpdateEnvironment(SeatLayoutTypeSeated)
	input.SeatLayoutType = SeatLayoutTypeGeneral

	output, err := usecase.UpdateEvent(context.Background(), input)

	if !errors.Is(err, ErrTicketingModeImmutable) {
		t.Fatalf("expected ErrTicketingModeImmutable, got: %v", err)
	}
	if output != nil {
		t.Fatalf("expected nil output on error, got: %v", output)
	}
}

func TestUpdateEvent_AllowSameMode_GeneralToGeneral(t *testing.T) {
	usecase, input, mockSetting := setupTestUpdateEnvironment(SeatLayoutTypeGeneral)
	input.SeatLayoutType = SeatLayoutTypeGeneral
	input.TicketTypes = []TicketTypeInput{
		{Name: "General Admission", Price: 100, Capacity: 50},
	}

	output, err := usecase.UpdateEvent(context.Background(), input)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if mockSetting.setting.SeatLayoutType != SeatLayoutTypeGeneral {
		t.Fatalf("expected SeatLayoutType GENERAL, got: %s", mockSetting.setting.SeatLayoutType)
	}
}

func TestUpdateEvent_AllowSameMode_SeatedToSeated(t *testing.T) {
	usecase, input, mockSetting := setupTestUpdateEnvironment(SeatLayoutTypeSeated)
	input.SeatLayoutType = SeatLayoutTypeSeated

	output, err := usecase.UpdateEvent(context.Background(), input)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if mockSetting.setting.SeatLayoutType != SeatLayoutTypeSeated {
		t.Fatalf("expected SeatLayoutType SEATED, got: %s", mockSetting.setting.SeatLayoutType)
	}
}

func TestUpdateEvent_OmittedMode_InheritsExistingMode_Seated(t *testing.T) {
	usecase, input, mockSetting := setupTestUpdateEnvironment(SeatLayoutTypeSeated)
	input.SeatLayoutType = "" // omitted / empty in request

	output, err := usecase.UpdateEvent(context.Background(), input)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if mockSetting.setting.SeatLayoutType != SeatLayoutTypeSeated {
		t.Fatalf("expected SeatLayoutType to remain SEATED, got: %s", mockSetting.setting.SeatLayoutType)
	}
}

func TestUpdateEvent_OmittedMode_InheritsExistingMode_General(t *testing.T) {
	usecase, input, mockSetting := setupTestUpdateEnvironment(SeatLayoutTypeGeneral)
	input.SeatLayoutType = "" // omitted / empty in request

	output, err := usecase.UpdateEvent(context.Background(), input)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if mockSetting.setting.SeatLayoutType != SeatLayoutTypeGeneral {
		t.Fatalf("expected SeatLayoutType to remain GENERAL, got: %s", mockSetting.setting.SeatLayoutType)
	}
}

func TestUpdateEvent_NormalizeModeInput(t *testing.T) {
	usecase, input, mockSetting := setupTestUpdateEnvironment(SeatLayoutTypeGeneral)
	input.SeatLayoutType = "  general  " // whitespace and lowercase

	output, err := usecase.UpdateEvent(context.Background(), input)

	if err != nil {
		t.Fatalf("expected nil error with normalized mode, got: %v", err)
	}
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if mockSetting.setting.SeatLayoutType != SeatLayoutTypeGeneral {
		t.Fatalf("expected SeatLayoutType GENERAL, got: %s", mockSetting.setting.SeatLayoutType)
	}
}
