package settlement_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/settlement"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Repositories ---

type mockFinanceJournalRepo struct {
	domain.FinanceJournalRepository
	subledgerBalance *domain.OrganizerSubledgerBalance
	eventBalances    []domain.EventSubledgerBalance
	singleEventBal   *domain.EventSubledgerBalance
	getOrgBalErr     error
	getEventBalsErr  error
	getSingleBalErr  error
}

func (m *mockFinanceJournalRepo) GetOrganizerBalance(ctx context.Context, organizerID uuid.UUID) (*domain.OrganizerSubledgerBalance, error) {
	if m.getOrgBalErr != nil {
		return nil, m.getOrgBalErr
	}
	return m.subledgerBalance, nil
}

func (m *mockFinanceJournalRepo) GetOrganizerEventBalances(ctx context.Context, organizerID uuid.UUID) ([]domain.EventSubledgerBalance, error) {
	if m.getEventBalsErr != nil {
		return nil, m.getEventBalsErr
	}
	return m.eventBalances, nil
}

func (m *mockFinanceJournalRepo) GetEventBalance(ctx context.Context, organizerID, eventID uuid.UUID) (*domain.EventSubledgerBalance, error) {
	if m.getSingleBalErr != nil {
		return nil, m.getSingleBalErr
	}
	return m.singleEventBal, nil
}

type mockOrganizerRepo struct {
	domain.OrganizerRepository
	organizer   *domain.Organizer
	findByIDErr error
}

func (m *mockOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.organizer, nil
}

func (m *mockOrganizerRepo) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.organizer, nil
}

func (m *mockOrganizerRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Organizer, error) {
	return m.FindByID(id)
}

type mockEventRepo struct {
	domain.EventRepository
	events  []*domain.OrganizerEvent
	total   int64
	findErr error
}

func (m *mockEventRepo) FindByOrganizerID(organizerID uuid.UUID, page, limit int, search, status string) ([]*domain.OrganizerEvent, int64, error) {
	if m.findErr != nil {
		return nil, 0, m.findErr
	}
	return m.events, m.total, nil
}

type mockBankAccountRepo struct {
	domain.OrganizerBankAccountRepository
	bankAccount *domain.OrganizerBankAccount
	findErr     error
}

func (m *mockBankAccountRepo) FindByOrganizerID(organizerID uuid.UUID) (*domain.OrganizerBankAccount, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.bankAccount, nil
}

type mockRefundRepo struct {
	domain.RefundRepository
	refundsByEvent map[uuid.UUID][]domain.Refund
	findErr        error
}

func (m *mockRefundRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Refund, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.refundsByEvent[eventID], nil
}

// --- Tests ---

func TestNewOrganizerBalanceUsecase_Validation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	jRepo := &mockFinanceJournalRepo{}
	oRepo := &mockOrganizerRepo{}
	eRepo := &mockEventRepo{}
	bRepo := &mockBankAccountRepo{}
	rRepo := &mockRefundRepo{}

	_, err := settlement.NewOrganizerBalanceUsecase(nil, oRepo, eRepo, bRepo, rRepo, logger)
	assert.ErrorIs(t, err, settlement.ErrNilFinanceJournalRepository)

	_, err = settlement.NewOrganizerBalanceUsecase(jRepo, nil, eRepo, bRepo, rRepo, logger)
	assert.ErrorIs(t, err, settlement.ErrNilOrganizerRepository)

	_, err = settlement.NewOrganizerBalanceUsecase(jRepo, oRepo, nil, bRepo, rRepo, logger)
	assert.ErrorIs(t, err, settlement.ErrNilEventRepository)

	_, err = settlement.NewOrganizerBalanceUsecase(jRepo, oRepo, eRepo, nil, rRepo, logger)
	assert.ErrorIs(t, err, settlement.ErrNilOrganizerBankAccountRepo)

	_, err = settlement.NewOrganizerBalanceUsecase(jRepo, oRepo, eRepo, bRepo, nil, logger)
	assert.ErrorIs(t, err, settlement.ErrNilRefundRepository)

	u, err := settlement.NewOrganizerBalanceUsecase(jRepo, oRepo, eRepo, bRepo, rRepo, nil)
	assert.NoError(t, err)
	assert.NotNil(t, u)
}

func TestOrganizerBalanceUsecase_GetOrganizerBalanceProjection_NilUserID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	u, err := settlement.NewOrganizerBalanceUsecase(
		&mockFinanceJournalRepo{},
		&mockOrganizerRepo{},
		&mockEventRepo{},
		&mockBankAccountRepo{},
		&mockRefundRepo{},
		logger,
	)
	require.NoError(t, err)

	_, err = u.GetOrganizerBalanceProjection(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, settlement.ErrInvalidUserID)
}

func TestOrganizerBalanceUsecase_GetOrganizerBalanceProjection_OrganizerNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	u, err := settlement.NewOrganizerBalanceUsecase(
		&mockFinanceJournalRepo{},
		&mockOrganizerRepo{findByIDErr: domain.ErrOrganizerNotFound},
		&mockEventRepo{},
		&mockBankAccountRepo{},
		&mockRefundRepo{},
		logger,
	)
	require.NoError(t, err)

	_, err = u.GetOrganizerBalanceProjection(context.Background(), uuid.New())
	assert.ErrorIs(t, err, settlement.ErrUnauthorizedOrganizer)
}

func TestOrganizerBalanceUsecase_GetOrganizerBalanceProjection_FullyEligible(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()

	startTime := "10:00:00"
	endTime := "12:00:00"
	pastDate := time.Now().UTC().AddDate(0, 0, -2) // Completed 2 days ago

	events := []*domain.OrganizerEvent{
		{
			ID:          eventID,
			OrganizerID: organizerID,
			Title:       "Rock Concert",
			Status:      domain.EventStatusPublished,
			EventDate:   pastDate,
			StartTime:   &startTime,
			EndTime:     &endTime,
			IsAllDay:    false,
		},
	}

	subledger := &domain.OrganizerSubledgerBalance{
		OrganizerID:     organizerID,
		GrossEarnings:   decimal.NewFromInt(10000),
		RefundsDeducted: decimal.Zero,
		SettledAmount:   decimal.Zero,
		NetEarnings:     decimal.NewFromInt(10000),
		NetPayable:      decimal.NewFromInt(10000),
	}

	eventBalances := []domain.EventSubledgerBalance{
		{
			EventID:         eventID,
			OrganizerID:     organizerID,
			GrossEarnings:   decimal.NewFromInt(10000),
			RefundsDeducted: decimal.Zero,
			SettledAmount:   decimal.Zero,
			NetEarnings:     decimal.NewFromInt(10000),
			NetPayable:      decimal.NewFromInt(10000),
		},
	}

	u, err := settlement.NewOrganizerBalanceUsecase(
		&mockFinanceJournalRepo{
			subledgerBalance: subledger,
			eventBalances:    eventBalances,
		},
		&mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
		&mockEventRepo{
			events: events,
			total:  1,
		},
		&mockBankAccountRepo{
			bankAccount: &domain.OrganizerBankAccount{
				ID:                 uuid.New(),
				OrganizerID:        organizerID,
				AccountNumberLast4: "4321",
			},
		},
		&mockRefundRepo{
			refundsByEvent: make(map[uuid.UUID][]domain.Refund),
		},
		logger,
	)
	require.NoError(t, err)

	projection, err := u.GetOrganizerBalanceProjection(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, projection)

	assert.Equal(t, organizerID, projection.OrganizerID)
	assert.Equal(t, domainfinance.CurrencyINR, projection.Currency)
	assert.True(t, projection.GrossEarnings.Equal(decimal.NewFromInt(10000)))
	assert.True(t, projection.OrganizerPayable.Equal(decimal.NewFromInt(10000)))
	assert.True(t, projection.AvailableBalance.Equal(decimal.NewFromInt(10000)))
	assert.True(t, projection.PendingBalance.IsZero())
	assert.True(t, projection.InFlightRefundHold.IsZero())
	assert.Equal(t, domain.SettlementEligible, projection.EligibilityStatus)
	assert.Empty(t, projection.EligibilityReasons)
	assert.True(t, projection.HasActiveBankAccount)

	require.Len(t, projection.EventBalances, 1)
	eb := projection.EventBalances[0]
	assert.Equal(t, eventID, eb.EventID)
	assert.Equal(t, domain.EventFinancialCompleted, eb.Lifecycle)
	assert.True(t, eb.AvailableBalance.Equal(decimal.NewFromInt(10000)))
	assert.True(t, eb.PendingBalance.IsZero())
	assert.True(t, eb.IsEligible)
	assert.Empty(t, eb.IneligibilityReasons)
}

func TestOrganizerBalanceUsecase_UpcomingEvent_HeldInPending(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()

	startTime := "18:00:00"
	futureDate := time.Now().UTC().AddDate(0, 0, 5) // In 5 days

	events := []*domain.OrganizerEvent{
		{
			ID:          eventID,
			OrganizerID: organizerID,
			Title:       "Future Festival",
			Status:      domain.EventStatusPublished,
			EventDate:   futureDate,
			StartTime:   &startTime,
			IsAllDay:    false,
		},
	}

	subledger := &domain.OrganizerSubledgerBalance{
		OrganizerID:     organizerID,
		GrossEarnings:   decimal.NewFromInt(5000),
		RefundsDeducted: decimal.Zero,
		SettledAmount:   decimal.Zero,
		NetEarnings:     decimal.NewFromInt(5000),
		NetPayable:      decimal.NewFromInt(5000),
	}

	eventBalances := []domain.EventSubledgerBalance{
		{
			EventID:         eventID,
			OrganizerID:     organizerID,
			GrossEarnings:   decimal.NewFromInt(5000),
			RefundsDeducted: decimal.Zero,
			SettledAmount:   decimal.Zero,
			NetEarnings:     decimal.NewFromInt(5000),
			NetPayable:      decimal.NewFromInt(5000),
		},
	}

	u, err := settlement.NewOrganizerBalanceUsecase(
		&mockFinanceJournalRepo{
			subledgerBalance: subledger,
			eventBalances:    eventBalances,
		},
		&mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
		&mockEventRepo{
			events: events,
			total:  1,
		},
		&mockBankAccountRepo{
			bankAccount: &domain.OrganizerBankAccount{
				ID:                 uuid.New(),
				AccountNumberLast4: "9999",
			},
		},
		&mockRefundRepo{
			refundsByEvent: make(map[uuid.UUID][]domain.Refund),
		},
		logger,
	)
	require.NoError(t, err)

	projection, err := u.GetOrganizerBalanceProjection(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, projection)

	assert.True(t, projection.AvailableBalance.IsZero())
	assert.True(t, projection.PendingBalance.Equal(decimal.NewFromInt(5000)))
	assert.Equal(t, domain.SettlementIneligible, projection.EligibilityStatus)
	assert.Contains(t, projection.EligibilityReasons, "NO_AVAILABLE_BALANCE")

	require.Len(t, projection.EventBalances, 1)
	eb := projection.EventBalances[0]
	assert.Equal(t, domain.EventFinancialUpcoming, eb.Lifecycle)
	assert.True(t, eb.AvailableBalance.IsZero())
	assert.True(t, eb.PendingBalance.Equal(decimal.NewFromInt(5000)))
	assert.False(t, eb.IsEligible)
	assert.Contains(t, eb.IneligibilityReasons, "EVENT_NOT_COMPLETED")
}

func TestOrganizerBalanceUsecase_CancelledEvent_FundsLockedInPending(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()

	events := []*domain.OrganizerEvent{
		{
			ID:          eventID,
			OrganizerID: organizerID,
			Title:       "Cancelled Gala",
			Status:      domain.EventStatusCancelled,
			EventDate:   time.Now().UTC().AddDate(0, 0, -1),
		},
	}

	subledger := &domain.OrganizerSubledgerBalance{
		OrganizerID:     organizerID,
		GrossEarnings:   decimal.NewFromInt(8000),
		RefundsDeducted: decimal.Zero,
		NetEarnings:     decimal.NewFromInt(8000),
		NetPayable:      decimal.NewFromInt(8000),
	}

	eventBalances := []domain.EventSubledgerBalance{
		{
			EventID:         eventID,
			OrganizerID:     organizerID,
			GrossEarnings:   decimal.NewFromInt(8000),
			RefundsDeducted: decimal.Zero,
			NetEarnings:     decimal.NewFromInt(8000),
			NetPayable:      decimal.NewFromInt(8000),
		},
	}

	refunds := []domain.Refund{
		{
			ID:     uuid.New(),
			Status: domain.RefundStatusPending,
			Amount: decimal.NewFromInt(4000),
		},
		{
			ID:     uuid.New(),
			Status: domain.RefundStatusProcessing,
			Amount: decimal.NewFromInt(4000),
		},
	}

	u, err := settlement.NewOrganizerBalanceUsecase(
		&mockFinanceJournalRepo{
			subledgerBalance: subledger,
			eventBalances:    eventBalances,
		},
		&mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "APPROVED",
			},
		},
		&mockEventRepo{
			events: events,
			total:  1,
		},
		&mockBankAccountRepo{
			bankAccount: &domain.OrganizerBankAccount{
				AccountNumberLast4: "1111",
			},
		},
		&mockRefundRepo{
			refundsByEvent: map[uuid.UUID][]domain.Refund{
				eventID: refunds,
			},
		},
		logger,
	)
	require.NoError(t, err)

	projection, err := u.GetOrganizerBalanceProjection(context.Background(), userID)
	require.NoError(t, err)

	assert.True(t, projection.AvailableBalance.IsZero())
	assert.True(t, projection.PendingBalance.Equal(decimal.NewFromInt(8000)))
	assert.True(t, projection.InFlightRefundHold.Equal(decimal.NewFromInt(8000)))
	assert.Equal(t, domain.SettlementIneligible, projection.EligibilityStatus)
	assert.Contains(t, projection.EligibilityReasons, "IN_FLIGHT_REFUNDS_PENDING")

	eb := projection.EventBalances[0]
	assert.Equal(t, domain.EventFinancialCancelled, eb.Lifecycle)
	assert.False(t, eb.IsEligible)
	assert.Contains(t, eb.IneligibilityReasons, "EVENT_CANCELLED")
	assert.Contains(t, eb.IneligibilityReasons, "IN_FLIGHT_REFUNDS_PENDING")
}

func TestOrganizerBalanceUsecase_InFlightRefundHold_DoesNotDoubleDeductSuccessOrFailed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()

	pastDate := time.Now().UTC().AddDate(0, 0, -3)

	events := []*domain.OrganizerEvent{
		{
			ID:          eventID,
			OrganizerID: organizerID,
			Title:       "Completed Show",
			Status:      domain.EventStatusPublished,
			EventDate:   pastDate,
		},
	}

	// Ledger already deducted completed refund: Gross=10000, RefundsDeducted=2000, NetPayable=8000
	subledger := &domain.OrganizerSubledgerBalance{
		OrganizerID:     organizerID,
		GrossEarnings:   decimal.NewFromInt(10000),
		RefundsDeducted: decimal.NewFromInt(2000),
		NetEarnings:     decimal.NewFromInt(8000),
		NetPayable:      decimal.NewFromInt(8000),
	}

	eventBalances := []domain.EventSubledgerBalance{
		{
			EventID:         eventID,
			OrganizerID:     organizerID,
			GrossEarnings:   decimal.NewFromInt(10000),
			RefundsDeducted: decimal.NewFromInt(2000),
			NetEarnings:     decimal.NewFromInt(8000),
			NetPayable:      decimal.NewFromInt(8000),
		},
	}

	// Operational refunds:
	// 1. PENDING: 1500 (must be held)
	// 2. SUCCESS: 2000 (already posted in ledger, MUST NOT BE HELD AGAIN)
	// 3. FAILED: 1000 (failed, MUST NOT BE HELD)
	refunds := []domain.Refund{
		{
			ID:     uuid.New(),
			Status: domain.RefundStatusPending,
			Amount: decimal.NewFromInt(1500),
		},
		{
			ID:     uuid.New(),
			Status: domain.RefundStatusSuccess,
			Amount: decimal.NewFromInt(2000),
		},
		{
			ID:     uuid.New(),
			Status: domain.RefundStatusFailed,
			Amount: decimal.NewFromInt(1000),
		},
	}

	u, err := settlement.NewOrganizerBalanceUsecase(
		&mockFinanceJournalRepo{
			subledgerBalance: subledger,
			eventBalances:    eventBalances,
		},
		&mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
		&mockEventRepo{
			events: events,
			total:  1,
		},
		&mockBankAccountRepo{
			bankAccount: &domain.OrganizerBankAccount{
				AccountNumberLast4: "5555",
			},
		},
		&mockRefundRepo{
			refundsByEvent: map[uuid.UUID][]domain.Refund{
				eventID: refunds,
			},
		},
		logger,
	)
	require.NoError(t, err)

	projection, err := u.GetOrganizerBalanceProjection(context.Background(), userID)
	require.NoError(t, err)

	// Available = NetPayable (8000) - InFlightHold (1500) = 6500
	assert.True(t, projection.AvailableBalance.Equal(decimal.NewFromInt(6500)), "Available should be 6500, got %s", projection.AvailableBalance)
	assert.True(t, projection.PendingBalance.Equal(decimal.NewFromInt(1500)), "Pending should be 1500, got %s", projection.PendingBalance)
	assert.True(t, projection.InFlightRefundHold.Equal(decimal.NewFromInt(1500)), "InFlightHold should be 1500, got %s", projection.InFlightRefundHold)
	assert.Equal(t, domain.SettlementPartiallyEligible, projection.EligibilityStatus)
	assert.Contains(t, projection.EligibilityReasons, "IN_FLIGHT_REFUNDS_PENDING")

	eb := projection.EventBalances[0]
	assert.True(t, eb.AvailableBalance.Equal(decimal.NewFromInt(6500)))
	assert.True(t, eb.PendingBalance.Equal(decimal.NewFromInt(1500)))
	assert.False(t, eb.IsEligible) // Not fully eligible because in-flight refund is pending
	assert.Contains(t, eb.IneligibilityReasons, "IN_FLIGHT_REFUNDS_PENDING")
}

func TestOrganizerBalanceUsecase_MissingBankAccount_Ineligible(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()

	u, err := settlement.NewOrganizerBalanceUsecase(
		&mockFinanceJournalRepo{
			subledgerBalance: &domain.OrganizerSubledgerBalance{
				OrganizerID: organizerID,
				NetPayable:  decimal.NewFromInt(10000),
			},
			eventBalances: []domain.EventSubledgerBalance{
				{
					EventID:    eventID,
					NetPayable: decimal.NewFromInt(10000),
				},
			},
		},
		&mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
		&mockEventRepo{
			events: []*domain.OrganizerEvent{
				{
					ID:        eventID,
					EventDate: time.Now().UTC().AddDate(0, 0, -2),
				},
			},
			total: 1,
		},
		&mockBankAccountRepo{
			findErr: domain.ErrOrganizerBankAccountNotFound,
		},
		&mockRefundRepo{
			refundsByEvent: make(map[uuid.UUID][]domain.Refund),
		},
		logger,
	)
	require.NoError(t, err)

	projection, err := u.GetOrganizerBalanceProjection(context.Background(), userID)
	require.NoError(t, err)

	assert.False(t, projection.HasActiveBankAccount)
	assert.Equal(t, domain.SettlementIneligible, projection.EligibilityStatus)
	assert.Contains(t, projection.EligibilityReasons, "NO_ACTIVE_BANK_ACCOUNT")
}

func TestOrganizerBalanceUsecase_UnapprovedOrganizer_Ineligible(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	organizerID := uuid.New()

	u, err := settlement.NewOrganizerBalanceUsecase(
		&mockFinanceJournalRepo{
			subledgerBalance: &domain.OrganizerSubledgerBalance{
				OrganizerID: organizerID,
				NetPayable:  decimal.NewFromInt(5000),
			},
		},
		&mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "PENDING_APPROVAL",
			},
		},
		&mockEventRepo{
			events: []*domain.OrganizerEvent{},
			total:  0,
		},
		&mockBankAccountRepo{
			bankAccount: &domain.OrganizerBankAccount{
				AccountNumberLast4: "1234",
			},
		},
		&mockRefundRepo{
			refundsByEvent: make(map[uuid.UUID][]domain.Refund),
		},
		logger,
	)
	require.NoError(t, err)

	projection, err := u.GetOrganizerBalanceProjection(context.Background(), userID)
	require.NoError(t, err)

	assert.Equal(t, domain.SettlementIneligible, projection.EligibilityStatus)
	assert.Contains(t, projection.EligibilityReasons, "ORGANIZER_ACCOUNT_NOT_APPROVED")
}
