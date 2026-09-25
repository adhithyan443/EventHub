package events_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	usecase "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/google/uuid"
)

// --- Mock Transaction Manager & Repositories ---

type mockTxManager struct {
	txRepos    domain.TransactionRepositories
	onRollback func()
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	err := fn(m.txRepos)
	if err != nil && m.onRollback != nil {
		m.onRollback()
	}
	return err
}

type mockTxRepos struct {
	userRepo                domain.UserRepository
	pendingRegistrationRepo domain.PendingRegistrationRepository
	passwordResetTokenRepo  domain.PasswordResetTokenRepository

	organizerAppRepo  domain.OrganizerApplicationRepository
	organizerRepo     domain.OrganizerRepository
	organizerProfRepo domain.OrganizerProfileRepository
	organizerAddrRepo domain.OrganizerAddressRepository
	organizerBankRepo domain.OrganizerBankAccountRepository

	categoryRepo domain.CategoryRepository
	venueRepo    domain.VenueRepository

	eventRepo             domain.EventRepository
	eventScheduleRepo     domain.EventScheduleRepository
	eventSettingRepo      domain.EventSettingRepository
	eventCancellationRepo domain.EventCancellationRepository
	eventContactRepo      domain.EventContactRepository

	seatLayoutRepo  domain.SeatLayoutRepository
	seatSectionRepo domain.SeatSectionRepository
	seatRowRepo     domain.SeatRowRepository
	seatRepo        domain.SeatRepository

	ticketTypeRepo domain.TicketTypeRepository
}

func (r *mockTxRepos) UserRepository() domain.UserRepository { return r.userRepo }
func (r *mockTxRepos) PendingRegistrationRepository() domain.PendingRegistrationRepository {
	return r.pendingRegistrationRepo
}
func (r *mockTxRepos) PasswordResetTokenRepository() domain.PasswordResetTokenRepository {
	return r.passwordResetTokenRepo
}
func (r *mockTxRepos) OrganizerApplicationRepository() domain.OrganizerApplicationRepository {
	return r.organizerAppRepo
}
func (r *mockTxRepos) OrganizerRepository() domain.OrganizerRepository { return r.organizerRepo }
func (r *mockTxRepos) OrganizerProfileRepository() domain.OrganizerProfileRepository {
	return r.organizerProfRepo
}
func (r *mockTxRepos) OrganizerAddressRepository() domain.OrganizerAddressRepository {
	return r.organizerAddrRepo
}
func (r *mockTxRepos) OrganizerBankAccountRepository() domain.OrganizerBankAccountRepository {
	return r.organizerBankRepo
}
func (r *mockTxRepos) CategoryRepository() domain.CategoryRepository { return r.categoryRepo }
func (r *mockTxRepos) VenueRepository() domain.VenueRepository       { return r.venueRepo }
func (r *mockTxRepos) EventRepository() domain.EventRepository       { return r.eventRepo }
func (r *mockTxRepos) EventScheduleRepository() domain.EventScheduleRepository {
	return r.eventScheduleRepo
}
func (r *mockTxRepos) EventSettingRepository() domain.EventSettingRepository {
	return r.eventSettingRepo
}
func (r *mockTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return r.eventCancellationRepo
}
func (r *mockTxRepos) EventContactRepository() domain.EventContactRepository {
	return r.eventContactRepo
}
func (r *mockTxRepos) SeatLayoutRepository() domain.SeatLayoutRepository   { return r.seatLayoutRepo }
func (r *mockTxRepos) SeatSectionRepository() domain.SeatSectionRepository { return r.seatSectionRepo }
func (r *mockTxRepos) SeatRowRepository() domain.SeatRowRepository         { return r.seatRowRepo }
func (r *mockTxRepos) SeatRepository() domain.SeatRepository               { return r.seatRepo }
func (r *mockTxRepos) TicketTypeRepository() domain.TicketTypeRepository {
	if r.ticketTypeRepo == nil {
		return &mockTicketTypeRepo{}
	}
	return r.ticketTypeRepo
}

// --- Specific Mocks ---

type mockOrganizerRepo struct {
	organizer *domain.Organizer
	err       error
}

func (m *mockOrganizerRepo) Create(o *domain.Organizer) error { return nil }
func (m *mockOrganizerRepo) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.organizer, nil
}
func (m *mockOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.organizer, nil
}

type mockCategoryRepo struct {
	categories []*domain.Category
	err        error
}

func (m *mockCategoryRepo) FindActive() ([]*domain.Category, error) {
	return m.categories, m.err
}

type mockVenueRepo struct {
	venue *domain.Venue
	err   error
}

func (m *mockVenueRepo) Create(v *domain.Venue) error { m.venue = v; return nil }
func (m *mockVenueRepo) FindByID(id uuid.UUID) (*domain.Venue, error) {
	if m.venue != nil && m.venue.ID == id {
		return m.venue, nil
	}
	if m.err != nil {
		return nil, m.err
	}
	return nil, domain.ErrVenueNotFound
}
func (m *mockVenueRepo) FindByGooglePlaceID(placeID string) (*domain.Venue, error) {
	if m.venue != nil && m.venue.GooglePlaceID == placeID {
		return m.venue, nil
	}
	return nil, domain.ErrVenueNotFound
}

type mockEventRepo struct {
	event     *domain.Event
	err       error
	updateErr error
	updated   bool
}

func (m *mockEventRepo) Create(e *domain.Event) error { return nil }
func (m *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.event, nil
}
func (m *mockEventRepo) Update(e *domain.Event) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.event = e
	m.updated = true
	return nil
}

type mockEventScheduleRepo struct {
	schedule  *domain.EventSchedule
	err       error
	updateErr error
	updated   bool
	created   bool
}

func (m *mockEventScheduleRepo) Create(s *domain.EventSchedule) error {
	m.schedule = s
	m.created = true
	return nil
}
func (m *mockEventScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.schedule == nil {
		return nil, domain.ErrEventNotFound
	}
	return m.schedule, nil
}
func (m *mockEventScheduleRepo) Update(s *domain.EventSchedule) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.schedule = s
	m.updated = true
	return nil
}

type mockEventSettingRepo struct {
	setting   *domain.EventSetting
	err       error
	updateErr error
	updated   bool
	created   bool
}

func (m *mockEventSettingRepo) Create(s *domain.EventSetting) error {
	m.setting = s
	m.created = true
	return nil
}
func (m *mockEventSettingRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSetting, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.setting == nil {
		return nil, domain.ErrEventNotFound
	}
	return m.setting, nil
}
func (m *mockEventSettingRepo) Update(s *domain.EventSetting) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.setting = s
	m.updated = true
	return nil
}

type mockEventCancellationRepo struct {
	cancellation *domain.EventCancellation
	err          error
	updateErr    error
	updated      bool
	created      bool
}

func (m *mockEventCancellationRepo) Create(c *domain.EventCancellation) error {
	m.cancellation = c
	m.created = true
	return nil
}
func (m *mockEventCancellationRepo) FindByEventID(eventID uuid.UUID) (*domain.EventCancellation, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.cancellation == nil {
		return nil, domain.ErrEventNotFound
	}
	return m.cancellation, nil
}
func (m *mockEventCancellationRepo) Update(c *domain.EventCancellation) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.cancellation = c
	m.updated = true
	return nil
}

type mockEventContactRepo struct {
	contact   *domain.EventContact
	err       error
	updateErr error
	updated   bool
	created   bool
	deleted   bool
}

func (m *mockEventContactRepo) Create(c *domain.EventContact) error {
	m.contact = c
	m.created = true
	return nil
}
func (m *mockEventContactRepo) FindByEventID(eventID uuid.UUID) (*domain.EventContact, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.contact == nil {
		return nil, domain.ErrEventNotFound
	}
	return m.contact, nil
}
func (m *mockEventContactRepo) Update(c *domain.EventContact) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.contact = c
	m.updated = true
	return nil
}
func (m *mockEventContactRepo) DeleteByEventID(eventID uuid.UUID) error {
	m.deleted = true
	m.contact = nil
	return nil
}

type mockTicketTypeRepo struct {
	ticketTypes      []*domain.TicketType
	findByEventIDErr error
	createErr        error
	updateErr        error
	deleteErr        error
	created          []*domain.TicketType
	updated          []*domain.TicketType
	deleted          []uuid.UUID
}

func (m *mockTicketTypeRepo) Create(ctx context.Context, t *domain.TicketType) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, t)
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
	if m.findByEventIDErr != nil {
		return nil, m.findByEventIDErr
	}
	var res []*domain.TicketType
	for _, t := range m.ticketTypes {
		if t.EventID == eventID {
			res = append(res, t)
		}
	}
	return res, nil
}

func (m *mockTicketTypeRepo) Update(ctx context.Context, t *domain.TicketType) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for i, existing := range m.ticketTypes {
		if existing.ID == t.ID {
			m.ticketTypes[i] = t
			m.updated = append(m.updated, t)
			return nil
		}
	}
	return domain.ErrTicketTypeNotFound
}

func (m *mockTicketTypeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, t := range m.ticketTypes {
		if t.ID == id {
			m.ticketTypes = append(m.ticketTypes[:i], m.ticketTypes[i+1:]...)
			m.deleted = append(m.deleted, id)
			return nil
		}
	}
	return domain.ErrTicketTypeNotFound
}

func (m *mockTicketTypeRepo) DeleteByEventID(ctx context.Context, eventID uuid.UUID) error {
	var remaining []*domain.TicketType
	for _, t := range m.ticketTypes {
		if t.EventID != eventID {
			remaining = append(remaining, t)
		}
	}
	m.ticketTypes = remaining
	return nil
}

// --- Test helper ---

func setupTestUsecase(repos *mockTxRepos, onRollback func()) *usecase.EventUsecase {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txMgr := &mockTxManager{txRepos: repos, onRollback: onRollback}
	return usecase.NewEventUsecase(txMgr, nil, logger)
}

func validUpdateInput(userID, eventID, categoryID uuid.UUID, venueID *uuid.UUID) usecase.UpdateEventInput {
	startTime, _ := time.Parse("15:04", "10:00")
	endTime, _ := time.Parse("15:04", "18:00")
	eventDate := time.Now().Add(48 * time.Hour)
	salesStart := time.Now().Add(-24 * time.Hour)
	salesEnd := time.Now().Add(24 * time.Hour)

	return usecase.UpdateEventInput{
		UserID:                    userID,
		EventID:                   eventID,
		CategoryID:                categoryID,
		VenueID:                   venueID,
		EventType:                 domain.EventTypePhysical,
		OnlineURL:                 "",
		Title:                     "Updated Tech Conference",
		Description:               "Updated Conference Description",
		Language:                  "English",
		AgeRestriction:            18,
		Visibility:                "PUBLIC",
		Highlights:                "Keynote, Networking",
		Rules:                     "No smoking",
		AttendeeInformation:       "Bring ID",
		EventDate:                 eventDate,
		StartTime:                 startTime,
		EndTime:                   endTime,
		IsAllDay:                  false,
		SeatLayoutType:            usecase.SeatLayoutTypeGeneral,
		BookingLimitPerUser:       5,
		SalesStartDate:            &salesStart,
		SalesEndDate:              &salesEnd,
		CancellationAllowed:       true,
		CancellationDeadlineHours: 24,
		RefundPolicy:              "STANDARD",
		RefundPercentage:          80,
		Contact: usecase.EventContactInput{
			Name:  "Jane Organizer",
			Phone: "+1987654321",
			Email: "jane@example.com",
		},
		TicketTypes: []usecase.TicketTypeInput{
			{
				Name:        "General Admission",
				Price:       50.0,
				Capacity:    100,
				Description: "Standard ticket",
			},
		},
	}
}

// --- Tests ---

func TestUpdateEvent_Success_UpdatesAllFourRelatedRecords(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Old Title",
		Description: "Old Description",
		Status:      domain.EventStatusDraft,
	}

	existingSchedule := &domain.EventSchedule{
		ID:        uuid.New(),
		EventID:   eventID,
		EventDate: time.Now().Add(24 * time.Hour),
		StartTime: time.Now(),
		EndTime:   time.Now().Add(2 * time.Hour),
	}

	existingSetting := &domain.EventSetting{
		ID:                  uuid.New(),
		EventID:             eventID,
		SeatLayoutType:      usecase.SeatLayoutTypeGeneral,
		BookingLimitPerUser: 2,
	}

	existingCancellation := &domain.EventCancellation{
		ID:                        uuid.New(),
		EventID:                   eventID,
		CancellationAllowed:       false,
		CancellationDeadlineHours: 0,
		RefundPolicy:              "NONE",
		RefundPercentage:          0,
	}

	existingContact := &domain.EventContact{
		ID:      uuid.New(),
		EventID: eventID,
		Name:    "Old Contact",
		Phone:   "+1000000000",
		Email:   "old@example.com",
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{schedule: existingSchedule},
		eventSettingRepo:      &mockEventSettingRepo{setting: existingSetting},
		eventCancellationRepo: &mockEventCancellationRepo{cancellation: existingCancellation},
		eventContactRepo:      &mockEventContactRepo{contact: existingContact},
	}

	uc := setupTestUsecase(repos, nil)
	input := validUpdateInput(userID, eventID, categoryID, &venueID)

	output, err := uc.UpdateEvent(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output == nil {
		t.Fatal("expected non-nil output")
	}

	// Verify main event was updated
	if output.Event.Title != "Updated Tech Conference" {
		t.Errorf("expected title to be 'Updated Tech Conference', got '%s'", output.Event.Title)
	}

	// 1. Verify schedule update
	if output.Schedule == nil {
		t.Fatal("expected non-nil schedule")
	}
	if !output.Schedule.EventDate.Equal(input.EventDate) {
		t.Errorf("schedule EventDate not updated: expected %v, got %v", input.EventDate, output.Schedule.EventDate)
	}
	if repos.eventScheduleRepo.(*mockEventScheduleRepo).updated != true {
		t.Error("expected schedule repository Update to have been called")
	}

	// 2. Verify setting update
	if output.Setting == nil {
		t.Fatal("expected non-nil setting")
	}
	if output.Setting.BookingLimitPerUser != 5 {
		t.Errorf("expected BookingLimitPerUser=5, got %d", output.Setting.BookingLimitPerUser)
	}
	if repos.eventSettingRepo.(*mockEventSettingRepo).updated != true {
		t.Error("expected setting repository Update to have been called")
	}

	// 3. Verify cancellation update
	if output.Cancellation == nil {
		t.Fatal("expected non-nil cancellation")
	}
	if !output.Cancellation.CancellationAllowed || output.Cancellation.CancellationDeadlineHours != 24 {
		t.Errorf("cancellation not updated properly: %+v", output.Cancellation)
	}
	if repos.eventCancellationRepo.(*mockEventCancellationRepo).updated != true {
		t.Error("expected cancellation repository Update to have been called")
	}

	// 4. Verify contact update
	if output.Contact == nil {
		t.Fatal("expected non-nil contact")
	}
	if output.Contact.Name != "Jane Organizer" || output.Contact.Email != "jane@example.com" {
		t.Errorf("contact not updated properly: %+v", output.Contact)
	}
	if repos.eventContactRepo.(*mockEventContactRepo).updated != true {
		t.Error("expected contact repository Update to have been called")
	}
}

func TestUpdateEvent_StatusNotDraft_ReturnsError(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Published Event",
		Status:      domain.EventStatusPublished, // NOT DRAFT
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{},
		eventSettingRepo:      &mockEventSettingRepo{},
		eventCancellationRepo: &mockEventCancellationRepo{},
		eventContactRepo:      &mockEventContactRepo{},
	}

	uc := setupTestUsecase(repos, nil)
	input := validUpdateInput(userID, eventID, categoryID, &venueID)

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, usecase.ErrEventNotEditable) {
		t.Fatalf("expected ErrEventNotEditable, got %v", err)
	}
}

func TestUpdateEvent_OwnershipMismatch_ReturnsUnauthorized(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	otherOrganizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: otherOrganizerID, // Owned by someone else
		Status:      domain.EventStatusDraft,
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{},
		eventSettingRepo:      &mockEventSettingRepo{},
		eventCancellationRepo: &mockEventCancellationRepo{},
		eventContactRepo:      &mockEventContactRepo{},
	}

	uc := setupTestUsecase(repos, nil)
	input := validUpdateInput(userID, eventID, categoryID, &venueID)

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, usecase.ErrUnauthorizedOrganizer) {
		t.Fatalf("expected ErrUnauthorizedOrganizer, got %v", err)
	}
}

func TestUpdateEvent_InactiveOrganizer_ReturnsUnauthorized(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		Status:      domain.EventStatusDraft,
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "SUSPENDED", // Inactive
			},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{},
		eventSettingRepo:      &mockEventSettingRepo{},
		eventCancellationRepo: &mockEventCancellationRepo{},
		eventContactRepo:      &mockEventContactRepo{},
	}

	uc := setupTestUsecase(repos, nil)
	input := validUpdateInput(userID, eventID, categoryID, &venueID)

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, usecase.ErrUnauthorizedOrganizer) {
		t.Fatalf("expected ErrUnauthorizedOrganizer, got %v", err)
	}
}

func TestUpdateEvent_AtomicRollback_OnRelatedUpdateFailure(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Old Title",
		Status:      domain.EventStatusDraft,
	}

	settingErr := errors.New("db disk failure")
	rolledBack := false

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:         &mockEventRepo{event: existingEvent},
		eventScheduleRepo: &mockEventScheduleRepo{schedule: &domain.EventSchedule{ID: uuid.New(), EventID: eventID}},
		eventSettingRepo: &mockEventSettingRepo{
			setting:   &domain.EventSetting{ID: uuid.New(), EventID: eventID},
			updateErr: settingErr, // Setting update fails!
		},
		eventCancellationRepo: &mockEventCancellationRepo{cancellation: &domain.EventCancellation{ID: uuid.New(), EventID: eventID}},
		eventContactRepo:      &mockEventContactRepo{contact: &domain.EventContact{ID: uuid.New(), EventID: eventID}},
	}

	uc := setupTestUsecase(repos, func() {
		rolledBack = true
	})

	input := validUpdateInput(userID, eventID, categoryID, &venueID)

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, settingErr) {
		t.Fatalf("expected settingErr %v, got %v", settingErr, err)
	}

	if !rolledBack {
		t.Error("expected transaction rollback to have been triggered")
	}
}

func TestUpdateEvent_EmptyContact_DeletesContact(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Old Title",
		Status:      domain.EventStatusDraft,
	}

	existingContact := &domain.EventContact{
		ID:      uuid.New(),
		EventID: eventID,
		Name:    "Old Contact",
		Phone:   "+1000000000",
		Email:   "old@example.com",
	}

	contactRepo := &mockEventContactRepo{contact: existingContact}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{schedule: &domain.EventSchedule{ID: uuid.New(), EventID: eventID}},
		eventSettingRepo:      &mockEventSettingRepo{setting: &domain.EventSetting{ID: uuid.New(), EventID: eventID}},
		eventCancellationRepo: &mockEventCancellationRepo{cancellation: &domain.EventCancellation{ID: uuid.New(), EventID: eventID}},
		eventContactRepo:      contactRepo,
	}

	uc := setupTestUsecase(repos, nil)
	input := validUpdateInput(userID, eventID, categoryID, &venueID)
	// Clear contact details
	input.Contact = usecase.EventContactInput{Name: "", Phone: "", Email: ""}

	output, err := uc.UpdateEvent(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contactRepo.deleted {
		t.Error("expected contact to be deleted via DeleteByEventID")
	}

	if output.Contact != nil {
		t.Errorf("expected output.Contact to be nil, got %+v", output.Contact)
	}
}

func TestUpdateEvent_ValidationFailures(t *testing.T) {
	userID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	uc := setupTestUsecase(&mockTxRepos{}, nil)

	t.Run("empty title", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.Title = ""
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventInput) {
			t.Fatalf("expected ErrInvalidEventInput, got %v", err)
		}
	})

	t.Run("zero event date", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.EventDate = time.Time{}
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSchedule) {
			t.Fatalf("expected ErrInvalidEventSchedule, got %v", err)
		}
	})

	t.Run("end time before start time", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		startTime, _ := time.Parse("15:04", "18:00")
		endTime, _ := time.Parse("15:04", "10:00")
		input.StartTime = startTime
		input.EndTime = endTime
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSchedule) {
			t.Fatalf("expected ErrInvalidEventSchedule, got %v", err)
		}
	})

	t.Run("invalid seat layout type", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.SeatLayoutType = "INVALID"
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSettings) {
			t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
		}
	})

	t.Run("seated layout for online event", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, nil)
		input.EventType = domain.EventTypeOnline
		input.OnlineURL = "https://example.com/stream"
		input.SeatLayoutType = usecase.SeatLayoutTypeSeated
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSettings) {
			t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
		}
	})

	t.Run("sales start after sales end", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		salesStart := time.Now().Add(24 * time.Hour)
		salesEnd := time.Now().Add(12 * time.Hour)
		input.SalesStartDate = &salesStart
		input.SalesEndDate = &salesEnd
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSettings) {
			t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
		}
	})

	t.Run("cancellation allowed with zero deadline", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.CancellationAllowed = true
		input.CancellationDeadlineHours = 0
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidCancellation) {
			t.Fatalf("expected ErrInvalidCancellation, got %v", err)
		}
	})

	t.Run("incomplete contact", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.Contact = usecase.EventContactInput{Name: "Only Name"}
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventInput) {
			t.Fatalf("expected ErrInvalidEventInput, got %v", err)
		}
	})

	t.Run("general admission without ticket types", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.SeatLayoutType = usecase.SeatLayoutTypeGeneral
		input.TicketTypes = nil
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSettings) {
			t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
		}
	})

	t.Run("ticket type with empty name", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.TicketTypes = []usecase.TicketTypeInput{{Name: "   ", Price: 10, Capacity: 50}}
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSettings) {
			t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
		}
	})

	t.Run("ticket type with negative price", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.TicketTypes = []usecase.TicketTypeInput{{Name: "VIP", Price: -1, Capacity: 50}}
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSettings) {
			t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
		}
	})

	t.Run("ticket type with zero capacity", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.TicketTypes = []usecase.TicketTypeInput{{Name: "VIP", Price: 100, Capacity: 0}}
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSettings) {
			t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
		}
	})

	t.Run("duplicate ticket type names", func(t *testing.T) {
		input := validUpdateInput(userID, eventID, categoryID, &venueID)
		input.TicketTypes = []usecase.TicketTypeInput{
			{Name: "General", Price: 10, Capacity: 50},
			{Name: "general", Price: 20, Capacity: 50},
		}
		_, err := uc.UpdateEvent(context.Background(), input)
		if !errors.Is(err, usecase.ErrInvalidEventSettings) {
			t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
		}
	})
}

func TestUpdateEvent_CategoryNotFound(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	otherCategoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Title",
		Status:      domain.EventStatusDraft,
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: otherCategoryID, Name: "Music", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo: &mockEventRepo{event: existingEvent},
	}

	uc := setupTestUsecase(repos, nil)
	input := validUpdateInput(userID, eventID, categoryID, &venueID)

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("expected ErrCategoryNotFound, got %v", err)
	}
}

func TestUpdateEvent_VenueNotFound(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Title",
		Status:      domain.EventStatusDraft,
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: nil, // Not found!
		},
		eventRepo: &mockEventRepo{event: existingEvent},
	}

	uc := setupTestUsecase(repos, nil)
	input := validUpdateInput(userID, eventID, categoryID, &venueID)

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, domain.ErrVenueNotFound) {
		t.Fatalf("expected ErrVenueNotFound, got %v", err)
	}
}

func TestUpdateEvent_MissingRelatedRecords_CreatesThem(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Title",
		Status:      domain.EventStatusDraft,
	}

	schedRepo := &mockEventScheduleRepo{}
	setRepo := &mockEventSettingRepo{}
	cancRepo := &mockEventCancellationRepo{}
	contRepo := &mockEventContactRepo{}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     schedRepo,
		eventSettingRepo:      setRepo,
		eventCancellationRepo: cancRepo,
		eventContactRepo:      contRepo,
	}

	uc := setupTestUsecase(repos, nil)
	input := validUpdateInput(userID, eventID, categoryID, &venueID)

	output, err := uc.UpdateEvent(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !schedRepo.created {
		t.Error("expected schedule to be created when not found")
	}
	if !setRepo.created {
		t.Error("expected setting to be created when not found")
	}
	if !cancRepo.created {
		t.Error("expected cancellation to be created when not found")
	}
	if !contRepo.created {
		t.Error("expected contact to be created when not found")
	}
	if output.Schedule == nil || output.Setting == nil || output.Cancellation == nil || output.Contact == nil {
		t.Error("expected all related outputs to be non-nil")
	}
}

func TestUpdateEvent_AtomicRollback_OnTicketSync_TicketTypeNotFound(t *testing.T) {
	// Reproduces the exact reported bug:
	// When ticket synchronization fails with ticket type not found,
	// all prior updates (event, schedule, setting, cancellation, contact)
	// must be rolled back atomically.
	userID := uuid.New()
	organizerID := uuid.New()
	eventID, _ := uuid.Parse("ded204c0-f0b6-42b9-9b00-f15a9cd6b322")
	categoryID := uuid.New()
	venueID := uuid.New()
	missingTicketID, _ := uuid.Parse("d01ee545-806a-4d1c-a39f-263a82f8bb36")

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Original Title In DB",
		Status:      domain.EventStatusDraft,
	}

	existingSchedule := &domain.EventSchedule{
		ID:        uuid.New(),
		EventID:   eventID,
		EventDate: time.Now().Add(24 * time.Hour),
	}

	existingSetting := &domain.EventSetting{
		ID:             uuid.New(),
		EventID:        eventID,
		SeatLayoutType: usecase.SeatLayoutTypeGeneral,
	}

	existingCancellation := &domain.EventCancellation{
		ID:      uuid.New(),
		EventID: eventID,
	}

	existingContact := &domain.EventContact{
		ID:      uuid.New(),
		EventID: eventID,
		Name:    "Original Contact",
	}

	// The event in DB has an existing ticket with a DIFFERENT ID
	dbTicketID := uuid.New()
	existingTicket := &domain.TicketType{
		ID:                dbTicketID,
		EventID:           eventID,
		Name:              "Old Ticket",
		Price:             25.0,
		TotalQuantity:     50,
		AvailableQuantity: 50,
	}

	ticketRepo := &mockTicketTypeRepo{
		ticketTypes: []*domain.TicketType{existingTicket},
	}

	eventRepo := &mockEventRepo{event: existingEvent}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             eventRepo,
		eventScheduleRepo:     &mockEventScheduleRepo{schedule: existingSchedule},
		eventSettingRepo:      &mockEventSettingRepo{setting: existingSetting},
		eventCancellationRepo: &mockEventCancellationRepo{cancellation: existingCancellation},
		eventContactRepo:      &mockEventContactRepo{contact: existingContact},
		ticketTypeRepo:        ticketRepo,
	}

	rolledBack := false
	uc := setupTestUsecase(repos, func() {
		rolledBack = true
	})

	input := validUpdateInput(userID, eventID, categoryID, &venueID)
	input.Title = "Attempted Updated Title"
	// Request refers to a ticket_type_id that does not exist in existingTicketTypes for this event
	input.TicketTypes = []usecase.TicketTypeInput{
		{
			ID:          &missingTicketID,
			Name:        "Nonexistent Ticket",
			Price:       99.0,
			Capacity:    200,
			Description: "Does not exist",
		},
	}

	output, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, domain.ErrTicketTypeNotFound) {
		t.Fatalf("expected ErrTicketTypeNotFound, got %v", err)
	}

	if output != nil {
		t.Errorf("expected output to be nil on error, got %+v", output)
	}

	if !rolledBack {
		t.Fatal("expected atomic transaction rollback when ticket sync fails with ticket type not found")
	}
}

func TestUpdateEvent_AtomicRollback_OnTicketSync_CreateFailure(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Original Title",
		Status:      domain.EventStatusDraft,
	}

	createErr := errors.New("db insert ticket failure")
	ticketRepo := &mockTicketTypeRepo{
		createErr: createErr,
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{schedule: &domain.EventSchedule{ID: uuid.New(), EventID: eventID}},
		eventSettingRepo:      &mockEventSettingRepo{setting: &domain.EventSetting{ID: uuid.New(), EventID: eventID}},
		eventCancellationRepo: &mockEventCancellationRepo{cancellation: &domain.EventCancellation{ID: uuid.New(), EventID: eventID}},
		eventContactRepo:      &mockEventContactRepo{contact: &domain.EventContact{ID: uuid.New(), EventID: eventID}},
		ticketTypeRepo:        ticketRepo,
	}

	rolledBack := false
	uc := setupTestUsecase(repos, func() {
		rolledBack = true
	})

	input := validUpdateInput(userID, eventID, categoryID, &venueID)
	input.TicketTypes = []usecase.TicketTypeInput{
		{
			ID:          nil,
			Name:        "New Ticket",
			Price:       30.0,
			Capacity:    50,
			Description: "Brand new",
		},
	}

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, createErr) {
		t.Fatalf("expected createErr %v, got %v", createErr, err)
	}

	if !rolledBack {
		t.Fatal("expected atomic transaction rollback when ticket create fails")
	}
}

func TestUpdateEvent_AtomicRollback_OnTicketSync_UpdateFailure(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()
	existingTicketID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Original Title",
		Status:      domain.EventStatusDraft,
	}

	existingTicket := &domain.TicketType{
		ID:                existingTicketID,
		EventID:           eventID,
		Name:              "General",
		Price:             20.0,
		TotalQuantity:     100,
		AvailableQuantity: 100,
	}

	updateErr := errors.New("db update ticket failure")
	ticketRepo := &mockTicketTypeRepo{
		ticketTypes: []*domain.TicketType{existingTicket},
		updateErr:   updateErr,
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{schedule: &domain.EventSchedule{ID: uuid.New(), EventID: eventID}},
		eventSettingRepo:      &mockEventSettingRepo{setting: &domain.EventSetting{ID: uuid.New(), EventID: eventID}},
		eventCancellationRepo: &mockEventCancellationRepo{cancellation: &domain.EventCancellation{ID: uuid.New(), EventID: eventID}},
		eventContactRepo:      &mockEventContactRepo{contact: &domain.EventContact{ID: uuid.New(), EventID: eventID}},
		ticketTypeRepo:        ticketRepo,
	}

	rolledBack := false
	uc := setupTestUsecase(repos, func() {
		rolledBack = true
	})

	input := validUpdateInput(userID, eventID, categoryID, &venueID)
	input.TicketTypes = []usecase.TicketTypeInput{
		{
			ID:          &existingTicketID,
			Name:        "General Updated",
			Price:       25.0,
			Capacity:    100,
			Description: "Updated desc",
		},
	}

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, updateErr) {
		t.Fatalf("expected updateErr %v, got %v", updateErr, err)
	}

	if !rolledBack {
		t.Fatal("expected atomic transaction rollback when ticket update fails")
	}
}

func TestUpdateEvent_AtomicRollback_OnTicketSync_DeleteFailure(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()
	existingTicketID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Original Title",
		Status:      domain.EventStatusDraft,
	}

	existingTicket := &domain.TicketType{
		ID:                existingTicketID,
		EventID:           eventID,
		Name:              "General",
		Price:             20.0,
		TotalQuantity:     100,
		AvailableQuantity: 100,
	}

	deleteErr := errors.New("db delete ticket failure")
	ticketRepo := &mockTicketTypeRepo{
		ticketTypes: []*domain.TicketType{existingTicket},
		deleteErr:   deleteErr,
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{schedule: &domain.EventSchedule{ID: uuid.New(), EventID: eventID}},
		eventSettingRepo:      &mockEventSettingRepo{setting: &domain.EventSetting{ID: uuid.New(), EventID: eventID}},
		eventCancellationRepo: &mockEventCancellationRepo{cancellation: &domain.EventCancellation{ID: uuid.New(), EventID: eventID}},
		eventContactRepo:      &mockEventContactRepo{contact: &domain.EventContact{ID: uuid.New(), EventID: eventID}},
		ticketTypeRepo:        ticketRepo,
	}

	rolledBack := false
	uc := setupTestUsecase(repos, func() {
		rolledBack = true
	})

	input := validUpdateInput(userID, eventID, categoryID, &venueID)
	input.TicketTypes = []usecase.TicketTypeInput{
		{
			ID:          nil,
			Name:        "Replacement Ticket",
			Price:       35.0,
			Capacity:    80,
			Description: "Brand new replacement",
		},
	}

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, deleteErr) {
		t.Fatalf("expected deleteErr %v, got %v", deleteErr, err)
	}

	if !rolledBack {
		t.Fatal("expected atomic transaction rollback when ticket delete fails")
	}
}

func TestUpdateEvent_AtomicRollback_OnTicketSync_FindFailure(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Original Title",
		Status:      domain.EventStatusDraft,
	}

	findErr := errors.New("db find ticket types failure")
	ticketRepo := &mockTicketTypeRepo{
		findByEventIDErr: findErr,
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{schedule: &domain.EventSchedule{ID: uuid.New(), EventID: eventID}},
		eventSettingRepo:      &mockEventSettingRepo{setting: &domain.EventSetting{ID: uuid.New(), EventID: eventID}},
		eventCancellationRepo: &mockEventCancellationRepo{cancellation: &domain.EventCancellation{ID: uuid.New(), EventID: eventID}},
		eventContactRepo:      &mockEventContactRepo{contact: &domain.EventContact{ID: uuid.New(), EventID: eventID}},
		ticketTypeRepo:        ticketRepo,
	}

	rolledBack := false
	uc := setupTestUsecase(repos, func() {
		rolledBack = true
	})

	input := validUpdateInput(userID, eventID, categoryID, &venueID)

	_, err := uc.UpdateEvent(context.Background(), input)
	if !errors.Is(err, findErr) {
		t.Fatalf("expected findErr %v, got %v", findErr, err)
	}

	if !rolledBack {
		t.Fatal("expected atomic transaction rollback when ticket FindByEventID fails")
	}
}

func TestUpdateEvent_TicketSync_Success_CreatesUpdatesDeletes(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()
	venueID := uuid.New()

	existingEvent := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		CategoryID:  categoryID,
		VenueID:     &venueID,
		EventType:   domain.EventTypePhysical,
		Title:       "Original Title",
		Status:      domain.EventStatusDraft,
	}

	ticketA := &domain.TicketType{
		ID:                uuid.New(),
		EventID:           eventID,
		Name:              "Ticket A",
		Price:             20.0,
		TotalQuantity:     100,
		AvailableQuantity: 100,
	}
	ticketB := &domain.TicketType{
		ID:                uuid.New(),
		EventID:           eventID,
		Name:              "Ticket B",
		Price:             50.0,
		TotalQuantity:     50,
		AvailableQuantity: 50,
	}

	ticketRepo := &mockTicketTypeRepo{
		ticketTypes: []*domain.TicketType{ticketA, ticketB},
	}

	repos := &mockTxRepos{
		organizerRepo: &mockOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"},
		},
		categoryRepo: &mockCategoryRepo{
			categories: []*domain.Category{{ID: categoryID, Name: "Tech", Status: domain.CategoryStatusActive}},
		},
		venueRepo: &mockVenueRepo{
			venue: &domain.Venue{ID: venueID, Name: "Hall A"},
		},
		eventRepo:             &mockEventRepo{event: existingEvent},
		eventScheduleRepo:     &mockEventScheduleRepo{schedule: &domain.EventSchedule{ID: uuid.New(), EventID: eventID}},
		eventSettingRepo:      &mockEventSettingRepo{setting: &domain.EventSetting{ID: uuid.New(), EventID: eventID}},
		eventCancellationRepo: &mockEventCancellationRepo{cancellation: &domain.EventCancellation{ID: uuid.New(), EventID: eventID}},
		eventContactRepo:      &mockEventContactRepo{contact: &domain.EventContact{ID: uuid.New(), EventID: eventID}},
		ticketTypeRepo:        ticketRepo,
	}

	rolledBack := false
	uc := setupTestUsecase(repos, func() {
		rolledBack = true
	})

	input := validUpdateInput(userID, eventID, categoryID, &venueID)
	// Ticket A is updated, Ticket C is created, Ticket B is omitted (should be deleted)
	input.TicketTypes = []usecase.TicketTypeInput{
		{
			ID:          &ticketA.ID,
			Name:        "Ticket A Updated",
			Price:       25.0,
			Capacity:    120,
			Description: "Updated desc",
		},
		{
			ID:          nil,
			Name:        "Ticket C New",
			Price:       75.0,
			Capacity:    40,
			Description: "New VIP ticket",
		},
	}

	output, err := uc.UpdateEvent(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rolledBack {
		t.Fatal("transaction should not roll back on success")
	}

	if len(output.TicketTypes) != 2 {
		t.Fatalf("expected 2 ticket types in output, got %d", len(output.TicketTypes))
	}

	// Verify Ticket B was deleted
	if len(ticketRepo.deleted) != 1 || ticketRepo.deleted[0] != ticketB.ID {
		t.Errorf("expected Ticket B (%s) to be deleted, got %v", ticketB.ID, ticketRepo.deleted)
	}

	// Verify Ticket A was updated
	if len(ticketRepo.updated) != 1 || ticketRepo.updated[0].Name != "Ticket A Updated" {
		t.Errorf("expected Ticket A to be updated, got %v", ticketRepo.updated)
	}

	// Verify Ticket C was created
	if len(ticketRepo.created) != 1 || ticketRepo.created[0].Name != "Ticket C New" {
		t.Errorf("expected Ticket C to be created, got %v", ticketRepo.created)
	}
}

// --- CreateEvent Test Helpers & Tests ---

func validCreateInput(userID, categoryID uuid.UUID) usecase.CreateEventInput {
	startTime, _ := time.Parse("15:04", "10:00")
	endTime, _ := time.Parse("15:04", "18:00")
	eventDate := time.Now().Add(48 * time.Hour).Truncate(24 * time.Hour)

	return usecase.CreateEventInput{
		UserID:         userID,
		CategoryID:     categoryID,
		EventType:      domain.EventTypePhysical,
		Title:          "Grand Tech Summit",
		Description:    "A comprehensive developer conference",
		Language:       "English",
		AgeRestriction: 18,
		EventDate:      eventDate,
		StartTime:      startTime,
		EndTime:        endTime,
		SeatLayoutType: usecase.SeatLayoutTypeGeneral,
		BookingLimitPerUser: 5,
		CancellationAllowed: false,
		Venue: usecase.VenueInput{
			GooglePlaceID: "ChIJ_TEST_VENUE_001",
			Name:          "Tech Arena",
			Address:       "123 Innovation Way",
			City:          "San Francisco",
			State:         "CA",
			Country:       "USA",
			PostalCode:    "94105",
			Latitude:      37.7749,
			Longitude:     -122.4194,
		},
		TicketTypes: []usecase.TicketTypeInput{
			{
				Name:        "General Admission",
				Price:       25.0,
				Capacity:    100,
				Description: "Standard entry",
			},
		},
		BannerReader:      strings.NewReader("fake-banner-data"),
		BannerContentType: "image/png",
		BannerSize:        1024,
	}
}

func validSeatLayoutInput() usecase.CreateSeatLayoutInput {
	return usecase.CreateSeatLayoutInput{
		LayoutName: "Main Auditorium",
		Sections: []usecase.SeatSectionInput{
			{
				Name:  "Balcony",
				Price: 50.0,
				Rows: []usecase.SeatRowInput{
					{
						RowName: "A",
						Seats:   10,
					},
				},
			},
		},
	}
}

func TestCreateEvent_SeatedWithTicketTypes_FailsValidationBeforeS3(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	repos := &mockTxRepos{}
	uc := setupTestUsecase(repos, nil)

	input := validCreateInput(userID, categoryID)
	input.SeatLayoutType = usecase.SeatLayoutTypeSeated
	input.SeatLayout = validSeatLayoutInput()
	input.TicketTypes = []usecase.TicketTypeInput{
		{
			Name:     "VIP",
			Price:    100.0,
			Capacity: 50,
		},
	}

	_, err := uc.CreateEvent(context.Background(), input)
	if !errors.Is(err, usecase.ErrInvalidEventSettings) {
		t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
	}
}

func TestCreateEvent_GeneralWithClientTicketID_FailsValidationBeforeS3(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	repos := &mockTxRepos{}
	uc := setupTestUsecase(repos, nil)

	ticketID := uuid.New()
	input := validCreateInput(userID, categoryID)
	input.SeatLayoutType = usecase.SeatLayoutTypeGeneral
	input.TicketTypes = []usecase.TicketTypeInput{
		{
			ID:       &ticketID,
			Name:     "General Admission",
			Price:    25.0,
			Capacity: 100,
		},
	}

	_, err := uc.CreateEvent(context.Background(), input)
	if !errors.Is(err, usecase.ErrInvalidEventSettings) {
		t.Fatalf("expected ErrInvalidEventSettings, got %v", err)
	}
}

func TestCreateEvent_OnlineWithSeatedLayout_FailsValidationBeforeS3(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	repos := &mockTxRepos{}
	uc := setupTestUsecase(repos, nil)

	input := validCreateInput(userID, categoryID)
	input.EventType = domain.EventTypeOnline
	input.OnlineURL = "https://example.com/event"
	input.Venue = usecase.VenueInput{}
	input.SeatLayoutType = usecase.SeatLayoutTypeSeated
	input.SeatLayout = validSeatLayoutInput()
	input.TicketTypes = nil

	_, err := uc.CreateEvent(context.Background(), input)
	if !errors.Is(err, usecase.ErrInvalidEventSettings) {
		t.Fatalf("expected ErrInvalidEventSettings for online seated event, got %v", err)
	}
}

func TestCreateEvent_GeneralWithSeatLayout_FailsValidationBeforeS3(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	repos := &mockTxRepos{}
	uc := setupTestUsecase(repos, nil)

	input := validCreateInput(userID, categoryID)
	input.SeatLayoutType = usecase.SeatLayoutTypeGeneral
	input.SeatLayout = validSeatLayoutInput()

	_, err := uc.CreateEvent(context.Background(), input)
	if !errors.Is(err, usecase.ErrInvalidEventSettings) {
		t.Fatalf("expected ErrInvalidEventSettings for general event with seat layout, got %v", err)
	}
}

func TestCreateEvent_GeneralWithoutTicketTypes_FailsValidationBeforeS3(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	repos := &mockTxRepos{}
	uc := setupTestUsecase(repos, nil)

	input := validCreateInput(userID, categoryID)
	input.SeatLayoutType = usecase.SeatLayoutTypeGeneral
	input.TicketTypes = nil

	_, err := uc.CreateEvent(context.Background(), input)
	if !errors.Is(err, usecase.ErrInvalidEventSettings) {
		t.Fatalf("expected ErrInvalidEventSettings for general event without ticket types, got %v", err)
	}
}

func TestCreateEvent_GeneralWithDuplicateTicketNames_FailsValidationBeforeS3(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	repos := &mockTxRepos{}
	uc := setupTestUsecase(repos, nil)

	input := validCreateInput(userID, categoryID)
	input.SeatLayoutType = usecase.SeatLayoutTypeGeneral
	input.TicketTypes = []usecase.TicketTypeInput{
		{Name: "Early Bird", Price: 15, Capacity: 50},
		{Name: "early bird", Price: 20, Capacity: 50},
	}

	_, err := uc.CreateEvent(context.Background(), input)
	if !errors.Is(err, usecase.ErrInvalidEventSettings) {
		t.Fatalf("expected ErrInvalidEventSettings for duplicate ticket names, got %v", err)
	}
}

func TestCreateEvent_SeatedWithInvalidSeatLayout_FailsValidationBeforeS3(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	repos := &mockTxRepos{}
	uc := setupTestUsecase(repos, nil)

	tests := []struct {
		name        string
		seatLayout  usecase.CreateSeatLayoutInput
		expectedErr error
	}{
		{
			name: "empty layout name",
			seatLayout: usecase.CreateSeatLayoutInput{
				LayoutName: "",
				Sections: []usecase.SeatSectionInput{
					{Name: "VIP", Price: 100, Rows: []usecase.SeatRowInput{{RowName: "A", Seats: 5}}},
				},
			},
			expectedErr: usecase.ErrInvalidSeatLayout,
		},
		{
			name: "zero sections",
			seatLayout: usecase.CreateSeatLayoutInput{
				LayoutName: "Main",
				Sections:   nil,
			},
			expectedErr: usecase.ErrInvalidSeatLayout,
		},
		{
			name: "section with empty name",
			seatLayout: usecase.CreateSeatLayoutInput{
				LayoutName: "Main",
				Sections: []usecase.SeatSectionInput{
					{Name: "", Price: 100, Rows: []usecase.SeatRowInput{{RowName: "A", Seats: 5}}},
				},
			},
			expectedErr: usecase.ErrInvalidSeatSection,
		},
		{
			name: "section with negative price",
			seatLayout: usecase.CreateSeatLayoutInput{
				LayoutName: "Main",
				Sections: []usecase.SeatSectionInput{
					{Name: "VIP", Price: -10, Rows: []usecase.SeatRowInput{{RowName: "A", Seats: 5}}},
				},
			},
			expectedErr: usecase.ErrInvalidSeatSection,
		},
		{
			name: "section with zero rows",
			seatLayout: usecase.CreateSeatLayoutInput{
				LayoutName: "Main",
				Sections: []usecase.SeatSectionInput{
					{Name: "VIP", Price: 100, Rows: nil},
				},
			},
			expectedErr: usecase.ErrInvalidSeatSection,
		},
		{
			name: "row with empty row name",
			seatLayout: usecase.CreateSeatLayoutInput{
				LayoutName: "Main",
				Sections: []usecase.SeatSectionInput{
					{Name: "VIP", Price: 100, Rows: []usecase.SeatRowInput{{RowName: "", Seats: 5}}},
				},
			},
			expectedErr: usecase.ErrInvalidSeatRow,
		},
		{
			name: "row with zero seats",
			seatLayout: usecase.CreateSeatLayoutInput{
				LayoutName: "Main",
				Sections: []usecase.SeatSectionInput{
					{Name: "VIP", Price: 100, Rows: []usecase.SeatRowInput{{RowName: "A", Seats: 0}}},
				},
			},
			expectedErr: usecase.ErrInvalidSeatRow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validCreateInput(userID, categoryID)
			input.SeatLayoutType = usecase.SeatLayoutTypeSeated
			input.SeatLayout = tt.seatLayout
			input.TicketTypes = nil

			_, err := uc.CreateEvent(context.Background(), input)
			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}
