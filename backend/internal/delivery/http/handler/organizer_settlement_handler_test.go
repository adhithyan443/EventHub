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
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOrganizerBalanceUsecase struct {
	projection *domain.OrganizerBalanceProjection
	err        error
	lastUserID uuid.UUID
}

func (m *mockOrganizerBalanceUsecase) GetOrganizerBalanceProjection(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.OrganizerBalanceProjection, error) {
	m.lastUserID = userID
	if m.err != nil {
		return nil, m.err
	}
	return m.projection, nil
}

type mockSettlementUsecase struct {
	createdSettlement *domain.Settlement
	listResult        *domain.SettlementListResult
	singleSettlement  *domain.Settlement
	requestErr        error
	listErr           error
	getErr            error
	approveErr        error
	rejectErr         error
	retryErr          error
}

func (m *mockSettlementUsecase) RequestSettlement(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, eventID *uuid.UUID) (*domain.Settlement, error) {
	if m.requestErr != nil {
		return nil, m.requestErr
	}
	return m.createdSettlement, nil
}

func (m *mockSettlementUsecase) ListOrganizerSettlements(ctx context.Context, userID uuid.UUID, page, limit int, status *domain.SettlementStatus) (*domain.SettlementListResult, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockSettlementUsecase) GetOrganizerSettlement(ctx context.Context, userID, settlementID uuid.UUID) (*domain.Settlement, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.singleSettlement, nil
}

func (m *mockSettlementUsecase) ListAdminSettlements(ctx context.Context, page, limit int, status *domain.SettlementStatus, organizerID *uuid.UUID) (*domain.SettlementListResult, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockSettlementUsecase) GetAdminSettlement(ctx context.Context, settlementID uuid.UUID) (*domain.Settlement, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.singleSettlement, nil
}

func (m *mockSettlementUsecase) ApproveSettlement(ctx context.Context, adminUserID, settlementID uuid.UUID) (*domain.Settlement, error) {
	if m.approveErr != nil {
		return nil, m.approveErr
	}
	return m.singleSettlement, nil
}

func (m *mockSettlementUsecase) RejectSettlement(ctx context.Context, adminUserID, settlementID uuid.UUID, reason string) (*domain.Settlement, error) {
	if m.rejectErr != nil {
		return nil, m.rejectErr
	}
	return m.singleSettlement, nil
}

func (m *mockSettlementUsecase) RetrySettlement(ctx context.Context, adminUserID, settlementID uuid.UUID) (*domain.Settlement, error) {
	if m.retryErr != nil {
		return nil, m.retryErr
	}
	return m.singleSettlement, nil
}

func setupOrganizerSettlementTestRouter(
	balUC domain.OrganizerBalanceUsecase,
	setUC domain.SettlementUsecase,
	logger *slog.Logger,
	authUserID *uuid.UUID,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler(logger))

	if authUserID != nil {
		router.Use(func(c *gin.Context) {
			c.Set("user_id", *authUserID)
			c.Next()
		})
	}

	h := handler.NewOrganizerSettlementHandler(balUC, setUC, logger)
	router.GET("/api/v1/organizers/settlements/balance", h.GetBalance)
	router.POST("/api/v1/organizers/settlements", h.RequestSettlement)
	router.GET("/api/v1/organizers/settlements", h.ListSettlements)
	router.GET("/api/v1/organizers/settlements/:id", h.GetSettlement)

	return router
}

func TestOrganizerSettlementHandler_GetBalance_MissingUserID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockBalUC := &mockOrganizerBalanceUsecase{}
	router := setupOrganizerSettlementTestRouter(mockBalUC, &mockSettlementUsecase{}, logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/settlements/balance", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOrganizerSettlementHandler_GetBalance_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	organizerID := uuid.New()
	userID := uuid.New()

	mockBalUC := &mockOrganizerBalanceUsecase{
		projection: &domain.OrganizerBalanceProjection{
			OrganizerID:      organizerID,
			AvailableBalance: decimal.NewFromInt(15000),
		},
	}

	router := setupOrganizerSettlementTestRouter(mockBalUC, &mockSettlementUsecase{}, logger, &userID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/settlements/balance", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestOrganizerSettlementHandler_RequestSettlement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	organizerID := uuid.New()
	settlementID := uuid.New()

	t.Run("Missing auth", func(t *testing.T) {
		router := setupOrganizerSettlementTestRouter(&mockOrganizerBalanceUsecase{}, &mockSettlementUsecase{}, logger, nil)
		body := bytes.NewBufferString(`{"amount":"5000.00"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizers/settlements", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid negative amount", func(t *testing.T) {
		router := setupOrganizerSettlementTestRouter(&mockOrganizerBalanceUsecase{}, &mockSettlementUsecase{}, logger, &userID)
		body := bytes.NewBufferString(`{"amount":"-100.00"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizers/settlements", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Insufficient available balance returns 409 Conflict", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			requestErr: domain.ErrInsufficientAvailableBalance,
		}
		router := setupOrganizerSettlementTestRouter(&mockOrganizerBalanceUsecase{}, mockSetUC, logger, &userID)
		body := bytes.NewBufferString(`{"amount":"50000.00"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizers/settlements", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("Successful settlement request creates 201 Created", func(t *testing.T) {
		now := time.Now().UTC()
		created := &domain.Settlement{
			ID:          settlementID,
			OrganizerID: organizerID,
			Amount:      decimal.NewFromInt(5000),
			Currency:    domainfinance.CurrencyINR,
			Status:      domain.SettlementStatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		mockSetUC := &mockSettlementUsecase{
			createdSettlement: created,
		}
		router := setupOrganizerSettlementTestRouter(&mockOrganizerBalanceUsecase{}, mockSetUC, logger, &userID)

		body := bytes.NewBufferString(`{"amount":"5000.00"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizers/settlements", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp struct {
			Success bool              `json:"success"`
			Data    domain.Settlement `json:"data"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, settlementID, resp.Data.ID)
		assert.Equal(t, domain.SettlementStatusPending, resp.Data.Status)
		assert.True(t, resp.Data.Amount.Equal(decimal.NewFromInt(5000)))
	})
}

func TestOrganizerSettlementHandler_ListAndGetSettlements(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	settlementID := uuid.New()

	t.Run("List settlements success", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			listResult: &domain.SettlementListResult{
				Settlements: []*domain.Settlement{
					{ID: settlementID, Amount: decimal.NewFromInt(1000), Status: domain.SettlementStatusPending},
				},
				Total: 1,
				Page:  1,
				Limit: 10,
			},
		}
		router := setupOrganizerSettlementTestRouter(&mockOrganizerBalanceUsecase{}, mockSetUC, logger, &userID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/settlements?page=1&limit=10", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Get settlement details success", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			singleSettlement: &domain.Settlement{
				ID:     settlementID,
				Amount: decimal.NewFromInt(1000),
				Status: domain.SettlementStatusCompleted,
			},
		}
		router := setupOrganizerSettlementTestRouter(&mockOrganizerBalanceUsecase{}, mockSetUC, logger, &userID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/settlements/"+settlementID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Get non-existent settlement returns 404", func(t *testing.T) {
		mockSetUC := &mockSettlementUsecase{
			getErr: domain.ErrSettlementNotFound,
		}
		router := setupOrganizerSettlementTestRouter(&mockOrganizerBalanceUsecase{}, mockSetUC, logger, &userID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/settlements/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
