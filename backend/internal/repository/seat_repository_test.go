package repository

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestSeatModel_GORMColumnMapping(t *testing.T) {
	namer := schema.NamingStrategy{}
	s, err := schema.Parse(&models.SeatModel{}, &sync.Map{}, namer)
	require.NoError(t, err)

	assert.Equal(t, "seats", s.Table)

	fieldDBNames := make(map[string]string)
	for _, f := range s.Fields {
		fieldDBNames[f.Name] = f.DBName
	}

	assert.Equal(t, "id", fieldDBNames["ID"])
	assert.Equal(t, "row_id", fieldDBNames["RowID"])
	assert.Equal(t, "column_id", fieldDBNames["ColumnID"])
	assert.Equal(t, "grid_col", fieldDBNames["GridCol"])
	assert.Equal(t, "seat_number", fieldDBNames["SeatNumber"])
	assert.Equal(t, "status", fieldDBNames["Status"])
	assert.Equal(t, "created_at", fieldDBNames["CreatedAt"])

	gridColField := s.FieldsByName["GridCol"]
	require.NotNil(t, gridColField)
	assert.True(t, gridColField.NotNull, "grid_col must be not null")
	// Regression guard: ensure default:1 is NOT present so zero values are not replaced
	assert.NotEqual(t, "1", gridColField.DefaultValue, "grid_col must not define default:1 in GORM tag")
	assert.False(t, gridColField.HasDefaultValue, "grid_col should not rely on GORM struct tag default")
}

func TestSeatRepository_Create_PreservesZeroGridCol(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		WithoutReturning: true,
	}), &gorm.Config{
		DryRun:                 true,
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)

	repo := NewSeatRepository(db, slog.Default())

	rowID := uuid.New()
	colID := uuid.New()
	seat := &domain.Seat{
		ID:         uuid.New(),
		RowID:      rowID,
		ColumnID:   &colID,
		GridCol:    0, // Explicit zero value
		SeatNumber: 1,
		Status:     domain.SeatStatusAvailable,
		CreatedAt:  time.Now(),
	}

	// 1. Verify via repository Create in dry run mode
	err = repo.Create(seat)
	require.NoError(t, err)
	assert.Equal(t, 0, seat.GridCol, "Seat.GridCol must remain 0 after Create")

	// 2. Inspect the exact SQL statement and bound variables to guarantee GORM does not substitute 1
	seatModel := toSeatModel(seat)
	stmt := db.Session(&gorm.Session{DryRun: true}).Create(seatModel).Statement

	// Find the index of grid_col in the generated column list
	var gridColIdx int = -1
	for idx, col := range stmt.Schema.DBNames {
		if col == "grid_col" {
			gridColIdx = idx
			break
		}
	}
	require.True(t, gridColIdx >= 0, "grid_col column must be present in statement schema")

	// Extract the bound parameter for grid_col from stmt.Vars
	// The fields in INSERT are: id, row_id, column_id, grid_col, seat_number, status, created_at
	var boundGridColVal interface{}
	for i, f := range stmt.Schema.Fields {
		if f.DBName == "grid_col" && i < len(stmt.Vars) {
			boundGridColVal = stmt.Vars[i]
			break
		}
	}
	require.NotNil(t, boundGridColVal, "bound variable for grid_col must exist")
	assert.Equal(t, 0, boundGridColVal, "GORM must bind 0 (not 1) for GridCol = 0")
}

func TestSeatRepository_Create_PreservesOneGridCol(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		WithoutReturning: true,
	}), &gorm.Config{
		DryRun:                 true,
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)

	repo := NewSeatRepository(db, slog.Default())

	rowID := uuid.New()
	colID := uuid.New()
	seat := &domain.Seat{
		ID:         uuid.New(),
		RowID:      rowID,
		ColumnID:   &colID,
		GridCol:    1, // Explicit 1 value
		SeatNumber: 2,
		Status:     domain.SeatStatusAvailable,
		CreatedAt:  time.Now(),
	}

	err = repo.Create(seat)
	require.NoError(t, err)
	assert.Equal(t, 1, seat.GridCol, "Seat.GridCol must remain 1 after Create")

	seatModel := toSeatModel(seat)
	stmt := db.Session(&gorm.Session{DryRun: true}).Create(seatModel).Statement

	var boundGridColVal interface{}
	for i, f := range stmt.Schema.Fields {
		if f.DBName == "grid_col" && i < len(stmt.Vars) {
			boundGridColVal = stmt.Vars[i]
			break
		}
	}
	require.NotNil(t, boundGridColVal, "bound variable for grid_col must exist")
	assert.Equal(t, 1, boundGridColVal, "GORM must bind 1 for GridCol = 1")
}

func TestSeatRepository_MappingRoundtrip(t *testing.T) {
	colID := uuid.New()
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name    string
		gridCol int
	}{
		{name: "zero grid_col", gridCol: 0},
		{name: "one grid_col", gridCol: 1},
		{name: "arbitrary grid_col", gridCol: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainSeat := &domain.Seat{
				ID:         uuid.New(),
				RowID:      uuid.New(),
				ColumnID:   &colID,
				GridCol:    tt.gridCol,
				SeatNumber: 1,
				Status:     domain.SeatStatusAvailable,
				CreatedAt:  now,
			}

			model := toSeatModel(domainSeat)
			assert.Equal(t, tt.gridCol, model.GridCol)

			roundtripDomain := toSeatDomain(model)
			assert.Equal(t, tt.gridCol, roundtripDomain.GridCol)
			assert.Equal(t, domainSeat.ID, roundtripDomain.ID)
			assert.Equal(t, domainSeat.RowID, roundtripDomain.RowID)
			assert.Equal(t, domainSeat.ColumnID, roundtripDomain.ColumnID)
			assert.Equal(t, domainSeat.SeatNumber, roundtripDomain.SeatNumber)
			assert.Equal(t, domainSeat.Status, roundtripDomain.Status)
		})
	}
}
