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

type mockPaymentRepoForHandlerTest struct {
	domain.PaymentRepository
	lastFilter    *domain.AdminPaymentFilter
	result        *domain.AdminPaymentListResult
	summaryResult *domain.AdminEventPaymentSummaryResult
	err           error
	summaryErr    error
}

func (m *mockPaymentRepoForHandlerTest) FindAdminPayments(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) (*domain.AdminPaymentListResult, error) {
	m.lastFilter = &filter
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *mockPaymentRepoForHandlerTest) GetEventPaymentSummaries(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) (*domain.AdminEventPaymentSummaryResult, error) {
	m.lastFilter = &filter
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	return m.summaryResult, nil
}

func setupAdminPaymentTestRouter(
	repo domain.PaymentRepository,
	jwtService *token.JWTService,
) (*gin.Engine, *mockPaymentRepoForHandlerTest) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	mockRepo, _ := repo.(*mockPaymentRepoForHandlerTest)
	adminPaymentUsecase, _ := payment.NewAdminPaymentUsecase(repo, logger)
	adminPaymentHandler := handler.NewAdminPaymentHandler(adminPaymentUsecase, logger)

	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(
		middleware.Auth(jwtService),
		middleware.RequireRole("ADMIN"),
	)
	admin.GET("/payments/summary-by-event", adminPaymentHandler.GetEventPaymentSummaries)
	admin.GET("/payments", adminPaymentHandler.ListPayments)

	return router, mockRepo
}

func TestAdminPaymentHandler_ListPayments(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminID := uuid.New()
	adminToken, err := jwtService.GenerateAccessToken(adminID, "ADMIN")
	require.NoError(t, err)

	customerToken, err := jwtService.GenerateAccessToken(uuid.New(), "CUSTOMER")
	require.NoError(t, err)

	t.Run("unauthorized request rejected with 401", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-admin role rejected with 403", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments", nil)
		req.Header.Set("Authorization", "Bearer "+customerToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("successful list with default pagination", func(t *testing.T) {
		expectedResult := &domain.AdminPaymentListResult{
			Payments: []domain.AdminPaymentListItem{
				{
					ID:            uuid.New().String(),
					BookingID:     uuid.New().String(),
					BookingNumber: "BK-99999",
					Customer: domain.AdminPaymentCustomerInfo{
						ID:    uuid.New(),
						Name:  "Jane Doe",
						Email: "jane@example.com",
					},
					Event: domain.AdminPaymentEventInfo{
						ID:    uuid.New(),
						Title: "Rock Concert",
					},
					Organizer: domain.AdminPaymentOrganizerInfo{
						ID:           uuid.New(),
						BusinessName: "Live Events LLC",
					},
					Amount:            decimal.NewFromInt(500),
					Currency:          "INR",
					Status:            domain.PaymentStatusSuccess,
					HasGatewayCapture: true,
					AllocationStatus:  domain.AllocationStatusAccounted,
					Method:            domain.PaymentMethodUPI,
					Provider:          domain.PaymentProviderRazorpay,
					CreatedAt:         time.Now().UTC(),
					UpdatedAt:         time.Now().UTC(),
				},
			},
			TotalRecords: 1,
			Page:         1,
			Limit:        10,
			TotalPages:   1,
		}

		repo := &mockPaymentRepoForHandlerTest{result: expectedResult}
		router, mockRepo := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, 1, mockRepo.lastFilter.Page)
		assert.Equal(t, 10, mockRepo.lastFilter.Limit)

		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["total_records"])
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(10), data["limit"])

		pagination := data["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["total_records"])
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(10), pagination["limit"])
		assert.Equal(t, float64(1), pagination["total_pages"])

		payments := data["payments"].([]interface{})
		assert.Len(t, payments, 1)
		first := payments[0].(map[string]interface{})
		assert.Equal(t, "BK-99999", first["booking_number"])
		assert.Equal(t, "ACCOUNTED", first["allocation_status"])
		assert.Equal(t, true, first["has_gateway_capture"])
	})

	t.Run("query parameters parsed correctly", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{
			result: &domain.AdminPaymentListResult{
				Payments:     []domain.AdminPaymentListItem{},
				TotalRecords: 0,
				Page:         2,
				Limit:        25,
				TotalPages:   0,
			},
		}
		router, mockRepo := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments?page=2&limit=25&status=SUCCESS&allocation_status=UNALLOCATED&search=concert&from=2026-11-01&to=2026-11-18", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.NotNil(t, mockRepo.lastFilter)
		assert.Equal(t, 2, mockRepo.lastFilter.Page)
		assert.Equal(t, 25, mockRepo.lastFilter.Limit)
		assert.Equal(t, "SUCCESS", mockRepo.lastFilter.Status)
		assert.Equal(t, "UNALLOCATED", mockRepo.lastFilter.AllocationStatus)
		assert.Equal(t, "concert", mockRepo.lastFilter.Search)
		require.NotNil(t, mockRepo.lastFilter.From)
		require.NotNil(t, mockRepo.lastFilter.To)
		assert.Equal(t, 2026, mockRepo.lastFilter.From.Year())
		assert.Equal(t, time.November, mockRepo.lastFilter.From.Month())
		assert.Equal(t, 1, mockRepo.lastFilter.From.Day())
		assert.Equal(t, 0, mockRepo.lastFilter.From.Hour())
		assert.Equal(t, 23, mockRepo.lastFilter.To.Hour())
		assert.Equal(t, 59, mockRepo.lastFilter.To.Minute())
	})

	t.Run("invalid page parameter rejected", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		// Page < 1
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments?page=-1", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		// Page > 100000
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments?page=100001", nil)
		req2.Header.Set("Authorization", "Bearer "+adminToken)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusBadRequest, rec2.Code)
	})

	t.Run("invalid limit parameter rejected", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments?limit=200", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid date parameter rejected", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments?from=not-a-date", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("usecase validation error returns 400", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments?allocation_status=NOT_VALID_STATUS", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})

	t.Run("unexpected repository failure returns 500 without leaking internal details", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{
			err: errors.New("fatal: database connection failed with credentials [super_secret]"),
		}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "INTERNAL_SERVER_ERROR", resp["code"])
		assert.Equal(t, "internal server error", resp["message"])
		assert.NotContains(t, rec.Body.String(), "super_secret")
	})

	t.Run("valid event_id parameter parsed and passed to filter", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{
			result: &domain.AdminPaymentListResult{
				Payments:     []domain.AdminPaymentListItem{},
				TotalRecords: 0,
				Page:         1,
				Limit:        10,
				TotalPages:   0,
			},
		}
		router, mockRepo := setupAdminPaymentTestRouter(repo, jwtService)

		evID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments?event_id="+evID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.NotNil(t, mockRepo.lastFilter)
		require.NotNil(t, mockRepo.lastFilter.EventID)
		assert.Equal(t, evID, *mockRepo.lastFilter.EventID)
	})

	t.Run("invalid event_id parameter rejected with 400", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments?event_id=not-a-valid-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})
}

func TestAdminPaymentHandler_GetEventPaymentSummaries(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminID := uuid.New()
	adminToken, err := jwtService.GenerateAccessToken(adminID, "ADMIN")
	require.NoError(t, err)

	customerToken, err := jwtService.GenerateAccessToken(uuid.New(), "CUSTOMER")
	require.NoError(t, err)

	t.Run("unauthorized request rejected with 401", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-admin role rejected with 403", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event", nil)
		req.Header.Set("Authorization", "Bearer "+customerToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("successful list with default pagination and envelope verification", func(t *testing.T) {
		evID := uuid.New()
		orgID := uuid.New()
		expectedResult := &domain.AdminEventPaymentSummaryResult{
			Summaries: []domain.AdminEventPaymentSummary{
				{
					EventID:                            evID,
					EventTitle:                         "Summer Fest 2026",
					OrganizerID:                        orgID,
					OrganizerBusinessName:              "Premier Festivals LLC",
					TotalPaymentsCount:                 5,
					SuccessfullyCompletedPaymentsCount: 3,
					GrossCapturedAmount:                decimal.NewFromInt(1500),
					Currency:                           "INR",
					PendingPaymentsCount:               1,
					FailedPaymentsCount:                1,
					RefundRequiredCount:                0,
					PaymentsWithSuccessfulRefundsCount: 0,
					UnallocatedCapturedCount:           1,
				},
			},
			OverallSummary: domain.AdminOverallPaymentSummary{
				TotalEvents:              1,
				GrossCapturedAmount:      decimal.NewFromInt(1500),
				Currency:                 "INR",
				RefundRequiredCount:      0,
				UnallocatedCapturedCount: 1,
			},
			TotalRecords: 1,
			Page:         1,
			Limit:        10,
			TotalPages:   1,
		}

		repo := &mockPaymentRepoForHandlerTest{summaryResult: expectedResult}
		router, mockRepo := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, 1, mockRepo.lastFilter.Page)
		assert.Equal(t, 10, mockRepo.lastFilter.Limit)

		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["total_records"])
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(10), data["limit"])
		assert.Equal(t, float64(1), data["total_pages"])

		overall := data["overall_summary"].(map[string]interface{})
		assert.Equal(t, float64(1), overall["total_events"])
		assert.Equal(t, "1500", overall["gross_captured_amount"].(string))
		assert.Equal(t, "INR", overall["currency"])
		assert.Equal(t, float64(0), overall["refund_required_count"])
		assert.Equal(t, float64(1), overall["unallocated_captured_count"])

		pagination := data["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["total_records"])
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(10), pagination["limit"])
		assert.Equal(t, float64(1), pagination["total_pages"])

		summaries := data["summaries"].([]interface{})
		assert.Len(t, summaries, 1)
		first := summaries[0].(map[string]interface{})
		assert.Equal(t, evID.String(), first["event_id"])
		assert.Equal(t, "Summer Fest 2026", first["event_title"])
		assert.Equal(t, "Premier Festivals LLC", first["organizer_business_name"])
		assert.Equal(t, float64(5), first["total_payments_count"])
		assert.Equal(t, float64(3), first["successfully_completed_payments_count"])
		assert.Equal(t, "1500", first["gross_captured_amount"].(string))
		assert.Equal(t, float64(1), first["unallocated_captured_count"])

		// Also check that 'events' array is present for compatibility
		events := data["events"].([]interface{})
		assert.Len(t, events, 1)
	})

	t.Run("query parameters parsed correctly", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{
			summaryResult: &domain.AdminEventPaymentSummaryResult{
				Summaries:    []domain.AdminEventPaymentSummary{},
				TotalRecords: 0,
				Page:         2,
				Limit:        25,
				TotalPages:   0,
			},
		}
		router, mockRepo := setupAdminPaymentTestRouter(repo, jwtService)

		evID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event?page=2&limit=25&search=festival&from=2026-11-01&to=2026-11-18&event_id="+evID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.NotNil(t, mockRepo.lastFilter)
		assert.Equal(t, 2, mockRepo.lastFilter.Page)
		assert.Equal(t, 25, mockRepo.lastFilter.Limit)
		assert.Equal(t, "festival", mockRepo.lastFilter.Search)
		require.NotNil(t, mockRepo.lastFilter.EventID)
		assert.Equal(t, evID, *mockRepo.lastFilter.EventID)
		require.NotNil(t, mockRepo.lastFilter.From)
		require.NotNil(t, mockRepo.lastFilter.To)
		assert.Equal(t, 2026, mockRepo.lastFilter.From.Year())
		assert.Equal(t, time.November, mockRepo.lastFilter.From.Month())
		assert.Equal(t, 1, mockRepo.lastFilter.From.Day())
		assert.Equal(t, 0, mockRepo.lastFilter.From.Hour())
		assert.Equal(t, 23, mockRepo.lastFilter.To.Hour())
		assert.Equal(t, 59, mockRepo.lastFilter.To.Minute())
	})

	t.Run("invalid page parameter rejected", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event?page=-1", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event?page=100001", nil)
		req2.Header.Set("Authorization", "Bearer "+adminToken)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusBadRequest, rec2.Code)
	})

	t.Run("invalid limit parameter rejected", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event?limit=0", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event?limit=101", nil)
		req2.Header.Set("Authorization", "Bearer "+adminToken)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusBadRequest, rec2.Code)
	})

	t.Run("invalid event_id parameter rejected", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event?event_id=not-a-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})

	t.Run("invalid date parameter rejected", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event?from=not-a-valid-date", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("usecase validation error returns 400 when from after to", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event?from=2026-11-20&to=2026-11-10", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})

	t.Run("handles empty result cleanly", func(t *testing.T) {
		emptyResult := &domain.AdminEventPaymentSummaryResult{
			Summaries:      []domain.AdminEventPaymentSummary{},
			OverallSummary: domain.AdminOverallPaymentSummary{},
			TotalRecords:   0,
			Page:           1,
			Limit:          10,
			TotalPages:     0,
		}
		repo := &mockPaymentRepoForHandlerTest{summaryResult: emptyResult}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp["success"].(bool))
		data := resp["data"].(map[string]interface{})
		assert.Empty(t, data["summaries"].([]interface{}))
		assert.Equal(t, float64(0), data["total_records"])
	})

	t.Run("unexpected repository failure returns 500 without leaking internal details", func(t *testing.T) {
		repo := &mockPaymentRepoForHandlerTest{
			summaryErr: errors.New("fatal: db timeout on summary query [sensitive_db_password]"),
		}
		router, _ := setupAdminPaymentTestRouter(repo, jwtService)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/payments/summary-by-event", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "INTERNAL_SERVER_ERROR", resp["code"])
		assert.Equal(t, "internal server error", resp["message"])
		assert.NotContains(t, rec.Body.String(), "sensitive_db_password")
	})
}
