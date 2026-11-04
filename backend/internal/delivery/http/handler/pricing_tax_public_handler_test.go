package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/pricingtax"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPricingTaxRepoForHandlerTest struct {
	domain.PricingTaxConfigurationRepository
	activeConfig *domain.PricingTaxConfiguration
	err          error
}

func (m *mockPricingTaxRepoForHandlerTest) FindActiveAt(ctx context.Context, t time.Time) (*domain.PricingTaxConfiguration, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.activeConfig, nil
}

type mockTxReposForPricingTaxTest struct {
	domain.TransactionRepositories
	repo *mockPricingTaxRepoForHandlerTest
}

func (r *mockTxReposForPricingTaxTest) PricingTaxConfigurationRepository() domain.PricingTaxConfigurationRepository {
	return r.repo
}

type mockTxManagerForPricingTaxTest struct {
	repos *mockTxReposForPricingTaxTest
}

func (m *mockTxManagerForPricingTaxTest) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

func TestGetPublicActiveConfiguration_SanitizedFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	signoffRef := "LEGAL-2026-TAX-001"
	createdBy := uuid.New()
	activeConfig := &domain.PricingTaxConfiguration{
		ID:                       uuid.New(),
		Version:                  2,
		Status:                   domain.PricingTaxStatusActive,
		PlatformFeeEnabled:       true,
		PlatformFeePercent:       decimal.NewFromFloat(7.00),
		TaxEnabled:               true,
		TaxPercent:               decimal.NewFromFloat(18.00),
		TaxBasis:                 domain.TaxBasisSubtotalOnly,
		TaxRulesValidated:        true,
		TaxLegalSignoffReference: &signoffRef,
		FreeTicketExempt:         true,
		CreatedBy:                &createdBy,
		ChangeReason:             "Internal administrative rate revision",
		EffectiveFrom:            time.Now().Add(-24 * time.Hour),
		CreatedAt:                time.Now().Add(-24 * time.Hour),
		UpdatedAt:                time.Now(),
	}

	repo := &mockPricingTaxRepoForHandlerTest{activeConfig: activeConfig}
	txMgr := &mockTxManagerForPricingTaxTest{repos: &mockTxReposForPricingTaxTest{repo: repo}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := pricingtax.NewPricingTaxUsecase(txMgr, logger)
	h := handler.NewAdminPricingTaxHandler(uc, logger)

	r := gin.New()
	r.GET("/api/v1/pricing/active", h.GetPublicActiveConfiguration)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pricing/active", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                                   `json:"success"`
		Data    handler.PublicPricingTaxConfigResponse `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.True(t, resp.Data.PlatformFeeEnabled)
	assert.Equal(t, "7.00", resp.Data.PlatformFeePercent.StringFixed(2))
	assert.True(t, resp.Data.TaxEnabled)
	assert.Equal(t, "18.00", resp.Data.TaxPercent.StringFixed(2))
	assert.Equal(t, domain.TaxBasisSubtotalOnly, resp.Data.TaxBasis)
	assert.True(t, resp.Data.FreeTicketExempt)

	// Ensure sensitive fields are not in the JSON response
	bodyStr := w.Body.String()
	assert.NotContains(t, bodyStr, "LEGAL-2026-TAX-001")
	assert.NotContains(t, bodyStr, "Internal administrative rate revision")
	assert.NotContains(t, bodyStr, createdBy.String())
}

func TestGetPublicActiveConfiguration_TaxNotValidated_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	activeConfig := &domain.PricingTaxConfiguration{
		PlatformFeeEnabled: true,
		PlatformFeePercent: decimal.NewFromFloat(5.00),
		TaxEnabled:         true,
		TaxPercent:         decimal.NewFromFloat(12.00),
		TaxBasis:           domain.TaxBasisSubtotalOnly,
		TaxRulesValidated:  false, // NOT validated legally!
		FreeTicketExempt:   true,
	}

	repo := &mockPricingTaxRepoForHandlerTest{activeConfig: activeConfig}
	txMgr := &mockTxManagerForPricingTaxTest{repos: &mockTxReposForPricingTaxTest{repo: repo}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := pricingtax.NewPricingTaxUsecase(txMgr, logger)
	h := handler.NewAdminPricingTaxHandler(uc, logger)

	r := gin.New()
	r.GET("/api/v1/pricing/active", h.GetPublicActiveConfiguration)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pricing/active", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                                   `json:"success"`
		Data    handler.PublicPricingTaxConfigResponse `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	// Because TaxRulesValidated was false, TaxEnabled must be false and TaxPercent 0
	assert.False(t, resp.Data.TaxEnabled)
	assert.Equal(t, "0.00", resp.Data.TaxPercent.StringFixed(2))
}
