package finance_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	usecasefinance "github.com/adhithyan443/EventHub/backend/internal/usecase/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// Test Mocks
// -----------------------------------------------------------------------------

type mockTxManager struct {
	withinTxFn func(fn func(tx domain.TransactionRepositories) error) error
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	if m.withinTxFn != nil {
		return m.withinTxFn(fn)
	}
	return fn(&mockTxRepos{})
}

type mockTxRepos struct {
	domain.TransactionRepositories
	accountRepo domain.FinanceAccountRepository
	journalRepo domain.FinanceJournalRepository
}

func (r *mockTxRepos) FinanceAccountRepository() domain.FinanceAccountRepository {
	return r.accountRepo
}

func (r *mockTxRepos) FinanceJournalRepository() domain.FinanceJournalRepository {
	return r.journalRepo
}

type mockAccountRepo struct {
	accountsByCode map[string]*domainfinance.Account
	findByCodeErr  map[string]error
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{
		accountsByCode: make(map[string]*domainfinance.Account),
		findByCodeErr:  make(map[string]error),
	}
}

func (m *mockAccountRepo) Create(ctx context.Context, account *domainfinance.Account) error {
	m.accountsByCode[account.Code()] = account
	return nil
}

func (m *mockAccountRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainfinance.Account, error) {
	for _, acc := range m.accountsByCode {
		if acc.ID() == id {
			return acc, nil
		}
	}
	return nil, domain.ErrFinanceAccountNotFound
}

func (m *mockAccountRepo) FindByCode(ctx context.Context, code string) (*domainfinance.Account, error) {
	if err, exists := m.findByCodeErr[code]; exists && err != nil {
		return nil, err
	}
	acc, exists := m.accountsByCode[code]
	if !exists {
		return nil, domain.ErrFinanceAccountNotFound
	}
	return acc, nil
}

type mockJournalRepo struct {
	recordedJournals []*domainfinance.JournalTransaction
	recordErr        error
}

func (m *mockJournalRepo) RecordJournal(
	ctx context.Context,
	journal *domainfinance.JournalTransaction,
) (*domainfinance.JournalTransaction, error) {
	if m.recordErr != nil {
		return nil, m.recordErr
	}
	m.recordedJournals = append(m.recordedJournals, journal)
	return journal, nil
}

func (m *mockJournalRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainfinance.JournalTransaction, error) {
	for _, j := range m.recordedJournals {
		if j.ID() == id {
			return j, nil
		}
	}
	return nil, domain.ErrFinanceJournalNotFound
}

func (m *mockJournalRepo) FindByIdempotencyKey(ctx context.Context, key string) (*domainfinance.JournalTransaction, error) {
	for _, j := range m.recordedJournals {
		if j.IdempotencyKey() == key {
			return j, nil
		}
	}
	return nil, domain.ErrFinanceJournalNotFound
}

func setupStandardAccounts(t *testing.T, repo *mockAccountRepo) {
	t.Helper()

	clearing, err := domainfinance.NewAccount(
		uuid.New(),
		domainfinance.AccountCodeRazorpayClearing,
		"Razorpay Clearing",
		domainfinance.AccountTypeAsset,
		domainfinance.NormalBalanceDebit,
		domainfinance.CurrencyINR,
		true,
	)
	require.NoError(t, err)

	organizer, err := domainfinance.NewAccount(
		uuid.New(),
		domainfinance.AccountCodeOrganizerPayable,
		"Organizer Payable",
		domainfinance.AccountTypeLiability,
		domainfinance.NormalBalanceCredit,
		domainfinance.CurrencyINR,
		true,
	)
	require.NoError(t, err)

	platformFee, err := domainfinance.NewAccount(
		uuid.New(),
		domainfinance.AccountCodePlatformFeeRevenue,
		"Platform Fee Revenue",
		domainfinance.AccountTypeRevenue,
		domainfinance.NormalBalanceCredit,
		domainfinance.CurrencyINR,
		true,
	)
	require.NoError(t, err)

	repo.accountsByCode[domainfinance.AccountCodeRazorpayClearing] = clearing
	repo.accountsByCode[domainfinance.AccountCodeOrganizerPayable] = organizer
	repo.accountsByCode[domainfinance.AccountCodePlatformFeeRevenue] = platformFee
}

func createValidInput() domain.RecordPaymentCaptureInput {
	return domain.RecordPaymentCaptureInput{
		PaymentID:     uuid.New(),
		PaymentAmount: decimal.NewFromInt(1050), // 1000 + 50
		Currency:      domainfinance.CurrencyINR,
		BookingID:     uuid.New(),
		Subtotal:      decimal.NewFromInt(1000),
		PlatformFee:   decimal.NewFromInt(50),
		Tax:           decimal.Zero,
		Discount:      decimal.Zero,
		Total:         decimal.NewFromInt(1050),
		EventID:       uuid.New(),
		OrganizerID:   uuid.New(),
		UserID:        uuid.New(),
		CapturedAt:    time.Now().UTC(),
	}
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

func TestNewFinanceUsecase(t *testing.T) {
	t.Run("Rejects nil txManager", func(t *testing.T) {
		uc, err := usecasefinance.NewFinanceUsecase(nil, slog.Default())
		require.Error(t, err)
		assert.Nil(t, uc)
		assert.Contains(t, err.Error(), "txManager cannot be nil")
	})

	t.Run("Accepts valid txManager and defaults nil logger", func(t *testing.T) {
		uc, err := usecasefinance.NewFinanceUsecase(&mockTxManager{}, nil)
		require.NoError(t, err)
		assert.NotNil(t, uc)
	})
}

func TestFinanceUsecase_ValidationRules(t *testing.T) {
	txMgr := &mockTxManager{}
	uc, err := usecasefinance.NewFinanceUsecase(txMgr, slog.Default())
	require.NoError(t, err)

	accRepo := newMockAccountRepo()
	setupStandardAccounts(t, accRepo)
	journalRepo := &mockJournalRepo{}
	txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: journalRepo}
	ctx := context.Background()

	t.Run("Rejects nil txRepos", func(t *testing.T) {
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, nil, createValidInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "txRepos cannot be nil")
	})

	t.Run("Rejects nil PaymentID", func(t *testing.T) {
		in := createValidInput()
		in.PaymentID = uuid.Nil
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payment ID cannot be nil")
	})

	t.Run("Rejects nil BookingID", func(t *testing.T) {
		in := createValidInput()
		in.BookingID = uuid.Nil
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "booking ID cannot be nil")
	})

	t.Run("Rejects nil EventID", func(t *testing.T) {
		in := createValidInput()
		in.EventID = uuid.Nil
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event ID cannot be nil")
	})

	t.Run("Rejects nil OrganizerID", func(t *testing.T) {
		in := createValidInput()
		in.OrganizerID = uuid.Nil
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "organizer ID cannot be nil")
	})

	t.Run("Rejects nil UserID", func(t *testing.T) {
		in := createValidInput()
		in.UserID = uuid.Nil
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be nil")
	})

	t.Run("Rejects unsupported currency", func(t *testing.T) {
		in := createValidInput()
		in.Currency = "USD"
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domainfinance.ErrUnsupportedCurrency))
	})

	t.Run("Rejects negative payment amount", func(t *testing.T) {
		in := createValidInput()
		in.PaymentAmount = decimal.NewFromInt(-100)
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payment amount cannot be negative")
	})

	t.Run("Rejects negative booking total", func(t *testing.T) {
		in := createValidInput()
		in.Total = decimal.NewFromInt(-100)
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "booking total cannot be negative")
	})

	t.Run("Rejects zero payment with positive booking total", func(t *testing.T) {
		in := createValidInput()
		in.PaymentAmount = decimal.Zero
		// in.Total is 2100
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrInconsistentBookingAmounts))
	})

	t.Run("Rejects positive payment with zero booking total", func(t *testing.T) {
		in := createValidInput()
		in.Total = decimal.Zero
		// in.PaymentAmount is 2100
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrInconsistentBookingAmounts))
	})

	t.Run("Rejects negative subtotal", func(t *testing.T) {
		in := createValidInput()
		in.Subtotal = decimal.NewFromInt(-100)
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subtotal cannot be negative")
	})

	t.Run("Rejects negative platform fee", func(t *testing.T) {
		in := createValidInput()
		in.PlatformFee = decimal.NewFromInt(-10)
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "platform fee cannot be negative")
	})

	t.Run("Rejects negative tax", func(t *testing.T) {
		in := createValidInput()
		in.Tax = decimal.NewFromInt(-5)
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tax cannot be negative")
	})

	t.Run("Rejects negative discount", func(t *testing.T) {
		in := createValidInput()
		in.Discount = decimal.NewFromInt(-50)
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "discount cannot be negative")
	})

	t.Run("Rejects fractional paise", func(t *testing.T) {
		in := createValidInput()
		in.PaymentAmount = decimal.RequireFromString("1050.123")
		in.Total = decimal.RequireFromString("1050.123")
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fractional paise")
	})

	t.Run("Rejects breakdown sum mismatch", func(t *testing.T) {
		in := createValidInput()
		// Subtotal 1000 + PlatformFee 50 != Total 1100
		in.Total = decimal.NewFromInt(1100)
		in.PaymentAmount = decimal.NewFromInt(1100)
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrInconsistentBookingAmounts))
		assert.Contains(t, err.Error(), "breakdown sum")
	})

	t.Run("Rejects payment amount mismatch against total", func(t *testing.T) {
		in := createValidInput()
		in.PaymentAmount = decimal.NewFromInt(999) // Total is 1050
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrInconsistentBookingAmounts))
		assert.Contains(t, err.Error(), "payment amount")
	})

	t.Run("Rejects positive tax with ErrTaxAccountNotConfigured", func(t *testing.T) {
		in := createValidInput()
		in.Tax = decimal.NewFromInt(50)
		in.Total = decimal.NewFromInt(1100) // 1000 + 50 + 50
		in.PaymentAmount = decimal.NewFromInt(1100)
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrTaxAccountNotConfigured), "Positive tax must return ErrTaxAccountNotConfigured")
	})

	t.Run("Rejects positive discount with ErrUnsupportedDiscountAllocation", func(t *testing.T) {
		in := createValidInput()
		in.Discount = decimal.NewFromInt(100)
		in.Total = decimal.NewFromInt(950) // 1000 + 50 - 100
		in.PaymentAmount = decimal.NewFromInt(950)
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrUnsupportedDiscountAllocation), "Positive discount must return ErrUnsupportedDiscountAllocation")
	})
}

func TestFinanceUsecase_AccountValidation(t *testing.T) {
	txMgr := &mockTxManager{}
	uc, err := usecasefinance.NewFinanceUsecase(txMgr, slog.Default())
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("Missing Razorpay Clearing account returns error", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)
		delete(accRepo.accountsByCode, domainfinance.AccountCodeRazorpayClearing)

		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: &mockJournalRepo{}}
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, createValidInput())
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrFinanceAccountNotFound))
	})

	t.Run("Inactive Razorpay Clearing account returns ErrFinanceAccountInactive", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)

		inactiveClearing, _ := domainfinance.NewAccount(
			uuid.New(),
			domainfinance.AccountCodeRazorpayClearing,
			"Razorpay Clearing",
			domainfinance.AccountTypeAsset,
			domainfinance.NormalBalanceDebit,
			domainfinance.CurrencyINR,
			false, // inactive
		)
		accRepo.accountsByCode[domainfinance.AccountCodeRazorpayClearing] = inactiveClearing

		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: &mockJournalRepo{}}
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, createValidInput())
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrFinanceAccountInactive))
	})

	t.Run("Missing Organizer Payable account returns error", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)
		delete(accRepo.accountsByCode, domainfinance.AccountCodeOrganizerPayable)

		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: &mockJournalRepo{}}
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, createValidInput())
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrFinanceAccountNotFound))
	})

	t.Run("Inactive Organizer Payable account returns ErrFinanceAccountInactive", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)

		inactiveOrg, _ := domainfinance.NewAccount(
			uuid.New(),
			domainfinance.AccountCodeOrganizerPayable,
			"Organizer Payable",
			domainfinance.AccountTypeLiability,
			domainfinance.NormalBalanceCredit,
			domainfinance.CurrencyINR,
			false, // inactive
		)
		accRepo.accountsByCode[domainfinance.AccountCodeOrganizerPayable] = inactiveOrg

		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: &mockJournalRepo{}}
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, createValidInput())
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrFinanceAccountInactive))
	})

	t.Run("Missing Platform Fee account when fee > 0 returns error", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)
		delete(accRepo.accountsByCode, domainfinance.AccountCodePlatformFeeRevenue)

		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: &mockJournalRepo{}}
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, createValidInput())
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrFinanceAccountNotFound))
	})

	t.Run("Inactive Platform Fee account when fee > 0 returns ErrFinanceAccountInactive", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)

		inactiveFee, _ := domainfinance.NewAccount(
			uuid.New(),
			domainfinance.AccountCodePlatformFeeRevenue,
			"Platform Fee Revenue",
			domainfinance.AccountTypeRevenue,
			domainfinance.NormalBalanceCredit,
			domainfinance.CurrencyINR,
			false, // inactive
		)
		accRepo.accountsByCode[domainfinance.AccountCodePlatformFeeRevenue] = inactiveFee

		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: &mockJournalRepo{}}
		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, createValidInput())
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrFinanceAccountInactive))
	})
}

func TestFinanceUsecase_SuccessfulJournalCreation(t *testing.T) {
	txMgr := &mockTxManager{}
	uc, err := usecasefinance.NewFinanceUsecase(txMgr, slog.Default())
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("Standard booking with platform fee: 1 Debit, 2 Credits", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)
		journalRepo := &mockJournalRepo{}
		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: journalRepo}

		in := createValidInput()
		in.Subtotal = decimal.NewFromInt(2000)
		in.PlatformFee = decimal.NewFromInt(100)
		in.Total = decimal.NewFromInt(2100)
		in.PaymentAmount = decimal.NewFromInt(2100)

		journal, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.NoError(t, err)
		require.NotNil(t, journal)

		// Header validation
		assert.Equal(t, domainfinance.TransactionTypePayment, journal.TransactionType())
		assert.Equal(t, domainfinance.FinancialEventTicketSaleCaptured, journal.FinancialEventType())
		assert.Equal(t, domainfinance.SourceTypePayment, journal.SourceType())
		assert.Equal(t, in.PaymentID.String(), journal.SourceID())
		assert.Equal(t, fmt.Sprintf("payment_capture:%s", in.PaymentID), journal.IdempotencyKey())
		assert.Equal(t, domainfinance.PostingStatusPosted, journal.Status())
		assert.NotNil(t, journal.PostedAt())
		assert.False(t, journal.IsReversed())

		// Double-entry balancing verification
		balanced, balErr := journal.IsBalanced()
		require.NoError(t, balErr)
		assert.True(t, balanced, "Journal must be balanced")

		entries := journal.Entries()
		require.Len(t, entries, 3, "Must have exactly 3 ledger entries (1 debit, 2 credits)")

		// Leg 1: Debit 1010 (Razorpay Clearing)
		debitLeg := entries[0]
		assert.Equal(t, domainfinance.EntryTypeDebit, debitLeg.EntryType())
		assert.Equal(t, accRepo.accountsByCode[domainfinance.AccountCodeRazorpayClearing].ID(), debitLeg.AccountID())
		assert.True(t, debitLeg.Amount().Amount().Equal(decimal.NewFromInt(2100)))
		assert.Nil(t, debitLeg.OrganizerID())

		// Leg 2: Credit 2010 (Organizer Payable)
		creditOrgLeg := entries[1]
		assert.Equal(t, domainfinance.EntryTypeCredit, creditOrgLeg.EntryType())
		assert.Equal(t, accRepo.accountsByCode[domainfinance.AccountCodeOrganizerPayable].ID(), creditOrgLeg.AccountID())
		assert.True(t, creditOrgLeg.Amount().Amount().Equal(decimal.NewFromInt(2000)))
		require.NotNil(t, creditOrgLeg.OrganizerID())
		assert.Equal(t, in.OrganizerID, *creditOrgLeg.OrganizerID())
		require.NotNil(t, creditOrgLeg.EventID())
		assert.Equal(t, in.EventID, *creditOrgLeg.EventID())

		// Leg 3: Credit 4010 (Platform Fee Revenue)
		creditFeeLeg := entries[2]
		assert.Equal(t, domainfinance.EntryTypeCredit, creditFeeLeg.EntryType())
		assert.Equal(t, accRepo.accountsByCode[domainfinance.AccountCodePlatformFeeRevenue].ID(), creditFeeLeg.AccountID())
		assert.True(t, creditFeeLeg.Amount().Amount().Equal(decimal.NewFromInt(100)))
		require.NotNil(t, creditFeeLeg.EventID())
		assert.Equal(t, in.EventID, *creditFeeLeg.EventID())

		// Verify repository recorded the journal
		assert.Len(t, journalRepo.recordedJournals, 1)
	})

	t.Run("Free platform fee: 1 Debit, 1 Credit", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)
		journalRepo := &mockJournalRepo{}
		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: journalRepo}

		in := createValidInput()
		in.Subtotal = decimal.NewFromInt(500)
		in.PlatformFee = decimal.Zero
		in.Total = decimal.NewFromInt(500)
		in.PaymentAmount = decimal.NewFromInt(500)

		journal, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.NoError(t, err)
		require.NotNil(t, journal)

		entries := journal.Entries()
		require.Len(t, entries, 2, "Must have exactly 2 ledger entries when platform fee is zero")

		assert.Equal(t, domainfinance.EntryTypeDebit, entries[0].EntryType())
		assert.Equal(t, accRepo.accountsByCode[domainfinance.AccountCodeRazorpayClearing].ID(), entries[0].AccountID())

		assert.Equal(t, domainfinance.EntryTypeCredit, entries[1].EntryType())
		assert.Equal(t, accRepo.accountsByCode[domainfinance.AccountCodeOrganizerPayable].ID(), entries[1].AccountID())
	})

	t.Run("Persistence error propagates", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)
		dbErr := errors.New("unique constraint violation on idempotency key")
		journalRepo := &mockJournalRepo{recordErr: dbErr}
		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: journalRepo}

		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, createValidInput())
		require.Error(t, err)
		assert.True(t, errors.Is(err, dbErr))
	})

	t.Run("Standalone RecordPaymentCapture runs inside transaction", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)
		journalRepo := &mockJournalRepo{}

		txCalled := false
		customTxMgr := &mockTxManager{
			withinTxFn: func(fn func(tx domain.TransactionRepositories) error) error {
				txCalled = true
				return fn(&mockTxRepos{accountRepo: accRepo, journalRepo: journalRepo})
			},
		}

		ucWithTx, err := usecasefinance.NewFinanceUsecase(customTxMgr, slog.Default())
		require.NoError(t, err)

		journal, err := ucWithTx.RecordPaymentCapture(ctx, createValidInput())
		require.NoError(t, err)
		require.NotNil(t, journal)
		assert.True(t, txCalled, "WithinTransaction must have been invoked")
	})

	t.Run("Standalone RecordPaymentCapture propagates transaction error", func(t *testing.T) {
		txErr := errors.New("tx serialization failure")
		customTxMgr := &mockTxManager{
			withinTxFn: func(fn func(tx domain.TransactionRepositories) error) error {
				return txErr
			},
		}

		ucWithTx, err := usecasefinance.NewFinanceUsecase(customTxMgr, slog.Default())
		require.NoError(t, err)

		journal, err := ucWithTx.RecordPaymentCapture(ctx, createValidInput())
		require.Error(t, err)
		assert.Nil(t, journal)
		assert.True(t, errors.Is(err, txErr))
	})

	t.Run("Free booking: zero payment and zero total returns nil journal successfully without repo recording", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)
		journalRepo := &mockJournalRepo{}
		txRepos := &mockTxRepos{accountRepo: accRepo, journalRepo: journalRepo}

		freeInput := createValidInput()
		freeInput.Subtotal = decimal.Zero
		freeInput.PlatformFee = decimal.Zero
		freeInput.Tax = decimal.Zero
		freeInput.Discount = decimal.Zero
		freeInput.Total = decimal.Zero
		freeInput.PaymentAmount = decimal.Zero

		journal, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, freeInput)
		require.NoError(t, err, "Free booking must return successfully without error")
		assert.Nil(t, journal, "Free booking must return nil journal")
		assert.Empty(t, journalRepo.recordedJournals, "No financial journal must be recorded for free booking")
	})

	t.Run("Free booking: standalone RecordPaymentCapture returns nil journal successfully", func(t *testing.T) {
		accRepo := newMockAccountRepo()
		setupStandardAccounts(t, accRepo)
		journalRepo := &mockJournalRepo{}

		txMgr := &mockTxManager{
			withinTxFn: func(fn func(tx domain.TransactionRepositories) error) error {
				return fn(&mockTxRepos{accountRepo: accRepo, journalRepo: journalRepo})
			},
		}

		ucWithTx, err := usecasefinance.NewFinanceUsecase(txMgr, slog.Default())
		require.NoError(t, err)

		freeInput := createValidInput()
		freeInput.Subtotal = decimal.Zero
		freeInput.PlatformFee = decimal.Zero
		freeInput.Tax = decimal.Zero
		freeInput.Discount = decimal.Zero
		freeInput.Total = decimal.Zero
		freeInput.PaymentAmount = decimal.Zero

		journal, err := ucWithTx.RecordPaymentCapture(ctx, freeInput)
		require.NoError(t, err)
		assert.Nil(t, journal)
		assert.Empty(t, journalRepo.recordedJournals)
	})

	t.Run("Free booking: rejected when breakdown is inconsistent with zero total", func(t *testing.T) {
		txRepos := &mockTxRepos{}
		in := createValidInput()
		in.Subtotal = decimal.NewFromInt(100) // positive subtotal
		in.PlatformFee = decimal.Zero
		in.Tax = decimal.Zero
		in.Discount = decimal.Zero
		in.Total = decimal.Zero
		in.PaymentAmount = decimal.Zero

		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrInconsistentBookingAmounts))
	})

	t.Run("Free booking: rejected when tax is positive", func(t *testing.T) {
		txRepos := &mockTxRepos{}
		in := createValidInput()
		in.Subtotal = decimal.Zero
		in.PlatformFee = decimal.Zero
		in.Tax = decimal.NewFromInt(10)
		in.Discount = decimal.NewFromInt(10) // sum is 0
		in.Total = decimal.Zero
		in.PaymentAmount = decimal.Zero

		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrTaxAccountNotConfigured))
	})

	t.Run("Free booking: rejected when discount is positive", func(t *testing.T) {
		txRepos := &mockTxRepos{}
		in := createValidInput()
		in.Subtotal = decimal.NewFromInt(50)
		in.PlatformFee = decimal.Zero
		in.Tax = decimal.Zero
		in.Discount = decimal.NewFromInt(50) // 50 - 50 = 0
		in.Total = decimal.Zero
		in.PaymentAmount = decimal.Zero

		_, err := uc.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrUnsupportedDiscountAllocation))
	})
}
