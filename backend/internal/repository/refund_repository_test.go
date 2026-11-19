package repository

import (
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestRefundModel_TableName(t *testing.T) {
	m := models.RefundModel{}
	assert.Equal(t, "refunds", m.TableName())
}

func TestRefundModel_MappingRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	gatewayID := "rfnd_test_roundtrip"
	d := &domain.Refund{
		ID:              uuid.New(),
		BookingID:       uuid.New(),
		PaymentID:       uuid.New(),
		GatewayRefundID: &gatewayID,
		Amount:          decimal.NewFromFloat(499.50),
		Status:          domain.RefundStatusProcessing,
		Reason:          "CUSTOMER_CANCELLATION",
		CreatedAt:       now,
		UpdatedAt:       now,
		ProcessedAt:     &now,
	}

	model := toRefundModel(d)
	assert.Equal(t, d.ID, model.ID)
	assert.Equal(t, d.BookingID, model.BookingID)
	assert.Equal(t, d.PaymentID, model.PaymentID)
	assert.Equal(t, d.GatewayRefundID, model.GatewayRefundID)
	assert.True(t, d.Amount.Equal(model.Amount))
	assert.Equal(t, d.Status, model.Status)
	assert.Equal(t, d.Reason, model.Reason)
	assert.Equal(t, d.CreatedAt, model.CreatedAt)
	assert.Equal(t, d.UpdatedAt, model.UpdatedAt)
	assert.Equal(t, d.ProcessedAt, model.ProcessedAt)

	roundtrip := toRefundDomain(model)
	assert.Equal(t, d.ID, roundtrip.ID)
	assert.Equal(t, d.BookingID, roundtrip.BookingID)
	assert.Equal(t, d.PaymentID, roundtrip.PaymentID)
	assert.Equal(t, d.GatewayRefundID, roundtrip.GatewayRefundID)
	assert.True(t, d.Amount.Equal(roundtrip.Amount))
	assert.Equal(t, d.Status, roundtrip.Status)
	assert.Equal(t, d.Reason, roundtrip.Reason)
	assert.Equal(t, d.CreatedAt, roundtrip.CreatedAt)
	assert.Equal(t, d.UpdatedAt, roundtrip.UpdatedAt)
	assert.Equal(t, d.ProcessedAt, roundtrip.ProcessedAt)
}
