package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
)

// --- Stub Repositories for Handler Testing ---

type stubBookingHandlerRepo struct {
	domain.BookingRepository
	findByUserFn    func(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error)
	findWithItemsFn func(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error)
}

func (s *stubBookingHandlerRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	if s.findByUserFn != nil {
		return s.findByUserFn(ctx, userID)
	}
	return nil, nil
}

func (s *stubBookingHandlerRepo) FindWithItemsByID(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
	if s.findWithItemsFn != nil {
		return s.findWithItemsFn(ctx, id)
	}
	return nil, nil, domain.ErrBookingNotFound
}

func (s *stubBookingHandlerRepo) FindItemsByBookingIDs(ctx context.Context, bookingIDs []uuid.UUID) (map[uuid.UUID][]domain.BookingItem, error) {
	return make(map[uuid.UUID][]domain.BookingItem), nil
}

type stubSeatHandlerRepo struct {
	domain.SeatRepository
}

func (s *stubSeatHandlerRepo) FindDetailsBySeatIDs(ctx context.Context, seatIDs []uuid.UUID) (map[uuid.UUID]domain.SeatDetail, error) {
	return make(map[uuid.UUID]domain.SeatDetail), nil
}

type stubUserHandlerRepo struct {
	domain.UserRepository
	user *domain.User
}

func (s *stubUserHandlerRepo) FindByID(id uuid.UUID) (*domain.User, error) {
	if s.user != nil {
		return s.user, nil
	}
	return &domain.User{ID: id, FullName: "Test Customer"}, nil
}

type stubEventHandlerRepo struct {
	domain.EventRepository
	event *domain.Event
}

func (s *stubEventHandlerRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if s.event != nil {
		return s.event, nil
	}
	return &domain.Event{ID: id, Title: "Test Festival", EventType: "ONLINE"}, nil
}

type stubScheduleHandlerRepo struct {
	domain.EventScheduleRepository
}

func (s *stubScheduleHandlerRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	return &domain.EventSchedule{EventID: eventID, EventDate: time.Now().Add(24 * time.Hour)}, nil
}

type stubVenueHandlerRepo struct {
	domain.VenueRepository
}

func (s *stubVenueHandlerRepo) FindByID(id uuid.UUID) (*domain.Venue, error) {
	return &domain.Venue{ID: id, Name: "Main Arena"}, nil
}

type stubTicketHandlerRepo struct {
	domain.TicketRepository
}

func (s *stubTicketHandlerRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}

type stubTicketTypeHandlerRepo struct {
	domain.TicketTypeRepository
}

func (s *stubTicketTypeHandlerRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	return &domain.TicketType{ID: id, Name: "General Admission"}, nil
}

type stubTxRepositories struct {
	domain.TransactionRepositories
	bookingRepo domain.BookingRepository
}

func (s *stubTxRepositories) BookingRepository() domain.BookingRepository {
	return s.bookingRepo
}

func (s *stubTxRepositories) SeatRepository() domain.SeatRepository {
	return &stubSeatHandlerRepo{}
}

func (s *stubTxRepositories) UserRepository() domain.UserRepository {
	return &stubUserHandlerRepo{}
}

func (s *stubTxRepositories) EventRepository() domain.EventRepository {
	return &stubEventHandlerRepo{}
}

func (s *stubTxRepositories) EventScheduleRepository() domain.EventScheduleRepository {
	return &stubScheduleHandlerRepo{}
}

func (s *stubTxRepositories) VenueRepository() domain.VenueRepository {
	return &stubVenueHandlerRepo{}
}

func (s *stubTxRepositories) TicketRepository() domain.TicketRepository {
	return &stubTicketHandlerRepo{}
}

func (s *stubTxRepositories) TicketTypeRepository() domain.TicketTypeRepository {
	return &stubTicketTypeHandlerRepo{}
}

type stubTxManager struct {
	txRepos domain.TransactionRepositories
}

func (s *stubTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(s.txRepos)
}

// --- Handler Test Fixture ---

type bookingHandlerFixture struct {
	repo    *stubBookingHandlerRepo
	handler *handler.BookingHandler
	router  *gin.Engine
	userID  uuid.UUID
}

func setupBookingHandlerFixture(t *testing.T) *bookingHandlerFixture {
	gin.SetMode(gin.TestMode)

	repo := &stubBookingHandlerRepo{}
	txRepos := &stubTxRepositories{bookingRepo: repo}
	txManager := &stubTxManager{txRepos: txRepos}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := booking.NewBookingUsecase(txManager, logger)
	qrService, err := ticket.NewQRService("0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	bh := handler.NewBookingHandler(uc, qrService, logger)

	router := gin.New()
	userID := uuid.New()

	return &bookingHandlerFixture{
		repo:    repo,
		handler: bh,
		router:  router,
		userID:  userID,
	}
}

// --- Handler Test Cases ---

func TestGetMyBookings_Success(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	expectedBookings := []domain.Booking{
		{
			ID:            uuid.New(),
			BookingNumber: "BK-20261025-001",
			UserID:        f.userID,
			EventID:       uuid.New(),
			Status:        domain.BookingStatusConfirmed,
			Total:         decimal.NewFromFloat(150.00),
			Subtotal:      decimal.NewFromFloat(130.00),
			PlatformFee:   decimal.Zero,
			Tax:           decimal.NewFromFloat(20.00),
			Discount:      decimal.Zero,
			CreatedAt:     time.Now(),
		},
		{
			ID:            uuid.New(),
			BookingNumber: "BK-20261025-002",
			UserID:        f.userID,
			EventID:       uuid.New(),
			Status:        domain.BookingStatusPending,
			Total:         decimal.NewFromFloat(75.50),
			Subtotal:      decimal.NewFromFloat(65.50),
			PlatformFee:   decimal.Zero,
			Tax:           decimal.NewFromFloat(10.00),
			Discount:      decimal.Zero,
			CreatedAt:     time.Now().Add(-1 * time.Hour),
		},
	}

	f.repo.findByUserFn = func(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
		assert.Equal(t, f.userID, userID)
		return expectedBookings, nil
	}

	f.router.GET("/api/v1/bookings", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.GetMyBookings(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []domain.Booking `json:"data"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, expectedBookings[0].BookingNumber, resp.Data[0].BookingNumber)
	assert.Equal(t, expectedBookings[1].BookingNumber, resp.Data[1].BookingNumber)
}

func TestGetMyBookings_Empty(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	f.repo.findByUserFn = func(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
		return []domain.Booking{}, nil
	}

	f.router.GET("/api/v1/bookings", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.GetMyBookings(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"data":[]`)
}

func TestGetMyBookings_Unauthorized(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	f.router.GET("/api/v1/bookings", f.handler.GetMyBookings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized")
}

func TestGetMyBookings_InternalServerError(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	f.repo.findByUserFn = func(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
		return nil, errors.New("database connection pool exhausted")
	}

	f.router.GET("/api/v1/bookings", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.GetMyBookings(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "database connection pool")
	assert.Contains(t, rec.Body.String(), "failed to retrieve bookings")
}

func TestGetBookingByID_Success(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	bookingID := uuid.New()
	eventID := uuid.New()
	ticketTypeID := uuid.New()

	expectedBooking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "BK-20261025-100",
		UserID:        f.userID,
		EventID:       eventID,
		Status:        domain.BookingStatusConfirmed,
		Total:         decimal.NewFromFloat(200.00),
		Subtotal:      decimal.NewFromFloat(180.00),
		PlatformFee:   decimal.Zero,
		Tax:           decimal.NewFromFloat(20.00),
		Discount:      decimal.Zero,
		CreatedAt:     time.Now(),
	}

	expectedItems := []domain.BookingItem{
		{
			ID:           uuid.New(),
			BookingID:    bookingID,
			TicketTypeID: &ticketTypeID,
			Quantity:     2,
			UnitPrice:    decimal.NewFromFloat(90.00),
			TotalPrice:   decimal.NewFromFloat(180.00),
		},
	}

	f.repo.findWithItemsFn = func(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
		if id == bookingID {
			return expectedBooking, expectedItems, nil
		}
		return nil, nil, domain.ErrBookingNotFound
	}

	f.router.GET("/api/v1/bookings/:id", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.GetBookingByID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+bookingID.String(), nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data domain.BookingWithItems `json:"data"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, bookingID, resp.Data.Booking.ID)
	assert.Equal(t, "BK-20261025-100", resp.Data.Booking.BookingNumber)
	assert.Equal(t, f.userID, resp.Data.Booking.UserID)
	assert.Equal(t, domain.BookingStatusConfirmed, resp.Data.Booking.Status)
	assert.Len(t, resp.Data.Items, 1)
	assert.Equal(t, 2, resp.Data.Items[0].Quantity)
}

func TestGetBookingByID_Unauthorized(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	bookingID := uuid.New()
	f.router.GET("/api/v1/bookings/:id", f.handler.GetBookingByID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+bookingID.String(), nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized")
}

func TestGetBookingByID_InvalidID(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	f.router.GET("/api/v1/bookings/:id", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.GetBookingByID(c)
	})

	// Test non-UUID string
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/not-a-valid-uuid", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid booking ID")

	// Test nil UUID (all zeros)
	reqNil := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/00000000-0000-0000-0000-000000000000", nil)
	recNil := httptest.NewRecorder()
	f.router.ServeHTTP(recNil, reqNil)

	assert.Equal(t, http.StatusBadRequest, recNil.Code)
	assert.Contains(t, recNil.Body.String(), "invalid booking ID")
}

func TestGetBookingByID_NotFound(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	missingID := uuid.New()
	f.repo.findWithItemsFn = func(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
		return nil, nil, domain.ErrBookingNotFound
	}

	f.router.GET("/api/v1/bookings/:id", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.GetBookingByID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+missingID.String(), nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "booking not found")
}

func TestGetBookingByID_CrossCustomerAccess_FailsClosed(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	otherUserID := uuid.New()
	bookingID := uuid.New()

	otherUserBooking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "BK-20261025-999",
		UserID:        otherUserID, // Owned by another user
		EventID:       uuid.New(),
		Status:        domain.BookingStatusConfirmed,
	}

	f.repo.findWithItemsFn = func(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
		return otherUserBooking, []domain.BookingItem{}, nil
	}

	f.router.GET("/api/v1/bookings/:id", func(c *gin.Context) {
		// Logged in as f.userID (NOT otherUserID)
		c.Set("user_id", f.userID)
		f.handler.GetBookingByID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+bookingID.String(), nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	// Must fail closed with 404 to avoid leaking booking existence across tenants
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "booking not found")
}

func TestGetBookingByID_UnexpectedDatabaseError(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	bookingID := uuid.New()
	f.repo.findWithItemsFn = func(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
		return nil, nil, errors.New("internal postgres error: connection refused")
	}

	f.router.GET("/api/v1/bookings/:id", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.GetBookingByID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+bookingID.String(), nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "postgres")
	assert.Contains(t, rec.Body.String(), "failed to retrieve booking")
}

func TestGetBookingQR_Success(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	bookingID := uuid.New()
	booking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "BK-20261025-QR1",
		UserID:        f.userID,
		EventID:       uuid.New(),
		Status:        domain.BookingStatusConfirmed,
	}

	f.repo.findWithItemsFn = func(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
		return booking, []domain.BookingItem{}, nil
	}

	f.router.GET("/api/v1/bookings/:id/qr", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.GetBookingQR(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+bookingID.String()+"/qr", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	assert.Equal(t, "private, no-store", rec.Header().Get("Cache-Control"))
	assert.NotEmpty(t, rec.Body.Bytes())
}

func TestGetBookingQR_CrossCustomerAccess_FailsClosed(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	otherUserID := uuid.New()
	bookingID := uuid.New()
	otherBooking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "BK-20261025-QR2",
		UserID:        otherUserID,
		EventID:       uuid.New(),
		Status:        domain.BookingStatusConfirmed,
	}

	f.repo.findWithItemsFn = func(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
		return otherBooking, []domain.BookingItem{}, nil
	}

	f.router.GET("/api/v1/bookings/:id/qr", func(c *gin.Context) {
		c.Set("user_id", f.userID) // authenticated as f.userID, not otherUserID
		f.handler.GetBookingQR(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+bookingID.String()+"/qr", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "booking not found")
}

func TestDownloadBookingPDF_Success(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	bookingID := uuid.New()
	eventID := uuid.New()
	seatID1 := uuid.New()

	booking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "BK-20261025-PDF1",
		UserID:        f.userID,
		EventID:       eventID,
		Status:        domain.BookingStatusConfirmed,
	}
	items := []domain.BookingItem{
		{
			ID:        uuid.New(),
			BookingID: bookingID,
			SeatID:    &seatID1,
			Quantity:  1,
		},
	}

	f.repo.findWithItemsFn = func(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
		return booking, items, nil
	}

	f.router.GET("/api/v1/bookings/:id/download", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.DownloadBookingPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+bookingID.String()+"/download", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment; filename=")
	assert.Equal(t, "private, no-store", rec.Header().Get("Cache-Control"))
	assert.NotEmpty(t, rec.Body.Bytes())
}

func TestDownloadBookingPDF_CrossCustomerAccess_FailsClosed(t *testing.T) {
	f := setupBookingHandlerFixture(t)

	otherUserID := uuid.New()
	bookingID := uuid.New()
	otherBooking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "BK-20261025-PDF2",
		UserID:        otherUserID,
		EventID:       uuid.New(),
		Status:        domain.BookingStatusConfirmed,
	}

	f.repo.findWithItemsFn = func(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
		return otherBooking, []domain.BookingItem{}, nil
	}

	f.router.GET("/api/v1/bookings/:id/download", func(c *gin.Context) {
		c.Set("user_id", f.userID)
		f.handler.DownloadBookingPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+bookingID.String()+"/download", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "booking not found")
}
