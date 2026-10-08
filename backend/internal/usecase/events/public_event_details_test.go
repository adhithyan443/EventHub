package events_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	eventUsecase "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/google/uuid"
)

// mockTxManager implements domain.TransactionManager for unit tests.
type mockTxManager struct {
	repos domain.TransactionRepositories
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

// mockTxRepos implements domain.TransactionRepositories.
type mockTxRepos struct {
	eventRepo        domain.EventRepository
	categoryRepo     domain.CategoryRepository
	scheduleRepo     domain.EventScheduleRepository
	settingRepo      domain.EventSettingRepository
	cancellationRepo domain.EventCancellationRepository
	contactRepo      domain.EventContactRepository
	venueRepo        domain.VenueRepository
	organizerRepo    domain.OrganizerRepository
	profileRepo      domain.OrganizerProfileRepository
	ticketTypeRepo   domain.TicketTypeRepository
	seatLayoutRepo   domain.SeatLayoutRepository
	seatSectionRepo  domain.SeatSectionRepository
}

func (m *mockTxRepos) UserRepository() domain.UserRepository { return nil }
func (m *mockTxRepos) PendingRegistrationRepository() domain.PendingRegistrationRepository {
	return nil
}
func (m *mockTxRepos) PasswordResetTokenRepository() domain.PasswordResetTokenRepository { return nil }
func (m *mockTxRepos) OrganizerApplicationRepository() domain.OrganizerApplicationRepository {
	return nil
}
func (m *mockTxRepos) OrganizerRepository() domain.OrganizerRepository { return m.organizerRepo }
func (m *mockTxRepos) OrganizerProfileRepository() domain.OrganizerProfileRepository {
	return m.profileRepo
}
func (m *mockTxRepos) OrganizerAddressRepository() domain.OrganizerAddressRepository { return nil }
func (m *mockTxRepos) OrganizerBankAccountRepository() domain.OrganizerBankAccountRepository {
	return nil
}
func (m *mockTxRepos) CategoryRepository() domain.CategoryRepository           { return m.categoryRepo }
func (m *mockTxRepos) VenueRepository() domain.VenueRepository                 { return m.venueRepo }
func (m *mockTxRepos) EventRepository() domain.EventRepository                 { return m.eventRepo }
func (m *mockTxRepos) EventScheduleRepository() domain.EventScheduleRepository { return m.scheduleRepo }
func (m *mockTxRepos) EventSettingRepository() domain.EventSettingRepository   { return m.settingRepo }
func (m *mockTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return m.cancellationRepo
}
func (m *mockTxRepos) EventContactRepository() domain.EventContactRepository { return m.contactRepo }
func (m *mockTxRepos) SeatLayoutRepository() domain.SeatLayoutRepository     { return m.seatLayoutRepo }
func (m *mockTxRepos) SeatSectionRepository() domain.SeatSectionRepository   { return m.seatSectionRepo }
func (m *mockTxRepos) SeatRowRepository() domain.SeatRowRepository           { return nil }
func (m *mockTxRepos) SeatRepository() domain.SeatRepository                 { return nil }
func (m *mockTxRepos) TicketTypeRepository() domain.TicketTypeRepository     { return m.ticketTypeRepo }

// Stubs for individual repositories
type stubEventRepo struct {
	event *domain.Event
	err   error
}

func (s *stubEventRepo) Create(event *domain.Event) error             { return nil }
func (s *stubEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) { return s.event, s.err }
func (s *stubEventRepo) Update(event *domain.Event) error             { return nil }
func (s *stubEventRepo) Delete(id uuid.UUID) error                    { return nil }
func (s *stubEventRepo) FindByOrganizerID(uuid.UUID, int, int, string, string) ([]*domain.OrganizerEvent, int64, error) {
	return nil, 0, nil
}
func (s *stubEventRepo) FindPublicEvents(domain.PublicEventFilter) ([]*domain.PublicEvent, int64, error) {
	return nil, 0, nil
}

type stubCategoryRepo struct {
	category *domain.Category
	err      error
}

func (s *stubCategoryRepo) FindActive() ([]*domain.Category, error)         { return nil, nil }
func (s *stubCategoryRepo) FindByID(id uuid.UUID) (*domain.Category, error) { return s.category, s.err }

type stubScheduleRepo struct {
	schedule *domain.EventSchedule
	err      error
}

func (s *stubScheduleRepo) Create(schedule *domain.EventSchedule) error { return nil }
func (s *stubScheduleRepo) FindByEventID(id uuid.UUID) (*domain.EventSchedule, error) {
	return s.schedule, s.err
}
func (s *stubScheduleRepo) Update(schedule *domain.EventSchedule) error { return nil }
func (s *stubScheduleRepo) DeleteByEventID(id uuid.UUID) error          { return nil }

type stubSettingRepo struct {
	setting *domain.EventSetting
	err     error
}

func (s *stubSettingRepo) Create(setting *domain.EventSetting) error { return nil }
func (s *stubSettingRepo) FindByEventID(id uuid.UUID) (*domain.EventSetting, error) {
	return s.setting, s.err
}
func (s *stubSettingRepo) Update(setting *domain.EventSetting) error { return nil }
func (s *stubSettingRepo) DeleteByEventID(id uuid.UUID) error        { return nil }

type stubCancellationRepo struct {
	cancellation *domain.EventCancellation
	err          error
}

func (s *stubCancellationRepo) Create(c *domain.EventCancellation) error { return nil }
func (s *stubCancellationRepo) FindByEventID(id uuid.UUID) (*domain.EventCancellation, error) {
	return s.cancellation, s.err
}
func (s *stubCancellationRepo) Update(c *domain.EventCancellation) error { return nil }
func (s *stubCancellationRepo) DeleteByEventID(id uuid.UUID) error       { return nil }

type stubContactRepo struct {
	contact *domain.EventContact
	err     error
}

func (s *stubContactRepo) Create(c *domain.EventContact) error { return nil }
func (s *stubContactRepo) FindByEventID(id uuid.UUID) (*domain.EventContact, error) {
	return s.contact, s.err
}
func (s *stubContactRepo) Update(c *domain.EventContact) error { return nil }
func (s *stubContactRepo) DeleteByEventID(id uuid.UUID) error  { return nil }

type stubVenueRepo struct {
	venue *domain.Venue
	err   error
}

func (s *stubVenueRepo) Create(v *domain.Venue) error                      { return nil }
func (s *stubVenueRepo) FindByID(id uuid.UUID) (*domain.Venue, error)      { return s.venue, s.err }
func (s *stubVenueRepo) FindByGooglePlaceID(string) (*domain.Venue, error) { return nil, nil }

type stubOrganizerRepo struct {
	organizer *domain.Organizer
	err       error
}

func (s *stubOrganizerRepo) Create(o *domain.Organizer) error { return nil }
func (s *stubOrganizerRepo) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	return s.organizer, s.err
}
func (s *stubOrganizerRepo) FindByUserID(id uuid.UUID) (*domain.Organizer, error) { return nil, nil }

type stubProfileRepo struct {
	profile *domain.OrganizerProfile
	err     error
}

func (s *stubProfileRepo) Create(p *domain.OrganizerProfile) error { return nil }
func (s *stubProfileRepo) FindByOrganizerID(id uuid.UUID) (*domain.OrganizerProfile, error) {
	return s.profile, s.err
}

type stubTicketTypeRepo struct {
	ticketTypes []*domain.TicketType
	err         error
}

func (s *stubTicketTypeRepo) Create(context.Context, *domain.TicketType) error { return nil }
func (s *stubTicketTypeRepo) FindByID(context.Context, uuid.UUID) (*domain.TicketType, error) {
	return nil, nil
}
func (s *stubTicketTypeRepo) FindByEventID(context.Context, uuid.UUID) ([]*domain.TicketType, error) {
	return s.ticketTypes, s.err
}
func (s *stubTicketTypeRepo) Update(context.Context, *domain.TicketType) error { return nil }
func (s *stubTicketTypeRepo) Delete(context.Context, uuid.UUID) error          { return nil }
func (s *stubTicketTypeRepo) DeleteByEventID(context.Context, uuid.UUID) error { return nil }

type stubSeatLayoutRepo struct {
	layout *domain.SeatLayout
	err    error
}

func (s *stubSeatLayoutRepo) Create(*domain.SeatLayout) error { return nil }
func (s *stubSeatLayoutRepo) Update(*domain.SeatLayout) error { return nil }
func (s *stubSeatLayoutRepo) FindByEventID(uuid.UUID) (*domain.SeatLayout, error) {
	return s.layout, s.err
}
func (s *stubSeatLayoutRepo) DeleteByEventID(uuid.UUID) error { return nil }

type stubSeatSectionRepo struct {
	stats []domain.SeatSectionWithStats
	err   error
}

func (s *stubSeatSectionRepo) Create(*domain.SeatSection) error     { return nil }
func (s *stubSeatSectionRepo) Update(*domain.SeatSection) error     { return nil }
func (s *stubSeatSectionRepo) Delete(uuid.UUID) error               { return nil }
func (s *stubSeatSectionRepo) DeleteBySeatLayoutID(uuid.UUID) error { return nil }
func (s *stubSeatSectionRepo) FindBySeatLayoutID(uuid.UUID) ([]domain.SeatSection, error) {
	return nil, nil
}
func (s *stubSeatSectionRepo) FindWithStatsBySeatLayoutID(uuid.UUID) ([]domain.SeatSectionWithStats, error) {
	return s.stats, s.err
}

func setupBaseUsecase(repos *mockTxRepos) *eventUsecase.EventUsecase {
	txManager := &mockTxManager{repos: repos}
	return eventUsecase.NewEventUsecase(txManager, nil, nil)
}

func createStandardMocks(eventID, categoryID, organizerID uuid.UUID) *mockTxRepos {
	return &mockTxRepos{
		categoryRepo: &stubCategoryRepo{
			category: &domain.Category{ID: categoryID, Name: "Music"},
		},
		scheduleRepo: &stubScheduleRepo{
			schedule: &domain.EventSchedule{
				ID:        uuid.New(),
				EventID:   eventID,
				EventDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
			},
		},
		settingRepo: &stubSettingRepo{
			setting: &domain.EventSetting{
				ID:                  uuid.New(),
				EventID:             eventID,
				SeatLayoutType:      "GENERAL",
				BookingLimitPerUser: 10,
			},
		},
		cancellationRepo: &stubCancellationRepo{
			cancellation: &domain.EventCancellation{
				ID:                        uuid.New(),
				EventID:                   eventID,
				CancellationAllowed:       true,
				CancellationDeadlineHours: 48,
				RefundPolicy:              "FULL",
				RefundPercentage:          100,
			},
		},
		contactRepo: &stubContactRepo{
			contact: &domain.EventContact{
				ID:      uuid.New(),
				EventID: eventID,
				Name:    "Event Team",
				Email:   "info@example.com",
				Phone:   "+91 9999999999",
			},
		},
		organizerRepo: &stubOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				Status: "ACTIVE",
			},
		},
		profileRepo: &stubProfileRepo{
			profile: &domain.OrganizerProfile{
				OrganizerID:  organizerID,
				BusinessName: "Event Masters",
			},
		},
		ticketTypeRepo: &stubTicketTypeRepo{
			ticketTypes: []*domain.TicketType{
				{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "General Admission",
					Price:             499,
					TotalQuantity:     100,
					AvailableQuantity: 50,
					Description:       "Standard pass",
				},
			},
		},
	}
}

func TestGetPublicEventDetails_PublishedAndPublic_Success(t *testing.T) {
	eventID := uuid.New()
	categoryID := uuid.New()
	organizerID := uuid.New()

	repos := createStandardMocks(eventID, categoryID, organizerID)
	repos.eventRepo = &stubEventRepo{
		event: &domain.Event{
			ID:          eventID,
			CategoryID:  categoryID,
			OrganizerID: organizerID,
			Title:       "Public Concert",
			Status:      domain.EventStatusPublished,
			Visibility:  "PUBLIC",
			EventType:   "PHYSICAL",
		},
	}

	uc := setupBaseUsecase(repos)
	details, err := uc.GetPublicEventDetails(context.Background(), eventID)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if details == nil {
		t.Fatal("expected non-nil event details")
	}

	if details.ID != eventID {
		t.Errorf("expected event ID %v, got %v", eventID, details.ID)
	}

	if details.Status != domain.EventStatusPublished {
		t.Errorf("expected status %s, got %s", domain.EventStatusPublished, details.Status)
	}

	if details.Visibility != "PUBLIC" {
		t.Errorf("expected visibility PUBLIC, got %s", details.Visibility)
	}

	if details.CategoryName != "Music" {
		t.Errorf("expected category Music, got %s", details.CategoryName)
	}

	if details.Organizer.Name != "Event Masters" || !details.Organizer.Verified {
		t.Errorf("expected verified organizer 'Event Masters', got %+v", details.Organizer)
	}
}

func TestGetPublicEventDetails_PublishedAndPrivate_ReturnsErrEventNotPublic(t *testing.T) {
	eventID := uuid.New()
	categoryID := uuid.New()
	organizerID := uuid.New()

	repos := createStandardMocks(eventID, categoryID, organizerID)
	repos.eventRepo = &stubEventRepo{
		event: &domain.Event{
			ID:          eventID,
			CategoryID:  categoryID,
			OrganizerID: organizerID,
			Title:       "Private Event",
			Status:      domain.EventStatusPublished,
			Visibility:  "PRIVATE",
		},
	}

	uc := setupBaseUsecase(repos)
	details, err := uc.GetPublicEventDetails(context.Background(), eventID)

	if !errors.Is(err, domain.ErrEventNotPublic) {
		t.Fatalf("expected ErrEventNotPublic, got %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details, got %+v", details)
	}
}

func TestGetPublicEventDetails_DraftAndPublic_ReturnsErrEventNotPublic(t *testing.T) {
	eventID := uuid.New()
	categoryID := uuid.New()
	organizerID := uuid.New()

	repos := createStandardMocks(eventID, categoryID, organizerID)
	repos.eventRepo = &stubEventRepo{
		event: &domain.Event{
			ID:          eventID,
			CategoryID:  categoryID,
			OrganizerID: organizerID,
			Title:       "Draft Event",
			Status:      domain.EventStatusDraft,
			Visibility:  "PUBLIC",
		},
	}

	uc := setupBaseUsecase(repos)
	details, err := uc.GetPublicEventDetails(context.Background(), eventID)

	if !errors.Is(err, domain.ErrEventNotPublic) {
		t.Fatalf("expected ErrEventNotPublic, got %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details, got %+v", details)
	}
}

func TestGetPublicEventDetails_CancelledAndPublic_ReturnsErrEventNotPublic(t *testing.T) {
	eventID := uuid.New()
	categoryID := uuid.New()
	organizerID := uuid.New()

	repos := createStandardMocks(eventID, categoryID, organizerID)
	repos.eventRepo = &stubEventRepo{
		event: &domain.Event{
			ID:          eventID,
			CategoryID:  categoryID,
			OrganizerID: organizerID,
			Title:       "Cancelled Event",
			Status:      domain.EventStatusCancelled,
			Visibility:  "PUBLIC",
		},
	}

	uc := setupBaseUsecase(repos)
	details, err := uc.GetPublicEventDetails(context.Background(), eventID)

	if !errors.Is(err, domain.ErrEventNotPublic) {
		t.Fatalf("expected ErrEventNotPublic, got %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details, got %+v", details)
	}
}

func TestGetPublicEventDetails_NonExistentEvent_ReturnsErrEventNotFound(t *testing.T) {
	eventID := uuid.New()

	repos := &mockTxRepos{
		eventRepo: &stubEventRepo{
			err: domain.ErrEventNotFound,
		},
	}

	uc := setupBaseUsecase(repos)
	details, err := uc.GetPublicEventDetails(context.Background(), eventID)

	if !errors.Is(err, domain.ErrEventNotFound) {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details, got %+v", details)
	}
}

func TestGetPublicEventDetails_RepositoryFailure_ReturnsError(t *testing.T) {
	eventID := uuid.New()
	dbErr := errors.New("database connection failed")

	repos := &mockTxRepos{
		eventRepo: &stubEventRepo{
			err: dbErr,
		},
	}

	uc := setupBaseUsecase(repos)
	details, err := uc.GetPublicEventDetails(context.Background(), eventID)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details, got %+v", details)
	}
}

func TestGetPublicEventDetails_GeneralTicketTiers(t *testing.T) {
	eventID := uuid.New()
	categoryID := uuid.New()
	organizerID := uuid.New()

	repos := createStandardMocks(eventID, categoryID, organizerID)
	repos.eventRepo = &stubEventRepo{
		event: &domain.Event{
			ID:          eventID,
			CategoryID:  categoryID,
			OrganizerID: organizerID,
			Title:       "General Admission Festival",
			Status:      domain.EventStatusPublished,
			Visibility:  "PUBLIC",
		},
	}
	repos.settingRepo = &stubSettingRepo{
		setting: &domain.EventSetting{
			EventID:        eventID,
			SeatLayoutType: "GENERAL",
		},
	}
	repos.ticketTypeRepo = &stubTicketTypeRepo{
		ticketTypes: []*domain.TicketType{
			{
				ID:                uuid.New(),
				Name:              "Tier 1 Early Bird",
				Price:             299,
				TotalQuantity:     200,
				AvailableQuantity: 150,
				Description:       "Early pass",
			},
			{
				ID:                uuid.New(),
				Name:              "Tier 2 Regular",
				Price:             499,
				TotalQuantity:     500,
				AvailableQuantity: 50, // <= 20% -> SELLING_FAST
				Description:       "Regular pass",
			},
		},
	}

	uc := setupBaseUsecase(repos)
	details, err := uc.GetPublicEventDetails(context.Background(), eventID)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(details.TicketTypes) != 2 {
		t.Fatalf("expected 2 ticket types, got %d", len(details.TicketTypes))
	}

	if details.TicketTypes[0].Status != "AVAILABLE" {
		t.Errorf("expected AVAILABLE status, got %s", details.TicketTypes[0].Status)
	}

	if details.TicketTypes[1].Status != "SELLING_FAST" {
		t.Errorf("expected SELLING_FAST status, got %s", details.TicketTypes[1].Status)
	}

	if details.StartingPrice != 299 {
		t.Errorf("expected starting price 299, got %f", details.StartingPrice)
	}
}

func TestGetPublicEventDetails_SeatedNormalizedTicketTiers_NoIndividualSeatsExposed(t *testing.T) {
	eventID := uuid.New()
	categoryID := uuid.New()
	organizerID := uuid.New()
	layoutID := uuid.New()

	repos := createStandardMocks(eventID, categoryID, organizerID)
	repos.eventRepo = &stubEventRepo{
		event: &domain.Event{
			ID:          eventID,
			CategoryID:  categoryID,
			OrganizerID: organizerID,
			Title:       "Seated Opera",
			Status:      domain.EventStatusPublished,
			Visibility:  "PUBLIC",
		},
	}
	repos.settingRepo = &stubSettingRepo{
		setting: &domain.EventSetting{
			EventID:        eventID,
			SeatLayoutType: "SEATED",
		},
	}
	repos.seatLayoutRepo = &stubSeatLayoutRepo{
		layout: &domain.SeatLayout{
			ID:      layoutID,
			EventID: eventID,
		},
	}
	repos.seatSectionRepo = &stubSeatSectionRepo{
		stats: []domain.SeatSectionWithStats{
			{
				ID:                uuid.New(),
				SeatLayoutID:      layoutID,
				Name:              "VIP Front Row",
				Price:             1499,
				TotalCapacity:     50,
				AvailableQuantity: 40,
			},
			{
				ID:                uuid.New(),
				SeatLayoutID:      layoutID,
				Name:              "Balcony General",
				Price:             799,
				TotalCapacity:     100,
				AvailableQuantity: 0, // SOLD_OUT
			},
		},
	}

	uc := setupBaseUsecase(repos)
	details, err := uc.GetPublicEventDetails(context.Background(), eventID)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(details.TicketTypes) != 2 {
		t.Fatalf("expected 2 normalized ticket types, got %d", len(details.TicketTypes))
	}

	if details.TicketTypes[0].Name != "VIP Front Row" || details.TicketTypes[0].Price != 1499 {
		t.Errorf("unexpected tier 0 data: %+v", details.TicketTypes[0])
	}

	if details.TicketTypes[1].Status != "SOLD_OUT" {
		t.Errorf("expected SOLD_OUT for tier 1, got %s", details.TicketTypes[1].Status)
	}

	if details.StartingPrice != 799 {
		t.Errorf("expected starting price 799, got %f", details.StartingPrice)
	}
}
