package repository

import (
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

func TestBookingModel_GORMColumnMapping(t *testing.T) {
	namer := schema.NamingStrategy{}
	s, err := schema.Parse(&models.BookingModel{}, &sync.Map{}, namer)
	require.NoError(t, err)

	assert.Equal(t, "bookings", s.Table)

	fieldDBNames := make(map[string]string)
	for _, f := range s.Fields {
		fieldDBNames[f.Name] = f.DBName
	}

	// Verify monetary and core columns match the documented bookings SQL schema (000002_create_bookings.up.sql)
	assert.Equal(t, "subtotal", fieldDBNames["Subtotal"])
	assert.Equal(t, "platform_fee", fieldDBNames["PlatformFee"])
	assert.Equal(t, "tax", fieldDBNames["Tax"], "Tax field must map to column 'tax', not 'tax_amount'")
	assert.Equal(t, "discount", fieldDBNames["Discount"], "Discount field must map to column 'discount', not 'discount_amount'")
	assert.Equal(t, "total", fieldDBNames["Total"], "Total field must map to column 'total', not 'total_amount'")
	assert.Equal(t, "booking_number", fieldDBNames["BookingNumber"])
	assert.Equal(t, "user_id", fieldDBNames["UserID"])
	assert.Equal(t, "event_id", fieldDBNames["EventID"])
	assert.Equal(t, "reservation_id", fieldDBNames["ReservationID"])
	assert.Equal(t, "status", fieldDBNames["Status"])

	// Ensure no obsolete _amount column names exist in the schema
	for name, dbName := range fieldDBNames {
		assert.NotEqual(t, "tax_amount", dbName, "Field %s mapped to invalid column tax_amount", name)
		assert.NotEqual(t, "discount_amount", dbName, "Field %s mapped to invalid column discount_amount", name)
		assert.NotEqual(t, "total_amount", dbName, "Field %s mapped to invalid column total_amount", name)
	}
}

func TestBookingModel_MappingRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	reservationID := uuid.New()

	original := &domain.Booking{
		ID:            uuid.New(),
		BookingNumber: "EVH-TEST123456",
		UserID:        uuid.New(),
		EventID:       uuid.New(),
		ReservationID: &reservationID,
		Subtotal:      decimal.NewFromInt(1000),
		PlatformFee:   decimal.NewFromInt(50),
		Tax:           decimal.NewFromInt(180),
		Discount:      decimal.NewFromInt(100),
		Total:         decimal.NewFromInt(1130),
		Status:        domain.BookingStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	model := bookingToModel(original)
	require.NotNil(t, model)

	assert.Equal(t, original.ID, model.ID)
	assert.Equal(t, original.BookingNumber, model.BookingNumber)
	assert.Equal(t, original.UserID, model.UserID)
	assert.Equal(t, original.EventID, model.EventID)
	assert.Equal(t, original.ReservationID, model.ReservationID)
	assert.True(t, original.Subtotal.Equal(model.Subtotal))
	assert.True(t, original.PlatformFee.Equal(model.PlatformFee))
	assert.True(t, original.Tax.Equal(model.Tax))
	assert.True(t, original.Discount.Equal(model.Discount))
	assert.True(t, original.Total.Equal(model.Total))
	assert.Equal(t, original.Status, model.Status)

	restored := bookingFromModel(model)
	require.NotNil(t, restored)

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.BookingNumber, restored.BookingNumber)
	assert.Equal(t, original.UserID, restored.UserID)
	assert.Equal(t, original.EventID, restored.EventID)
	assert.Equal(t, original.ReservationID, restored.ReservationID)
	assert.True(t, original.Subtotal.Equal(restored.Subtotal))
	assert.True(t, original.PlatformFee.Equal(restored.PlatformFee))
	assert.True(t, original.Tax.Equal(restored.Tax))
	assert.True(t, original.Discount.Equal(restored.Discount))
	assert.True(t, original.Total.Equal(restored.Total))
	assert.Equal(t, original.Status, restored.Status)
}
