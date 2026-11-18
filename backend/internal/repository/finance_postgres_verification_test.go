package repository_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	ErrIdempotencyConflict = domain.ErrIdempotencyConflict
	ErrConcurrentPosting   = domain.ErrConcurrentPosting
)

type CanonicalEntry struct {
	AccountID   string `json:"account_id"`
	EntryType   string `json:"entry_type"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	OrganizerID string `json:"organizer_id,omitempty"`
	EventID     string `json:"event_id,omitempty"`
}

type CanonicalJournal struct {
	TransactionType     string           `json:"transaction_type"`
	FinancialEventType  string           `json:"financial_event_type"`
	SourceType          string           `json:"source_type"`
	SourceID            string           `json:"source_id"`
	Currency            string           `json:"currency"`
	Description         string           `json:"description"`
	ReversalReferenceID string           `json:"reversal_reference_id,omitempty"`
	Entries             []CanonicalEntry `json:"entries"`
}

func computeJournalFingerprint(j *finance.JournalTransaction) (string, error) {
	if j == nil {
		return "", errors.New("nil journal")
	}

	entries := j.Entries()
	canonicalEntries := make([]CanonicalEntry, len(entries))
	for i, e := range entries {
		var orgID string
		if e.OrganizerID() != nil {
			orgID = e.OrganizerID().String()
		}
		var evtID string
		if e.EventID() != nil {
			evtID = e.EventID().String()
		}

		canonicalEntries[i] = CanonicalEntry{
			AccountID:   e.AccountID().String(),
			EntryType:   string(e.EntryType()),
			Amount:      e.Amount().Amount().StringFixed(2),
			Currency:    e.Currency(),
			OrganizerID: orgID,
			EventID:     evtID,
		}
	}

	sort.Slice(canonicalEntries, func(i, j int) bool {
		if canonicalEntries[i].AccountID != canonicalEntries[j].AccountID {
			return canonicalEntries[i].AccountID < canonicalEntries[j].AccountID
		}
		if canonicalEntries[i].EntryType != canonicalEntries[j].EntryType {
			return canonicalEntries[i].EntryType < canonicalEntries[j].EntryType
		}
		if canonicalEntries[i].Amount != canonicalEntries[j].Amount {
			return canonicalEntries[i].Amount < canonicalEntries[j].Amount
		}
		if canonicalEntries[i].OrganizerID != canonicalEntries[j].OrganizerID {
			return canonicalEntries[i].OrganizerID < canonicalEntries[j].OrganizerID
		}
		return canonicalEntries[i].EventID < canonicalEntries[j].EventID
	})

	var revRef string
	if j.ReversalReferenceID() != nil {
		revRef = j.ReversalReferenceID().String()
	}

	canonical := CanonicalJournal{
		TransactionType:     string(j.TransactionType()),
		FinancialEventType:  string(j.FinancialEventType()),
		SourceType:          string(j.SourceType()),
		SourceID:            j.SourceID(),
		Currency:            j.Currency(),
		Description:         j.Description(),
		ReversalReferenceID: revRef,
		Entries:             canonicalEntries,
	}

	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}

type FinanceAccountModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code          string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name          string    `gorm:"type:varchar(255);not null"`
	Type          string    `gorm:"type:varchar(50);not null"`
	NormalBalance string    `gorm:"type:varchar(10);not null"`
	Currency      string    `gorm:"type:varchar(3);not null;default:'INR'"`
	IsActive      bool      `gorm:"not null;default:true"`
	CreatedAt     time.Time `gorm:"not null"`
}

func (FinanceAccountModel) TableName() string {
	return "finance_accounts"
}

type FinanceIdempotencyKeyModel struct {
	IdempotencyKey string    `gorm:"type:varchar(255);primaryKey"`
	Fingerprint    string    `gorm:"type:char(64);not null"`
	JournalID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	CreatedAt      time.Time `gorm:"not null"`
}

func (FinanceIdempotencyKeyModel) TableName() string {
	return "finance_idempotency_keys"
}

type FinanceJournalModel struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	TransactionType        string     `gorm:"type:varchar(50);not null"`
	FinancialEventType     string     `gorm:"type:varchar(50);not null"`
	SourceType             string     `gorm:"type:varchar(50);not null"`
	SourceID               string     `gorm:"type:varchar(255);not null"`
	IdempotencyKey         string     `gorm:"type:varchar(255);not null;uniqueIndex"`
	IdempotencyFingerprint string     `gorm:"type:char(64);not null"`
	Currency               string     `gorm:"type:varchar(3);not null;default:'INR'"`
	Description            string     `gorm:"type:text;not null"`
	Status                 string     `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	CreatedAt              time.Time  `gorm:"not null"`
	PostedAt               *time.Time `gorm:"type:timestamptz"`
	IsReversed             bool       `gorm:"not null;default:false"`
	ReversalReferenceID    *uuid.UUID `gorm:"type:uuid"`
	ReversedByJournalID    *uuid.UUID `gorm:"type:uuid"`
	ReversedAt             *time.Time `gorm:"type:timestamptz"`
}

func (FinanceJournalModel) TableName() string {
	return "finance_journals"
}

type FinanceLedgerEntryModel struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey"`
	JournalID   uuid.UUID       `gorm:"type:uuid;not null;index"`
	AccountID   uuid.UUID       `gorm:"type:uuid;not null;index"`
	EntryType   string          `gorm:"type:varchar(10);not null"`
	Amount      decimal.Decimal `gorm:"type:numeric(20,2);not null"`
	Currency    string          `gorm:"type:varchar(3);not null;default:'INR'"`
	OrganizerID *uuid.UUID      `gorm:"type:uuid;index"`
	EventID     *uuid.UUID      `gorm:"type:uuid;index"`
	CreatedAt   time.Time       `gorm:"not null"`
}

func (FinanceLedgerEntryModel) TableName() string {
	return "finance_ledger_entries"
}

type txCtxKey struct{}

func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

func TxFromCtx(ctx context.Context) (*gorm.DB, bool) {
	if tx, ok := ctx.Value(txCtxKey{}).(*gorm.DB); ok && tx != nil {
		return tx, true
	}
	return nil, false
}

type TestFinanceRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewTestFinanceRepository(db *gorm.DB, logger *slog.Logger) *TestFinanceRepository {
	return &TestFinanceRepository{db: db, logger: logger}
}

func (r *TestFinanceRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := TxFromCtx(ctx); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *TestFinanceRepository) isTx(db *gorm.DB) bool {
	if db == nil || db.Statement == nil {
		return false
	}
	_, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok
}

func (r *TestFinanceRepository) executeInTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	db := r.getDB(ctx)
	if r.isTx(db) {
		return fn(db)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

func (r *TestFinanceRepository) RecordJournal(ctx context.Context, journal *finance.JournalTransaction) (*finance.JournalTransaction, error) {
	db := r.getDB(ctx)
	repo := repository.NewFinanceJournalRepository(db, r.logger)
	return repo.RecordJournal(ctx, journal)
}

const testSchemaPrefix = "test_finance_run_"

var (
	testExecutionSchema string
	testSchemaCreated   bool
	testExecutionMu     sync.Mutex
)

// validateSafeTestSchema ensures that operations only touch uniquely generated test schemas
// and strictly protects production, application, or system schemas from being modified or dropped.
func validateSafeTestSchema(schemaName string) error {
	if schemaName == "" {
		return errors.New("test schema name cannot be empty")
	}

	// Strictly prohibit dropping or modifying any production, application, or system schemas
	protectedSchemas := map[string]bool{
		"public":             true,
		"information_schema": true,
		"pg_catalog":         true,
		"pg_toast":           true,
		"eventhub":           true,
		"finance":            true,
	}
	if protectedSchemas[strings.ToLower(schemaName)] {
		return fmt.Errorf("safety check triggered: refusing to drop or modify protected schema %q", schemaName)
	}

	if !strings.HasPrefix(schemaName, testSchemaPrefix) {
		return fmt.Errorf("safety check triggered: schema %q does not have required test prefix %q", schemaName, testSchemaPrefix)
	}

	for _, r := range schemaName {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Errorf("invalid identifier character in schema name %q", schemaName)
		}
	}

	return nil
}

// validateDatabaseName ensures the database name does not indicate a production database.
func validateDatabaseName(dbname string) error {
	dbnameLower := strings.ToLower(strings.TrimSpace(dbname))
	if dbnameLower == "" {
		return nil
	}
	productionKeywords := []string{"prod", "production", "live"}
	for _, kw := range productionKeywords {
		if strings.Contains(dbnameLower, kw) {
			return fmt.Errorf("safety check triggered: database name %q indicates a production database; integration tests rejected", dbname)
		}
	}
	return nil
}

// validateTargetDatabase inspects the DSN, environment flags, and database metadata
// to ensure tests run only against safe local/test databases and never against production targets.
func validateTargetDatabase(dsn string) error {
	// 1. Check APP_ENV / ENVIRONMENT
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if appEnv == "production" || appEnv == "prod" || appEnv == "live" {
		return fmt.Errorf("safety check triggered: refusing to run integration tests when APP_ENV=%q", appEnv)
	}

	// 2. Parse DSN to inspect host and dbname
	var host, dbname string
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return fmt.Errorf("invalid TEST_DATABASE_URL: %w", err)
		}
		host = u.Hostname()
		dbname = strings.TrimPrefix(u.Path, "/")
	} else {
		// Key-value DSN format
		for _, part := range strings.Fields(dsn) {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				k := strings.ToLower(strings.TrimSpace(kv[0]))
				v := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
				if k == "host" {
					host = v
				} else if k == "dbname" {
					dbname = v
				}
			}
		}
	}

	if host == "" {
		host = "localhost"
	}

	hostLower := strings.ToLower(host)

	// 3. Reject known production / remote cloud indicators
	productionHostPatterns := []string{
		"rds.amazonaws.com",
		"cloudsql",
		"database.azure.com",
		"cockroachlabs.cloud",
		"neon.tech",
		"supabase.co",
		"render.com",
		"railway.app",
		"prod", "production", "live",
	}
	for _, pattern := range productionHostPatterns {
		if strings.Contains(hostLower, pattern) {
			return fmt.Errorf("safety check triggered: host %q contains production indicator %q; integration tests rejected", host, pattern)
		}
	}

	// 4. Reject production database names
	if err := validateDatabaseName(dbname); err != nil {
		return err
	}

	// 5. Enforce safe target host: must be local, test container, or explicitly test-designated
	safeHostPrefixes := []string{"localhost", "127.0.0.1", "::1", "0.0.0.0", "postgres", "test", "docker"}
	isSafeHost := false
	for _, sh := range safeHostPrefixes {
		if hostLower == sh || strings.HasPrefix(hostLower, sh) || strings.Contains(hostLower, "test") || strings.Contains(hostLower, "local") {
			isSafeHost = true
			break
		}
	}
	if !isSafeHost {
		return fmt.Errorf("safety check triggered: host %q is not recognized as a safe local or test database host", host)
	}

	return nil
}

// generateUniqueTestSchemaName creates a unique schema name for each test run.
func generateUniqueTestSchemaName() string {
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return fmt.Sprintf("%s%d_%x", testSchemaPrefix, time.Now().UnixNano(), time.Now().UnixNano()&0xFFFF)
	}
	return fmt.Sprintf("%s%d_%x", testSchemaPrefix, time.Now().UnixNano(), randomBytes)
}

func loadEnvFiles() {
	candidates := []string{
		".env.test",
		".env",
		"../.env.test",
		"../.env",
		"../../.env.test",
		"../../.env",
		"../../../.env.test",
		"../../../.env",
		"../../../../.env.test",
		"../../../../.env",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
		}
	}
}

// getBaseTestDSN retrieves the PostgreSQL connection configuration from the environment.
// TEST_DATABASE_URL is explicitly required; fallback to DATABASE_URL is strictly prohibited.
func getBaseTestDSN() (string, error) {
	loadEnvFiles()

	rawURL := os.Getenv("TEST_DATABASE_URL")
	if strings.TrimSpace(rawURL) == "" {
		return "", errors.New("TEST_DATABASE_URL environment variable is explicitly required for finance integration tests; fallback to DATABASE_URL is disabled for safety")
	}

	dsn := strings.Trim(strings.TrimSpace(rawURL), `"'`)
	if err := validateTargetDatabase(dsn); err != nil {
		return "", fmt.Errorf("target database safety validation failed: %w", err)
	}

	return dsn, nil
}

// buildDSNWithSearchPath appends the test schema to the search_path of the DSN.
func buildDSNWithSearchPath(baseDSN string, schemaName string) (string, error) {
	if err := validateSafeTestSchema(schemaName); err != nil {
		return "", err
	}

	if strings.HasPrefix(baseDSN, "postgres://") || strings.HasPrefix(baseDSN, "postgresql://") {
		u, err := url.Parse(baseDSN)
		if err != nil {
			return "", fmt.Errorf("failed to parse database URL: %w", err)
		}
		q := u.Query()
		q.Set("search_path", schemaName)
		u.RawQuery = q.Encode()
		return u.String(), nil
	}

	// Key-value DSN format
	parts := strings.Fields(baseDSN)
	var filtered []string
	for _, p := range parts {
		if !strings.HasPrefix(strings.ToLower(p), "search_path=") {
			filtered = append(filtered, p)
		}
	}
	filtered = append(filtered, fmt.Sprintf("search_path=%s", schemaName))
	return strings.Join(filtered, " "), nil
}

// dropTestSchema safely removes only the specific uniquely named test schema.
func dropTestSchema(schemaName string) error {
	if err := validateSafeTestSchema(schemaName); err != nil {
		return fmt.Errorf("safety validation failed: %w", err)
	}

	baseDSN, err := getBaseTestDSN()
	if err != nil {
		return fmt.Errorf("cannot resolve base DSN for cleanup: %w", err)
	}

	rootDB, err := gorm.Open(postgres.Open(baseDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database for cleanup of schema %s: %w", schemaName, err)
	}

	sqlDB, err := rootDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB for cleanup: %w", err)
	}
	defer sqlDB.Close()

	var currentDB string
	if err := rootDB.Raw("SELECT current_database();").Scan(&currentDB).Error; err != nil {
		return fmt.Errorf("failed to inspect database name during cleanup: %w", err)
	}
	if err := validateDatabaseName(currentDB); err != nil {
		return fmt.Errorf("cleanup aborted by safety check: %w", err)
	}

	dropSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", schemaName)
	if err := rootDB.Exec(dropSQL).Error; err != nil {
		return fmt.Errorf("failed to drop test schema %s: %w", schemaName, err)
	}

	return nil
}

func TestMain(m *testing.M) {
	testExecutionMu.Lock()
	testExecutionSchema = generateUniqueTestSchemaName()
	testExecutionMu.Unlock()

	code := m.Run()

	// Clean up ONLY the schema created by this test execution run; report errors clearly
	testExecutionMu.Lock()
	schemaToDrop := testExecutionSchema
	wasCreated := testSchemaCreated
	testExecutionMu.Unlock()

	if wasCreated && schemaToDrop != "" {
		if err := dropTestSchema(schemaToDrop); err != nil {
			fmt.Fprintf(os.Stderr, "CLEANUP ERROR: failed to clean up test schema %q: %v\n", schemaToDrop, err)
		} else {
			fmt.Fprintf(os.Stdout, "CLEANUP SUCCESS: dropped test schema %q\n", schemaToDrop)
		}
	}

	os.Exit(code)
}

func loadMigrationSQL(filename string) (string, error) {
	candidates := []string{
		filepath.Join("migrations", filename),
		filepath.Join("..", "migrations", filename),
		filepath.Join("..", "..", "migrations", filename),
		filepath.Join("..", "..", "..", "migrations", filename),
		filepath.Join("backend", "migrations", filename),
	}
	for _, path := range candidates {
		if content, err := os.ReadFile(path); err == nil {
			return string(content), nil
		}
	}
	return "", fmt.Errorf("migration file %q not found in any candidate path", filename)
}

func initTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	testExecutionMu.Lock()
	if testExecutionSchema == "" {
		testExecutionSchema = generateUniqueTestSchemaName()
	}
	schema := testExecutionSchema
	testExecutionMu.Unlock()

	require.NoError(t, validateSafeTestSchema(schema))

	baseDSN, err := getBaseTestDSN()
	require.NoError(t, err, "Must resolve base test database configuration from TEST_DATABASE_URL")

	rootDB, err := gorm.Open(postgres.Open(baseDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to root test database")

	// Pre-creation safety check: ensure the target database name is safe
	var currentDB string
	require.NoError(t, rootDB.Raw("SELECT current_database();").Scan(&currentDB).Error)
	require.NoError(t, validateDatabaseName(currentDB), "Target database must not be a production database")

	// Create the unique schema for this test run
	require.NoError(t, rootDB.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schema)).Error)

	testExecutionMu.Lock()
	testSchemaCreated = true
	testExecutionMu.Unlock()

	// Connect with search_path strictly set to this unique test schema
	dsn, err := buildDSNWithSearchPath(baseDSN, schema)
	require.NoError(t, err)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to test schema database")

	// Safety check 1: Verify current_schema() matches the unique test schema and is NEVER public
	var curSchema string
	require.NoError(t, db.Raw("SELECT current_schema()").Scan(&curSchema).Error)
	require.Equal(t, schema, curSchema, "Safety check: current_schema() must match test schema exactly")
	require.NotEqual(t, "public", curSchema, "Safety check: current_schema() must NEVER be public")

	// Safety check 2: Explicitly set search_path on the connection session
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path = %s;", schema)).Error)

	// Clean up any existing tables in this schema using the actual DOWN migration
	downSQL, err := loadMigrationSQL("000017_create_finance_ledger.down.sql")
	require.NoError(t, err, "Failed to load finance ledger down migration")
	_ = db.Exec(downSQL).Error

	// Apply the canonical UP migration file into the isolated test schema
	upSQL, err := loadMigrationSQL("000017_create_finance_ledger.up.sql")
	require.NoError(t, err, "Failed to load finance ledger up migration")
	require.NoError(t, db.Exec(upSQL).Error, "Failed to apply migration 000017_create_finance_ledger.up.sql")

	return db
}

func setupAccounts(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	accA := uuid.New()
	accB := uuid.New()

	err := db.Exec(`
		INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
		VALUES 
			(?, '1010', 'Cash Clearing', 'ASSET', 'DEBIT', 'INR', TRUE, NOW()),
			(?, '2010', 'Ticket Sales Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW())
	`, accA, accB).Error
	require.NoError(t, err)

	return accA, accB
}

func buildValidJournal(
	t *testing.T,
	id uuid.UUID,
	txType finance.TransactionType,
	eventType finance.FinancialEventType,
	sourceType finance.SourceType,
	sourceID string,
	idempotencyKey string,
	currency string,
	description string,
	reversalReferenceID *uuid.UUID,
	createdAt time.Time,
	entries ...finance.LedgerEntry,
) *finance.JournalTransaction {
	t.Helper()
	j, err := finance.NewJournalTransaction(
		id,
		txType,
		eventType,
		sourceType,
		sourceID,
		idempotencyKey,
		currency,
		description,
		reversalReferenceID,
		createdAt,
	)
	require.NoError(t, err)

	for _, e := range entries {
		err := j.AddEntry(e)
		require.NoError(t, err)
	}

	err = j.Post(createdAt)
	require.NoError(t, err)

	return j
}

// -------------------------------------------------------------
// OBJECTIVE 1: STRENGTHEN REVERSAL INTEGRITY TESTS
// -------------------------------------------------------------

func TestFinance_Reversal_ValidBidirectionalCommit(t *testing.T) {
	db := initTestDatabase(t)
	repo := NewTestFinanceRepository(db, slog.Default())
	accA, accB := setupAccounts(t, db)

	amt, _ := finance.NewMoney(decimal.NewFromInt(100), finance.CurrencyINR)
	now := time.Now().UTC().Truncate(time.Second)

	d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt, nil, nil, now)
	c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt, nil, nil, now)
	origJ := buildValidJournal(t, uuid.New(), finance.TransactionTypePayment, finance.FinancialEventTicketSaleCaptured, finance.SourceTypePayment, "s1", "k_orig", finance.CurrencyINR, "Orig", nil, now, d1, c1)

	_, err := repo.RecordJournal(context.Background(), origJ)
	require.NoError(t, err)

	revJ, err := origJ.CreateReversal("k_rev_valid", "Valid Reversal", now)
	require.NoError(t, err)

	resRev, err := repo.RecordJournal(context.Background(), revJ)
	require.NoError(t, err)
	require.NotNil(t, resRev)

	var origDB, revDB FinanceJournalModel
	require.NoError(t, db.Where("id = ?", origJ.ID()).First(&origDB).Error)
	require.NoError(t, db.Where("id = ?", revJ.ID()).First(&revDB).Error)

	assert.True(t, origDB.IsReversed)
	assert.Equal(t, revJ.ID(), *origDB.ReversedByJournalID)
	assert.Equal(t, origJ.ID(), *revDB.ReversalReferenceID)
	assert.Equal(t, revDB.PostedAt.Unix(), origDB.ReversedAt.Unix())
	assert.Equal(t, "POSTED", revDB.Status)
	assert.Equal(t, "POSTED", origDB.Status)
}

func TestFinance_Reversal_ForeignKeyEnforcement(t *testing.T) {
	db := initTestDatabase(t)

	nonExistentUUID := uuid.New()
	jID := uuid.New()

	t.Run("Reject reversal_reference_id pointing to non-existent journal", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
			VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's1', 'k_fk_fail', 'fp1', 'INR', 'Rev', ?, 'DRAFT', NOW(), FALSE)
		`, jID, nonExistentUUID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fk_finance_journals_reversal_ref")
	})

	t.Run("Reject reversed_by_journal_id pointing to non-existent journal", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversed_by_journal_id, status, created_at, is_reversed)
			VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's2', 'k_fk_fail2', 'fp2', 'INR', 'Orig', ?, 'DRAFT', NOW(), FALSE)
		`, uuid.New(), nonExistentUUID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fk_finance_journals_reversed_by")
	})
}

func TestFinance_Reversal_RejectWhenOriginalIsDraft(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	tx := db.Begin()
	defer tx.Rollback()

	origID := uuid.New()
	revID := uuid.New()

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_orig_draft', 'fp1', 'INR', 'Draft Orig', 'DRAFT', NOW(), FALSE)
	`, origID).Error)

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
		VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's1', 'k_rev_draft', 'fp2', 'INR', 'Reversal', ?, 'DRAFT', NOW(), FALSE)
	`, revID, origID).Error)

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()),
			(?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), revID, accA, uuid.New(), revID, accB).Error)

	require.NoError(t, tx.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", revID).Error)

	commitErr := tx.Commit().Error
	require.Error(t, commitErr)
	assert.Contains(t, commitErr.Error(), "original journal")
	assert.Contains(t, commitErr.Error(), "must be POSTED to be reversed")
}

func TestFinance_Reversal_RejectWhenOriginalNotMarkedReversed(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	origID := uuid.New()
	revID := uuid.New()

	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_orig_unrev', 'fp1', 'INR', 'Orig', 'DRAFT', NOW(), FALSE)
	`, origID).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()), (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), origID, accA, uuid.New(), origID, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", origID).Error)

	tx := db.Begin()
	defer tx.Rollback()

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
		VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's1', 'k_rev_unrev', 'fp2', 'INR', 'Rev', ?, 'DRAFT', NOW(), FALSE)
	`, revID, origID).Error)
	require.NoError(t, tx.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW()), (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW())
	`, uuid.New(), revID, accA, uuid.New(), revID, accB).Error)
	require.NoError(t, tx.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", revID).Error)

	commitErr := tx.Commit().Error
	require.Error(t, commitErr)
	assert.Contains(t, commitErr.Error(), "must be marked is_reversed = TRUE")
}

func TestFinance_Reversal_RejectMutualReferenceMismatch(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	orig1 := uuid.New()
	orig2 := uuid.New()
	rev1 := uuid.New()
	rev2 := uuid.New()

	// 1. Set up orig1 as a valid POSTED journal with entries
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_orig1', 'fp1', 'INR', 'Orig1', 'DRAFT', NOW(), FALSE)
	`, orig1).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()), (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), orig1, accA, uuid.New(), orig1, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", orig1).Error)

	// 2. Set up orig2 as a valid POSTED journal with entries
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's2', 'k_orig2', 'fp2', 'INR', 'Orig2', 'DRAFT', NOW(), FALSE)
	`, orig2).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 200.00, 'INR', NOW()), (?, ?, ?, 'CREDIT', 200.00, 'INR', NOW())
	`, uuid.New(), orig2, accA, uuid.New(), orig2, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", orig2).Error)

	// 3. Cross-reference mismatch:
	// rev1 points reversal_reference_id to orig2, but orig1 points reversed_by_journal_id to rev1.
	// rev2 points reversal_reference_id to orig1, but orig2 points reversed_by_journal_id to rev2.
	// All foreign keys and unique constraints are satisfied, but the mutual references cross-match!
	tx := db.Begin()
	defer tx.Rollback()

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
		VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's1_rev', 'k_rev1', 'fp_r1', 'INR', 'Rev1', ?, 'DRAFT', NOW(), FALSE)
	`, rev1, orig2).Error)
	require.NoError(t, tx.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW()), (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW())
	`, uuid.New(), rev1, accA, uuid.New(), rev1, accB).Error)

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
		VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's2_rev', 'k_rev2', 'fp_r2', 'INR', 'Rev2', ?, 'DRAFT', NOW(), FALSE)
	`, rev2, orig1).Error)
	require.NoError(t, tx.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'CREDIT', 200.00, 'INR', NOW()), (?, ?, ?, 'DEBIT', 200.00, 'INR', NOW())
	`, uuid.New(), rev2, accA, uuid.New(), rev2, accB).Error)

	now := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, tx.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?", now, rev1).Error)
	require.NoError(t, tx.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?", now, rev2).Error)
	require.NoError(t, tx.Exec("UPDATE finance_journals SET is_reversed = TRUE, reversed_by_journal_id = ?, reversed_at = ? WHERE id = ?", rev1, now, orig1).Error)
	require.NoError(t, tx.Exec("UPDATE finance_journals SET is_reversed = TRUE, reversed_by_journal_id = ?, reversed_at = ? WHERE id = ?", rev2, now, orig2).Error)

	commitErr := tx.Commit().Error
	require.Error(t, commitErr)
	assert.Contains(t, commitErr.Error(), "mutual reference mismatch")
}

func TestFinance_Reversal_RejectWhenReversalIsDraft(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	origID := uuid.New()
	revID := uuid.New()

	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_orig_draft_rev', 'fp1', 'INR', 'Orig', 'DRAFT', NOW(), FALSE)
	`, origID).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()), (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), origID, accA, uuid.New(), origID, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", origID).Error)

	tx := db.Begin()
	defer tx.Rollback()

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
		VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's1', 'k_rev_still_draft', 'fp2', 'INR', 'Rev', ?, 'DRAFT', NOW(), FALSE)
	`, revID, origID).Error)

	require.NoError(t, tx.Exec("UPDATE finance_journals SET is_reversed = TRUE, reversed_by_journal_id = ?, reversed_at = NOW() WHERE id = ?", revID, origID).Error)

	commitErr := tx.Commit().Error
	require.Error(t, commitErr)
	assert.Contains(t, commitErr.Error(), "must be in POSTED status")
}

func TestFinance_Reversal_SelfReferencingRejection(t *testing.T) {
	db := initTestDatabase(t)
	selfID := uuid.New()

	err := db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
		VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's1', 'k_self_rev', 'fp1', 'INR', 'Self Rev', ?, 'DRAFT', NOW(), FALSE)
	`, selfID, selfID).Error

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chk_finance_journals_no_self_reversal")
}

// -------------------------------------------------------------
// OBJECTIVE 2: JOURNAL LIFECYCLE INVARIANTS TESTS
// -------------------------------------------------------------

func TestFinance_Lifecycle_DraftMustHaveNullPostedAt(t *testing.T) {
	db := initTestDatabase(t)
	jID := uuid.New()

	err := db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, posted_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_draft_posted_at', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), NOW(), FALSE)
	`, jID).Error

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chk_finance_journals_lifecycle_timestamps")
}

func TestFinance_Lifecycle_PostedMustHaveNonNullPostedAt(t *testing.T) {
	db := initTestDatabase(t)
	jID := uuid.New()

	err := db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, posted_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_posted_null_at', 'fp1', 'INR', 'desc', 'POSTED', NOW(), NULL, FALSE)
	`, jID).Error

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chk_finance_journals_lifecycle_timestamps")
}

func TestFinance_Lifecycle_PostedAtCannotBeBeforeCreatedAt(t *testing.T) {
	db := initTestDatabase(t)
	jID := uuid.New()

	err := db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, posted_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_posted_past', 'fp1', 'INR', 'desc', 'POSTED', NOW(), NOW() - INTERVAL '1 hour', FALSE)
	`, jID).Error

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chk_finance_journals_lifecycle_timestamps")
}

func TestFinance_Lifecycle_RejectPostedToDraftTransition(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	jID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_posted_to_draft', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()), (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), jID, accA, uuid.New(), jID, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", jID).Error)

	err := db.Exec("UPDATE finance_journals SET status = 'DRAFT', posted_at = NULL WHERE id = ?", jID).Error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot change status of POSTED journal transaction")
}

func TestFinance_Lifecycle_RejectPostedAtMutationOnPostedJournal(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	jID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_mutate_posted_at', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()), (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), jID, accA, uuid.New(), jID, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", jID).Error)

	err := db.Exec("UPDATE finance_journals SET posted_at = NOW() + INTERVAL '10 minutes' WHERE id = ?", jID).Error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update immutable fields of POSTED journal transaction")
}

func TestFinance_Lifecycle_RejectInvalidStatus(t *testing.T) {
	db := initTestDatabase(t)
	jID := uuid.New()

	err := db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_invalid_status', 'fp1', 'INR', 'desc', 'PENDING', NOW(), FALSE)
	`, jID).Error

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chk_finance_journals_")
}

func TestFinance_Lifecycle_RejectDescriptionMutationOnPostedJournal(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	jID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_mutate_desc', 'fp1', 'INR', 'original description', 'DRAFT', NOW(), FALSE)
	`, jID).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()), (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), jID, accA, uuid.New(), jID, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", jID).Error)

	err := db.Exec("UPDATE finance_journals SET description = 'modified description' WHERE id = ?", jID).Error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update immutable fields of POSTED journal transaction")
}

func TestFinance_Lifecycle_EnforceExactOneTimeReversalTransition(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	jID := uuid.New()
	dummyRevID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_exact_rev_trans', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()), (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), jID, accA, uuid.New(), jID, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", jID).Error)

	t.Run("Reject partial transition with is_reversed=TRUE but null reversed_by_journal_id", func(t *testing.T) {
		err := db.Exec("UPDATE finance_journals SET is_reversed = TRUE WHERE id = ?", jID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reversal metadata transition on journal")
	})

	t.Run("Reject partial transition with non-null reversed_by_journal_id but is_reversed=FALSE", func(t *testing.T) {
		err := db.Exec("UPDATE finance_journals SET reversed_by_journal_id = ? WHERE id = ?", dummyRevID, jID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reversal metadata transition on journal")
	})

	t.Run("Reject partial transition with non-null reversed_at but is_reversed=FALSE", func(t *testing.T) {
		err := db.Exec("UPDATE finance_journals SET reversed_at = NOW() WHERE id = ?", jID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reversal metadata transition on journal")
	})
}

func TestFinance_Lifecycle_PreventSubsequentReversalMetadataMutation(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	origID := uuid.New()
	revID := uuid.New()

	// 1. Post original journal
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_orig_subseq', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, origID).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()), (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), origID, accA, uuid.New(), origID, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", origID).Error)

	// 2. Perform exact valid one-time reversal
	tx := db.Begin()
	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
		VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's1', 'k_rev_subseq', 'fp2', 'INR', 'desc', ?, 'DRAFT', NOW(), FALSE)
	`, revID, origID).Error)
	require.NoError(t, tx.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'CREDIT', 100.00, 'INR', NOW()), (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW())
	`, uuid.New(), revID, accA, uuid.New(), revID, accB).Error)
	require.NoError(t, tx.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", revID).Error)
	require.NoError(t, tx.Exec("UPDATE finance_journals SET is_reversed = TRUE, reversed_by_journal_id = ?, reversed_at = NOW() WHERE id = ?", revID, origID).Error)
	require.NoError(t, tx.Commit().Error)

	// 3. Test subsequent mutation attempts on already reversed journal
	t.Run("Reject un-reversing (is_reversed = FALSE)", func(t *testing.T) {
		err := db.Exec("UPDATE finance_journals SET is_reversed = FALSE WHERE id = ?", origID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot modify reversal metadata on already reversed journal")
	})

	t.Run("Reject modifying reversed_by_journal_id", func(t *testing.T) {
		err := db.Exec("UPDATE finance_journals SET reversed_by_journal_id = ? WHERE id = ?", uuid.New(), origID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot modify reversal metadata on already reversed journal")
	})

	t.Run("Reject modifying reversed_at", func(t *testing.T) {
		err := db.Exec("UPDATE finance_journals SET reversed_at = NOW() + INTERVAL '1 hour' WHERE id = ?", origID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot modify reversal metadata on already reversed journal")
	})
}

// -------------------------------------------------------------
// OBJECTIVE 3: STRENGTHEN IDEMPOTENCY INTEGRITY TESTS
// -------------------------------------------------------------

func TestFinance_Idempotency_RejectFingerprintMismatchWithJournal(t *testing.T) {
	db := initTestDatabase(t)

	tx := db.Begin()
	defer tx.Rollback()

	jID := uuid.New()
	key := "k_fp_mismatch"
	fp1 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fp2 := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_idempotency_keys (idempotency_key, fingerprint, journal_id, created_at)
		VALUES (?, ?, ?, NOW())
	`, key, fp1, jID).Error)

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', ?, ?, 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID, key, fp2).Error)

	commitErr := tx.Commit().Error
	require.Error(t, commitErr, "Commit must fail because composite FK is violated")
	assert.Contains(t, commitErr.Error(), "fk_finance_idempotency_keys_journal_composite")
}

func TestFinance_Idempotency_RejectKeyMismatchWithJournal(t *testing.T) {
	db := initTestDatabase(t)

	tx := db.Begin()
	defer tx.Rollback()

	jID := uuid.New()
	fp := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_idempotency_keys (idempotency_key, fingerprint, journal_id, created_at)
		VALUES ('k_first', ?, ?, NOW())
	`, fp, jID).Error)

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_second', ?, 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID, fp).Error)

	commitErr := tx.Commit().Error
	require.Error(t, commitErr)
	assert.Contains(t, commitErr.Error(), "fk_finance_idempotency_keys_journal_composite")
}

func TestFinance_Idempotency_RejectDuplicateJournalAssociation(t *testing.T) {
	db := initTestDatabase(t)

	jID := uuid.New()
	fp := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_one', ?, 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID, fp).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_idempotency_keys (idempotency_key, fingerprint, journal_id, created_at)
		VALUES ('k_one', ?, ?, NOW())
	`, fp, jID).Error)

	err := db.Exec(`
		INSERT INTO finance_idempotency_keys (idempotency_key, fingerprint, journal_id, created_at)
		VALUES ('k_two', ?, ?, NOW())
	`, fp, jID).Error

	require.Error(t, err)
	assert.Contains(t, err.Error(), "uq_finance_idempotency_keys_journal_id")
}

func TestFinance_Idempotency_RejectDuplicateKeyAcrossJournals(t *testing.T) {
	db := initTestDatabase(t)

	j1 := uuid.New()
	j2 := uuid.New()
	fp := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_shared', ?, 'INR', 'desc1', 'DRAFT', NOW(), FALSE)
	`, j1, fp).Error)

	err := db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's2', 'k_shared', ?, 'INR', 'desc2', 'DRAFT', NOW(), FALSE)
	`, j2, fp).Error

	require.Error(t, err)
	assert.Contains(t, err.Error(), "uq_finance_journals_idempotency_key")
}

func TestFinance_Idempotency_ValidCompositeAssociation(t *testing.T) {
	db := initTestDatabase(t)

	tx := db.Begin()
	defer tx.Rollback()

	jID := uuid.New()
	key := "k_valid_composite"
	fp := "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_idempotency_keys (idempotency_key, fingerprint, journal_id, created_at)
		VALUES (?, ?, ?, NOW())
	`, key, fp, jID).Error)

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', ?, ?, 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID, key, fp).Error)

	require.NoError(t, tx.Commit().Error)
}

func TestFinance_Idempotency_RejectDuplicatePrimaryKey(t *testing.T) {
	db := initTestDatabase(t)

	j1 := uuid.New()
	j2 := uuid.New()
	key := "k_duplicate_pk"
	fp := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', ?, ?, 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, j1, key, fp).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_idempotency_keys (idempotency_key, fingerprint, journal_id, created_at)
		VALUES (?, ?, ?, NOW())
	`, key, fp, j1).Error)

	err := db.Exec(`
		INSERT INTO finance_idempotency_keys (idempotency_key, fingerprint, journal_id, created_at)
		VALUES (?, ?, ?, NOW())
	`, key, fp, j2).Error

	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr))
	assert.Equal(t, "23505", pgErr.Code)
	assert.Equal(t, "pk_finance_idempotency_keys", pgErr.ConstraintName)
}

// -------------------------------------------------------------
// CORE CONCURRENCY & PERSISTENCE VERIFICATION TESTS
// -------------------------------------------------------------

func TestFinance_PostedEntryMutationAndOwnershipImmutability(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	jID := uuid.New()
	e1 := uuid.New()
	e2 := uuid.New()

	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k1', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()),
			(?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, e1, jID, accA, e2, jID, accB).Error)

	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", jID).Error)

	t.Run("Reject INSERT into POSTED journal", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 50.00, 'INR', NOW())
		`, uuid.New(), jID, accA).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot insert or update ledger entry for already POSTED journal")
	})

	t.Run("Reject UPDATE of entry amount on POSTED journal", func(t *testing.T) {
		err := db.Exec("UPDATE finance_ledger_entries SET amount = 999.00 WHERE id = ?", e1).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot insert or update ledger entry for already POSTED journal")
	})

	t.Run("Reject DELETE of entry from POSTED journal", func(t *testing.T) {
		err := db.Exec("DELETE FROM finance_ledger_entries WHERE id = ?", e2).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete ledger entry from already POSTED journal")
	})

	t.Run("Reject journal_id mutation on ledger entry", func(t *testing.T) {
		otherJID := uuid.New()
		err := db.Exec("UPDATE finance_ledger_entries SET journal_id = ? WHERE id = ?", otherJID, e1).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "journal_id on ledger entries is strictly immutable")
	})
}

func TestFinance_ConcurrentEntryInsertionVsPosting(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	draftID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_draft_conc', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, draftID).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()),
			(?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), draftID, accA, uuid.New(), draftID, accB).Error)

	var wg sync.WaitGroup
	wg.Add(2)
	var postErr, insertErr error

	go func() {
		defer wg.Done()
		tx := db.Begin()
		defer tx.Rollback()

		res := tx.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ? AND status = 'DRAFT'", draftID)
		if res.Error != nil {
			postErr = res.Error
			return
		}
		time.Sleep(50 * time.Millisecond)
		postErr = tx.Commit().Error
	}()

	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		tx := db.Begin()
		defer tx.Rollback()

		res := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 50.00, 'INR', NOW())
		`, uuid.New(), draftID, accA)
		if res.Error != nil {
			insertErr = res.Error
			return
		}
		insertErr = tx.Commit().Error
	}()

	wg.Wait()

	require.NoError(t, postErr, "Posting should succeed")
	require.Error(t, insertErr, "Entry insertion must be rejected once journal is posted")
	assert.Contains(t, insertErr.Error(), "cannot insert or update ledger entry for already POSTED journal")

	var entryCount int64
	db.Table("finance_ledger_entries").Where("journal_id = ?", draftID).Count(&entryCount)
	assert.Equal(t, int64(2), entryCount, "Exactly 2 original entries must remain")
}

func TestFinance_ConcurrentAccountDeactivationVsPosting(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	draftID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_deact', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, draftID).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()),
			(?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), draftID, accA, uuid.New(), draftID, accB).Error)

	var wg sync.WaitGroup
	wg.Add(2)
	var deactErr, postErr error

	go func() {
		defer wg.Done()
		tx := db.Begin()
		defer tx.Rollback()

		res := tx.Exec("UPDATE finance_accounts SET is_active = FALSE WHERE id = ?", accA)
		if res.Error != nil {
			deactErr = res.Error
			return
		}
		time.Sleep(50 * time.Millisecond)
		deactErr = tx.Commit().Error
	}()

	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		tx := db.Begin()
		defer tx.Rollback()

		res := tx.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", draftID)
		if res.Error != nil {
			postErr = res.Error
			return
		}
		postErr = tx.Commit().Error
	}()

	wg.Wait()

	require.NoError(t, deactErr)
	require.Error(t, postErr, "Posting must be rejected because Account A is deactivated")
	assert.Contains(t, postErr.Error(), "is inactive; cannot post journal")
}

func TestFinance_DeferredUnbalancedJournalRejection(t *testing.T) {
	db := initTestDatabase(t)
	accA, _ := setupAccounts(t, db)

	tx := db.Begin()
	defer tx.Rollback()

	jID := uuid.New()
	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_unbal', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID).Error)

	require.NoError(t, tx.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES (?, ?, ?, 'DEBIT', 100.00, 'INR', NOW())
	`, uuid.New(), jID, accA).Error)

	require.NoError(t, tx.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", jID).Error)

	commitErr := tx.Commit().Error
	require.Error(t, commitErr)
	assert.Contains(t, commitErr.Error(), "must have at least 2 ledger entries")
}

func TestFinance_DuplicateAndConcurrentReversalBehavior(t *testing.T) {
	db := initTestDatabase(t)
	repo := NewTestFinanceRepository(db, slog.Default())
	accA, accB := setupAccounts(t, db)

	amt, _ := finance.NewMoney(decimal.NewFromInt(100), finance.CurrencyINR)
	now := time.Now().UTC().Truncate(time.Second)

	d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt, nil, nil, now)
	c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt, nil, nil, now)
	origJ := buildValidJournal(t, uuid.New(), finance.TransactionTypePayment, finance.FinancialEventTicketSaleCaptured, finance.SourceTypePayment, "s1", "k_rev_base", finance.CurrencyINR, "Orig", nil, now, d1, c1)

	_, err := repo.RecordJournal(context.Background(), origJ)
	require.NoError(t, err)

	rev1, err := origJ.CreateReversal("rev_k1", "Reversal 1", now)
	require.NoError(t, err)
	rev2, err := origJ.CreateReversal("rev_k2", "Reversal 2", now)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)
	var err1, err2 error

	go func() {
		defer wg.Done()
		_, err1 = repo.RecordJournal(context.Background(), rev1)
	}()

	go func() {
		defer wg.Done()
		_, err2 = repo.RecordJournal(context.Background(), rev2)
	}()

	wg.Wait()

	successCount := 0
	failCount := 0
	for _, e := range []error{err1, err2} {
		if e == nil {
			successCount++
		} else if errors.Is(e, finance.ErrJournalAlreadyReversed) {
			failCount++
		}
	}
	assert.Equal(t, 1, successCount, "Exactly one concurrent reversal must succeed")
	assert.Equal(t, 1, failCount, "Conflicting reversal must fail with ErrJournalAlreadyReversed")

	var revCount int64
	db.Table("finance_journals").Where("reversal_reference_id = ?", origJ.ID()).Count(&revCount)
	assert.Equal(t, int64(1), revCount, "Exactly 1 reversal journal must persist")
}

func TestFinance_Reversal_DuplicateReversalReferenceIndexRejection(t *testing.T) {
	db := initTestDatabase(t)
	repo := NewTestFinanceRepository(db, slog.Default())
	accA, accB := setupAccounts(t, db)

	amt, _ := finance.NewMoney(decimal.NewFromInt(100), finance.CurrencyINR)
	now := time.Now().UTC().Truncate(time.Second)

	d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt, nil, nil, now)
	c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt, nil, nil, now)
	origJ := buildValidJournal(t, uuid.New(), finance.TransactionTypePayment, finance.FinancialEventTicketSaleCaptured, finance.SourceTypePayment, "s1", "k_rev_idx_test", finance.CurrencyINR, "Orig", nil, now, d1, c1)

	_, err := repo.RecordJournal(context.Background(), origJ)
	require.NoError(t, err)

	rev1, err := origJ.CreateReversal("rev_k1", "Reversal 1", now)
	require.NoError(t, err)
	rev2, err := origJ.CreateReversal("rev_k2", "Reversal 2", now)
	require.NoError(t, err)

	// 1. First reversal succeeds
	_, err = repo.RecordJournal(context.Background(), rev1)
	require.NoError(t, err)

	// 2. Second sequential reversal with same reversal_reference_id must fail and map to finance.ErrJournalAlreadyReversed
	_, err = repo.RecordJournal(context.Background(), rev2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, finance.ErrJournalAlreadyReversed), "Must map to finance.ErrJournalAlreadyReversed")

	// 3. Raw SQL direct verification: index idx_finance_journals_unique_reversal_ref explicitly fails
	revRawID := uuid.New()
	errRaw := db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
		VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's_raw', 'k_raw_rev', 'fp_raw', 'INR', 'Raw Rev', ?, 'DRAFT', NOW(), FALSE)
	`, revRawID, origJ.ID()).Error
	require.Error(t, errRaw)
	var pgErr *pgconn.PgError
	require.True(t, errors.As(errRaw, &pgErr))
	assert.Equal(t, "23505", pgErr.Code)
	assert.Equal(t, "idx_finance_journals_unique_reversal_ref", pgErr.ConstraintName)
}

func TestFinance_FullRollbackAfterReversalFailure(t *testing.T) {
	db := initTestDatabase(t)
	accA, accB := setupAccounts(t, db)

	origID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_orig_rb', 'fp1', 'INR', 'Orig', 'DRAFT', NOW(), FALSE)
	`, origID).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 100.00, 'INR', NOW()),
			(?, ?, ?, 'CREDIT', 100.00, 'INR', NOW())
	`, uuid.New(), origID, accA, uuid.New(), origID, accB).Error)
	require.NoError(t, db.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", origID).Error)

	tx := db.Begin()
	defer tx.Rollback()

	revID := uuid.New()
	require.NoError(t, tx.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, reversal_reference_id, status, created_at, is_reversed)
		VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', 's1', 'k_rev_rb', 'fp1', 'INR', 'Rev', ?, 'DRAFT', NOW(), FALSE)
	`, revID, origID).Error)
	require.NoError(t, tx.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES 
			(?, ?, ?, 'CREDIT', 100.00, 'INR', NOW()),
			(?, ?, ?, 'DEBIT', 100.00, 'INR', NOW())
	`, uuid.New(), revID, accA, uuid.New(), revID, accB).Error)
	require.NoError(t, tx.Exec("UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?", revID).Error)

	res := tx.Exec("UPDATE finance_journals SET is_reversed = TRUE, reversed_by_journal_id = ?, reversed_at = NOW() WHERE id = ? AND is_reversed = TRUE", revID, origID)
	require.NoError(t, res.Error)
	assert.Equal(t, int64(0), res.RowsAffected)

	tx.Rollback()

	var count int64
	db.Table("finance_journals").Where("id = ?", revID).Count(&count)
	assert.Equal(t, int64(0), count, "Reversal journal must not persist")

	var origReversed bool
	db.Table("finance_journals").Select("is_reversed").Where("id = ?", origID).Scan(&origReversed)
	assert.False(t, origReversed, "Original journal must remain unreversed")
}

func TestFinance_ConcurrentIdenticalIdempotencyRequests(t *testing.T) {
	db := initTestDatabase(t)
	repo := NewTestFinanceRepository(db, slog.Default())
	accA, accB := setupAccounts(t, db)

	amt, _ := finance.NewMoney(decimal.NewFromInt(100), finance.CurrencyINR)
	now := time.Now().UTC().Truncate(time.Second)
	key := "idem_conc_" + uuid.New().String()

	d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt, nil, nil, now)
	c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt, nil, nil, now)
	j := buildValidJournal(t, uuid.New(), finance.TransactionTypePayment, finance.FinancialEventTicketSaleCaptured, finance.SourceTypePayment, "s1", key, finance.CurrencyINR, "desc", nil, now, d1, c1)

	concurrency := 10
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	results := make([]*finance.JournalTransaction, concurrency)

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = repo.RecordJournal(context.Background(), j)
		}(i)
	}
	wg.Wait()

	for i := 0; i < concurrency; i++ {
		require.NoError(t, errs[i], "Goroutine %d must succeed", i)
		require.NotNil(t, results[i])
		assert.Equal(t, j.ID(), results[i].ID())
		assert.Equal(t, finance.PostingStatusPosted, results[i].Status())
	}

	var jCount int64
	db.Table("finance_journals").Where("idempotency_key = ?", key).Count(&jCount)
	assert.Equal(t, int64(1), jCount, "Exactly 1 journal in DB")

	var eCount int64
	db.Table("finance_ledger_entries").Where("journal_id = ?", j.ID()).Count(&eCount)
	assert.Equal(t, int64(2), eCount, "Exactly 2 ledger entries in DB")

	var kCount int64
	db.Table("finance_idempotency_keys").Where("idempotency_key = ?", key).Count(&kCount)
	assert.Equal(t, int64(1), kCount, "Exactly 1 idempotency record in DB")
}

func TestFinance_ConflictingPayloadWithSameIdempotencyKey(t *testing.T) {
	db := initTestDatabase(t)
	repo := NewTestFinanceRepository(db, slog.Default())
	accA, accB := setupAccounts(t, db)

	key := "idem_conflict_" + uuid.New().String()
	now := time.Now().UTC().Truncate(time.Second)

	amt100, _ := finance.NewMoney(decimal.NewFromInt(100), finance.CurrencyINR)
	amt200, _ := finance.NewMoney(decimal.NewFromInt(200), finance.CurrencyINR)

	d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
	c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt100, nil, nil, now)
	j1 := buildValidJournal(t, uuid.New(), finance.TransactionTypePayment, finance.FinancialEventTicketSaleCaptured, finance.SourceTypePayment, "s1", key, finance.CurrencyINR, "desc1", nil, now, d1, c1)

	d2, _ := finance.NewDebitEntry(uuid.New(), accA, amt200, nil, nil, now)
	c2, _ := finance.NewCreditEntry(uuid.New(), accB, amt200, nil, nil, now)
	j2 := buildValidJournal(t, uuid.New(), finance.TransactionTypePayment, finance.FinancialEventTicketSaleCaptured, finance.SourceTypePayment, "s1", key, finance.CurrencyINR, "desc2", nil, now, d2, c2)

	res1, err1 := repo.RecordJournal(context.Background(), j1)
	require.NoError(t, err1)
	require.NotNil(t, res1)

	res2, err2 := repo.RecordJournal(context.Background(), j2)
	require.Error(t, err2)
	assert.Nil(t, res2)
	assert.True(t, errors.Is(err2, ErrIdempotencyConflict), "Must return ErrIdempotencyConflict")
}

func TestFinance_TransactionIntegrationWithCallerContext(t *testing.T) {
	db := initTestDatabase(t)
	repo := NewTestFinanceRepository(db, slog.Default())
	accA, accB := setupAccounts(t, db)

	amt, _ := finance.NewMoney(decimal.NewFromInt(100), finance.CurrencyINR)
	now := time.Now().UTC().Truncate(time.Second)
	key := "idem_tx_test_" + uuid.New().String()

	d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt, nil, nil, now)
	c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt, nil, nil, now)
	j := buildValidJournal(t, uuid.New(), finance.TransactionTypePayment, finance.FinancialEventTicketSaleCaptured, finance.SourceTypePayment, "s1", key, finance.CurrencyINR, "tx desc", nil, now, d1, c1)

	t.Run("Atomic rollback when subsequent step in transaction fails", func(t *testing.T) {
		txErr := db.Transaction(func(txDB *gorm.DB) error {
			ctx := WithTx(context.Background(), txDB)
			_, err := repo.RecordJournal(ctx, j)
			if err != nil {
				return err
			}
			return errors.New("subsequent booking failure")
		})

		require.Error(t, txErr)

		var jCount int64
		db.Table("finance_journals").Where("id = ?", j.ID()).Count(&jCount)
		assert.Equal(t, int64(0), jCount, "Journal must not persist when outer transaction rolls back")

		var kCount int64
		db.Table("finance_idempotency_keys").Where("idempotency_key = ?", key).Count(&kCount)
		assert.Equal(t, int64(0), kCount, "Idempotency key must not persist when outer transaction rolls back")
	})

	t.Run("Atomic commit when all steps succeed", func(t *testing.T) {
		txErr := db.Transaction(func(txDB *gorm.DB) error {
			ctx := WithTx(context.Background(), txDB)
			_, err := repo.RecordJournal(ctx, j)
			return err
		})

		require.NoError(t, txErr)

		var jCount int64
		db.Table("finance_journals").Where("id = ?", j.ID()).Count(&jCount)
		assert.Equal(t, int64(1), jCount, "Journal must persist when outer transaction commits")
	})
}

// -------------------------------------------------------------
// DATABASE SAFETY VALIDATION TESTS
// -------------------------------------------------------------

func TestFinance_DatabaseSafety_RequireTestDatabaseURL(t *testing.T) {
	oldVal, hasOld := os.LookupEnv("TEST_DATABASE_URL")
	defer func() {
		if hasOld {
			_ = os.Setenv("TEST_DATABASE_URL", oldVal)
		} else {
			_ = os.Unsetenv("TEST_DATABASE_URL")
		}
	}()

	_ = os.Unsetenv("TEST_DATABASE_URL")

	_, err := getBaseTestDSN()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TEST_DATABASE_URL environment variable is explicitly required")
	assert.Contains(t, err.Error(), "fallback to DATABASE_URL is disabled")
}

func TestFinance_DatabaseSafety_RejectProductionDatabaseTargets(t *testing.T) {
	testCases := []struct {
		name        string
		env         string
		dsn         string
		expectError string
	}{
		{
			name:        "Reject when APP_ENV is production",
			env:         "production",
			dsn:         "host=localhost port=5432 user=postgres password=root dbname=eventhub_test sslmode=disable",
			expectError: "refusing to run integration tests when APP_ENV=\"production\"",
		},
		{
			name:        "Reject when APP_ENV is prod",
			env:         "prod",
			dsn:         "host=localhost port=5432 user=postgres password=root dbname=eventhub_test sslmode=disable",
			expectError: "refusing to run integration tests when APP_ENV=\"prod\"",
		},
		{
			name:        "Reject AWS RDS cloud host",
			env:         "test",
			dsn:         "host=my-cluster.12345.rds.amazonaws.com port=5432 user=postgres password=root dbname=eventhub sslmode=disable",
			expectError: "contains production indicator \"rds.amazonaws.com\"",
		},
		{
			name:        "Reject production hostname keyword",
			env:         "test",
			dsn:         "host=prod-database.internal port=5432 user=postgres password=root dbname=eventhub sslmode=disable",
			expectError: "contains production indicator \"prod\"",
		},
		{
			name:        "Reject production database name",
			env:         "test",
			dsn:         "host=localhost port=5432 user=postgres password=root dbname=eventhub_production sslmode=disable",
			expectError: "database name \"eventhub_production\" indicates a production database",
		},
		{
			name:        "Reject unrecognized remote IP host",
			env:         "test",
			dsn:         "host=198.51.100.25 port=5432 user=postgres password=root dbname=eventhub sslmode=disable",
			expectError: "is not recognized as a safe local or test database host",
		},
		{
			name:        "Accept safe localhost key-value DSN",
			env:         "test",
			dsn:         "host=localhost port=5432 user=postgres password=root dbname=eventhub sslmode=disable",
			expectError: "",
		},
		{
			name:        "Accept safe 127.0.0.1 postgres URL",
			env:         "test",
			dsn:         "postgres://postgres:root@127.0.0.1:5432/eventhub_test?sslmode=disable",
			expectError: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldEnv, hasEnv := os.LookupEnv("APP_ENV")
			defer func() {
				if hasEnv {
					_ = os.Setenv("APP_ENV", oldEnv)
				} else {
					_ = os.Unsetenv("APP_ENV")
				}
			}()

			_ = os.Setenv("APP_ENV", tc.env)
			err := validateTargetDatabase(tc.dsn)
			if tc.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// -------------------------------------------------------------
// OBJECTIVE: MIGRATION & ACCOUNT CONSTRAINT VERIFICATION TESTS
// -------------------------------------------------------------

func TestFinance_Accounts_SchemaAndConstraints(t *testing.T) {
	db := initTestDatabase(t)

	t.Run("Insert all valid account types with corresponding normal balances", func(t *testing.T) {
		validAccounts := []struct {
			code          string
			name          string
			accType       string
			normalBalance string
		}{
			{"1010", "Razorpay Clearing", "ASSET", "DEBIT"},
			{"1020", "Operating Bank", "ASSET", "DEBIT"},
			{"2010", "Organizer Payable", "LIABILITY", "CREDIT"},
			{"2020", "Customer Refund Payable", "LIABILITY", "CREDIT"},
			{"3010", "Retained Earnings", "EQUITY", "CREDIT"},
			{"4010", "Platform Fee Revenue", "REVENUE", "CREDIT"},
			{"5010", "Payment Gateway Expense", "EXPENSE", "DEBIT"},
		}

		for _, acc := range validAccounts {
			err := db.Exec(`
				INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
				VALUES (?, ?, ?, ?, ?, 'INR', TRUE, NOW())
			`, uuid.New(), acc.code, acc.name, acc.accType, acc.normalBalance).Error
			require.NoError(t, err, "Failed to insert valid account %s (%s, %s)", acc.name, acc.accType, acc.normalBalance)
		}
	})

	t.Run("Reject invalid account type", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, '9991', 'Invalid Type Account', 'UNKNOWN', 'DEBIT', 'INR', TRUE, NOW())
		`, uuid.New()).Error
		require.Error(t, err)
		var pgErr *pgconn.PgError
		require.True(t, errors.As(err, &pgErr))
		assert.Equal(t, "23514", pgErr.Code)
		assert.True(t, strings.HasPrefix(pgErr.ConstraintName, "chk_finance_accounts_"))
	})

	t.Run("Reject invalid normal balance", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, '9992', 'Invalid Balance Account', 'ASSET', 'UNKNOWN', 'INR', TRUE, NOW())
		`, uuid.New()).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_accounts_normal_balance")
	})

	t.Run("Reject ASSET with normal_balance CREDIT", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, '9993', 'Mismatched Asset', 'ASSET', 'CREDIT', 'INR', TRUE, NOW())
		`, uuid.New()).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_accounts_normal_balance_by_type")
	})

	t.Run("Reject EXPENSE with normal_balance CREDIT", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, '9994', 'Mismatched Expense', 'EXPENSE', 'CREDIT', 'INR', TRUE, NOW())
		`, uuid.New()).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_accounts_normal_balance_by_type")
	})

	t.Run("Reject LIABILITY with normal_balance DEBIT", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, '9995', 'Mismatched Liability', 'LIABILITY', 'DEBIT', 'INR', TRUE, NOW())
		`, uuid.New()).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_accounts_normal_balance_by_type")
	})

	t.Run("Reject REVENUE with normal_balance DEBIT", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, '9996', 'Mismatched Revenue', 'REVENUE', 'DEBIT', 'INR', TRUE, NOW())
		`, uuid.New()).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_accounts_normal_balance_by_type")
	})

	t.Run("Reject EQUITY with normal_balance DEBIT", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, '9997', 'Mismatched Equity', 'EQUITY', 'DEBIT', 'INR', TRUE, NOW())
		`, uuid.New()).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_accounts_normal_balance_by_type")
	})

	t.Run("Reject non-INR currency in account", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, '9998', 'USD Account', 'ASSET', 'DEBIT', 'USD', TRUE, NOW())
		`, uuid.New()).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_accounts_currency")
	})

	t.Run("Reject duplicate account code", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()
		err := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, 'DUPL_CODE', 'Account 1', 'ASSET', 'DEBIT', 'INR', TRUE, NOW())
		`, id1).Error
		require.NoError(t, err)

		err2 := db.Exec(`
			INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
			VALUES (?, 'DUPL_CODE', 'Account 2', 'ASSET', 'DEBIT', 'INR', TRUE, NOW())
		`, id2).Error
		require.Error(t, err2)
		var pgErr *pgconn.PgError
		require.True(t, errors.As(err2, &pgErr))
		assert.Equal(t, "23505", pgErr.Code)
	})
}

func TestFinance_LedgerEntries_Constraints(t *testing.T) {
	db := initTestDatabase(t)
	accA, _ := setupAccounts(t, db)

	jID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 's1', 'k_le_constraints', 'fp1', 'INR', 'desc', 'DRAFT', NOW(), FALSE)
	`, jID).Error)

	t.Run("Reject entry with invalid entry_type", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'INVALID', 100.00, 'INR', NOW())
		`, uuid.New(), jID, accA).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_ledger_entries_entry_type")
	})

	t.Run("Reject entry with zero amount", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 0.00, 'INR', NOW())
		`, uuid.New(), jID, accA).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_ledger_entries_positive_amount")
	})

	t.Run("Reject entry with negative amount", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', -50.00, 'INR', NOW())
		`, uuid.New(), jID, accA).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_ledger_entries_positive_amount")
	})

	t.Run("Reject entry with non-INR currency", func(t *testing.T) {
		err := db.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 100.00, 'USD', NOW())
		`, uuid.New(), jID, accA).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chk_finance_ledger_entries_currency")
	})
}

func TestFinance_Migration_UpAndDownLifecycle(t *testing.T) {
	db := initTestDatabase(t)

	downSQL, err := loadMigrationSQL("000017_create_finance_ledger.down.sql")
	require.NoError(t, err)
	upSQL, err := loadMigrationSQL("000017_create_finance_ledger.up.sql")
	require.NoError(t, err)

	// Verify DOWN drops all objects cleanly
	require.NoError(t, db.Exec(downSQL).Error, "Down migration must execute cleanly")

	// Verify UP applies cleanly from clean state
	require.NoError(t, db.Exec(upSQL).Error, "Up migration must execute cleanly after Down")
}
