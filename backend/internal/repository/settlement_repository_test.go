package repository_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettlementRepository_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	upSQL, err := loadMigrationSQL("000019_create_settlements.up.sql")
	require.NoError(t, err)
	require.NoError(t, db.Exec(upSQL).Error, "Failed to apply migration 000019_create_settlements.up.sql")

	repo := repository.NewSettlementRepository(db, slog.Default())
	ctx := context.Background()

	orgA := uuid.New()
	orgB := uuid.New()
	eventA1 := uuid.New()
	eventA2 := uuid.New()

	t.Run("Create and FindByID", func(t *testing.T) {
		sID := uuid.New()
		now := time.Now().UTC().Truncate(time.Second)
		s := &domain.Settlement{
			ID:          sID,
			OrganizerID: orgA,
			EventID:     &eventA1,
			Amount:      decimal.NewFromInt(5000),
			Currency:    "INR",
			Status:      domain.SettlementStatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		err := repo.Create(ctx, s)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, sID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, sID, found.ID)
		assert.Equal(t, orgA, found.OrganizerID)
		require.NotNil(t, found.EventID)
		assert.Equal(t, eventA1, *found.EventID)
		assert.True(t, found.Amount.Equal(decimal.NewFromInt(5000)))
		assert.Equal(t, domain.SettlementStatusPending, found.Status)

		// Non-existent ID returns ErrSettlementNotFound
		missing, err := repo.FindByID(ctx, uuid.New())
		assert.ErrorIs(t, err, domain.ErrSettlementNotFound)
		assert.Nil(t, missing)
	})

	t.Run("FindByIDForUpdate row lock", func(t *testing.T) {
		sID := uuid.New()
		now := time.Now().UTC().Truncate(time.Second)
		s := &domain.Settlement{
			ID:          sID,
			OrganizerID: orgA,
			Amount:      decimal.NewFromInt(2500),
			Currency:    "INR",
			Status:      domain.SettlementStatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		require.NoError(t, repo.Create(ctx, s))

		locked, err := repo.FindByIDForUpdate(ctx, sID)
		require.NoError(t, err)
		require.NotNil(t, locked)
		assert.Equal(t, sID, locked.ID)
	})

	t.Run("Update settlement status and payout reference", func(t *testing.T) {
		sID := uuid.New()
		now := time.Now().UTC().Truncate(time.Second)
		s := &domain.Settlement{
			ID:          sID,
			OrganizerID: orgA,
			Amount:      decimal.NewFromInt(12000),
			Currency:    "INR",
			Status:      domain.SettlementStatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		require.NoError(t, repo.Create(ctx, s))

		ref := "pout_mock_test_123"
		adminID := uuid.New()
		approvedAt := time.Now().UTC().Truncate(time.Second)
		processedAt := time.Now().UTC().Truncate(time.Second)

		s.Status = domain.SettlementStatusCompleted
		s.PayoutReference = &ref
		s.ApprovedBy = &adminID
		s.ApprovedAt = &approvedAt
		s.ProcessedAt = &processedAt

		require.NoError(t, repo.Update(ctx, s))

		found, err := repo.FindByID(ctx, sID)
		require.NoError(t, err)
		assert.Equal(t, domain.SettlementStatusCompleted, found.Status)
		require.NotNil(t, found.PayoutReference)
		assert.Equal(t, ref, *found.PayoutReference)
		require.NotNil(t, found.ApprovedBy)
		assert.Equal(t, adminID, *found.ApprovedBy)
	})

	t.Run("List with filtering and pagination", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		// Insert distinct records for orgA and orgB
		sOrgA := &domain.Settlement{
			ID:          uuid.New(),
			OrganizerID: orgA,
			Amount:      decimal.NewFromInt(3000),
			Currency:    "INR",
			Status:      domain.SettlementStatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		sOrgB := &domain.Settlement{
			ID:          uuid.New(),
			OrganizerID: orgB,
			Amount:      decimal.NewFromInt(4000),
			Currency:    "INR",
			Status:      domain.SettlementStatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		require.NoError(t, repo.Create(ctx, sOrgA))
		require.NoError(t, repo.Create(ctx, sOrgB))

		// Filter by OrgA
		resOrgA, err := repo.List(ctx, domain.SettlementFilter{
			OrganizerID: &orgA,
			Page:        1,
			Limit:       10,
		})
		require.NoError(t, err)
		require.NotNil(t, resOrgA)
		assert.GreaterOrEqual(t, resOrgA.Total, int64(1))
		for _, item := range resOrgA.Settlements {
			assert.Equal(t, orgA, item.OrganizerID, "List must only contain settlements for OrgA")
		}

		// Filter by OrgB
		resOrgB, err := repo.List(ctx, domain.SettlementFilter{
			OrganizerID: &orgB,
			Page:        1,
			Limit:       10,
		})
		require.NoError(t, err)
		require.NotNil(t, resOrgB)
		assert.Equal(t, int64(1), resOrgB.Total)
		assert.Equal(t, orgB, resOrgB.Settlements[0].OrganizerID)
	})

	t.Run("GetInFlightSettlementAmount calculates committed liabilities accurately", func(t *testing.T) {
		orgHoldTest := uuid.New()
		now := time.Now().UTC().Truncate(time.Second)

		// In-flight settlements: PENDING (5,000), APPROVED (3,000), PROCESSING (2,000) -> Total In-Flight = 10,000
		s1 := &domain.Settlement{ID: uuid.New(), OrganizerID: orgHoldTest, Amount: decimal.NewFromInt(5000), Currency: "INR", Status: domain.SettlementStatusPending, CreatedAt: now, UpdatedAt: now}
		s2 := &domain.Settlement{ID: uuid.New(), OrganizerID: orgHoldTest, Amount: decimal.NewFromInt(3000), Currency: "INR", Status: domain.SettlementStatusApproved, CreatedAt: now, UpdatedAt: now}
		s3 := &domain.Settlement{ID: uuid.New(), OrganizerID: orgHoldTest, Amount: decimal.NewFromInt(2000), Currency: "INR", Status: domain.SettlementStatusProcessing, CreatedAt: now, UpdatedAt: now}

		// Terminal / non-in-flight settlements: COMPLETED (8,000), REJECTED (4,000), FAILED (1,000) -> MUST NOT BE IN-FLIGHT
		s4 := &domain.Settlement{ID: uuid.New(), OrganizerID: orgHoldTest, Amount: decimal.NewFromInt(8000), Currency: "INR", Status: domain.SettlementStatusCompleted, CreatedAt: now, UpdatedAt: now}
		s5 := &domain.Settlement{ID: uuid.New(), OrganizerID: orgHoldTest, Amount: decimal.NewFromInt(4000), Currency: "INR", Status: domain.SettlementStatusRejected, CreatedAt: now, UpdatedAt: now}
		s6 := &domain.Settlement{ID: uuid.New(), OrganizerID: orgHoldTest, Amount: decimal.NewFromInt(1000), Currency: "INR", Status: domain.SettlementStatusFailed, CreatedAt: now, UpdatedAt: now}

		for _, item := range []*domain.Settlement{s1, s2, s3, s4, s5, s6} {
			require.NoError(t, repo.Create(ctx, item))
		}

		inFlight, err := repo.GetInFlightSettlementAmount(ctx, orgHoldTest)
		require.NoError(t, err)
		assert.True(t, inFlight.Equal(decimal.NewFromInt(10000)), "Expected in-flight 10000, got %s", inFlight)
	})

	t.Run("GetInFlightSettlementAmountForEvent isolates event liabilities", func(t *testing.T) {
		orgEvtTest := uuid.New()
		evtTarget := eventA2
		now := time.Now().UTC().Truncate(time.Second)

		// Target event settlements: PENDING (3,500), PROCESSING (1,500) -> 5,000
		s1 := &domain.Settlement{ID: uuid.New(), OrganizerID: orgEvtTest, EventID: &evtTarget, Amount: decimal.NewFromInt(3500), Currency: "INR", Status: domain.SettlementStatusPending, CreatedAt: now, UpdatedAt: now}
		s2 := &domain.Settlement{ID: uuid.New(), OrganizerID: orgEvtTest, EventID: &evtTarget, Amount: decimal.NewFromInt(1500), Currency: "INR", Status: domain.SettlementStatusProcessing, CreatedAt: now, UpdatedAt: now}

		// Other event or general settlements
		otherEvt := uuid.New()
		s3 := &domain.Settlement{ID: uuid.New(), OrganizerID: orgEvtTest, EventID: &otherEvt, Amount: decimal.NewFromInt(9000), Currency: "INR", Status: domain.SettlementStatusPending, CreatedAt: now, UpdatedAt: now}

		require.NoError(t, repo.Create(ctx, s1))
		require.NoError(t, repo.Create(ctx, s2))
		require.NoError(t, repo.Create(ctx, s3))

		inFlightEvt, err := repo.GetInFlightSettlementAmountForEvent(ctx, evtTarget)
		require.NoError(t, err)
		assert.True(t, inFlightEvt.Equal(decimal.NewFromInt(5000)), "Expected event in-flight 5000, got %s", inFlightEvt)
	})
}
