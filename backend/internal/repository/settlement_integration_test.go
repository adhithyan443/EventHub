package repository_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/adhithyan443/EventHub/backend/internal/service/payout"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/settlement"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrganizerRepoForIntegration provides minimal organizer lookup for integration tests.
type mockOrganizerRepoForIntegration struct {
	domain.OrganizerRepository
	organizer *domain.Organizer
}

func (m *mockOrganizerRepoForIntegration) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if m.organizer != nil && m.organizer.UserID == userID {
		return m.organizer, nil
	}
	return nil, domain.ErrOrganizerNotFound
}

// mockEventRepoForIntegration provides empty or completed events.
type mockEventRepoForIntegration struct {
	domain.EventRepository
	events []*domain.OrganizerEvent
}

func (m *mockEventRepoForIntegration) FindByOrganizerID(organizerID uuid.UUID, page, limit int, search, status string) ([]*domain.OrganizerEvent, int64, error) {
	return m.events, int64(len(m.events)), nil
}

// mockBankAccountRepoForIntegration provides bank account info.
type mockBankAccountRepoForIntegration struct {
	domain.OrganizerBankAccountRepository
	bankAccount *domain.OrganizerBankAccount
}

func (m *mockBankAccountRepoForIntegration) FindByOrganizerID(organizerID uuid.UUID) (*domain.OrganizerBankAccount, error) {
	return m.bankAccount, nil
}

// mockRefundRepoForIntegration provides empty refunds.
type mockRefundRepoForIntegration struct {
	domain.RefundRepository
}

func (m *mockRefundRepoForIntegration) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Refund, error) {
	return nil, nil
}

func TestSettlementWorkflow_EndToEnd_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	acc1010, acc2010 := setupAccounts(t, db)

	upSQL, err := loadMigrationSQL("000019_create_settlements.up.sql")
	require.NoError(t, err)
	require.NoError(t, db.Exec(upSQL).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS organizers (
			id uuid PRIMARY KEY,
			user_id uuid NOT NULL,
			status character varying(30) NOT NULL,
			created_at timestamp with time zone NOT NULL,
			updated_at timestamp with time zone NOT NULL
		);
	`).Error)

	logger := slog.Default()
	ctx := context.Background()

	// 2. Initialize Repositories and Transaction Manager
	journalRepo := repository.NewFinanceJournalRepository(db, logger)
	settlementRepo := repository.NewSettlementRepository(db, logger)
	txManager := repository.NewTransactionManager(db, logger)

	// 3. Setup test organizer and event
	userID := uuid.New()
	adminID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()

	require.NoError(t, db.Exec(`
		INSERT INTO organizers (id, user_id, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', NOW(), NOW())
	`, organizerID, userID).Error)

	oRepo := &mockOrganizerRepoForIntegration{
		organizer: &domain.Organizer{
			ID:     organizerID,
			UserID: userID,
			Status: "ACTIVE",
		},
	}
	bRepo := &mockBankAccountRepoForIntegration{
		bankAccount: &domain.OrganizerBankAccount{
			ID:                 uuid.New(),
			OrganizerID:        organizerID,
			AccountNumberLast4: "5678",
			IFSCCode:           "HDFC0000001",
		},
	}

	pastDate := time.Now().UTC().AddDate(0, 0, -5)
	startTime := "10:00:00"
	endTime := "18:00:00"
	eRepo := &mockEventRepoForIntegration{
		events: []*domain.OrganizerEvent{
			{
				ID:          eventID,
				OrganizerID: organizerID,
				Title:       "Completed Music Festival",
				Status:      domain.EventStatusPublished,
				EventDate:   pastDate,
				StartTime:   &startTime,
				EndTime:     &endTime,
			},
		},
	}
	rRepo := &mockRefundRepoForIntegration{}

	// 4. Seed initial ticket sale in the ledger: ₹20,000 INR
	now := time.Now().UTC().Truncate(time.Second)
	saleAmt, _ := domainfinance.NewMoney(decimal.NewFromInt(20000), domainfinance.CurrencyINR)
	dSale, _ := domainfinance.NewDebitEntry(uuid.New(), acc1010, saleAmt, nil, nil, now)
	cSale, _ := domainfinance.NewCreditEntry(uuid.New(), acc2010, saleAmt, &organizerID, &eventID, now)
	saleJournal := buildValidJournal(t, uuid.New(), domainfinance.TransactionTypePayment,
		domainfinance.FinancialEventTicketSaleCaptured, domainfinance.SourceTypePayment, "pay_e2e_1",
		"idem_sale_e2e_"+uuid.New().String(), domainfinance.CurrencyINR, "Ticket sale 20k", nil, now, dSale, cSale)
	_, err = journalRepo.RecordJournal(ctx, saleJournal)
	require.NoError(t, err)

	// 5. Initialize OrganizerBalanceUsecase and wire settlementRepo
	balanceUC, err := settlement.NewOrganizerBalanceUsecase(journalRepo, oRepo, eRepo, bRepo, rRepo, logger)
	require.NoError(t, err)
	balanceUC.SetSettlementRepository(settlementRepo)

	// Initial balance check: ₹20,000 available
	proj0, err := balanceUC.GetOrganizerBalanceProjection(ctx, userID)
	require.NoError(t, err)
	assert.True(t, proj0.OrganizerPayable.Equal(decimal.NewFromInt(20000)))
	assert.True(t, proj0.AvailableBalance.Equal(decimal.NewFromInt(20000)))
	assert.True(t, proj0.InFlightSettlementHold.IsZero())

	// 6. Initialize SettlementUsecase with MockPayoutProvider
	payoutProvider := payout.NewMockPayoutProvider()
	settlementUC, err := settlement.NewSettlementUsecase(txManager, settlementRepo, oRepo, bRepo, balanceUC, payoutProvider, logger)
	require.NoError(t, err)

	// 7. Step A: Organizer requests settlement of ₹12,000
	s1, err := settlementUC.RequestSettlement(ctx, userID, decimal.NewFromInt(12000), nil)
	require.NoError(t, err)
	require.NotNil(t, s1)
	assert.Equal(t, domain.SettlementStatusPending, s1.Status)
	assert.True(t, s1.Amount.Equal(decimal.NewFromInt(12000)))

	// 8. Step B: In-Flight Hold & Concurrency Overspending Protection
	// Project balance again: Available should now be ₹8,000 (20k - 12k hold)!
	proj1, err := balanceUC.GetOrganizerBalanceProjection(ctx, userID)
	require.NoError(t, err)
	assert.True(t, proj1.OrganizerPayable.Equal(decimal.NewFromInt(20000)))
	assert.True(t, proj1.InFlightSettlementHold.Equal(decimal.NewFromInt(12000)))
	assert.True(t, proj1.AvailableBalance.Equal(decimal.NewFromInt(8000)), "Available balance must be reduced by in-flight settlement hold")

	// Attempt second settlement exceeding remaining available balance (e.g. ₹10,000 > ₹8,000)
	_, err = settlementUC.RequestSettlement(ctx, userID, decimal.NewFromInt(10000), nil)
	assert.ErrorIs(t, err, domain.ErrInsufficientAvailableBalance, "Concurrent/subsequent overspending must be rejected")

	// 9. Step C: Admin approves the ₹12,000 settlement
	approved, err := settlementUC.ApproveSettlement(ctx, adminID, s1.ID)
	require.NoError(t, err)
	require.NotNil(t, approved)
	assert.Equal(t, domain.SettlementStatusCompleted, approved.Status)
	assert.NotNil(t, approved.PayoutReference)
	assert.Contains(t, *approved.PayoutReference, "pout_mock_")
	assert.NotNil(t, approved.ProcessedAt)

	// 10. Step D: Verify Financial Ledger Posting & Subledger Balances
	subledger, err := journalRepo.GetOrganizerBalance(ctx, organizerID)
	require.NoError(t, err)
	assert.True(t, subledger.GrossEarnings.Equal(decimal.NewFromInt(20000)))
	assert.True(t, subledger.SettledAmount.Equal(decimal.NewFromInt(12000)), "Settled amount in ledger must be ₹12,000")
	assert.True(t, subledger.NetPayable.Equal(decimal.NewFromInt(8000)), "Net payable to organizer in ledger must now be ₹8,000")
	assert.True(t, subledger.TotalCredits.Equal(decimal.NewFromInt(20000)))
	assert.True(t, subledger.TotalDebits.Equal(decimal.NewFromInt(12000)))

	// 11. Step E: Verify Final Projection
	projFinal, err := balanceUC.GetOrganizerBalanceProjection(ctx, userID)
	require.NoError(t, err)
	assert.True(t, projFinal.OrganizerPayable.Equal(decimal.NewFromInt(8000)))
	assert.True(t, projFinal.SettledAmount.Equal(decimal.NewFromInt(12000)))
	assert.True(t, projFinal.InFlightSettlementHold.IsZero(), "Settlement is completed, in-flight hold is released")
	assert.True(t, projFinal.AvailableBalance.Equal(decimal.NewFromInt(8000)))

	// 12. Step F: State Machine Guard (Re-approving completed settlement must fail)
	_, err = settlementUC.ApproveSettlement(ctx, adminID, s1.ID)
	assert.ErrorIs(t, err, domain.ErrInvalidSettlementStatusTransition)
}
