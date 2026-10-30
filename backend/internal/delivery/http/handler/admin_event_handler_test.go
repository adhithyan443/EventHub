package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAdminEventRepoForHandlerTest struct {
	domain.EventRepository
	events       map[uuid.UUID]*domain.Event
	listResult   []*domain.AdminEventListItem
	statusCounts domain.AdminEventStatusCounts
	deletedIDs   []uuid.UUID
}

func (m *mockAdminEventRepoForHandlerTest) FindAllEvents(filter domain.AdminEventFilter) ([]*domain.AdminEventListItem, int64, error) {
	return m.listResult, int64(len(m.listResult)), nil
}

func (m *mockAdminEventRepoForHandlerTest) GetAdminStatusCounts() (domain.AdminEventStatusCounts, error) {
	return m.statusCounts, nil
}

func (m *mockAdminEventRepoForHandlerTest) FindByID(id uuid.UUID) (*domain.Event, error) {
	if ev, ok := m.events[id]; ok {
		copied := *ev
		return &copied, nil
	}
	return nil, domain.ErrEventNotFound
}

func (m *mockAdminEventRepoForHandlerTest) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) {
	return m.FindByID(id)
}

func (m *mockAdminEventRepoForHandlerTest) Update(ev *domain.Event) error {
	if _, ok := m.events[ev.ID]; !ok {
		return domain.ErrEventNotFound
	}
	copied := *ev
	m.events[ev.ID] = &copied
	return nil
}

func (m *mockAdminEventRepoForHandlerTest) Delete(id uuid.UUID) error {
	delete(m.events, id)
	m.deletedIDs = append(m.deletedIDs, id)
	return nil
}

type mockAdminTxManagerForHandler struct {
	txRepos *mockAdminTxReposForHandler
}

func (m *mockAdminTxManagerForHandler) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.txRepos)
}

type mockAdminTxReposForHandler struct {
	domain.TransactionRepositories
	eventRepo        domain.EventRepository
	bookingRepo      domain.BookingRepository
	reservationRepo  domain.SeatReservationRepository
	organizerRepo    domain.OrganizerRepository
	profileRepo      domain.OrganizerProfileRepository
	userRepo         domain.UserRepository
	categoryRepo     domain.CategoryRepository
	venueRepo        domain.VenueRepository
	scheduleRepo     domain.EventScheduleRepository
	settingRepo      domain.EventSettingRepository
	cancellationRepo domain.EventCancellationRepository
	contactRepo      domain.EventContactRepository
	ticketTypeRepo   domain.TicketTypeRepository
	ticketRepo       domain.TicketRepository
	seatLayoutRepo   domain.SeatLayoutRepository
	seatSectionRepo  domain.SeatSectionRepository
}

func (m *mockAdminTxReposForHandler) EventRepository() domain.EventRepository { return m.eventRepo }
func (m *mockAdminTxReposForHandler) BookingRepository() domain.BookingRepository {
	return m.bookingRepo
}
func (m *mockAdminTxReposForHandler) TicketRepository() domain.TicketRepository {
	if m.ticketRepo == nil {
		return &mockAdminTicketRepoForHandler{}
	}
	return m.ticketRepo
}
func (m *mockAdminTxReposForHandler) SeatReservationRepository() domain.SeatReservationRepository {
	return m.reservationRepo
}
func (m *mockAdminTxReposForHandler) OrganizerRepository() domain.OrganizerRepository {
	return m.organizerRepo
}
func (m *mockAdminTxReposForHandler) OrganizerProfileRepository() domain.OrganizerProfileRepository {
	return m.profileRepo
}
func (m *mockAdminTxReposForHandler) UserRepository() domain.UserRepository { return m.userRepo }
func (m *mockAdminTxReposForHandler) CategoryRepository() domain.CategoryRepository {
	return m.categoryRepo
}
func (m *mockAdminTxReposForHandler) VenueRepository() domain.VenueRepository { return m.venueRepo }
func (m *mockAdminTxReposForHandler) EventScheduleRepository() domain.EventScheduleRepository {
	return m.scheduleRepo
}
func (m *mockAdminTxReposForHandler) EventSettingRepository() domain.EventSettingRepository {
	return m.settingRepo
}
func (m *mockAdminTxReposForHandler) EventCancellationRepository() domain.EventCancellationRepository {
	return m.cancellationRepo
}
func (m *mockAdminTxReposForHandler) EventContactRepository() domain.EventContactRepository {
	return m.contactRepo
}
func (m *mockAdminTxReposForHandler) TicketTypeRepository() domain.TicketTypeRepository {
	return m.ticketTypeRepo
}
func (m *mockAdminTxReposForHandler) SeatLayoutRepository() domain.SeatLayoutRepository {
	return m.seatLayoutRepo
}
func (m *mockAdminTxReposForHandler) SeatSectionRepository() domain.SeatSectionRepository {
	return m.seatSectionRepo
}

type mockAdminBookingRepoForHandler struct {
	domain.BookingRepository
	confirmedCount int64
}

func (m *mockAdminBookingRepoForHandler) CountConfirmedByEventID(ctx context.Context, id uuid.UUID) (int64, error) {
	return m.confirmedCount, nil
}
func (m *mockAdminBookingRepoForHandler) CountByEventID(ctx context.Context, id uuid.UUID) (int64, error) {
	return m.confirmedCount, nil
}
func (m *mockAdminBookingRepoForHandler) GetBookingStatsByEventID(ctx context.Context, id uuid.UUID) (domain.AdminEventBookingStats, error) {
	return domain.AdminEventBookingStats{}, nil
}

type mockAdminTicketRepoForHandler struct {
	domain.TicketRepository
}

func (m *mockAdminTicketRepoForHandler) CountByEventID(ctx context.Context, id uuid.UUID) (int64, error) {
	return 0, nil
}

type mockAdminResRepoForHandler struct {
	domain.SeatReservationRepository
	activeCount int64
}

func (m *mockAdminResRepoForHandler) CountActiveByEventID(ctx context.Context, id uuid.UUID, now time.Time) (int64, error) {
	return m.activeCount, nil
}

type mockAdminScheduleRepoForHandler struct {
	domain.EventScheduleRepository
	schedules map[uuid.UUID]*domain.EventSchedule
}

func (m *mockAdminScheduleRepoForHandler) FindByEventID(id uuid.UUID) (*domain.EventSchedule, error) {
	return m.schedules[id], nil
}
func (m *mockAdminScheduleRepoForHandler) DeleteByEventID(id uuid.UUID) error {
	delete(m.schedules, id)
	return nil
}

type mockAdminSettingRepoForHandler struct {
	domain.EventSettingRepository
	settings map[uuid.UUID]*domain.EventSetting
}

func (m *mockAdminSettingRepoForHandler) FindByEventID(id uuid.UUID) (*domain.EventSetting, error) {
	return m.settings[id], nil
}
func (m *mockAdminSettingRepoForHandler) DeleteByEventID(id uuid.UUID) error {
	delete(m.settings, id)
	return nil
}

type mockAdminCancellationRepoForHandler struct {
	domain.EventCancellationRepository
	cancellations map[uuid.UUID]*domain.EventCancellation
}

func (m *mockAdminCancellationRepoForHandler) FindByEventID(id uuid.UUID) (*domain.EventCancellation, error) {
	return m.cancellations[id], nil
}
func (m *mockAdminCancellationRepoForHandler) DeleteByEventID(id uuid.UUID) error {
	delete(m.cancellations, id)
	return nil
}

type mockAdminContactRepoForHandler struct {
	domain.EventContactRepository
	contacts map[uuid.UUID]*domain.EventContact
}

func (m *mockAdminContactRepoForHandler) FindByEventID(id uuid.UUID) (*domain.EventContact, error) {
	return m.contacts[id], nil
}
func (m *mockAdminContactRepoForHandler) DeleteByEventID(id uuid.UUID) error {
	delete(m.contacts, id)
	return nil
}

type mockAdminTicketTypeRepoForHandler struct {
	domain.TicketTypeRepository
}

func (m *mockAdminTicketTypeRepoForHandler) FindByEventID(ctx context.Context, id uuid.UUID) ([]*domain.TicketType, error) {
	return nil, nil
}
func (m *mockAdminTicketTypeRepoForHandler) DeleteByEventID(ctx context.Context, id uuid.UUID) error {
	return nil
}

type mockAdminSeatLayoutRepoForHandler struct {
	domain.SeatLayoutRepository
}

func (m *mockAdminSeatLayoutRepoForHandler) FindByEventID(id uuid.UUID) (*domain.SeatLayout, error) {
	return nil, domain.ErrSeatLayoutNotFound
}
func (m *mockAdminSeatLayoutRepoForHandler) DeleteByEventID(id uuid.UUID) error {
	return nil
}

type mockAdminOrgRepoForHandler struct {
	domain.OrganizerRepository
}

func (m *mockAdminOrgRepoForHandler) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	return &domain.Organizer{ID: id, UserID: uuid.New(), Status: "ACTIVE"}, nil
}

type mockAdminProfileRepoForHandler struct {
	domain.OrganizerProfileRepository
}

func (m *mockAdminProfileRepoForHandler) FindByOrganizerID(id uuid.UUID) (*domain.OrganizerProfile, error) {
	return &domain.OrganizerProfile{OrganizerID: id, BusinessName: "Test Org"}, nil
}

type mockAdminUserRepoForHandler struct {
	domain.UserRepository
}

func (m *mockAdminUserRepoForHandler) FindByID(id uuid.UUID) (*domain.User, error) {
	return &domain.User{ID: id, Email: "test@org.com"}, nil
}

type mockAdminCatRepoForHandler struct {
	domain.CategoryRepository
}

func (m *mockAdminCatRepoForHandler) FindByID(id uuid.UUID) (*domain.Category, error) {
	return &domain.Category{ID: id, Name: "General", Status: domain.CategoryStatusActive}, nil
}

func setupAdminEventTestRouter(
	eventRepo *mockAdminEventRepoForHandlerTest,
	bookingRepo *mockAdminBookingRepoForHandler,
	resRepo *mockAdminResRepoForHandler,
	jwtService *token.JWTService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	txRepos := &mockAdminTxReposForHandler{
		eventRepo:        eventRepo,
		bookingRepo:      bookingRepo,
		reservationRepo:  resRepo,
		organizerRepo:    &mockAdminOrgRepoForHandler{},
		profileRepo:      &mockAdminProfileRepoForHandler{},
		userRepo:         &mockAdminUserRepoForHandler{},
		categoryRepo:     &mockAdminCatRepoForHandler{},
		scheduleRepo:     &mockAdminScheduleRepoForHandler{schedules: map[uuid.UUID]*domain.EventSchedule{}},
		settingRepo:      &mockAdminSettingRepoForHandler{settings: map[uuid.UUID]*domain.EventSetting{}},
		cancellationRepo: &mockAdminCancellationRepoForHandler{cancellations: map[uuid.UUID]*domain.EventCancellation{}},
		contactRepo:      &mockAdminContactRepoForHandler{contacts: map[uuid.UUID]*domain.EventContact{}},
		ticketTypeRepo:   &mockAdminTicketTypeRepoForHandler{},
		ticketRepo:       &mockAdminTicketRepoForHandler{},
		seatLayoutRepo:   &mockAdminSeatLayoutRepoForHandler{},
	}
	txMgr := &mockAdminTxManagerForHandler{txRepos: txRepos}

	adminEventUsecase := events.NewAdminEventUsecase(txMgr, eventRepo, nil, logger)
	adminEventHandler := handler.NewAdminEventHandler(adminEventUsecase, logger)

	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(
		middleware.Auth(jwtService),
		middleware.RequireRole("ADMIN"),
	)
	admin.GET("/events", adminEventHandler.ListEvents)
	admin.GET("/events/:id", adminEventHandler.GetEventDetails)
	admin.PUT("/events/:id", adminEventHandler.UpdateEvent)
	admin.PATCH("/events/:id/publish", adminEventHandler.PublishEvent)
	admin.DELETE("/events/:id", adminEventHandler.DeleteEvent)

	return router
}

func TestAdminEventHandler_ListEvents(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	require.NoError(t, err)

	customerToken, err := jwtService.GenerateAccessToken(uuid.New(), "CUSTOMER")
	require.NoError(t, err)

	eventID := uuid.New()
	eventRepo := &mockAdminEventRepoForHandlerTest{
		listResult: []*domain.AdminEventListItem{
			{
				ID:                    eventID,
				Title:                 "Global Developer Conference",
				OrganizerBusinessName: "DevCorp",
				Status:                "PUBLISHED",
			},
		},
		statusCounts: domain.AdminEventStatusCounts{
			Total:     15,
			Published: 10,
			Draft:     4,
			Cancelled: 1,
		},
	}
	bookingRepo := &mockAdminBookingRepoForHandler{}
	resRepo := &mockAdminResRepoForHandler{}

	router := setupAdminEventTestRouter(eventRepo, bookingRepo, resRepo, jwtService)

	t.Run("authorized admin successfully retrieves events list", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/events?page=1&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				Events       []domain.AdminEventListItem   `json:"events"`
				Total        int64                         `json:"total"`
				StatusCounts domain.AdminEventStatusCounts `json:"status_counts"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.True(t, res.Success)
		assert.Len(t, res.Data.Events, 1)
		assert.Equal(t, "Global Developer Conference", res.Data.Events[0].Title)
		assert.Equal(t, int64(15), res.Data.StatusCounts.Total)
	})

	t.Run("non-admin customer is rejected with 403 Forbidden", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/events", nil)
		req.Header.Set("Authorization", "Bearer "+customerToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestAdminEventHandler_DeleteEvent(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	require.NoError(t, err)

	eventID := uuid.New()
	eventRepo := &mockAdminEventRepoForHandlerTest{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:     eventID,
				Title:  "Deletable Event",
				Status: domain.EventStatusDraft,
			},
		},
	}

	t.Run("zero bookings allows successful deletion", func(t *testing.T) {
		bookingRepo := &mockAdminBookingRepoForHandler{confirmedCount: 0}
		resRepo := &mockAdminResRepoForHandler{activeCount: 0}
		router := setupAdminEventTestRouter(eventRepo, bookingRepo, resRepo, jwtService)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/events/"+eventID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, eventRepo.deletedIDs, eventID)
	})

	t.Run("confirmed bookings returns 409 Conflict", func(t *testing.T) {
		bookedEventID := uuid.New()
		eventRepo.events[bookedEventID] = &domain.Event{
			ID:     bookedEventID,
			Title:  "Booked Event",
			Status: domain.EventStatusPublished,
		}
		bookingRepo := &mockAdminBookingRepoForHandler{confirmedCount: 2}
		resRepo := &mockAdminResRepoForHandler{activeCount: 0}
		router := setupAdminEventTestRouter(eventRepo, bookingRepo, resRepo, jwtService)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/events/"+bookedEventID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errRes struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &errRes)
		require.NoError(t, err)
		assert.Equal(t, "CONFLICT", errRes.Code)
		assert.Contains(t, errRes.Message, "cannot be deleted because it has existing bookings, tickets, or active reservations")
	})
}

func TestAdminEventHandler_UpdateEvent(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	require.NoError(t, err)

	publishedEventID := uuid.New()
	eventRepo := &mockAdminEventRepoForHandlerTest{
		events: map[uuid.UUID]*domain.Event{
			publishedEventID: {
				ID:          publishedEventID,
				Title:       "Original Title",
				Description: "Original Description",
				Status:      domain.EventStatusPublished,
			},
		},
	}
	bookingRepo := &mockAdminBookingRepoForHandler{}
	resRepo := &mockAdminResRepoForHandler{}
	router := setupAdminEventTestRouter(eventRepo, bookingRepo, resRepo, jwtService)

	t.Run("updating descriptive metadata on published event succeeds", func(t *testing.T) {
		body := map[string]interface{}{
			"title":       "Updated Descriptive Title",
			"description": "Updated Descriptive Description",
		}
		bodyBytes, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/events/"+publishedEventID.String(), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("attempting structural change on published event returns 400 Bad Request", func(t *testing.T) {
		futureDate := time.Now().Add(48 * time.Hour)
		body := map[string]interface{}{
			"event_date": futureDate,
		}
		bodyBytes, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/events/"+publishedEventID.String(), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
