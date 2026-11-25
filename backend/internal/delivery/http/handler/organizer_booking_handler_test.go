package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOrganizerBookingService struct {
	listBookingsResult  *domain.OrganizerBookingListResult
	listBookingsErr     error
	bookingStatsResult  *domain.OrganizerBookingStats
	bookingStatsErr     error
	bookingDetailResult *domain.OrganizerBookingDetail
	bookingDetailErr    error
	listAttendeesResult *domain.OrganizerAttendeeListResult
	listAttendeesErr    error
	attendeeStatsResult *domain.OrganizerAttendeeStats
	attendeeStatsErr    error

	capturedUserID    uuid.UUID
	capturedBookingID uuid.UUID
	capturedBFilter   domain.OrganizerBookingFilter
	capturedAFilter   domain.OrganizerAttendeeFilter
}

func (m *mockOrganizerBookingService) ListBookings(ctx context.Context, userID uuid.UUID, filter domain.OrganizerBookingFilter) (*domain.OrganizerBookingListResult, error) {
	m.capturedUserID = userID
	m.capturedBFilter = filter
	if m.listBookingsErr != nil {
		return nil, m.listBookingsErr
	}
	return m.listBookingsResult, nil
}

func (m *mockOrganizerBookingService) GetBookingStats(ctx context.Context, userID uuid.UUID, filter domain.OrganizerBookingFilter) (*domain.OrganizerBookingStats, error) {
	m.capturedUserID = userID
	m.capturedBFilter = filter
	if m.bookingStatsErr != nil {
		return nil, m.bookingStatsErr
	}
	return m.bookingStatsResult, nil
}

func (m *mockOrganizerBookingService) GetBookingDetail(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID) (*domain.OrganizerBookingDetail, error) {
	m.capturedUserID = userID
	m.capturedBookingID = bookingID
	if m.bookingDetailErr != nil {
		return nil, m.bookingDetailErr
	}
	return m.bookingDetailResult, nil
}

func (m *mockOrganizerBookingService) ListAttendees(ctx context.Context, userID uuid.UUID, filter domain.OrganizerAttendeeFilter) (*domain.OrganizerAttendeeListResult, error) {
	m.capturedUserID = userID
	m.capturedAFilter = filter
	if m.listAttendeesErr != nil {
		return nil, m.listAttendeesErr
	}
	return m.listAttendeesResult, nil
}

func (m *mockOrganizerBookingService) GetAttendeeStats(ctx context.Context, userID uuid.UUID, filter domain.OrganizerAttendeeFilter) (*domain.OrganizerAttendeeStats, error) {
	m.capturedUserID = userID
	m.capturedAFilter = filter
	if m.attendeeStatsErr != nil {
		return nil, m.attendeeStatsErr
	}
	return m.attendeeStatsResult, nil
}

func setupTestRouter(svc OrganizerBookingService, authenticatedUser *uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(slog.Default()))

	api := r.Group("/api/v1/organizers")
	if authenticatedUser != nil {
		api.Use(func(c *gin.Context) {
			c.Set("user_id", *authenticatedUser)
			c.Next()
		})
	}

	h := NewOrganizerBookingHandler(svc, slog.Default())

	bookings := api.Group("/bookings")
	{
		bookings.GET("/stats", h.GetBookingStats)
		bookings.GET("", h.ListBookings)
		bookings.GET("/:id", h.GetBookingDetail)
	}

	attendees := api.Group("/attendees")
	{
		attendees.GET("/stats", h.GetAttendeeStats)
		attendees.GET("", h.ListAttendees)
	}

	return r
}

// ---------------------------------------------------------------------------
// 1. Bookings Listing Tests
// ---------------------------------------------------------------------------

func TestOrganizerBookingHandler_ListBookings(t *testing.T) {
	userID := uuid.New()

	t.Run("successful listing returns 200 with standard pagination envelope", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			listBookingsResult: &domain.OrganizerBookingListResult{
				Bookings: []domain.OrganizerBookingListItem{
					{
						ID:                   uuid.New(),
						BookingNumber:        "ORD-TEST-100",
						CreatedAt:            time.Now(),
						OriginalBookingTotal: decimal.NewFromInt(1200),
						Currency:             "INR",
						BookingStatus:        "CONFIRMED",
						PaymentStatus:        "SUCCESS",
					},
				},
				TotalRecords: 1,
				Page:         1,
				Limit:        10,
				TotalPages:   1,
			},
		}

		router := setupTestRouter(mockSvc, &userID)
		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings?page=1&limit=10&booking_status=CONFIRMED&payment_status=SUCCESS", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["total_records"])
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(10), data["limit"])

		pagination := data["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["total_records"])

		assert.Equal(t, userID, mockSvc.capturedUserID)
		assert.Equal(t, "CONFIRMED", mockSvc.capturedBFilter.BookingStatus)
		assert.Equal(t, "SUCCESS", mockSvc.capturedBFilter.PaymentStatus)
	})

	t.Run("invalid page parameter returns 400 VALIDATION_ERROR", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings?page=-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})

	t.Run("invalid limit parameter returns 400 VALIDATION_ERROR", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings?limit=150", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})

	t.Run("invalid event_id returns 400 VALIDATION_ERROR", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings?event_id=not-a-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})

	t.Run("invalid date parameter returns 400 VALIDATION_ERROR", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings?from=invalid-date", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthenticated request returns 401 UNAUTHORIZED", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{}
		router := setupTestRouter(mockSvc, nil) // No user_id set

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "UNAUTHORIZED", resp["code"])
	})

	t.Run("unauthorized organizer returns 403 FORBIDDEN", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			listBookingsErr: booking.ErrUnauthorizedOrganizer,
		}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "FORBIDDEN", resp["code"])
	})

	t.Run("foreign event access returns 403 FORBIDDEN", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			listBookingsErr: booking.ErrUnauthorizedEvent,
		}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("unexpected usecase error returns 500 INTERNAL_SERVER_ERROR", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			listBookingsErr: errors.New("db query timeout"),
		}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "INTERNAL_SERVER_ERROR", resp["code"])
	})
}

// ---------------------------------------------------------------------------
// 2. Booking Statistics Tests
// ---------------------------------------------------------------------------

func TestOrganizerBookingHandler_GetBookingStats(t *testing.T) {
	userID := uuid.New()

	t.Run("successful statistics retrieval returns 200", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			bookingStatsResult: &domain.OrganizerBookingStats{
				TotalBookings:       5,
				ConfirmedBookings:   4,
				CancelledBookings:   1,
				TotalTicketsSold:    10,
				GrossCapturedAmount: decimal.NewFromInt(5000),
				TotalRefundedAmount: decimal.NewFromInt(1000),
				NetCapturedAmount:   decimal.NewFromInt(4000),
				Currency:            "INR",
			},
		}

		router := setupTestRouter(mockSvc, &userID)
		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(5), data["total_bookings"])
		assert.Equal(t, float64(10), data["total_tickets_sold"])
	})

	t.Run("invalid date range from usecase returns 400 VALIDATION_ERROR", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			bookingStatsErr: booking.ErrInvalidDateRange,
		}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings/stats?from=2026-11-20&to=2026-11-10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})
}

// ---------------------------------------------------------------------------
// 3. Booking Detail Tests
// ---------------------------------------------------------------------------

func TestOrganizerBookingHandler_GetBookingDetail(t *testing.T) {
	userID := uuid.New()
	bookingID := uuid.New()

	t.Run("successful retrieval returns 200 with detail", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			bookingDetailResult: &domain.OrganizerBookingDetail{
				ID:            bookingID,
				BookingNumber: "ORD-DETAIL-999",
				BookingStatus: "CONFIRMED",
				Financials: domain.OrganizerBookingFinancials{
					OriginalBookingTotal:     decimal.NewFromInt(3000),
					SuccessfulCapturedAmount: decimal.NewFromInt(3000),
					NetCapturedAfterRefunds:  decimal.NewFromInt(3000),
					Currency:                 "INR",
				},
			},
		}

		router := setupTestRouter(mockSvc, &userID)
		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings/"+bookingID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp["success"].(bool))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, bookingID.String(), data["id"])
		assert.Equal(t, "ORD-DETAIL-999", data["booking_number"])
		assert.Equal(t, bookingID, mockSvc.capturedBookingID)
	})

	t.Run("invalid booking ID parameter returns 400 VALIDATION_ERROR", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings/invalid-booking-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})

	t.Run("booking not found returns 404 NOT_FOUND", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			bookingDetailErr: domain.ErrBookingNotFound,
		}
		router := setupTestRouter(mockSvc, &userID)

		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings/"+bookingID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "NOT_FOUND", resp["code"])
	})
}

// ---------------------------------------------------------------------------
// 4. Attendee Listing & Statistics Tests
// ---------------------------------------------------------------------------

func TestOrganizerBookingHandler_Attendees(t *testing.T) {
	userID := uuid.New()

	t.Run("successful attendees listing returns 200 with envelope", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			listAttendeesResult: &domain.OrganizerAttendeeListResult{
				Attendees: []domain.OrganizerAttendeeListItem{
					{
						TicketID:        uuid.New(),
						TicketCode:      "TCK-2026-VIP",
						AdmissionNumber: 1,
						TicketTypeName:  "VIP",
						Status:          "VALID",
					},
				},
				TotalRecords: 1,
				Page:         1,
				Limit:        10,
				TotalPages:   1,
			},
		}

		router := setupTestRouter(mockSvc, &userID)
		req, _ := http.NewRequest("GET", "/api/v1/organizers/attendees?check_in_status=VALID&page=1&limit=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp["success"].(bool))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["total_records"])
		assert.Equal(t, "VALID", mockSvc.capturedAFilter.CheckInStatus)
	})

	t.Run("successful attendee stats returns 200", func(t *testing.T) {
		mockSvc := &mockOrganizerBookingService{
			attendeeStatsResult: &domain.OrganizerAttendeeStats{
				TotalIssuedTickets:       20,
				ActiveEligibleAdmissions: 18,
				CheckedInTickets:         6,
				PendingCheckInTickets:    12,
				CancelledTickets:         2,
				CheckInRate:              decimal.NewFromFloat(33.33),
			},
		}

		router := setupTestRouter(mockSvc, &userID)
		req, _ := http.NewRequest("GET", "/api/v1/organizers/attendees/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(20), data["total_issued_tickets"])
		assert.Equal(t, float64(6), data["checked_in_tickets"])
	})
}

// ---------------------------------------------------------------------------
// 5. Static Route Resolution Test (/stats vs /:id)
// ---------------------------------------------------------------------------

func TestOrganizerBookingHandler_RouteResolution(t *testing.T) {
	userID := uuid.New()
	statsBookingCalled := false
	detailBookingCalled := false

	mockSvc := &mockOrganizerBookingService{
		bookingStatsResult: &domain.OrganizerBookingStats{
			TotalBookings: 1,
		},
		bookingDetailResult: &domain.OrganizerBookingDetail{
			ID: uuid.New(),
		},
	}

	router := setupTestRouter(mockSvc, &userID)

	t.Run("/bookings/stats strictly calls GetBookingStats, not GetBookingDetail", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/organizers/bookings/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// capturedBookingID must not be parsed from "stats"
		assert.Equal(t, uuid.Nil, mockSvc.capturedBookingID)
		_ = statsBookingCalled
		_ = detailBookingCalled
	})

	t.Run("/attendees/stats strictly calls GetAttendeeStats, not ListAttendees", func(t *testing.T) {
		mockSvc.attendeeStatsResult = &domain.OrganizerAttendeeStats{
			TotalIssuedTickets: 5,
		}
		req, _ := http.NewRequest("GET", "/api/v1/organizers/attendees/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(5), data["total_issued_tickets"])
	})
}
