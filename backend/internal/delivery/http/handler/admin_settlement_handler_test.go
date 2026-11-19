package handler_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func setupAdminSettlementTestRouter(
	setUC domain.SettlementUsecase,
	logger *slog.Logger,
	adminUserID *uuid.UUID,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler(logger))

	if adminUserID != nil {
		router.Use(func(c *gin.Context) {
			c.Set("user_id", *adminUserID)
			c.Next()
		})
	}

	h := handler.NewAdminSettlementHandler(setUC, logger)
	router.GET("/api/v1/admin/settlements", h.ListSettlements)
	router.GET("/api/v1/admin/settlements/:id", h.GetSettlement)
	router.POST("/api/v1/admin/settlements/:id/approve", h.ApproveSettlement)
	router.POST("/api/v1/admin/settlements/:id/reject", h.RejectSettlement)
	router.POST("/api/v1/admin/settlements/:id/retry", h.RetrySettlement)

	return router
}

func TestAdminSettlementHandler_ListAndGet(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminUserID := uuid.New()
	settlementID := uuid.New()

	t.Run("List settlements success", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			listResult: &domain.SettlementListResult{
				Settlements: []*domain.Settlement{
					{ID: settlementID, Amount: decimal.NewFromInt(5000), Status: domain.SettlementStatusPending},
				},
				Total: 1,
				Page:  1,
				Limit: 10,
			},
		}
		router := setupAdminSettlementTestRouter(mockSetUC, logger, &adminUserID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settlements?page=1&limit=10&status=PENDING", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Get settlement success", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			singleSettlement: &domain.Settlement{
				ID:     settlementID,
				Amount: decimal.NewFromInt(5000),
				Status: domain.SettlementStatusPending,
			},
		}
		router := setupAdminSettlementTestRouter(mockSetUC, logger, &adminUserID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settlements/"+settlementID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Get non-existent settlement returns 404", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			getErr: domain.ErrSettlementNotFound,
		}
		router := setupAdminSettlementTestRouter(mockSetUC, logger, &adminUserID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settlements/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestAdminSettlementHandler_ApproveSettlement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminUserID := uuid.New()
	settlementID := uuid.New()
	ref := "pout_mock_123456"

	t.Run("Approve settlement success", func(t *testing.T) {
		now := time.Now().UTC()
		mockSetUC := &mockSettlementUsecase{
			singleSettlement: &domain.Settlement{
				ID:              settlementID,
				Status:          domain.SettlementStatusCompleted,
				ApprovedBy:      &adminUserID,
				ApprovedAt:      &now,
				ProcessedAt:     &now,
				PayoutReference: &ref,
			},
		}
		router := setupAdminSettlementTestRouter(mockSetUC, logger, &adminUserID)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settlements/"+settlementID.String()+"/approve", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Approve invalid status transition returns 409", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			approveErr: domain.ErrInvalidSettlementStatusTransition,
		}
		router := setupAdminSettlementTestRouter(mockSetUC, logger, &adminUserID)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settlements/"+settlementID.String()+"/approve", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("Approve when payout provider fails returns 502", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			approveErr: domain.ErrPayoutFailed,
		}
		router := setupAdminSettlementTestRouter(mockSetUC, logger, &adminUserID)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settlements/"+settlementID.String()+"/approve", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadGateway, rec.Code)
	})
}

func TestAdminSettlementHandler_RejectSettlement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminUserID := uuid.New()
	settlementID := uuid.New()

	t.Run("Reject missing reason returns 400", func(t *testing.T) {
		router := setupAdminSettlementTestRouter(&mockSettlementUsecase{}, logger, &adminUserID)

		body := bytes.NewBufferString(`{"reason":""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settlements/"+settlementID.String()+"/reject", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Reject success returns 200", func(t *testing.T) {
		reason := "Suspected duplicate claim"
		mockSetUC := &mockSettlementUsecase{
			singleSettlement: &domain.Settlement{
				ID:              settlementID,
				Status:          domain.SettlementStatusRejected,
				RejectionReason: &reason,
			},
		}
		router := setupAdminSettlementTestRouter(mockSetUC, logger, &adminUserID)

		body := bytes.NewBufferString(`{"reason":"Suspected duplicate claim"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settlements/"+settlementID.String()+"/reject", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestAdminSettlementHandler_RetrySettlement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminUserID := uuid.New()
	settlementID := uuid.New()

	t.Run("Retry success", func(t *testing.T) {
		ref := "pout_mock_retry_success"
		mockSetUC := &mockSettlementUsecase{
			singleSettlement: &domain.Settlement{
				ID:              settlementID,
				Status:          domain.SettlementStatusCompleted,
				PayoutReference: &ref,
			},
		}
		router := setupAdminSettlementTestRouter(mockSetUC, logger, &adminUserID)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settlements/"+settlementID.String()+"/retry", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Retry invalid status returns 409", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			retryErr: domain.ErrInvalidSettlementStatusTransition,
		}
		router := setupAdminSettlementTestRouter(mockSetUC, logger, &adminUserID)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settlements/"+settlementID.String()+"/retry", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}
