package settlement_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/service/payout"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/settlement"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Transaction Repositories & Manager ---

type mockTxRepos struct {
	domain.TransactionRepositories
	settlementRepo     domain.SettlementRepository
	organizerRepo      domain.OrganizerRepository
	financeAccountRepo domain.FinanceAccountRepository
	financeJournalRepo domain.FinanceJournalRepository
}

func (m *mockTxRepos) SettlementRepository() domain.SettlementRepository {
	return m.settlementRepo
}

func (m *mockTxRepos) OrganizerRepository() domain.OrganizerRepository {
	return m.organizerRepo
}

func (m *mockTxRepos) FinanceAccountRepository() domain.FinanceAccountRepository {
	return m.financeAccountRepo
}

func (m *mockTxRepos) FinanceJournalRepository() domain.FinanceJournalRepository {
	return m.financeJournalRepo
}

type mockTxManager struct {
	txRepos *mockTxRepos
	err     error
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(m.txRepos)
}

// --- Mock Settlement Repository ---

type mockSettlementRepo struct {
	settlements       map[uuid.UUID]*domain.Settlement
	inFlightAmount    decimal.Decimal
	inFlightEvtAmount decimal.Decimal
	createErr         error
	updateErr         error
	findErr           error
}

func newMockSettlementRepo() *mockSettlementRepo {
	return &mockSettlementRepo{
		settlements: make(map[uuid.UUID]*domain.Settlement),
	}
}

func (m *mockSettlementRepo) Create(ctx context.Context, s *domain.Settlement) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.settlements[s.ID] = s
	return nil
}

func (m *mockSettlementRepo) Update(ctx context.Context, s *domain.Settlement) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.settlements[s.ID] = s
	return nil
}

func (m *mockSettlementRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Settlement, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	s, exists := m.settlements[id]
	if !exists {
		return nil, domain.ErrSettlementNotFound
	}
	return s, nil
}

func (m *mockSettlementRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Settlement, error) {
	return m.FindByID(ctx, id)
}

func (m *mockSettlementRepo) List(ctx context.Context, filter domain.SettlementFilter) (*domain.SettlementListResult, error) {
	var list []*domain.Settlement
	for _, s := range m.settlements {
		if filter.OrganizerID != nil && s.OrganizerID != *filter.OrganizerID {
			continue
		}
		if filter.EventID != nil && (s.EventID == nil || *s.EventID != *filter.EventID) {
			continue
		}
		if filter.Status != nil && s.Status != *filter.Status {
			continue
		}
		list = append(list, s)
	}
	return &domain.SettlementListResult{
		Settlements: list,
		Total:       int64(len(list)),
		Page:        filter.Page,
		Limit:       filter.Limit,
	}, nil
}

func (m *mockSettlementRepo) GetInFlightSettlementAmount(ctx context.Context, organizerID uuid.UUID) (decimal.Decimal, error) {
	return m.inFlightAmount, nil
}

func (m *mockSettlementRepo) GetInFlightSettlementAmountForEvent(ctx context.Context, eventID uuid.UUID) (decimal.Decimal, error) {
	return m.inFlightEvtAmount, nil
}

// --- Mock Finance Account Repository ---

type mockFinanceAccountRepo struct {
	accounts map[string]*domainfinance.Account
}

func (m *mockFinanceAccountRepo) FindByCode(ctx context.Context, code string) (*domainfinance.Account, error) {
	acc, exists := m.accounts[code]
	if !exists {
		return nil, errors.New("account not found")
	}
	return acc, nil
}

func (m *mockFinanceAccountRepo) Create(ctx context.Context, account *domainfinance.Account) error {
	return nil
}

func (m *mockFinanceAccountRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainfinance.Account, error) {
	return nil, nil
}

// --- Mock Organizer Balance Usecase ---

type mockOrgBalanceUC struct {
	projection *domain.OrganizerBalanceProjection
	err        error
}

func (m *mockOrgBalanceUC) GetOrganizerBalanceProjection(ctx context.Context, userID uuid.UUID) (*domain.OrganizerBalanceProjection, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.projection, nil
}

// --- Tests ---

func TestNewSettlementUsecase_Validation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txMgr := &mockTxManager{}
	sRepo := newMockSettlementRepo()
	oRepo := &mockOrganizerRepo{}
	bRepo := &mockBankAccountRepo{}
	balUC := &mockOrgBalanceUC{}
	pProvider := payout.NewMockPayoutProvider()

	_, err := settlement.NewSettlementUsecase(nil, sRepo, oRepo, bRepo, balUC, pProvider, logger)
	assert.ErrorIs(t, err, settlement.ErrNilTransactionManager)

	_, err = settlement.NewSettlementUsecase(txMgr, nil, oRepo, bRepo, balUC, pProvider, logger)
	assert.ErrorIs(t, err, settlement.ErrNilSettlementRepository)

	_, err = settlement.NewSettlementUsecase(txMgr, sRepo, nil, bRepo, balUC, pProvider, logger)
	assert.ErrorIs(t, err, settlement.ErrNilOrganizerRepository)

	_, err = settlement.NewSettlementUsecase(txMgr, sRepo, oRepo, nil, balUC, pProvider, logger)
	assert.ErrorIs(t, err, settlement.ErrNilOrganizerBankAccountRepo)

	_, err = settlement.NewSettlementUsecase(txMgr, sRepo, oRepo, bRepo, nil, pProvider, logger)
	assert.ErrorIs(t, err, settlement.ErrNilOrganizerBalanceUsecase)

	_, err = settlement.NewSettlementUsecase(txMgr, sRepo, oRepo, bRepo, balUC, nil, logger)
	assert.ErrorIs(t, err, settlement.ErrNilPayoutProvider)
}

func TestSettlementStatus_CanTransitionTo(t *testing.T) {
	assert.True(t, domain.SettlementStatusPending.CanTransitionTo(domain.SettlementStatusApproved))
	assert.True(t, domain.SettlementStatusPending.CanTransitionTo(domain.SettlementStatusRejected))
	assert.False(t, domain.SettlementStatusPending.CanTransitionTo(domain.SettlementStatusCompleted))
	assert.False(t, domain.SettlementStatusPending.CanTransitionTo(domain.SettlementStatusProcessing))

	assert.True(t, domain.SettlementStatusApproved.CanTransitionTo(domain.SettlementStatusProcessing))
	assert.False(t, domain.SettlementStatusApproved.CanTransitionTo(domain.SettlementStatusCompleted))
	assert.False(t, domain.SettlementStatusApproved.CanTransitionTo(domain.SettlementStatusRejected))

	assert.True(t, domain.SettlementStatusProcessing.CanTransitionTo(domain.SettlementStatusCompleted))
	assert.True(t, domain.SettlementStatusProcessing.CanTransitionTo(domain.SettlementStatusFailed))
	assert.False(t, domain.SettlementStatusProcessing.CanTransitionTo(domain.SettlementStatusPending))

	assert.True(t, domain.SettlementStatusFailed.CanTransitionTo(domain.SettlementStatusProcessing))
	assert.True(t, domain.SettlementStatusFailed.CanTransitionTo(domain.SettlementStatusRejected))
	assert.False(t, domain.SettlementStatusFailed.CanTransitionTo(domain.SettlementStatusCompleted))

	assert.False(t, domain.SettlementStatusCompleted.CanTransitionTo(domain.SettlementStatusProcessing))
	assert.False(t, domain.SettlementStatusCompleted.CanTransitionTo(domain.SettlementStatusFailed))
	assert.False(t, domain.SettlementStatusCompleted.CanTransitionTo(domain.SettlementStatusApproved))

	assert.False(t, domain.SettlementStatusRejected.CanTransitionTo(domain.SettlementStatusApproved))
	assert.False(t, domain.SettlementStatusRejected.CanTransitionTo(domain.SettlementStatusProcessing))
}

func TestSettlementUsecase_RequestSettlement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()

	setupUsecase := func(availBal decimal.Decimal, eventAvailBal decimal.Decimal, isEventEligible bool) (*settlement.SettlementUsecaseImpl, *mockSettlementRepo) {
		sRepo := newMockSettlementRepo()
		oRepo := &mockOrganizerRepo{
			organizer: &domain.Organizer{
				ID:     organizerID,
				UserID: userID,
				Status: "ACTIVE",
			},
		}
		bRepo := &mockBankAccountRepo{
			bankAccount: &domain.OrganizerBankAccount{
				AccountNumberLast4: "1234",
				IFSCCode:           "HDFC0001234",
			},
		}
		balUC := &mockOrgBalanceUC{
			projection: &domain.OrganizerBalanceProjection{
				OrganizerID:      organizerID,
				AvailableBalance: availBal,
				EventBalances: []domain.EventBalanceProjection{
					{
						EventID:          eventID,
						AvailableBalance: eventAvailBal,
						IsEligible:       isEventEligible,
					},
				},
			},
		}
		pProvider := payout.NewMockPayoutProvider()
		txRepos := &mockTxRepos{settlementRepo: sRepo, organizerRepo: oRepo}
		txMgr := &mockTxManager{txRepos: txRepos}

		uc, err := settlement.NewSettlementUsecase(txMgr, sRepo, oRepo, bRepo, balUC, pProvider, logger)
		require.NoError(t, err)
		return uc, sRepo
	}

	t.Run("Reject zero or negative amount", func(t *testing.T) {
		uc, _ := setupUsecase(decimal.NewFromInt(10000), decimal.Zero, false)
		_, err := uc.RequestSettlement(context.Background(), userID, decimal.Zero, nil)
		assert.ErrorIs(t, err, domain.ErrInvalidSettlementAmount)

		_, err = uc.RequestSettlement(context.Background(), userID, decimal.NewFromInt(-100), nil)
		assert.ErrorIs(t, err, domain.ErrInvalidSettlementAmount)
	})

	t.Run("Reject amount exceeding available balance", func(t *testing.T) {
		uc, _ := setupUsecase(decimal.NewFromInt(5000), decimal.Zero, false)
		_, err := uc.RequestSettlement(context.Background(), userID, decimal.NewFromInt(6000), nil)
		assert.ErrorIs(t, err, domain.ErrInsufficientAvailableBalance)
	})

	t.Run("Reject event settlement if event is not eligible", func(t *testing.T) {
		uc, _ := setupUsecase(decimal.NewFromInt(10000), decimal.NewFromInt(5000), false)
		_, err := uc.RequestSettlement(context.Background(), userID, decimal.NewFromInt(3000), &eventID)
		assert.ErrorIs(t, err, domain.ErrEventNotEligibleForSettlement)
	})

	t.Run("Reject event settlement exceeding event available balance", func(t *testing.T) {
		uc, _ := setupUsecase(decimal.NewFromInt(10000), decimal.NewFromInt(3000), true)
		_, err := uc.RequestSettlement(context.Background(), userID, decimal.NewFromInt(4000), &eventID)
		assert.ErrorIs(t, err, domain.ErrInsufficientAvailableBalance)
	})

	t.Run("Successful general settlement creation", func(t *testing.T) {
		uc, sRepo := setupUsecase(decimal.NewFromInt(10000), decimal.Zero, false)
		s, err := uc.RequestSettlement(context.Background(), userID, decimal.NewFromInt(5000), nil)
		require.NoError(t, err)
		require.NotNil(t, s)

		assert.Equal(t, organizerID, s.OrganizerID)
		assert.Nil(t, s.EventID)
		assert.True(t, s.Amount.Equal(decimal.NewFromInt(5000)))
		assert.Equal(t, domain.SettlementStatusPending, s.Status)

		// Verify stored in repo
		stored, err := sRepo.FindByID(context.Background(), s.ID)
		require.NoError(t, err)
		assert.Equal(t, s.ID, stored.ID)
	})

	t.Run("Successful event-specific settlement creation", func(t *testing.T) {
		uc, sRepo := setupUsecase(decimal.NewFromInt(10000), decimal.NewFromInt(8000), true)
		s, err := uc.RequestSettlement(context.Background(), userID, decimal.NewFromInt(4000), &eventID)
		require.NoError(t, err)
		require.NotNil(t, s)

		assert.Equal(t, organizerID, s.OrganizerID)
		assert.Equal(t, &eventID, s.EventID)
		assert.True(t, s.Amount.Equal(decimal.NewFromInt(4000)))
		assert.Equal(t, domain.SettlementStatusPending, s.Status)

		stored, err := sRepo.FindByID(context.Background(), s.ID)
		require.NoError(t, err)
		assert.Equal(t, s.ID, stored.ID)
	})
}

func TestSettlementUsecase_ApproveSettlement_SuccessAndLedgerPosting(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminUserID := uuid.New()
	organizerID := uuid.New()
	settlementID := uuid.New()

	sRepo := newMockSettlementRepo()
	sRepo.settlements[settlementID] = &domain.Settlement{
		ID:          settlementID,
		OrganizerID: organizerID,
		Amount:      decimal.NewFromInt(5000),
		Currency:    domainfinance.CurrencyINR,
		Status:      domain.SettlementStatusPending,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	oRepo := &mockOrganizerRepo{
		organizer: &domain.Organizer{
			ID:     organizerID,
			Status: "ACTIVE",
		},
	}
	bRepo := &mockBankAccountRepo{
		bankAccount: &domain.OrganizerBankAccount{
			AccountNumberLast4: "9876",
			IFSCCode:           "SBIN0001234",
		},
	}
	balUC := &mockOrgBalanceUC{}
	pProvider := payout.NewMockPayoutProvider()

	// Financial Accounts
	acc2010ID := uuid.New()
	acc1010ID := uuid.New()
	acc2010, _ := domainfinance.NewAccount(acc2010ID, domainfinance.AccountCodeOrganizerPayable, "Organizer Payable",
		domainfinance.AccountTypeLiability, domainfinance.NormalBalanceCredit, domainfinance.CurrencyINR, true)
	acc1010, _ := domainfinance.NewAccount(acc1010ID, domainfinance.AccountCodeRazorpayClearing, "Razorpay Clearing",
		domainfinance.AccountTypeAsset, domainfinance.NormalBalanceDebit, domainfinance.CurrencyINR, true)

	accRepo := &mockFinanceAccountRepo{
		accounts: map[string]*domainfinance.Account{
			domainfinance.AccountCodeOrganizerPayable: acc2010,
			domainfinance.AccountCodeRazorpayClearing: acc1010,
		},
	}

	jRepo := &mockFinanceJournalRepo{}
	var recordedJournal *domainfinance.JournalTransaction

	// Hook RecordJournal to capture posted journal
	recordedJRepo := &mockFinanceJournalWithRecording{
		onRecord: func(j *domainfinance.JournalTransaction) (*domainfinance.JournalTransaction, error) {
			recordedJournal = j
			// Reconstitute as POSTED
			now := time.Now().UTC()
			return domainfinance.ReconstituteJournalTransaction(
				j.ID(),
				j.TransactionType(),
				j.FinancialEventType(),
				j.SourceType(),
				j.SourceID(),
				j.IdempotencyKey(),
				j.Currency(),
				j.Description(),
				j.Entries(),
				nil,
				domainfinance.PostingStatusPosted,
				now,
				&now,
				false,
				nil,
				nil,
			)
		},
	}

	txRepos := &mockTxRepos{
		settlementRepo:     sRepo,
		organizerRepo:      oRepo,
		financeAccountRepo: accRepo,
		financeJournalRepo: recordedJRepo,
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc, err := settlement.NewSettlementUsecase(txMgr, sRepo, oRepo, bRepo, balUC, pProvider, logger)
	require.NoError(t, err)

	completed, err := uc.ApproveSettlement(context.Background(), adminUserID, settlementID)
	require.NoError(t, err)
	require.NotNil(t, completed)

	// 1. Verify settlement updated to COMPLETED with payout ref
	assert.Equal(t, domain.SettlementStatusCompleted, completed.Status)
	assert.NotNil(t, completed.ApprovedBy)
	assert.Equal(t, adminUserID, *completed.ApprovedBy)
	assert.NotNil(t, completed.ApprovedAt)
	assert.NotNil(t, completed.ProcessedAt)
	assert.NotNil(t, completed.PayoutReference)
	assert.Contains(t, *completed.PayoutReference, "pout_mock_")

	// 2. Verify double-entry journal was posted to ledger
	require.NotNil(t, recordedJournal, "Settlement journal must be recorded in ledger")
	assert.Equal(t, domainfinance.TransactionTypePayout, recordedJournal.TransactionType())
	assert.Equal(t, domainfinance.FinancialEventOrganizerPayoutSettled, recordedJournal.FinancialEventType())
	assert.Equal(t, domainfinance.SourceTypeSettlement, recordedJournal.SourceType())
	assert.Equal(t, settlementID.String(), recordedJournal.SourceID())
	assert.Equal(t, 2, len(recordedJournal.Entries()))

	// Verify entries: Debit 2010 (Payable), Credit 1010 (Cash Clearing)
	var hasDebit2010, hasCredit1010 bool
	for _, entry := range recordedJournal.Entries() {
		if entry.AccountID() == acc2010ID && entry.EntryType() == domainfinance.EntryTypeDebit {
			hasDebit2010 = true
			assert.True(t, entry.Amount().Amount().Equal(decimal.NewFromInt(5000)))
			require.NotNil(t, entry.OrganizerID())
			assert.Equal(t, organizerID, *entry.OrganizerID())
		}
		if entry.AccountID() == acc1010ID && entry.EntryType() == domainfinance.EntryTypeCredit {
			hasCredit1010 = true
			assert.True(t, entry.Amount().Amount().Equal(decimal.NewFromInt(5000)))
		}
	}
	assert.True(t, hasDebit2010, "Journal must debit Account 2010")
	assert.True(t, hasCredit1010, "Journal must credit Account 1010")

	_ = jRepo
}

func TestSettlementUsecase_ApproveSettlement_PayoutFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminUserID := uuid.New()
	organizerID := uuid.New()
	settlementID := uuid.New()

	sRepo := newMockSettlementRepo()
	sRepo.settlements[settlementID] = &domain.Settlement{
		ID:          settlementID,
		OrganizerID: organizerID,
		Amount:      decimal.NewFromInt(5000),
		Currency:    domainfinance.CurrencyINR,
		Status:      domain.SettlementStatusPending,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	oRepo := &mockOrganizerRepo{organizer: &domain.Organizer{ID: organizerID, Status: "ACTIVE"}}
	bRepo := &mockBankAccountRepo{bankAccount: &domain.OrganizerBankAccount{AccountNumberLast4: "9876"}}
	balUC := &mockOrgBalanceUC{}

	pProvider := payout.NewMockPayoutProvider()
	pProvider.SetFailure(settlementID, "insufficient funds in clearing account")

	txRepos := &mockTxRepos{settlementRepo: sRepo, organizerRepo: oRepo}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc, err := settlement.NewSettlementUsecase(txMgr, sRepo, oRepo, bRepo, balUC, pProvider, logger)
	require.NoError(t, err)

	res, err := uc.ApproveSettlement(context.Background(), adminUserID, settlementID)
	assert.ErrorIs(t, err, domain.ErrPayoutFailed)
	require.NotNil(t, res)

	// Verify settlement status is FAILED and failure reason recorded
	assert.Equal(t, domain.SettlementStatusFailed, res.Status)
	assert.NotNil(t, res.FailureReason)
	assert.Equal(t, "insufficient funds in clearing account", *res.FailureReason)
	assert.Nil(t, res.ProcessedAt)
	assert.Nil(t, res.PayoutReference)
}

func TestSettlementUsecase_RejectSettlement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminUserID := uuid.New()
	organizerID := uuid.New()
	settlementID := uuid.New()

	sRepo := newMockSettlementRepo()
	sRepo.settlements[settlementID] = &domain.Settlement{
		ID:          settlementID,
		OrganizerID: organizerID,
		Amount:      decimal.NewFromInt(3000),
		Status:      domain.SettlementStatusPending,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	oRepo := &mockOrganizerRepo{}
	txRepos := &mockTxRepos{settlementRepo: sRepo, organizerRepo: oRepo}
	txMgr := &mockTxManager{txRepos: txRepos}
	pProvider := payout.NewMockPayoutProvider()

	uc, err := settlement.NewSettlementUsecase(txMgr, sRepo, oRepo, &mockBankAccountRepo{}, &mockOrgBalanceUC{}, pProvider, logger)
	require.NoError(t, err)

	t.Run("Reject without reason fails", func(t *testing.T) {
		_, err := uc.RejectSettlement(context.Background(), adminUserID, settlementID, "   ")
		assert.ErrorIs(t, err, domain.ErrRejectionReasonRequired)
	})

	t.Run("Successful rejection", func(t *testing.T) {
		rejected, err := uc.RejectSettlement(context.Background(), adminUserID, settlementID, "Bank account details mismatch")
		require.NoError(t, err)
		assert.Equal(t, domain.SettlementStatusRejected, rejected.Status)
		require.NotNil(t, rejected.RejectionReason)
		assert.Equal(t, "Bank account details mismatch", *rejected.RejectionReason)
		require.NotNil(t, rejected.RejectedAt)
	})

	t.Run("Rejecting already rejected settlement fails transition", func(t *testing.T) {
		_, err := uc.RejectSettlement(context.Background(), adminUserID, settlementID, "Another reason")
		assert.ErrorIs(t, err, domain.ErrInvalidSettlementStatusTransition)
	})

	t.Run("Rejecting failed settlement succeeds and releases hold", func(t *testing.T) {
		failedSettlementID := uuid.New()
		sRepo.settlements[failedSettlementID] = &domain.Settlement{
			ID:          failedSettlementID,
			OrganizerID: organizerID,
			Amount:      decimal.NewFromInt(1500),
			Status:      domain.SettlementStatusFailed,
		}
		rejected, err := uc.RejectSettlement(context.Background(), adminUserID, failedSettlementID, "Bank details unrecoverable")
		require.NoError(t, err)
		assert.Equal(t, domain.SettlementStatusRejected, rejected.Status)
		require.NotNil(t, rejected.RejectionReason)
		assert.Equal(t, "Bank details unrecoverable", *rejected.RejectionReason)
	})
}

func TestSettlementUsecase_RetrySettlement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminUserID := uuid.New()
	organizerID := uuid.New()
	settlementID := uuid.New()

	sRepo := newMockSettlementRepo()
	sRepo.settlements[settlementID] = &domain.Settlement{
		ID:          settlementID,
		OrganizerID: organizerID,
		Amount:      decimal.NewFromInt(2000),
		Currency:    domainfinance.CurrencyINR,
		Status:      domain.SettlementStatusFailed,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	userID := uuid.New()
	oRepo := &mockOrganizerRepo{organizer: &domain.Organizer{ID: organizerID, UserID: userID, Status: "ACTIVE"}}
	bRepo := &mockBankAccountRepo{bankAccount: &domain.OrganizerBankAccount{AccountNumberLast4: "1234"}}
	balUC := &mockOrgBalanceUC{
		projection: &domain.OrganizerBalanceProjection{
			OrganizerID:      organizerID,
			AvailableBalance: decimal.NewFromInt(5000),
		},
	}
	pProvider := payout.NewMockPayoutProvider()

	acc2010ID := uuid.New()
	acc1010ID := uuid.New()
	acc2010, _ := domainfinance.NewAccount(acc2010ID, domainfinance.AccountCodeOrganizerPayable, "Organizer Payable",
		domainfinance.AccountTypeLiability, domainfinance.NormalBalanceCredit, domainfinance.CurrencyINR, true)
	acc1010, _ := domainfinance.NewAccount(acc1010ID, domainfinance.AccountCodeRazorpayClearing, "Razorpay Clearing",
		domainfinance.AccountTypeAsset, domainfinance.NormalBalanceDebit, domainfinance.CurrencyINR, true)

	accRepo := &mockFinanceAccountRepo{
		accounts: map[string]*domainfinance.Account{
			domainfinance.AccountCodeOrganizerPayable: acc2010,
			domainfinance.AccountCodeRazorpayClearing: acc1010,
		},
	}

	recordedJRepo := &mockFinanceJournalWithRecording{
		onRecord: func(j *domainfinance.JournalTransaction) (*domainfinance.JournalTransaction, error) {
			now := time.Now().UTC()
			return domainfinance.ReconstituteJournalTransaction(
				j.ID(),
				j.TransactionType(),
				j.FinancialEventType(),
				j.SourceType(),
				j.SourceID(),
				j.IdempotencyKey(),
				j.Currency(),
				j.Description(),
				j.Entries(),
				nil,
				domainfinance.PostingStatusPosted,
				now,
				&now,
				false,
				nil,
				nil,
			)
		},
	}

	txRepos := &mockTxRepos{
		settlementRepo:     sRepo,
		organizerRepo:      oRepo,
		financeAccountRepo: accRepo,
		financeJournalRepo: recordedJRepo,
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc, err := settlement.NewSettlementUsecase(txMgr, sRepo, oRepo, bRepo, balUC, pProvider, logger)
	require.NoError(t, err)

	t.Run("Retry on non-FAILED settlement fails", func(t *testing.T) {
		nonFailedID := uuid.New()
		sRepo.settlements[nonFailedID] = &domain.Settlement{
			ID:          nonFailedID,
			OrganizerID: organizerID,
			Amount:      decimal.NewFromInt(1000),
			Status:      domain.SettlementStatusApproved,
		}
		_, err := uc.RetrySettlement(context.Background(), adminUserID, nonFailedID)
		assert.ErrorIs(t, err, domain.ErrInvalidSettlementStatusTransition)
	})

	t.Run("Retry when available balance is insufficient fails", func(t *testing.T) {
		insufficientID := uuid.New()
		sRepo.settlements[insufficientID] = &domain.Settlement{
			ID:          insufficientID,
			OrganizerID: organizerID,
			Amount:      decimal.NewFromInt(99999), // Exceeds available balance (5000)
			Status:      domain.SettlementStatusFailed,
		}
		_, err := uc.RetrySettlement(context.Background(), adminUserID, insufficientID)
		assert.ErrorIs(t, err, domain.ErrInsufficientAvailableBalance)
	})

	t.Run("Successful retry executes payout and marks completed", func(t *testing.T) {
		retried, err := uc.RetrySettlement(context.Background(), adminUserID, settlementID)
		require.NoError(t, err)
		assert.Equal(t, domain.SettlementStatusCompleted, retried.Status)
		assert.NotNil(t, retried.PayoutReference)
		assert.NotNil(t, retried.ProcessedAt)
	})
}

func TestSettlementUsecase_OrganizerAuthorization(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userA := uuid.New()
	userB := uuid.New()
	orgA := uuid.New()
	orgB := uuid.New()
	settlementA := uuid.New()

	sRepo := newMockSettlementRepo()
	sRepo.settlements[settlementA] = &domain.Settlement{
		ID:          settlementA,
		OrganizerID: orgA,
		Amount:      decimal.NewFromInt(1000),
		Status:      domain.SettlementStatusPending,
	}

	oRepo := &mockOrganizerRepoWithMap{
		organizers: map[uuid.UUID]*domain.Organizer{
			userA: {ID: orgA, UserID: userA, Status: "ACTIVE"},
			userB: {ID: orgB, UserID: userB, Status: "ACTIVE"},
		},
	}

	txRepos := &mockTxRepos{settlementRepo: sRepo, organizerRepo: oRepo}
	txMgr := &mockTxManager{txRepos: txRepos}
	pProvider := payout.NewMockPayoutProvider()

	uc, err := settlement.NewSettlementUsecase(txMgr, sRepo, oRepo, &mockBankAccountRepo{}, &mockOrgBalanceUC{}, pProvider, logger)
	require.NoError(t, err)

	// User A can access settlement A
	s, err := uc.GetOrganizerSettlement(context.Background(), userA, settlementA)
	require.NoError(t, err)
	assert.Equal(t, settlementA, s.ID)

	// User B cannot access settlement A
	_, err = uc.GetOrganizerSettlement(context.Background(), userB, settlementA)
	assert.ErrorIs(t, err, domain.ErrSettlementNotFound)
}

// --- Helper mocks ---

type mockFinanceJournalWithRecording struct {
	domain.FinanceJournalRepository
	onRecord func(j *domainfinance.JournalTransaction) (*domainfinance.JournalTransaction, error)
}

func (m *mockFinanceJournalWithRecording) RecordJournal(ctx context.Context, j *domainfinance.JournalTransaction) (*domainfinance.JournalTransaction, error) {
	if m.onRecord != nil {
		return m.onRecord(j)
	}
	return j, nil
}

type mockOrganizerRepoWithMap struct {
	domain.OrganizerRepository
	organizers map[uuid.UUID]*domain.Organizer
}

func (m *mockOrganizerRepoWithMap) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	o, exists := m.organizers[userID]
	if !exists {
		return nil, domain.ErrOrganizerNotFound
	}
	return o, nil
}

func (m *mockOrganizerRepoWithMap) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	for _, o := range m.organizers {
		if o.ID == id {
			return o, nil
		}
	}
	return nil, domain.ErrOrganizerNotFound
}

func (m *mockOrganizerRepoWithMap) FindByIDForUpdate(id uuid.UUID) (*domain.Organizer, error) {
	return m.FindByID(id)
}
