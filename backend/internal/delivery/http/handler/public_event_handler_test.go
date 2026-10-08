package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	eventUsecase "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockTxManager struct {
	repos domain.TransactionRepositories
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

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

type stubEventRepo struct {
	event *domain.Event
	err   error
}

func (s *stubEventRepo) Create(*domain.Event) error                { return nil }
func (s *stubEventRepo) FindByID(uuid.UUID) (*domain.Event, error) { return s.event, s.err }
func (s *stubEventRepo) Update(*domain.Event) error                { return nil }
func (s *stubEventRepo) Delete(uuid.UUID) error                    { return nil }
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

func (s *stubCategoryRepo) FindActive() ([]*domain.Category, error)      { return nil, nil }
func (s *stubCategoryRepo) FindByID(uuid.UUID) (*domain.Category, error) { return s.category, s.err }

type stubScheduleRepo struct {
	schedule *domain.EventSchedule
	err      error
}

func (s *stubScheduleRepo) Create(*domain.EventSchedule) error { return nil }
func (s *stubScheduleRepo) FindByEventID(uuid.UUID) (*domain.EventSchedule, error) {
	return s.schedule, s.err
}
func (s *stubScheduleRepo) Update(*domain.EventSchedule) error { return nil }
func (s *stubScheduleRepo) DeleteByEventID(uuid.UUID) error    { return nil }

type stubSettingRepo struct {
	setting *domain.EventSetting
	err     error
}

func (s *stubSettingRepo) Create(*domain.EventSetting) error { return nil }
func (s *stubSettingRepo) FindByEventID(uuid.UUID) (*domain.EventSetting, error) {
	return s.setting, s.err
}
func (s *stubSettingRepo) Update(*domain.EventSetting) error { return nil }
func (s *stubSettingRepo) DeleteByEventID(uuid.UUID) error   { return nil }

type stubCancellationRepo struct {
	cancellation *domain.EventCancellation
	err          error
}

func (s *stubCancellationRepo) Create(*domain.EventCancellation) error { return nil }
func (s *stubCancellationRepo) FindByEventID(uuid.UUID) (*domain.EventCancellation, error) {
	return s.cancellation, s.err
}
func (s *stubCancellationRepo) Update(*domain.EventCancellation) error { return nil }
func (s *stubCancellationRepo) DeleteByEventID(uuid.UUID) error        { return nil }

type stubContactRepo struct {
	contact *domain.EventContact
	err     error
}

func (s *stubContactRepo) Create(*domain.EventContact) error { return nil }
func (s *stubContactRepo) FindByEventID(uuid.UUID) (*domain.EventContact, error) {
	return s.contact, s.err
}
func (s *stubContactRepo) Update(*domain.EventContact) error { return nil }
func (s *stubContactRepo) DeleteByEventID(uuid.UUID) error   { return nil }

type stubOrganizerRepo struct {
	organizer *domain.Organizer
	err       error
}

func (s *stubOrganizerRepo) Create(*domain.Organizer) error { return nil }
func (s *stubOrganizerRepo) FindByID(uuid.UUID) (*domain.Organizer, error) {
	return s.organizer, s.err
}
func (s *stubOrganizerRepo) FindByUserID(uuid.UUID) (*domain.Organizer, error) { return nil, nil }

type stubProfileRepo struct {
	profile *domain.OrganizerProfile
	err     error
}

func (s *stubProfileRepo) Create(*domain.OrganizerProfile) error { return nil }
func (s *stubProfileRepo) FindByOrganizerID(uuid.UUID) (*domain.OrganizerProfile, error) {
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

func setupTestRouter(repos *mockTxRepos) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	txManager := &mockTxManager{repos: repos}
	uc := eventUsecase.NewEventUsecase(txManager, nil, nil)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventHandler := handler.NewEventHandler(uc, logger)

	router.GET("/api/v1/events/:eventId", eventHandler.GetPublicEventDetails)
	return router
}

func TestGetPublicEventDetailsHandler_InvalidUUID(t *testing.T) {
	router := setupTestRouter(&mockTxRepos{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp["success"] != false || resp["message"] != "invalid event id" {
		t.Errorf("unexpected response body: %+v", resp)
	}
}

func TestGetPublicEventDetailsHandler_NonExistentEvent(t *testing.T) {
	eventID := uuid.New()
	router := setupTestRouter(&mockTxRepos{
		eventRepo: &stubEventRepo{err: domain.ErrEventNotFound},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+eventID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp["success"] != false || resp["message"] != "event not found" {
		t.Errorf("unexpected response body: %+v", resp)
	}
}

func TestGetPublicEventDetailsHandler_DraftOrPrivate_Returns404Indistinguishable(t *testing.T) {
	eventID := uuid.New()
	router := setupTestRouter(&mockTxRepos{
		eventRepo: &stubEventRepo{
			event: &domain.Event{
				ID:         eventID,
				Status:     domain.EventStatusDraft,
				Visibility: "PUBLIC",
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+eventID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// DRAFT events MUST return 404 with identical "event not found" message
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for DRAFT event, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp["success"] != false || resp["message"] != "event not found" {
		t.Errorf("expected indistinguishable 'event not found' message, got: %+v", resp)
	}
}

func TestGetPublicEventDetailsHandler_Success200(t *testing.T) {
	eventID := uuid.New()
	categoryID := uuid.New()
	organizerID := uuid.New()

	router := setupTestRouter(&mockTxRepos{
		eventRepo: &stubEventRepo{
			event: &domain.Event{
				ID:          eventID,
				CategoryID:  categoryID,
				OrganizerID: organizerID,
				Title:       "Live Open Air Concert",
				Description: "Spectacular concert",
				Status:      domain.EventStatusPublished,
				Visibility:  "PUBLIC",
				EventType:   "PHYSICAL",
			},
		},
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
				BookingLimitPerUser: 4,
			},
		},
		cancellationRepo: &stubCancellationRepo{
			cancellation: &domain.EventCancellation{
				ID:                  uuid.New(),
				EventID:             eventID,
				CancellationAllowed: true,
			},
		},
		contactRepo: &stubContactRepo{
			contact: &domain.EventContact{
				ID:      uuid.New(),
				EventID: eventID,
				Name:    "Concert Support",
				Email:   "support@concert.com",
				Phone:   "+1234567890",
			},
		},
		organizerRepo: &stubOrganizerRepo{
			organizer: &domain.Organizer{ID: organizerID, Status: "ACTIVE"},
		},
		profileRepo: &stubProfileRepo{
			profile: &domain.OrganizerProfile{OrganizerID: organizerID, BusinessName: "Live Events Ltd"},
		},
		ticketTypeRepo: &stubTicketTypeRepo{
			ticketTypes: []*domain.TicketType{
				{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "General Entry",
					Price:             500,
					TotalQuantity:     100,
					AvailableQuantity: 80,
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+eventID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                      `json:"success"`
		Data    domain.PublicEventDetails `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Fatal("expected success true")
	}

	if resp.Data.Title != "Live Open Air Concert" {
		t.Errorf("expected title 'Live Open Air Concert', got '%s'", resp.Data.Title)
	}

	if resp.Data.CategoryName != "Music" {
		t.Errorf("expected category 'Music', got '%s'", resp.Data.CategoryName)
	}

	if resp.Data.Organizer.Name != "Live Events Ltd" || !resp.Data.Organizer.Verified {
		t.Errorf("unexpected organizer: %+v", resp.Data.Organizer)
	}

	if len(resp.Data.TicketTypes) != 1 || resp.Data.TicketTypes[0].Price != 500 {
		t.Errorf("unexpected ticket types: %+v", resp.Data.TicketTypes)
	}
}
