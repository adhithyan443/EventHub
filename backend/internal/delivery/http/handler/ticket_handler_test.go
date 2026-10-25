package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Repositories for Handler Testing ---

type stubTicketRepo struct {
	ticket *domain.Ticket
	err    error
}

func (s *stubTicketRepo) Create(ctx context.Context, t *domain.Ticket) error       { return s.err }
func (s *stubTicketRepo) CreateMany(ctx context.Context, ts []domain.Ticket) error { return s.err }
func (s *stubTicketRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.ticket != nil && s.ticket.ID == id {
		return s.ticket, nil
	}
	return nil, domain.ErrTicketNotFound
}
func (s *stubTicketRepo) FindByCode(ctx context.Context, code string) (*domain.Ticket, error) {
	return nil, nil
}
func (s *stubTicketRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (s *stubTicketRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (s *stubTicketRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return s.err
}

type stubEventRepo struct {
	event *domain.Event
	err   error
}

func (s *stubEventRepo) Create(event *domain.Event) error { return s.err }
func (s *stubEventRepo) Update(event *domain.Event) error { return s.err }
func (s *stubEventRepo) Delete(id uuid.UUID) error        { return s.err }
func (s *stubEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.event != nil && s.event.ID == id {
		return s.event, nil
	}
	return nil, domain.ErrEventNotFound
}
func (s *stubEventRepo) FindByOrganizerID(organizerID uuid.UUID, page int, limit int, status string, search string) ([]*domain.OrganizerEvent, int64, error) {
	return nil, 0, nil
}
func (s *stubEventRepo) FindPublicEvents(filter domain.PublicEventFilter) ([]*domain.PublicEvent, int64, error) {
	return nil, 0, nil
}

type stubScheduleRepo struct {
	schedule *domain.EventSchedule
	err      error
}

func (s *stubScheduleRepo) Create(schedule *domain.EventSchedule) error { return s.err }
func (s *stubScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.schedule != nil && s.schedule.EventID == eventID {
		return s.schedule, nil
	}
	return nil, domain.ErrEventNotFound
}
func (s *stubScheduleRepo) Update(schedule *domain.EventSchedule) error { return s.err }
func (s *stubScheduleRepo) DeleteByEventID(eventID uuid.UUID) error     { return s.err }

type stubUserRepo struct {
	user *domain.User
	err  error
}

func (s *stubUserRepo) Create(user *domain.User) error { return s.err }
func (s *stubUserRepo) FindByID(id uuid.UUID) (*domain.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user != nil && s.user.ID == id {
		return s.user, nil
	}
	return nil, domain.ErrUserNotFound
}
func (s *stubUserRepo) FindByEmail(email string) (*domain.User, error) { return nil, nil }
func (s *stubUserRepo) FindByPhone(phone string) (*domain.User, error) { return nil, nil }
func (s *stubUserRepo) Update(user *domain.User) error                 { return s.err }

type stubVenueRepo struct {
	venue *domain.Venue
	err   error
}

func (s *stubVenueRepo) Create(venue *domain.Venue) error { return s.err }
func (s *stubVenueRepo) FindByID(id uuid.UUID) (*domain.Venue, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.venue != nil && s.venue.ID == id {
		return s.venue, nil
	}
	return nil, domain.ErrVenueNotFound
}
func (s *stubVenueRepo) FindByGooglePlaceID(placeID string) (*domain.Venue, error) { return nil, nil }

type stubTicketTypeRepo struct {
	ticketType *domain.TicketType
	err        error
}

func (s *stubTicketTypeRepo) Create(ctx context.Context, ticketType *domain.TicketType) error {
	return s.err
}
func (s *stubTicketTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.ticketType != nil && s.ticketType.ID == id {
		return s.ticketType, nil
	}
	return nil, domain.ErrTicketTypeNotFound
}
func (s *stubTicketTypeRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.TicketType, error) {
	return nil, nil
}
func (s *stubTicketTypeRepo) Update(ctx context.Context, ticketType *domain.TicketType) error {
	return s.err
}
func (s *stubTicketTypeRepo) Delete(ctx context.Context, id uuid.UUID) error { return s.err }
func (s *stubTicketTypeRepo) DeleteByEventID(ctx context.Context, eventID uuid.UUID) error {
	return s.err
}
func (s *stubTicketTypeRepo) ReserveQuantityIfAvailable(ctx context.Context, ticketTypeID uuid.UUID, quantity int) error {
	return s.err
}
func (s *stubTicketTypeRepo) ReleaseReservedQuantity(ctx context.Context, ticketTypeID uuid.UUID, quantity int) error {
	return s.err
}

// --- Handler Test Fixture ---

type handlerFixture struct {
	ticketRepo   *stubTicketRepo
	eventRepo    *stubEventRepo
	scheduleRepo *stubScheduleRepo
	userRepo     *stubUserRepo
	venueRepo    *stubVenueRepo
	typeRepo     *stubTicketTypeRepo
	handler      *handler.TicketHandler
	router       *gin.Engine
	userID       uuid.UUID
	ticketID     uuid.UUID
	eventID      uuid.UUID
	venueID      uuid.UUID
	ticketTypeID uuid.UUID
}

func setupHandlerFixture(t *testing.T) *handlerFixture {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	ticketID := uuid.New()
	eventID := uuid.New()
	venueID := uuid.New()
	ticketTypeID := uuid.New()

	ticketModel := &domain.Ticket{
		ID:              ticketID,
		TicketCode:      "EH-TCK-999",
		BookingID:       uuid.New(),
		UserID:          userID,
		EventID:         eventID,
		TicketTypeID:    &ticketTypeID,
		AdmissionNumber: 7,
		Price:           decimal.NewFromInt(200),
		Status:          domain.TicketStatusValid,
		IssuedAt:        time.Now(),
	}

	eventModel := &domain.Event{
		ID:        eventID,
		Title:     "Global Tech Summit",
		EventType: domain.EventTypePhysical,
		VenueID:   &venueID,
	}

	eventDate := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	scheduleModel := &domain.EventSchedule{
		ID:        uuid.New(),
		EventID:   eventID,
		EventDate: eventDate,
		StartTime: startTime,
	}

	userModel := &domain.User{
		ID:       userID,
		FullName: "Alex Mercer",
		Email:    "alex@example.com",
	}

	venueModel := &domain.Venue{
		ID:         venueID,
		Name:       "Silicon Convention Center",
		Address:    "500 Innovation Blvd",
		City:       "San Jose",
		State:      "CA",
		PostalCode: "95110",
	}

	ticketTypeModel := &domain.TicketType{
		ID:      ticketTypeID,
		EventID: eventID,
		Name:    "Conference Delegate",
		Price:   decimal.NewFromInt(200),
	}

	ticketRepo := &stubTicketRepo{ticket: ticketModel}
	eventRepo := &stubEventRepo{event: eventModel}
	scheduleRepo := &stubScheduleRepo{schedule: scheduleModel}
	userRepo := &stubUserRepo{user: userModel}
	venueRepo := &stubVenueRepo{venue: venueModel}
	typeRepo := &stubTicketTypeRepo{ticketType: ticketTypeModel}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := ticket.NewTicketUsecase(ticketRepo, eventRepo, scheduleRepo, userRepo, venueRepo, typeRepo, logger)

	qrService, err := ticket.NewQRService("very-secure-signing-secret-at-least-32-chars")
	require.NoError(t, err)

	th := handler.NewTicketHandler(uc, qrService)

	router := gin.New()

	return &handlerFixture{
		ticketRepo:   ticketRepo,
		eventRepo:    eventRepo,
		scheduleRepo: scheduleRepo,
		userRepo:     userRepo,
		venueRepo:    venueRepo,
		typeRepo:     typeRepo,
		handler:      th,
		router:       router,
		userID:       userID,
		ticketID:     ticketID,
		eventID:      eventID,
		venueID:      venueID,
		ticketTypeID: ticketTypeID,
	}
}

// --- Handler Test Cases ---

func TestDownloadTicketPDF_Success(t *testing.T) {
	f := setupHandlerFixture(t)

	f.router.GET("/api/v1/tickets/:id/download", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.DownloadTicketPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+f.ticketID.String()+"/download", nil)
	rec := httptest.NewRecorder()

	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment; filename=")
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "global-tech-summit")
	assert.Contains(t, rec.Header().Get("Content-Disposition"), f.ticketID.String()+".pdf")
	assert.Equal(t, "private, no-store", rec.Header().Get("Cache-Control"))
	assert.NotEmpty(t, rec.Header().Get("Content-Length"))

	body := rec.Body.Bytes()
	require.True(t, len(body) > 100)
	// PDF files start with the magic header %PDF-
	assert.True(t, bytes.HasPrefix(body, []byte("%PDF-")), "Response body must be a valid PDF file")
}

func TestDownloadTicketPDF_Success_AltContextKey_userID(t *testing.T) {
	f := setupHandlerFixture(t)

	f.router.GET("/api/v1/tickets/:id/download", func(c *gin.Context) {
		// Testing backward compatibility with userID context key
		c.Set("userID", f.userID)
		f.handler.DownloadTicketPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+f.ticketID.String()+"/download", nil)
	rec := httptest.NewRecorder()

	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.True(t, bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")))
}

func TestDownloadTicketPDF_Unauthorized(t *testing.T) {
	f := setupHandlerFixture(t)

	f.router.GET("/api/v1/tickets/:id/download", func(c *gin.Context) {
		// No user_id set in context
		f.handler.DownloadTicketPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+f.ticketID.String()+"/download", nil)
	rec := httptest.NewRecorder()

	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", resp["error"])
}

func TestDownloadTicketPDF_InvalidTicketID(t *testing.T) {
	f := setupHandlerFixture(t)

	f.router.GET("/api/v1/tickets/:id/download", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.DownloadTicketPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/not-a-uuid/download", nil)
	rec := httptest.NewRecorder()

	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid ticket ID", resp["error"])
}

func TestDownloadTicketPDF_TicketNotFound(t *testing.T) {
	f := setupHandlerFixture(t)

	f.router.GET("/api/v1/tickets/:id/download", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.DownloadTicketPDF(c)
	})

	// Request a non-existent ticket ID
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+uuid.New().String()+"/download", nil)
	rec := httptest.NewRecorder()

	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ticket not found", resp["error"])
}

func TestDownloadTicketPDF_CrossCustomerAccess_Returns404(t *testing.T) {
	f := setupHandlerFixture(t)

	otherCustomer := uuid.New()
	f.router.GET("/api/v1/tickets/:id/download", func(c *gin.Context) {
		// Logged in as other customer trying to access f.ticketID
		c.Set("user_id", otherCustomer)
		f.handler.DownloadTicketPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+f.ticketID.String()+"/download", nil)
	rec := httptest.NewRecorder()

	f.router.ServeHTTP(rec, req)

	// Must fail closed with 404 without leaking whether ticket exists
	assert.Equal(t, http.StatusNotFound, rec.Code)
	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ticket not found", resp["error"])
}

func TestDownloadTicketPDF_EventNotFound_Returns404(t *testing.T) {
	f := setupHandlerFixture(t)

	// Make event repo return ErrEventNotFound
	f.eventRepo.event = nil

	f.router.GET("/api/v1/tickets/:id/download", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.DownloadTicketPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+f.ticketID.String()+"/download", nil)
	rec := httptest.NewRecorder()

	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "event not found", resp["error"])
}

func TestDownloadTicketPDF_UnexpectedError_Returns500WithoutInternalDetails(t *testing.T) {
	f := setupHandlerFixture(t)

	// Simulate unexpected internal database error
	f.ticketRepo.err = errors.New("critical postgres connection timeout")

	f.router.GET("/api/v1/tickets/:id/download", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.DownloadTicketPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+f.ticketID.String()+"/download", nil)
	rec := httptest.NewRecorder()

	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	// Internal database details must NOT be returned in response body
	assert.Equal(t, "failed to retrieve ticket details", resp["error"])
	assert.NotContains(t, rec.Body.String(), "critical postgres connection timeout")
}

func TestDownloadTicketPDF_MissingOptionalVenueOrTicketType(t *testing.T) {
	f := setupHandlerFixture(t)

	// Physical event with no venue configured and general admission ticket
	f.eventRepo.event.VenueID = nil
	f.ticketRepo.ticket.TicketTypeID = nil
	f.scheduleRepo.schedule = nil

	f.router.GET("/api/v1/tickets/:id/download", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.DownloadTicketPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+f.ticketID.String()+"/download", nil)
	rec := httptest.NewRecorder()

	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.True(t, bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")))
}
