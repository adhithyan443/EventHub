package database_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/database"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// mockFinanceAccountRepo implements domain.FinanceAccountRepository for deterministic unit testing.
type mockFinanceAccountRepo struct {
	mu             sync.Mutex
	accountsByCode map[string]*finance.Account
	accountsByID   map[uuid.UUID]*finance.Account

	findByCodeErr     map[string]error
	createErr         map[string]error
	createHooks       map[string]func(acc *finance.Account)
	findByCodeCallCnt map[string]int
	createCallCnt     map[string]int
}

func newMockFinanceAccountRepo() *mockFinanceAccountRepo {
	return &mockFinanceAccountRepo{
		accountsByCode:    make(map[string]*finance.Account),
		accountsByID:      make(map[uuid.UUID]*finance.Account),
		findByCodeErr:     make(map[string]error),
		createErr:         make(map[string]error),
		createHooks:       make(map[string]func(acc *finance.Account)),
		findByCodeCallCnt: make(map[string]int),
		createCallCnt:     make(map[string]int),
	}
}

func (m *mockFinanceAccountRepo) Create(ctx context.Context, account *finance.Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if account == nil {
		return errors.New("account cannot be nil")
	}

	m.createCallCnt[account.Code()]++

	if err, exists := m.createErr[account.Code()]; exists && err != nil {
		if hook, hasHook := m.createHooks[account.Code()]; hasHook {
			hook(account)
		}
		return err
	}

	if _, exists := m.accountsByCode[account.Code()]; exists {
		return domain.ErrFinanceAccountAlreadyExists
	}

	m.accountsByCode[account.Code()] = account
	m.accountsByID[account.ID()] = account
	return nil
}

func (m *mockFinanceAccountRepo) FindByID(ctx context.Context, id uuid.UUID) (*finance.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	acc, exists := m.accountsByID[id]
	if !exists {
		return nil, domain.ErrFinanceAccountNotFound
	}
	return acc, nil
}

func (m *mockFinanceAccountRepo) FindByCode(ctx context.Context, code string) (*finance.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanCode := strings.TrimSpace(code)
	m.findByCodeCallCnt[cleanCode]++

	if err, exists := m.findByCodeErr[cleanCode]; exists && err != nil {
		return nil, err
	}

	acc, exists := m.accountsByCode[cleanCode]
	if !exists {
		return nil, domain.ErrFinanceAccountNotFound
	}
	return acc, nil
}

// -----------------------------------------------------------------------------
// Unit Tests: ValidateFinanceAccountConfig
// -----------------------------------------------------------------------------

func TestValidateFinanceAccountConfig(t *testing.T) {
	expected := database.ExpectedFinanceAccount{
		Code:          "1010",
		Name:          "Razorpay Clearing",
		Type:          finance.AccountTypeAsset,
		NormalBalance: finance.NormalBalanceDebit,
		Currency:      finance.CurrencyINR,
		IsActive:      true,
	}

	t.Run("Valid account matching expected config passes", func(t *testing.T) {
		acc, err := finance.NewAccount(
			uuid.New(),
			expected.Code,
			expected.Name,
			expected.Type,
			expected.NormalBalance,
			expected.Currency,
			expected.IsActive,
		)
		require.NoError(t, err)

		err = database.ValidateFinanceAccountConfig(acc, expected)
		assert.NoError(t, err)
	})

	t.Run("Nil account returns error", func(t *testing.T) {
		err := database.ValidateFinanceAccountConfig(nil, expected)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("Code mismatch returns error", func(t *testing.T) {
		acc, err := finance.NewAccount(
			uuid.New(),
			"1020",
			expected.Name,
			expected.Type,
			expected.NormalBalance,
			expected.Currency,
			expected.IsActive,
		)
		require.NoError(t, err)

		err = database.ValidateFinanceAccountConfig(acc, expected)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "code mismatch")
	})

	t.Run("Type mismatch returns error", func(t *testing.T) {
		// Valid domain account with liability type
		acc, err := finance.NewAccount(
			uuid.New(),
			expected.Code,
			expected.Name,
			finance.AccountTypeLiability,
			finance.NormalBalanceCredit,
			expected.Currency,
			expected.IsActive,
		)
		require.NoError(t, err)

		err = database.ValidateFinanceAccountConfig(acc, expected)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "type mismatch")
	})

	t.Run("Normal balance mismatch returns error", func(t *testing.T) {
		// Create expected with conflicting normal balance
		conflictingExpected := expected
		conflictingExpected.NormalBalance = finance.NormalBalanceCredit

		acc, err := finance.NewAccount(
			uuid.New(),
			expected.Code,
			expected.Name,
			expected.Type,
			expected.NormalBalance,
			expected.Currency,
			expected.IsActive,
		)
		require.NoError(t, err)

		err = database.ValidateFinanceAccountConfig(acc, conflictingExpected)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "normal balance mismatch")
	})

	t.Run("Currency mismatch returns error", func(t *testing.T) {
		conflictingExpected := expected
		conflictingExpected.Currency = "USD"

		acc, err := finance.NewAccount(
			uuid.New(),
			expected.Code,
			expected.Name,
			expected.Type,
			expected.NormalBalance,
			expected.Currency,
			expected.IsActive,
		)
		require.NoError(t, err)

		err = database.ValidateFinanceAccountConfig(acc, conflictingExpected)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "currency mismatch")
	})

	t.Run("Active status mismatch returns error", func(t *testing.T) {
		acc, err := finance.NewAccount(
			uuid.New(),
			expected.Code,
			expected.Name,
			expected.Type,
			expected.NormalBalance,
			expected.Currency,
			false, // inactive
		)
		require.NoError(t, err)

		err = database.ValidateFinanceAccountConfig(acc, expected)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "active status mismatch")
	})

	t.Run("Name mismatch returns error", func(t *testing.T) {
		acc, err := finance.NewAccount(
			uuid.New(),
			expected.Code,
			"Different Account Name",
			expected.Type,
			expected.NormalBalance,
			expected.Currency,
			expected.IsActive,
		)
		require.NoError(t, err)

		err = database.ValidateFinanceAccountConfig(acc, expected)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "name mismatch")
	})
}

// -----------------------------------------------------------------------------
// Unit Tests: SeedFinanceAccountsWithRepo
// -----------------------------------------------------------------------------

func TestSeedFinanceAccountsWithRepo(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("Clean initial seed of all 7 accounts", func(t *testing.T) {
		repo := newMockFinanceAccountRepo()

		err := database.SeedFinanceAccountsWithRepo(ctx, repo, logger)
		require.NoError(t, err)

		assert.Len(t, repo.accountsByCode, len(database.StandardFinanceAccounts))
		for _, exp := range database.StandardFinanceAccounts {
			acc, exists := repo.accountsByCode[exp.Code]
			require.True(t, exists, "Account %s must be seeded", exp.Code)
			assert.Equal(t, exp.Name, acc.Name())
			assert.Equal(t, exp.Type, acc.Type())
			assert.Equal(t, exp.NormalBalance, acc.NormalBalance())
			assert.Equal(t, exp.Currency, acc.Currency())
			assert.Equal(t, exp.IsActive, acc.IsActive())
		}
	})

	t.Run("Idempotent re-run on already seeded database", func(t *testing.T) {
		repo := newMockFinanceAccountRepo()

		// First run: seeds all accounts
		err := database.SeedFinanceAccountsWithRepo(ctx, repo, logger)
		require.NoError(t, err)

		initialCreateCounts := make(map[string]int)
		for k, v := range repo.createCallCnt {
			initialCreateCounts[k] = v
		}

		// Second run: must detect all existing accounts and skip creation safely
		err = database.SeedFinanceAccountsWithRepo(ctx, repo, logger)
		require.NoError(t, err)

		// Create must not have been called again for any account
		for k, v := range repo.createCallCnt {
			assert.Equal(t, initialCreateCounts[k], v, "Create call count must not increase for %s", k)
		}
	})

	t.Run("Detection and error return on configuration mismatch", func(t *testing.T) {
		repo := newMockFinanceAccountRepo()

		// Pre-populate an account with mismatched type (EXPENSE instead of ASSET)
		tamperedAcc, err := finance.NewAccount(
			uuid.New(),
			finance.AccountCodeRazorpayClearing, // 1010
			"Razorpay Clearing",
			finance.AccountTypeExpense,
			finance.NormalBalanceDebit,
			finance.CurrencyINR,
			true,
		)
		require.NoError(t, err)
		repo.accountsByCode[tamperedAcc.Code()] = tamperedAcc
		repo.accountsByID[tamperedAcc.ID()] = tamperedAcc

		err = database.SeedFinanceAccountsWithRepo(ctx, repo, logger)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "type mismatch")
	})

	t.Run("Concurrent insert race condition handled gracefully", func(t *testing.T) {
		repo := newMockFinanceAccountRepo()

		// Simulate race condition for 1010:
		// 1. Initial FindByCode returns ErrFinanceAccountNotFound (default for empty repo)
		// 2. Create returns ErrFinanceAccountAlreadyExists, and hook simulates another worker inserting the valid account
		repo.createErr[finance.AccountCodeRazorpayClearing] = domain.ErrFinanceAccountAlreadyExists
		repo.createHooks[finance.AccountCodeRazorpayClearing] = func(acc *finance.Account) {
			validAcc, _ := finance.NewAccount(
				uuid.New(),
				finance.AccountCodeRazorpayClearing,
				"Razorpay Clearing",
				finance.AccountTypeAsset,
				finance.NormalBalanceDebit,
				finance.CurrencyINR,
				true,
			)
			repo.accountsByCode[finance.AccountCodeRazorpayClearing] = validAcc
		}

		err := database.SeedFinanceAccountsWithRepo(ctx, repo, logger)
		require.NoError(t, err, "Seeder must recover from concurrent insert and validate successfully")

		// Verify account exists and is valid
		acc, exists := repo.accountsByCode[finance.AccountCodeRazorpayClearing]
		require.True(t, exists)
		assert.Equal(t, "Razorpay Clearing", acc.Name())
	})

	t.Run("Concurrent insert with mismatched configuration fails fatally", func(t *testing.T) {
		repo := newMockFinanceAccountRepo()

		// Simulate race condition where the concurrently inserted account is invalid/mismatched
		repo.createErr[finance.AccountCodeRazorpayClearing] = domain.ErrFinanceAccountAlreadyExists
		repo.createHooks[finance.AccountCodeRazorpayClearing] = func(acc *finance.Account) {
			mismatchedAcc, _ := finance.NewAccount(
				uuid.New(),
				finance.AccountCodeRazorpayClearing,
				"Wrong Name",
				finance.AccountTypeAsset,
				finance.NormalBalanceDebit,
				finance.CurrencyINR,
				true,
			)
			repo.accountsByCode[finance.AccountCodeRazorpayClearing] = mismatchedAcc
		}

		err := database.SeedFinanceAccountsWithRepo(ctx, repo, logger)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "name mismatch")
	})

	t.Run("Concurrent insert with lookup failure propagates error", func(t *testing.T) {
		repo := newMockFinanceAccountRepo()

		dbErr := errors.New("database connection reset")
		repo.createErr[finance.AccountCodeRazorpayClearing] = domain.ErrFinanceAccountAlreadyExists
		repo.createHooks[finance.AccountCodeRazorpayClearing] = func(acc *finance.Account) {
			repo.findByCodeErr[finance.AccountCodeRazorpayClearing] = dbErr
		}

		err := database.SeedFinanceAccountsWithRepo(ctx, repo, logger)
		require.Error(t, err)
		assert.True(t, errors.Is(err, dbErr))
	})

	t.Run("Initial lookup error aborts seeding", func(t *testing.T) {
		repo := newMockFinanceAccountRepo()
		dbErr := errors.New("timeout querying postgres")
		repo.findByCodeErr[finance.AccountCodeRazorpayClearing] = dbErr

		err := database.SeedFinanceAccountsWithRepo(ctx, repo, logger)
		require.Error(t, err)
		assert.True(t, errors.Is(err, dbErr))
	})

	t.Run("Unexpected creation error aborts seeding", func(t *testing.T) {
		repo := newMockFinanceAccountRepo()
		dbErr := errors.New("disk full")
		repo.createErr[finance.AccountCodeRazorpayClearing] = dbErr

		err := database.SeedFinanceAccountsWithRepo(ctx, repo, logger)
		require.Error(t, err)
		assert.True(t, errors.Is(err, dbErr))
	})
}

// -----------------------------------------------------------------------------
// PostgreSQL Integration Tests
// -----------------------------------------------------------------------------

func getTestDSN() string {
	candidates := []string{
		".env.test",
		".env",
		"../.env.test",
		"../.env",
		"../../.env.test",
		"../../.env",
		"../../../.env.test",
		"../../../.env",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
		}
	}

	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		raw = os.Getenv("DATABASE_URL")
	}
	clean := strings.Trim(strings.TrimSpace(raw), `"'`)
	if clean != "" && (strings.Contains(clean, "localhost") || strings.Contains(clean, "127.0.0.1")) {
		return clean
	}
	return ""
}

func loadMigrationSQL(filename string) (string, error) {
	candidates := []string{
		filepath.Join("..", "..", "migrations", filename),
		filepath.Join("..", "migrations", filename),
		filepath.Join("migrations", filename),
		filepath.Join("backend", "migrations", filename),
	}
	for _, p := range candidates {
		if content, err := os.ReadFile(p); err == nil {
			return string(content), nil
		}
	}
	return "", fmt.Errorf("migration file %s not found", filename)
}

func TestSeedFinanceAccounts_Integration(t *testing.T) {
	baseDSN := getTestDSN()
	if baseDSN == "" {
		t.Skip("No test database DSN available; skipping PostgreSQL integration test")
	}

	rootDB, err := gorm.Open(postgres.Open(baseDSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "Failed to connect to postgres for seeding integration test")

	// Create an isolated test schema
	schema := fmt.Sprintf("test_seed_%d_%d", time.Now().UnixNano(), os.Getpid())
	require.NoError(t, rootDB.Exec(fmt.Sprintf("CREATE SCHEMA %s;", schema)).Error)

	defer func() {
		_ = rootDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", schema)).Error
	}()

	// Connect to the isolated schema
	schemaDSN := fmt.Sprintf("%s search_path=%s", baseDSN, schema)
	schemaDB, err := gorm.Open(postgres.Open(schemaDSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	// Apply migration 000017
	upSQL, err := loadMigrationSQL("000017_create_finance_ledger.up.sql")
	require.NoError(t, err)
	require.NoError(t, schemaDB.Exec(upSQL).Error)

	logger := slog.Default()

	t.Run("Clean initial seed populates all 7 accounts in Postgres", func(t *testing.T) {
		err := database.SeedFinanceAccounts(schemaDB, logger)
		require.NoError(t, err)

		var count int64
		require.NoError(t, schemaDB.Table("finance_accounts").Count(&count).Error)
		assert.Equal(t, int64(7), count, "Must seed exactly 7 standard accounts")

		// Verify each account row
		for _, exp := range database.StandardFinanceAccounts {
			var row struct {
				ID            uuid.UUID
				Code          string
				Name          string
				Type          string
				NormalBalance string
				Currency      string
				IsActive      bool
			}
			err := schemaDB.Table("finance_accounts").Where("code = ?", exp.Code).First(&row).Error
			require.NoError(t, err, "Account %s must exist in DB", exp.Code)
			assert.Equal(t, exp.Name, row.Name)
			assert.Equal(t, string(exp.Type), row.Type)
			assert.Equal(t, string(exp.NormalBalance), row.NormalBalance)
			assert.Equal(t, exp.Currency, row.Currency)
			assert.Equal(t, exp.IsActive, row.IsActive)
		}
	})

	t.Run("Idempotent re-run on already seeded database succeeds without changes", func(t *testing.T) {
		err := database.SeedFinanceAccounts(schemaDB, logger)
		require.NoError(t, err, "Idempotent re-run must succeed without errors")

		var count int64
		require.NoError(t, schemaDB.Table("finance_accounts").Count(&count).Error)
		assert.Equal(t, int64(7), count, "Account count must remain exactly 7")
	})

	t.Run("Configuration mismatch in database returns fatal error", func(t *testing.T) {
		// Tamper with account name for 1010 directly in Postgres
		err := schemaDB.Exec("UPDATE finance_accounts SET name = 'Altered Name' WHERE code = '1010';").Error
		require.NoError(t, err)

		err = database.SeedFinanceAccounts(schemaDB, logger)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "name mismatch")

		// Restore for other tests
		err = schemaDB.Exec("UPDATE finance_accounts SET name = 'Razorpay Clearing' WHERE code = '1010';").Error
		require.NoError(t, err)
	})

	t.Run("Tampering with is_active flag returns fatal error", func(t *testing.T) {
		// Set account 2010 to inactive
		err := schemaDB.Exec("UPDATE finance_accounts SET is_active = FALSE WHERE code = '2010';").Error
		require.NoError(t, err)

		err = database.SeedFinanceAccounts(schemaDB, logger)
		require.Error(t, err)
		assert.True(t, errors.Is(err, database.ErrFinanceAccountConfigMismatch))
		assert.Contains(t, err.Error(), "active status mismatch")

		// Restore
		err = schemaDB.Exec("UPDATE finance_accounts SET is_active = TRUE WHERE code = '2010';").Error
		require.NoError(t, err)
	})
}
