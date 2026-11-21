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

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOrganizerPaymentRepoForHandlerTest struct {
	domain.PaymentRepository
	lastFilter     *domain.OrganizerPaymentFilter
	listResult     *domain.OrganizerPaymentListResult
	summaryResult  *domain.OrganizerEventPaymentSummaryResult
	overviewResult *domain.OrganizerPaymentOverview
	listErr        error
	summaryErr     error
	overviewErr    error
}

func (m *mockOrganizerPaymentRepoForHandlerTest) FindOrganizerPayments(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerPaymentListResult, error) {
	m.lastFilter = &filter
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockOrganizerPaymentRepoForHandlerTest) GetOrganizerEventPaymentSummaries(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerEventPaymentSummaryResult, error) {
	m.lastFilter = &filter
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	return m.summaryResult, nil
}

func (m *mockOrganizerPaymentRepoForHandlerTest) GetOrganizerPaymentOverview(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerPaymentOverview, error) {
	m.lastFilter = &filter
	if m.overviewErr != nil {
		return nil, m.overviewErr
	}
	return m.overviewResult, nil
}

type mockOrganizerRepoForOrgPaymentHandlerTest struct {
	domain.OrganizerRepository
	organizers map[uuid.UUID]*domain.Organizer
	err        error
}

func (m *mockOrganizerRepoForOrgPaymentHandlerTest) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if m.err != nil {
		return nil, m.err
	}
	if org, ok := m.organizers[userID]; ok {
		return org, nil
	}
	return nil, domain.ErrOrganizerNotFound
}

type mockEventRepoForOrgPaymentHandlerTest struct {
	domain.EventRepository
	events map[uuid.UUID]*domain.Event
	err    error
}

func (m *mockEventRepoForOrgPaymentHandlerTest) FindByID(id uuid.UUID) (*domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	if ev, ok := m.events[id]; ok {
		return ev, nil
	}
	return nil, domain.ErrEventNotFound
}

func setupOrganizerPaymentTestRouter(
	payRepo domain.PaymentRepository,
	orgRepo domain.OrganizerRepository,
	eventRepo domain.EventRepository,
	jwtService *token.JWTService,
) (*gin.Engine, *mockOrganizerPaymentRepoForHandlerTest) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	mockRepo, _ := payRepo.(*mockOrganizerPaymentRepoForHandlerTest)
	uc, _ := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, eventRepo, logger)
	h := handler.NewOrganizerPaymentHandler(uc, logger)

	api := router.Group("/api/v1")
	organizer := api.Group("/organizers")
	organizer.Use(
		middleware.Auth(jwtService),
	)

	payments := organizer.Group("/payments")
	payments.Use(middleware.RequireRole("ORGANIZER"))
	{
		payments.GET("/overview", h.GetPaymentOverview)
		payments.GET("/summary-by-event", h.GetEventPaymentSummaries)
		payments.GET("", h.ListPayments)
	}

	return router, mockRepo
}

func TestOrganizerPaymentHandler_AuthenticationAndAuthorization(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	orgUserID := uuid.New()
	orgID := uuid.New()
	organizerToken, err := jwtService.GenerateAccessToken(orgUserID, "ORGANIZER")
	require.NoError(t, err)

	customerToken, err := jwtService.GenerateAccessToken(uuid.New(), "CUSTOMER")
	require.NoError(t, err)

	orgRepo := &mockOrganizerRepoForOrgPaymentHandlerTest{
		organizers: map[uuid.UUID]*domain.Organizer{
			orgUserID: {
				ID:     orgID,
				UserID: orgUserID,
				Status: "ACTIVE",
			},
		},
	}
	payRepo := &mockOrganizerPaymentRepoForHandlerTest{
		overviewResult: &domain.OrganizerPaymentOverview{},
		summaryResult:  &domain.OrganizerEventPaymentSummaryResult{},
		listResult:     &domain.OrganizerPaymentListResult{},
	}

	router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, nil, jwtService)

	endpoints := []string{
		"/api/v1/organizers/payments/overview",
		"/api/v1/organizers/payments/summary-by-event",
		"/api/v1/organizers/payments",
	}

	for _, endpoint := range endpoints {
		t.Run("unauthenticated request returns 401 on "+endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})

		t.Run("non-organizer role returns 403 on "+endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+customerToken)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusForbidden, rec.Code)
		})

		t.Run("valid organizer token returns 200 on "+endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+organizerToken)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestOrganizerPaymentHandler_StatusAndOwnershipErrors(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	orgUserID := uuid.New()
	orgToken, err := jwtService.GenerateAccessToken(orgUserID, "ORGANIZER")
	require.NoError(t, err)

	payRepo := &mockOrganizerPaymentRepoForHandlerTest{}

	t.Run("unapproved organizer returns 403 forbidden", func(t *testing.T) {
		orgRepo := &mockOrganizerRepoForOrgPaymentHandlerTest{
			organizers: map[uuid.UUID]*domain.Organizer{
				orgUserID: {
					ID:     uuid.New(),
					UserID: orgUserID,
					Status: "PENDING",
				},
			},
		}
		router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, nil, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/payments/overview", nil)
		req.Header.Set("Authorization", "Bearer "+orgToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("missing organizer record returns 403 forbidden", func(t *testing.T) {
		orgRepo := &mockOrganizerRepoForOrgPaymentHandlerTest{
			organizers: make(map[uuid.UUID]*domain.Organizer),
		}
		router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, nil, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/payments/overview", nil)
		req.Header.Set("Authorization", "Bearer "+orgToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("foreign event filtering returns 403 forbidden", func(t *testing.T) {
		orgID := uuid.New()
		foreignOrgID := uuid.New()
		foreignEventID := uuid.New()

		orgRepo := &mockOrganizerRepoForOrgPaymentHandlerTest{
			organizers: map[uuid.UUID]*domain.Organizer{
				orgUserID: {
					ID:     orgID,
					UserID: orgUserID,
					Status: "ACTIVE",
				},
			},
		}
		eventRepo := &mockEventRepoForOrgPaymentHandlerTest{
			events: map[uuid.UUID]*domain.Event{
				foreignEventID: {
					ID:          foreignEventID,
					OrganizerID: foreignOrgID,
				},
			},
		}
		router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, eventRepo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/payments?event_id="+foreignEventID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+orgToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("event not found returns 404 not found", func(t *testing.T) {
		orgID := uuid.New()
		missingEventID := uuid.New()

		orgRepo := &mockOrganizerRepoForOrgPaymentHandlerTest{
			organizers: map[uuid.UUID]*domain.Organizer{
				orgUserID: {
					ID:     orgID,
					UserID: orgUserID,
					Status: "ACTIVE",
				},
			},
		}
		eventRepo := &mockEventRepoForOrgPaymentHandlerTest{
			events: make(map[uuid.UUID]*domain.Event),
		}
		router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, eventRepo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/payments?event_id="+missingEventID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+orgToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestOrganizerPaymentHandler_Validation(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	orgUserID := uuid.New()
	orgID := uuid.New()
	orgToken, err := jwtService.GenerateAccessToken(orgUserID, "ORGANIZER")
	require.NoError(t, err)

	orgRepo := &mockOrganizerRepoForOrgPaymentHandlerTest{
		organizers: map[uuid.UUID]*domain.Organizer{
			orgUserID: {
				ID:     orgID,
				UserID: orgUserID,
				Status: "ACTIVE",
			},
		},
	}
	payRepo := &mockOrganizerPaymentRepoForHandlerTest{}
	router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, nil, jwtService)

	invalidTestCases := []struct {
		name string
		url  string
	}{
		{"invalid page text", "/api/v1/organizers/payments?page=abc"},
		{"invalid page negative", "/api/v1/organizers/payments?page=-5"},
		{"invalid limit text", "/api/v1/organizers/payments?limit=xyz"},
		{"invalid limit zero", "/api/v1/organizers/payments?limit=0"},
		{"invalid limit exceeding 100", "/api/v1/organizers/payments?limit=101"},
		{"invalid event ID non-uuid", "/api/v1/organizers/payments?event_id=not-a-uuid"},
		{"invalid event ID nil uuid", "/api/v1/organizers/payments?event_id=00000000-0000-0000-0000-000000000000"},
		{"invalid from date format", "/api/v1/organizers/payments?from=bad-date"},
		{"invalid to date format", "/api/v1/organizers/payments?to=bad-date"},
		{"invalid date range from after to", "/api/v1/organizers/payments?from=2026-11-20&to=2026-11-10"},
		{"invalid payment status", "/api/v1/organizers/payments?status=INVALID_STATUS"},
		{"invalid allocation status", "/api/v1/organizers/payments?allocation_status=INVALID_ALLOC"},
	}

	for _, tc := range invalidTestCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			req.Header.Set("Authorization", "Bearer "+orgToken)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestOrganizerPaymentHandler_SuccessfulResponses(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	orgUserID := uuid.New()
	orgID := uuid.New()
	orgToken, err := jwtService.GenerateAccessToken(orgUserID, "ORGANIZER")
	require.NoError(t, err)

	orgRepo := &mockOrganizerRepoForOrgPaymentHandlerTest{
		organizers: map[uuid.UUID]*domain.Organizer{
			orgUserID: {
				ID:     orgID,
				UserID: orgUserID,
				Status: "ACTIVE",
			},
		},
	}

	t.Run("GetPaymentOverview returns 200 with data envelope", func(t *testing.T) {
		overview := &domain.OrganizerPaymentOverview{
			GrossCapturedAmount: decimal.NewFromInt(2500),
			TotalTicketsSold:    10,
			TotalEvents:         3,
		}
		payRepo := &mockOrganizerPaymentRepoForHandlerTest{
			overviewResult: overview,
		}
		router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, nil, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/payments/overview?status=SUCCESS", nil)
		req.Header.Set("Authorization", "Bearer "+orgToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, true, body["success"])
		data := body["data"].(map[string]interface{})
		assert.Equal(t, "2500", data["gross_captured_amount"])
		assert.Equal(t, float64(10), data["total_tickets_sold"])
		assert.Equal(t, float64(3), data["total_events"])
		assert.Equal(t, "SUCCESS", payRepo.lastFilter.Status)
	})

	t.Run("GetEventPaymentSummaries returns 200 with summaries and overview", func(t *testing.T) {
		eventID := uuid.New()
		summaryResult := &domain.OrganizerEventPaymentSummaryResult{
			Summaries: []domain.OrganizerEventPaymentSummary{
				{
					EventID:             eventID,
					EventTitle:          "Summer Rock",
					TicketsSold:         4,
					GrossCapturedAmount: decimal.NewFromInt(1000),
				},
			},
			Overview: domain.OrganizerPaymentOverview{
				TotalTicketsSold:    4,
				GrossCapturedAmount: decimal.NewFromInt(1000),
			},
			TotalRecords: 1,
			Page:         1,
			Limit:        10,
			TotalPages:   1,
		}
		payRepo := &mockOrganizerPaymentRepoForHandlerTest{
			summaryResult: summaryResult,
		}
		router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, nil, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/payments/summary-by-event?page=1&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+orgToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, true, body["success"])
		data := body["data"].(map[string]interface{})
		summaries := data["summaries"].([]interface{})
		assert.Len(t, summaries, 1)
		assert.Equal(t, float64(1), data["total_records"])
	})

	t.Run("ListPayments returns 200 with payments list and pagination", func(t *testing.T) {
		paymentID := uuid.New().String()
		listResult := &domain.OrganizerPaymentListResult{
			Payments: []domain.OrganizerPaymentListItem{
				{
					ID:            paymentID,
					BookingNumber: "BK-100",
					Amount:        decimal.NewFromInt(500),
					Status:        domain.PaymentStatusSuccess,
				},
			},
			TotalRecords: 1,
			Page:         1,
			Limit:        10,
			TotalPages:   1,
		}
		payRepo := &mockOrganizerPaymentRepoForHandlerTest{
			listResult: listResult,
		}
		router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, nil, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/payments?search=BK-100", nil)
		req.Header.Set("Authorization", "Bearer "+orgToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, true, body["success"])
		data := body["data"].(map[string]interface{})
		paymentsList := data["payments"].([]interface{})
		assert.Len(t, paymentsList, 1)
		assert.Equal(t, "BK-100", payRepo.lastFilter.Search)
	})

	t.Run("Repository unexpected failure returns 500", func(t *testing.T) {
		payRepo := &mockOrganizerPaymentRepoForHandlerTest{
			listErr: errors.New("database disk full"),
		}
		router, _ := setupOrganizerPaymentTestRouter(payRepo, orgRepo, nil, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/payments", nil)
		req.Header.Set("Authorization", "Bearer "+orgToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
