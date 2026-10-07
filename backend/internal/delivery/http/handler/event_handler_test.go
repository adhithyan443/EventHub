package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	eventUsecase "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockTxManagerForHandler struct {
	repos domain.TransactionRepositories
}

func (m *mockTxManagerForHandler) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

type mockTxReposForHandler struct {
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

func (m *mockTxReposForHandler) UserRepository() domain.UserRepository { return m.userRepo }
func (m *mockTxReposForHandler) PendingRegistrationRepository() domain.PendingRegistrationRepository {
	return m.pendingRegRepo
}
func (m *mockTxReposForHandler) PasswordResetTokenRepository() domain.PasswordResetTokenRepository {
	return m.passwordResetRepo
}
func (m *mockTxReposForHandler) OrganizerApplicationRepository() domain.OrganizerApplicationRepository {
	return m.organizerAppRepo
}
func (m *mockTxReposForHandler) OrganizerRepository() domain.OrganizerRepository {
	return m.organizerRepo
}
func (m *mockTxReposForHandler) OrganizerProfileRepository() domain.OrganizerProfileRepository {
	return m.organizerProfileRepo
}
func (m *mockTxReposForHandler) OrganizerAddressRepository() domain.OrganizerAddressRepository {
	return m.organizerAddressRepo
}
func (m *mockTxReposForHandler) OrganizerBankAccountRepository() domain.OrganizerBankAccountRepository {
	return m.organizerBankRepo
}
func (m *mockTxReposForHandler) CategoryRepository() domain.CategoryRepository { return m.categoryRepo }
func (m *mockTxReposForHandler) VenueRepository() domain.VenueRepository       { return m.venueRepo }
func (m *mockTxReposForHandler) EventRepository() domain.EventRepository       { return m.eventRepo }
func (m *mockTxReposForHandler) EventScheduleRepository() domain.EventScheduleRepository {
	return m.eventScheduleRepo
}
func (m *mockTxReposForHandler) EventSettingRepository() domain.EventSettingRepository {
	return m.eventSettingRepo
}
func (m *mockTxReposForHandler) EventCancellationRepository() domain.EventCancellationRepository {
	return m.eventCancellationRepo
}
func (m *mockTxReposForHandler) EventContactRepository() domain.EventContactRepository {
	return m.eventContactRepo
}
func (m *mockTxReposForHandler) SeatLayoutRepository() domain.SeatLayoutRepository {
	return m.seatLayoutRepo
}
func (m *mockTxReposForHandler) SeatSectionRepository() domain.SeatSectionRepository {
	return m.seatSectionRepo
}
func (m *mockTxReposForHandler) SeatRowRepository() domain.SeatRowRepository { return m.seatRowRepo }
func (m *mockTxReposForHandler) SeatRepository() domain.SeatRepository       { return m.seatRepo }
func (m *mockTxReposForHandler) TicketTypeRepository() domain.TicketTypeRepository {
	return m.ticketTypeRepo
}

type mockOrganizerRepoForHandler struct {
	organizer *domain.Organizer
}

func (m *mockOrganizerRepoForHandler) Create(organizer *domain.Organizer) error { return nil }
func (m *mockOrganizerRepoForHandler) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	return m.organizer, nil
}
func (m *mockOrganizerRepoForHandler) Update(organizer *domain.Organizer) error { return nil }
func (m *mockOrganizerRepoForHandler) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	return m.organizer, nil
}

type mockEventRepoForHandler struct {
	event *domain.Event
}

func (m *mockEventRepoForHandler) Create(event *domain.Event) error { return nil }
func (m *mockEventRepoForHandler) FindByID(id uuid.UUID) (*domain.Event, error) {
	return m.event, nil
}
func (m *mockEventRepoForHandler) Update(event *domain.Event) error { return nil }
func (m *mockEventRepoForHandler) Delete(id uuid.UUID) error        { return nil }
func (m *mockEventRepoForHandler) FindByOrganizerID(organizerID uuid.UUID, page, limit int, status, search string) ([]*domain.OrganizerEvent, int64, error) {
	return nil, 0, nil
}

type mockEventSettingRepoForHandler struct {
	setting *domain.EventSetting
}

func (m *mockEventSettingRepoForHandler) Create(setting *domain.EventSetting) error { return nil }
func (m *mockEventSettingRepoForHandler) FindByEventID(eventID uuid.UUID) (*domain.EventSetting, error) {
	return m.setting, nil
}
func (m *mockEventSettingRepoForHandler) Update(setting *domain.EventSetting) error { return nil }
func (m *mockEventSettingRepoForHandler) DeleteByEventID(eventID uuid.UUID) error   { return nil }

func TestEventHandler_UpdateEvent_ModeChangeRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()

	mockOrg := &mockOrganizerRepoForHandler{
		organizer: &domain.Organizer{
			ID:     organizerID,
			UserID: userID,
			Status: "ACTIVE",
		},
	}
	mockEvt := &mockEventRepoForHandler{
		event: &domain.Event{
			ID:          eventID,
			OrganizerID: organizerID,
			CategoryID:  categoryID,
			EventType:   domain.EventTypePhysical,
			Title:       "Original Title",
			Status:      domain.EventStatusDraft,
		},
	}
	mockSetting := &mockEventSettingRepoForHandler{
		setting: &domain.EventSetting{
			ID:             uuid.New(),
			EventID:        eventID,
			SeatLayoutType: "GENERAL",
		},
	}

	repos := &mockTxReposForHandler{
		organizerRepo:    mockOrg,
		eventRepo:        mockEvt,
		eventSettingRepo: mockSetting,
	}

	txManager := &mockTxManagerForHandler{repos: repos}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	usecase := eventUsecase.NewEventUsecase(txManager, nil, logger)
	h := NewEventHandler(usecase, logger)

	router := gin.New()
	router.Use(middleware.ErrorHandler(logger))
	// Middleware simulating authentication (setting user_id)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.PUT("/organizers/events/:id", h.UpdateEvent)

	now := time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(24 * time.Hour)
	salesStart := now.Add(-5 * 24 * time.Hour)
	salesEnd := now.Add(-2 * 24 * time.Hour)

	reqBody := UpdateEventRequest{
		CategoryID:          categoryID,
		EventType:           domain.EventTypePhysical,
		Title:               "Updated Title",
		Description:         "Updated Description",
		EventDate:           now.Format("2006-01-02"),
		StartTime:           "10:00",
		EndTime:             "12:00",
		SeatLayoutType:      "SEATED", // Attempted mode change from GENERAL to SEATED
		BookingLimitPerUser: 4,
		SalesStartDate:      salesStart.Format("2006-01-02"),
		SalesEndDate:        salesEnd.Format("2006-01-02"),
		Venue: VenueRequest{
			GooglePlaceID: "place_123",
			Name:          "Venue",
			Address:       "Address",
			City:          "City",
			State:         "State",
			Country:       "Country",
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, "/organizers/events/"+eventID.String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got: %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp["code"] != "VALIDATION_ERROR" {
		t.Fatalf("expected error code VALIDATION_ERROR, got: %v", resp["code"])
	}

	if resp["message"] != "ticketing mode cannot be changed after event creation" {
		t.Fatalf("expected message 'ticketing mode cannot be changed after event creation', got: %v", resp["message"])
	}
}
