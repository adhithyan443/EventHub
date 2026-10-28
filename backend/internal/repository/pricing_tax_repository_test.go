package repository

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestPricingTaxModel_GORMColumnMapping(t *testing.T) {
	namer := schema.NamingStrategy{}
	s, err := schema.Parse(&models.PricingTaxModel{}, &sync.Map{}, namer)
	require.NoError(t, err)

	assert.Equal(t, "pricing_tax_configurations", s.Table)

	fieldDBNames := make(map[string]string)
	for _, f := range s.Fields {
		fieldDBNames[f.Name] = f.DBName
	}

	assert.Equal(t, "id", fieldDBNames["ID"])
	assert.Equal(t, "version", fieldDBNames["Version"])
	assert.Equal(t, "status", fieldDBNames["Status"])
	assert.Equal(t, "platform_fee_enabled", fieldDBNames["PlatformFeeEnabled"])
	assert.Equal(t, "platform_fee_percent", fieldDBNames["PlatformFeePercent"])
	assert.Equal(t, "tax_enabled", fieldDBNames["TaxEnabled"])
	assert.Equal(t, "tax_percent", fieldDBNames["TaxPercent"])
	assert.Equal(t, "tax_basis", fieldDBNames["TaxBasis"])
	assert.Equal(t, "tax_rules_validated", fieldDBNames["TaxRulesValidated"])
	assert.Equal(t, "tax_legal_signoff_reference", fieldDBNames["TaxLegalSignoffReference"])
	assert.Equal(t, "free_ticket_exempt", fieldDBNames["FreeTicketExempt"])
	assert.Equal(t, "effective_from", fieldDBNames["EffectiveFrom"])
	assert.Equal(t, "effective_to", fieldDBNames["EffectiveTo"])
	assert.Equal(t, "cancelled_at", fieldDBNames["CancelledAt"])
	assert.Equal(t, "cancelled_by", fieldDBNames["CancelledBy"])
	assert.Equal(t, "created_by", fieldDBNames["CreatedBy"])
	assert.Equal(t, "change_reason", fieldDBNames["ChangeReason"])
	assert.Equal(t, "created_at", fieldDBNames["CreatedAt"])
	assert.Equal(t, "updated_at", fieldDBNames["UpdatedAt"])
}

func TestPricingTaxModel_MappingRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	legalRef := "LEGAL-MEMO-2026-GST-01"
	adminID := uuid.New()

	original := &domain.PricingTaxConfiguration{
		ID:                       uuid.New(),
		Version:                  3,
		Status:                   domain.PricingTaxStatusActive,
		PlatformFeeEnabled:       true,
		PlatformFeePercent:       decimal.NewFromFloat(3.50),
		TaxEnabled:               true,
		TaxPercent:               decimal.NewFromFloat(18.00),
		TaxBasis:                 domain.TaxBasisPlatformFeeOnly,
		TaxRulesValidated:        true,
		TaxLegalSignoffReference: &legalRef,
		FreeTicketExempt:         true,
		EffectiveFrom:            now,
		EffectiveTo:              nil,
		CreatedBy:                &adminID,
		ChangeReason:             "Annual pricing update",
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	model := models.PricingTaxModelFromDomain(original)
	require.NotNil(t, model)

	restored := model.ToDomain()
	require.NotNil(t, restored)

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.Version, restored.Version)
	assert.Equal(t, original.Status, restored.Status)
	assert.Equal(t, original.PlatformFeeEnabled, restored.PlatformFeeEnabled)
	assert.True(t, original.PlatformFeePercent.Equal(restored.PlatformFeePercent))
	assert.Equal(t, original.TaxEnabled, restored.TaxEnabled)
	assert.True(t, original.TaxPercent.Equal(restored.TaxPercent))
	assert.Equal(t, original.TaxBasis, restored.TaxBasis)
	assert.Equal(t, original.TaxRulesValidated, restored.TaxRulesValidated)
	assert.Equal(t, original.TaxLegalSignoffReference, restored.TaxLegalSignoffReference)
	assert.Equal(t, original.FreeTicketExempt, restored.FreeTicketExempt)
	assert.Equal(t, original.EffectiveFrom, restored.EffectiveFrom)
	assert.Nil(t, restored.EffectiveTo)
	assert.Equal(t, original.CreatedBy, restored.CreatedBy)
	assert.Equal(t, original.ChangeReason, restored.ChangeReason)
}

// Database Integration Tests
func getTestDB(t *testing.T) *gorm.DB {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=localhost port=5432 user=postgres password=root dbname=eventhub sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		t.Skip("PostgreSQL database not available for integration tests")
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil || sqlDB.Ping() != nil {
		t.Skip("PostgreSQL database ping failed; skipping integration tests")
		return nil
	}

	return db
}

func TestPricingTaxRepository_PostgreSQL_ExclusionAndAdvisoryLock(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	// Run inside a rolled-back transaction so no test data pollutes the database
	tx := db.Begin()
	defer tx.Rollback()

	// Ensure table exists in tx
	err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS public.pricing_tax_configurations (
			id UUID PRIMARY KEY,
			version INTEGER NOT NULL UNIQUE,
			status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
			platform_fee_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			platform_fee_percent NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
			tax_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			tax_percent NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
			tax_basis VARCHAR(32) NOT NULL DEFAULT 'UNVALIDATED',
			tax_rules_validated BOOLEAN NOT NULL DEFAULT FALSE,
			tax_legal_signoff_reference VARCHAR(128) NULL,
			free_ticket_exempt BOOLEAN NOT NULL DEFAULT TRUE,
			effective_from TIMESTAMP WITH TIME ZONE NOT NULL,
			effective_to TIMESTAMP WITH TIME ZONE NULL,
			cancelled_at TIMESTAMP WITH TIME ZONE NULL,
			cancelled_by UUID NULL,
			created_by UUID NULL,
			change_reason VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL
		);
	`).Error
	require.NoError(t, err)

	repo := NewPricingTaxRepository(tx, logger)

	// Test Advisory Lock
	err = repo.AcquireAdvisoryLock(ctx)
	require.NoError(t, err, "Acquiring transaction advisory lock must succeed")

	// Test ActiveAt query on empty table returns ErrPricingConfigurationUnavailable
	_, err = repo.FindActiveAt(ctx, time.Now().UTC())
	require.ErrorIs(t, err, domain.ErrPricingConfigurationUnavailable)
}
