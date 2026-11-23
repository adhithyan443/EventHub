package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOrgTxManager struct {
	repos *mockOrgTxRepos
}

func (m *mockOrgTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

type mockOrgTxRepos struct {
	domain.TransactionRepositories
	eventRepo        *mockOrgEventRepo
	cancellationRepo *mockOrgCancellationRepo
	organizerRepo    *mockOrgOrganizerRepo
	bookingRepo      *mockOrgBookingRepo
	ticketRepo       *mockOrgTicketRepo
	refundRepo       *mockOrgRefundRepo
}

func (m *mockOrgTxRepos) EventRepository() domain.EventRepository { return m.eventRepo }
func (m *mockOrgTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return m.cancellationRepo
}
func (m *mockOrgTxRepos) OrganizerRepository() domain.OrganizerRepository { return m.organizerRepo }
func (m *mockOrgTxRepos) BookingRepository() domain.BookingRepository     { return m.bookingRepo }
func (m *mockOrgTxRepos) TicketRepository() domain.TicketRepository       { return m.ticketRepo }
func (m *mockOrgTxRepos) RefundRepository() domain.RefundRepository       { return m.refundRepo }
func (m *mockOrgTxRepos) SettlementRepository() domain.SettlementRepository {
	return &mockOrgSettlementRepo{}
}
func (m *mockOrgTxRepos) EventScheduleRepository() domain.EventScheduleRepository {
	return &mockOrgScheduleRepo{}
}
func (m *mockOrgTxRepos) SeatReservationRepository() domain.SeatReservationRepository {
	return &mockOrgReservationRepo{}
}
func (m *mockOrgTxRepos) CheckoutRepository() domain.CheckoutRepository {
	return nil
}
func (m *mockOrgTxRepos) SeatRepository() domain.SeatRepository {
	return nil
}
func (m *mockOrgTxRepos) TicketTypeRepository() domain.TicketTypeRepository {
	return nil
}
func (m *mockOrgTxRepos) UserRepository() domain.UserRepository {
	return nil
}
func (m *mockOrgTxRepos) PaymentRepository() domain.PaymentRepository {
	return nil
}

type mockOrgReservationRepo struct {
	domain.SeatReservationRepository
}

func (m *mockOrgReservationRepo) FindActiveByEventID(ctx context.Context, id uuid.UUID) ([]domain.SeatReservation, error) {
	return nil, nil
}

type mockOrgSettlementRepo struct {
	domain.SettlementRepository
}

func (m *mockOrgSettlementRepo) List(ctx context.Context, filter domain.SettlementFilter) (*domain.SettlementListResult, error) {
	return &domain.SettlementListResult{Settlements: nil, Total: 0}, nil
}
func (m *mockOrgSettlementRepo) Update(ctx context.Context, s *domain.Settlement) error {
	return nil
}

type mockOrgScheduleRepo struct {
	domain.EventScheduleRepository
}

func (m *mockOrgScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	return nil, domain.ErrEventNotFound
}

type mockOrgEventRepo struct {
	domain.EventRepository
	mu     sync.Mutex
	events map[uuid.UUID]*domain.Event
}

func (m *mockOrgEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ev, ok := m.events[id]; ok {
		copied := *ev
		return &copied, nil
	}
	return nil, domain.ErrEventNotFound
}
func (m *mockOrgEventRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) {
	return m.FindByID(id)
}
func (m *mockOrgEventRepo) Update(ev *domain.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.events[ev.ID]; !ok {
		return domain.ErrEventNotFound
	}
	copied := *ev
	m.events[ev.ID] = &copied
	return nil
}

type mockOrgCancellationRepo struct {
	domain.EventCancellationRepository
	cancellations map[uuid.UUID]*domain.EventCancellation
}

func (m *mockOrgCancellationRepo) FindByEventID(id uuid.UUID) (*domain.EventCancellation, error) {
	if c, ok := m.cancellations[id]; ok {
		return c, nil
	}
	return nil, domain.ErrEventNotFound
}
func (m *mockOrgCancellationRepo) Create(c *domain.EventCancellation) error {
	m.cancellations[c.EventID] = c
	return nil
}
func (m *mockOrgCancellationRepo) Update(c *domain.EventCancellation) error {
	m.cancellations[c.EventID] = c
	return nil
}

type mockOrgOrganizerRepo struct {
	domain.OrganizerRepository
	organizers map[uuid.UUID]*domain.Organizer
}

func (m *mockOrgOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	for _, o := range m.organizers {
		if o.UserID == userID {
			return o, nil
		}
	}
	return nil, domain.ErrOrganizerNotFound
}

func (m *mockOrgOrganizerRepo) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	for _, o := range m.organizers {
		if o.ID == id {
			return o, nil
		}
	}
	return nil, domain.ErrOrganizerNotFound
}

type mockOrgBookingRepo struct {
	domain.BookingRepository
}

func (m *mockOrgBookingRepo) FindByEventID(ctx context.Context, id uuid.UUID) ([]domain.Booking, error) {
	return nil, nil
}

type mockOrgTicketRepo struct {
	domain.TicketRepository
}

func (m *mockOrgTicketRepo) FindByEventID(ctx context.Context, id uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}

type mockOrgRefundRepo struct {
	domain.RefundRepository
}

func (m *mockOrgRefundRepo) FindByEventID(ctx context.Context, id uuid.UUID) ([]domain.Refund, error) {
	return nil, nil
}

func setupOrganizerEventTestRouter(
	eventRepo *mockOrgEventRepo,
	cancellationRepo *mockOrgCancellationRepo,
	organizerRepo *mockOrgOrganizerRepo,
) (*gin.Engine, *events.EventUsecase) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	txRepos := &mockOrgTxRepos{
		eventRepo:        eventRepo,
		cancellationRepo: cancellationRepo,
		organizerRepo:    organizerRepo,
		bookingRepo:      &mockOrgBookingRepo{},
		ticketRepo:       &mockOrgTicketRepo{},
		refundRepo:       &mockOrgRefundRepo{},
	}
	txMgr := &mockOrgTxManager{repos: txRepos}

	uc := events.NewEventUsecase(txMgr, nil, logger)
	h := handler.NewEventHandler(uc, logger)

	api := router.Group("/api/v1/organizers")
	// Simulate auth middleware setting user_id
	api.Use(func(c *gin.Context) {
		uidStr := c.GetHeader("X-User-ID")
		if uidStr != "" {
			uid, err := uuid.Parse(uidStr)
			if err == nil {
				c.Set("user_id", uid)
			}
		}
		c.Next()
	})

	api.POST("/events/:id/cancel", h.CancelEvent)
	api.PATCH("/events/:id/cancel", h.CancelEvent)
	api.GET("/events/:id/cancellation", h.GetEventCancellationStatus)
	api.GET("/events/:id/cancellation-preview", h.GetCancellationPreview)

	return router, uc
}

func TestEventHandler_OrganizerEventCancellation(t *testing.T) {
	ownerUserID := uuid.New()
	ownerOrgID := uuid.New()
	eventID := uuid.New()

	otherUserID := uuid.New()
	otherOrgID := uuid.New()

	eventRepo := &mockOrgEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:          eventID,
				OrganizerID: ownerOrgID,
				Title:       "Organizer Showcase",
				Status:      domain.EventStatusPublished,
			},
		},
	}
	cancellationRepo := &mockOrgCancellationRepo{
		cancellations: make(map[uuid.UUID]*domain.EventCancellation),
	}
	organizerRepo := &mockOrgOrganizerRepo{
		organizers: map[uuid.UUID]*domain.Organizer{
			ownerOrgID: {ID: ownerOrgID, UserID: ownerUserID, Status: "ACTIVE"},
			otherOrgID: {ID: otherOrgID, UserID: otherUserID, Status: "ACTIVE"},
		},
	}

	router, _ := setupOrganizerEventTestRouter(eventRepo, cancellationRepo, organizerRepo)

	t.Run("owner organizer cancels event successfully", func(t *testing.T) {
		body := map[string]interface{}{
			"reason": "Family emergency for keynote speaker",
		}
		bodyBytes, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/organizers/events/"+eventID.String()+"/cancel", bytes.NewBuffer(bodyBytes))
		req.Header.Set("X-User-ID", ownerUserID.String())
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
		assert.Equal(t, "event cancelled successfully", resp["message"])
	})

	t.Run("different organizer is forbidden from cancelling", func(t *testing.T) {
		bodyBytes, _ := json.Marshal(map[string]interface{}{"reason": "Malicious attempt"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/organizers/events/"+eventID.String()+"/cancel", bytes.NewBuffer(bodyBytes))
		req.Header.Set("X-User-ID", otherUserID.String())
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("cancellation fails with 400 when reason is missing", func(t *testing.T) {
		bodyBytes, _ := json.Marshal(map[string]interface{}{"reason": "   "})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/organizers/events/"+eventID.String()+"/cancel", bytes.NewBuffer(bodyBytes))
		req.Header.Set("X-User-ID", ownerUserID.String())
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "cancellation reason is required", resp["message"])
	})

	t.Run("cancellation fails with 409 when event is already cancelled", func(t *testing.T) {
		// Event was cancelled in the first subtest
		bodyBytes, _ := json.Marshal(map[string]interface{}{"reason": "Another cancellation attempt"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/organizers/events/"+eventID.String()+"/cancel", bytes.NewBuffer(bodyBytes))
		req.Header.Set("X-User-ID", ownerUserID.String())
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "event is already cancelled", resp["message"])
	})

	t.Run("owner organizer gets cancellation status successfully", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/organizers/events/"+eventID.String()+"/cancellation", nil)
		req.Header.Set("X-User-ID", ownerUserID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("different organizer is forbidden from viewing cancellation status", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/organizers/events/"+eventID.String()+"/cancellation", nil)
		req.Header.Set("X-User-ID", otherUserID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("owner organizer gets cancellation preview successfully", func(t *testing.T) {
		previewEventID := uuid.New()
		eventRepo.events[previewEventID] = &domain.Event{
			ID:          previewEventID,
			OrganizerID: ownerOrgID,
			Title:       "Live Tech Showcase",
			Status:      domain.EventStatusPublished,
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/organizers/events/"+previewEventID.String()+"/cancellation-preview", nil)
		req.Header.Set("X-User-ID", ownerUserID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
		data := resp["data"].(map[string]interface{})
		eligibility := data["eligibility"].(map[string]interface{})
		assert.True(t, eligibility["is_cancellable"].(bool))
	})

	t.Run("already cancelled event preview returns 200 OK with is_cancellable false", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/organizers/events/"+eventID.String()+"/cancellation-preview", nil)
		req.Header.Set("X-User-ID", ownerUserID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
		data := resp["data"].(map[string]interface{})
		eligibility := data["eligibility"].(map[string]interface{})
		assert.False(t, eligibility["is_cancellable"].(bool))
		reasons := eligibility["blocking_reasons"].([]interface{})
		assert.Contains(t, reasons, "event is already cancelled")
	})

	t.Run("draft event returns 200 OK with is_cancellable false in preview", func(t *testing.T) {
		draftEventID := uuid.New()
		eventRepo.events[draftEventID] = &domain.Event{
			ID:          draftEventID,
			OrganizerID: ownerOrgID,
			Title:       "Draft Summit",
			Status:      domain.EventStatusDraft,
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/organizers/events/"+draftEventID.String()+"/cancellation-preview", nil)
		req.Header.Set("X-User-ID", ownerUserID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
		data := resp["data"].(map[string]interface{})
		eligibility := data["eligibility"].(map[string]interface{})
		assert.False(t, eligibility["is_cancellable"].(bool))
		reasons := eligibility["blocking_reasons"].([]interface{})
		assert.NotEmpty(t, reasons)
	})

	t.Run("different organizer is forbidden from viewing preview", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/organizers/events/"+eventID.String()+"/cancellation-preview", nil)
		req.Header.Set("X-User-ID", otherUserID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid UUID in preview returns 400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/organizers/events/not-a-uuid/cancellation-preview", nil)
		req.Header.Set("X-User-ID", ownerUserID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("nonexistent event in preview returns 404", func(t *testing.T) {
		missingID := uuid.New()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/organizers/events/"+missingID.String()+"/cancellation-preview", nil)
		req.Header.Set("X-User-ID", ownerUserID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
