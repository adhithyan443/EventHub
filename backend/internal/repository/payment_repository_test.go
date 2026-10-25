package repository

import (
	"strings"
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

func TestPaymentModel_GORMColumnMapping(t *testing.T) {
	namer := schema.NamingStrategy{}
	s, err := schema.Parse(&models.PaymentModel{}, &sync.Map{}, namer)
	require.NoError(t, err)

	assert.Equal(t, "payments", s.Table)

	fieldDBNames := make(map[string]string)
	for _, f := range s.Fields {
		fieldDBNames[f.Name] = f.DBName
	}

	// Verify all columns match the canonical payments schema (migrations 000004, 000005, 000006)
	assert.Equal(t, "id", fieldDBNames["ID"])
	assert.Equal(t, "booking_id", fieldDBNames["BookingID"])
	assert.Equal(t, "transaction_id", fieldDBNames["TransactionID"])
	assert.Equal(t, "provider", fieldDBNames["Provider"])
	assert.Equal(t, "provider_order_id", fieldDBNames["ProviderOrderID"])
	assert.Equal(t, "payment_method", fieldDBNames["PaymentMethod"])
	assert.Equal(t, "amount", fieldDBNames["Amount"])
	assert.Equal(t, "status", fieldDBNames["Status"])
	assert.Equal(t, "paid_at", fieldDBNames["PaidAt"])
	assert.Equal(t, "created_at", fieldDBNames["CreatedAt"])
	assert.Equal(t, "updated_at", fieldDBNames["UpdatedAt"])

	// Explicitly verify DeletedAt and deleted_at do NOT exist
	assert.Nil(t, s.FieldsByName["DeletedAt"], "DeletedAt field must not exist on PaymentModel")
	for name, dbName := range fieldDBNames {
		assert.NotEqual(t, "deleted_at", dbName, "Field %s must not map to deleted_at", name)
	}
}

func TestPaymentModel_NoDeletedAtInGeneratedSQL(t *testing.T) {
	// Initialize GORM in DryRun mode without connecting to a live database
	db, err := gorm.Open(postgres.New(postgres.Config{
		WithoutReturning: true,
	}), &gorm.Config{
		DryRun:                 true,
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)

	bookingID := uuid.New().String()

	// 1. Find by booking query
	stmtFind := db.Session(&gorm.Session{DryRun: true}).
		Where("booking_id = ?", bookingID).
		Order("created_at DESC").
		Find(&[]models.PaymentModel{}).
		Statement
	require.NotEmpty(t, stmtFind.SQL.String())
	assert.False(t, strings.Contains(strings.ToLower(stmtFind.SQL.String()), "deleted_at"),
		"Find query must not contain deleted_at clause, got: %s", stmtFind.SQL.String())

	// 2. First by ID query
	stmtFirst := db.Session(&gorm.Session{DryRun: true}).
		First(&models.PaymentModel{}, "id = ?", uuid.New().String()).
		Statement
	require.NotEmpty(t, stmtFirst.SQL.String())
	assert.False(t, strings.Contains(strings.ToLower(stmtFirst.SQL.String()), "deleted_at"),
		"First query must not contain deleted_at clause, got: %s", stmtFirst.SQL.String())

	// 3. Create query
	stmtCreate := db.Session(&gorm.Session{DryRun: true}).
		Create(&models.PaymentModel{
			ID:            uuid.New().String(),
			BookingID:     bookingID,
			PaymentMethod: "CARD",
			Amount:        decimal.NewFromInt(500),
			Status:        "PENDING",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}).
		Statement
	require.NotEmpty(t, stmtCreate.SQL.String())
	assert.False(t, strings.Contains(strings.ToLower(stmtCreate.SQL.String()), "deleted_at"),
		"Create query must not contain deleted_at column, got: %s", stmtCreate.SQL.String())

	// 4. Update query
	stmtUpdate := db.Session(&gorm.Session{DryRun: true}).
		Model(&models.PaymentModel{}).
		Where("id = ?", uuid.New().String()).
		Updates(map[string]interface{}{
			"status": "SUCCESS",
		}).
		Statement
	require.NotEmpty(t, stmtUpdate.SQL.String())
	assert.False(t, strings.Contains(strings.ToLower(stmtUpdate.SQL.String()), "deleted_at"),
		"Update query must not contain deleted_at clause, got: %s", stmtUpdate.SQL.String())

	// 5. UpdateProviderOrderID conditional query
	stmtUpdateOrderID := db.Session(&gorm.Session{DryRun: true}).
		Model(&models.PaymentModel{}).
		Where("id = ? AND status = ?", uuid.New().String(), string(domain.PaymentStatusPending)).
		Update("provider_order_id", "order_12345").
		Statement
	require.NotEmpty(t, stmtUpdateOrderID.SQL.String())
	sqlStr := strings.ToLower(stmtUpdateOrderID.SQL.String())
	assert.Contains(t, sqlStr, "provider_order_id", "Must update provider_order_id column")
	assert.Contains(t, sqlStr, "status", "Must check status = PENDING")
}

func TestPaymentModel_MappingRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	txID := "tx_12345"
	orderID := "order_67890"

	domainPayment := &domain.Payment{
		ID:              uuid.New().String(),
		BookingID:       uuid.New().String(),
		TransactionID:   &txID,
		Provider:        domain.PaymentProviderRazorpay,
		ProviderOrderID: &orderID,
		Method:          domain.PaymentMethodCard,
		Amount:          decimal.NewFromInt(1250),
		Status:          domain.PaymentStatusPending,
		PaidAt:          &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	model := toPaymentModel(domainPayment)
	require.NotNil(t, model)

	assert.Equal(t, domainPayment.ID, model.ID)
	assert.Equal(t, domainPayment.BookingID, model.BookingID)
	assert.Equal(t, domainPayment.TransactionID, model.TransactionID)
	assert.Equal(t, string(domainPayment.Provider), model.Provider)
	assert.Equal(t, domainPayment.ProviderOrderID, model.ProviderOrderID)
	assert.Equal(t, string(domainPayment.Method), model.PaymentMethod)
	assert.True(t, domainPayment.Amount.Equal(model.Amount))
	assert.Equal(t, string(domainPayment.Status), model.Status)
	assert.Equal(t, domainPayment.PaidAt, model.PaidAt)
	assert.Equal(t, domainPayment.CreatedAt, model.CreatedAt)
	assert.Equal(t, domainPayment.UpdatedAt, model.UpdatedAt)

	restored := toPaymentDomain(model)
	require.NotNil(t, restored)

	assert.Equal(t, domainPayment.ID, restored.ID)
	assert.Equal(t, domainPayment.BookingID, restored.BookingID)
	assert.Equal(t, domainPayment.TransactionID, restored.TransactionID)
	assert.Equal(t, domainPayment.Provider, restored.Provider)
	assert.Equal(t, domainPayment.ProviderOrderID, restored.ProviderOrderID)
	assert.Equal(t, domainPayment.Method, restored.Method)
	assert.True(t, domainPayment.Amount.Equal(restored.Amount))
	assert.Equal(t, domainPayment.Status, restored.Status)
	assert.Equal(t, domainPayment.PaidAt, restored.PaidAt)
	assert.Equal(t, domainPayment.CreatedAt, restored.CreatedAt)
	assert.Equal(t, domainPayment.UpdatedAt, restored.UpdatedAt)
}

func TestPaymentRepository_MarkRefundRequired_DryRun(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		WithoutReturning: true,
	}), &gorm.Config{
		DryRun:                 true,
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)

	now := time.Now()
	stmt := db.Session(&gorm.Session{DryRun: true}).
		Model(&models.PaymentModel{}).
		Where(
			"id = ? AND status = ?",
			uuid.New().String(),
			string(domain.PaymentStatusPending),
		).
		Updates(map[string]interface{}{
			"transaction_id": "pay_tx_999",
			"status":         string(domain.PaymentStatusRefundRequired),
			"paid_at":        now,
		}).
		Statement

	require.NotEmpty(t, stmt.SQL.String())
	sqlStr := strings.ToLower(stmt.SQL.String())
	assert.Contains(t, sqlStr, "status", "Must include status in query")
	assert.Contains(t, sqlStr, "transaction_id", "Must set transaction_id")
	assert.Contains(t, sqlStr, "paid_at", "Must set paid_at")

	// Verify parameterized variables include the REFUND_REQUIRED status and transaction ID
	var foundStatus, foundTxID bool
	for _, v := range stmt.Vars {
		if v == string(domain.PaymentStatusRefundRequired) {
			foundStatus = true
		}
		if v == "pay_tx_999" {
			foundTxID = true
		}
	}
	assert.True(t, foundStatus, "Vars must include REFUND_REQUIRED status")
	assert.True(t, foundTxID, "Vars must include transaction_id")
}
