package events_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
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
