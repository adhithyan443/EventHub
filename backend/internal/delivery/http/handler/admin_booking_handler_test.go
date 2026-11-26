package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAdminBookingService struct {
	listBookingsResult  *domain.AdminBookingListResult
	listBookingsErr     error
	bookingStatsResult  *domain.AdminBookingStats
	bookingStatsErr     error
	bookingDetailResult *domain.AdminBookingDetail
	bookingDetailErr    error
	listAttendeesResult *domain.AdminAttendeeListResult
	listAttendeesErr    error
	attendeeStatsResult *domain.AdminAttendeeStats
	attendeeStatsErr    error

	capturedBFilter   domain.AdminBookingFilter
	capturedAFilter   domain.AdminAttendeeFilter
	capturedBookingID uuid.UUID
}

func (m *mockAdminBookingService) ListBookings(ctx context.Context, filter domain.AdminBookingFilter) (*domain.AdminBookingListResult, error) {
	m.capturedBFilter = filter
	if m.listBookingsErr != nil {
		return nil, m.listBookingsErr
	}
	return m.listBookingsResult, nil
}

func (m *mockAdminBookingService) GetBookingStats(ctx context.Context, filter domain.AdminBookingFilter) (*domain.AdminBookingStats, error) {
	m.capturedBFilter = filter
	if m.bookingStatsErr != nil {
		return nil, m.bookingStatsErr
	}
	return m.bookingStatsResult, nil
}

func (m *mockAdminBookingService) GetBookingDetail(ctx context.Context, bookingID uuid.UUID) (*domain.AdminBookingDetail, error) {
	m.capturedBookingID = bookingID
	if m.bookingDetailErr != nil {
		return nil, m.bookingDetailErr
	}
	return m.bookingDetailResult, nil
}

func (m *mockAdminBookingService) ListAttendees(ctx context.Context, filter domain.AdminAttendeeFilter) (*domain.AdminAttendeeListResult, error) {
	m.capturedAFilter = filter
	if m.listAttendeesErr != nil {
		return nil, m.listAttendeesErr
	}
	return m.listAttendeesResult, nil
}

func (m *mockAdminBookingService) GetAttendeeStats(ctx context.Context, filter domain.AdminAttendeeFilter) (*domain.AdminAttendeeStats, error) {
	m.capturedAFilter = filter
	if m.attendeeStatsErr != nil {
		return nil, m.attendeeStatsErr
	}
	return m.attendeeStatsResult, nil
}

func setupAdminBookingTestRouter(svc AdminBookingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.ErrorHandler(nil))

	h := NewAdminBookingHandler(svc, nil)

	admin := r.Group("/api/v1/admin")
	{
		admin.GET("/bookings", h.ListBookings)
		admin.GET("/bookings/stats", h.GetBookingStats)
		admin.GET("/bookings/:id", h.GetBookingDetail)

		admin.GET("/attendees", h.ListAttendees)
		admin.GET("/attendees/stats", h.GetAttendeeStats)
	}

	return r
}

func TestAdminBookingHandler_ListBookings(t *testing.T) {
	svc := &mockAdminBookingService{
		listBookingsResult: &domain.AdminBookingListResult{
			Bookings: []domain.AdminBookingListItem{
				{
					ID:            uuid.New(),
					BookingNumber: "BK-ADMIN-1",
					Organizer: domain.AdminOrganizerBrief{
						BusinessName: "Global Live",
					},
					OriginalBookingTotal: decimal.NewFromInt(1500),
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

	r := setupAdminBookingTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/bookings?page=1&limit=10&search=BK-ADMIN", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]interface{})
	bookings := data["bookings"].([]interface{})
	assert.Equal(t, 1, len(bookings))
	first := bookings[0].(map[string]interface{})
	assert.Equal(t, "BK-ADMIN-1", first["booking_number"])
}

func TestAdminBookingHandler_GetBookingStats(t *testing.T) {
	svc := &mockAdminBookingService{
		bookingStatsResult: &domain.AdminBookingStats{
			TotalBookings:       50,
			ConfirmedBookings:   45,
			CancelledBookings:   5,
			TotalTicketsSold:    90,
			GrossCapturedAmount: decimal.NewFromInt(90000),
			TotalRefundedAmount: decimal.NewFromInt(5000),
			NetCapturedAmount:   decimal.NewFromInt(85000),
			Currency:            "INR",
		},
	}

	r := setupAdminBookingTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/bookings/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(50), data["total_bookings"])
}

func TestAdminBookingHandler_GetBookingDetail(t *testing.T) {
	bID := uuid.New()
	svc := &mockAdminBookingService{
		bookingDetailResult: &domain.AdminBookingDetail{
			ID:            bID,
			BookingNumber: "BK-DETAIL-1",
			Organizer: domain.AdminOrganizerBrief{
				BusinessName: "Supreme Events",
			},
		},
	}

	r := setupAdminBookingTestRouter(svc)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/bookings/"+bID.String(), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "BK-DETAIL-1", data["booking_number"])
	})

	t.Run("invalid uuid", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/bookings/invalid-id", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAdminBookingHandler_Attendees(t *testing.T) {
	svc := &mockAdminBookingService{
		listAttendeesResult: &domain.AdminAttendeeListResult{
			Attendees: []domain.AdminAttendeeListItem{
				{
					TicketID:        uuid.New(),
					TicketCode:      "TCK-ATT-1",
					AdmissionNumber: 1,
					Status:          "VALID",
					Organizer: domain.AdminOrganizerBrief{
						BusinessName: "Superstar Con",
					},
				},
			},
			TotalRecords: 1,
			Page:         1,
			Limit:        10,
			TotalPages:   1,
		},
		attendeeStatsResult: &domain.AdminAttendeeStats{
			TotalIssuedTickets:       100,
			ActiveEligibleAdmissions: 95,
			CheckedInTickets:         70,
			PendingCheckInTickets:    25,
			CancelledTickets:         5,
			CheckInRate:              decimal.NewFromFloat(73.68),
		},
	}

	r := setupAdminBookingTestRouter(svc)

	t.Run("list attendees", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/attendees?page=1&limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["total_records"])
	})

	t.Run("attendee stats", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/attendees/stats", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(100), data["total_issued_tickets"])
	})
}
