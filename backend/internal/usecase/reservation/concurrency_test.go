package reservation_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/database"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/reservation"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	testDBOnce      sync.Once
	sharedTestDB    *gorm.DB
	sharedTestDBErr error
)

// findMigrationsDir traverses up from startDir until it finds the folder containing the "migrations" directory.
func findMigrationsDir(startDir string) string {
	dir := startDir
	for {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// resolveTestDatabaseURL resolves the connection string for the dedicated test database without hardcoding credentials.
func resolveTestDatabaseURL() string {
	if testURL := os.Getenv("TEST_DATABASE_URL"); testURL != "" {
		return testURL
	}

	// If DATABASE_URL is not in the environment, try loading .env from parent directories
	if os.Getenv("DATABASE_URL") == "" {
		if wd, err := os.Getwd(); err == nil {
			dir := wd
			for {
				envPath := filepath.Join(dir, ".env")
				if info, err := os.Stat(envPath); err == nil && !info.IsDir() {
					_ = godotenv.Load(envPath)
					break
				}
				parent := filepath.Dir(dir)
				if parent == dir {
					break
				}
				dir = parent
			}
		}
	}

	if devURL := os.Getenv("DATABASE_URL"); devURL != "" {
		// Replace dbname parameter with eventhub_concurrency_test to safely reuse connection credentials
		if strings.Contains(devURL, "dbname=") {
			re := regexp.MustCompile(`dbname=\S+`)
			return re.ReplaceAllString(devURL, "dbname=eventhub_concurrency_test")
		}
		if u, err := url.Parse(devURL); err == nil && u.Scheme != "" {
			u.Path = "/eventhub_concurrency_test"
			return u.String()
		}
	}

	host := os.Getenv("PGHOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PGPORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("PGUSER")
	if user == "" {
		user = "postgres"
	}
	pass := os.Getenv("PGPASSWORD")
	if pass != "" {
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=eventhub_concurrency_test sslmode=disable", host, port, user, pass)
	}

	return fmt.Sprintf("host=%s port=%s user=%s dbname=eventhub_concurrency_test sslmode=disable", host, port, user)
}

func getIsolatedTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	testDBOnce.Do(func() {
		dbURL := resolveTestDatabaseURL()

		db, err := database.Connect(dbURL)
		if err != nil {
			sharedTestDBErr = fmt.Errorf("failed to connect to test database at %s: %w", dbURL, err)
			return
		}

		sqlDB, err := db.DB()
		if err != nil {
			sharedTestDBErr = fmt.Errorf("failed to obtain sql.DB: %w", err)
			return
		}

		if err := sqlDB.Ping(); err != nil {
			sharedTestDBErr = fmt.Errorf("failed to ping test database: %w", err)
			return
		}

		// Safety check: Validate connected database name from the database engine itself.
		// NEVER run migrations or tests on development ("eventhub") or production databases.
		var currentDB string
		if err := db.Raw("SELECT current_database()").Scan(&currentDB).Error; err != nil {
			sharedTestDBErr = fmt.Errorf("failed to verify current database name: %w", err)
			return
		}

		currentDBLower := strings.ToLower(currentDB)
		if currentDBLower == "eventhub" || currentDBLower == "postgres" || !strings.Contains(currentDBLower, "test") {
			sharedTestDBErr = fmt.Errorf("safety violation: Refusing to run tests/migrations on non-test database %q (must contain 'test')", currentDB)
			return
		}

		// Ensure the working directory allows file://migrations in database.Migrate(db) to resolve correctly
		origWd, err := os.Getwd()
		if err != nil {
			sharedTestDBErr = fmt.Errorf("failed to get working directory: %w", err)
			return
		}

		migrationsParentDir := findMigrationsDir(origWd)
		if migrationsParentDir == "" {
			sharedTestDBErr = fmt.Errorf("could not locate backend migrations directory from %s", origWd)
			return
		}

		if err := os.Chdir(migrationsParentDir); err != nil {
			sharedTestDBErr = fmt.Errorf("failed to chdir to %s for migrations: %w", migrationsParentDir, err)
			return
		}

		// In PostgreSQL, 000001_baseline.up.sql sets session search_path = '' (via pg_dump output).
		// Because database.Migrate executes all migrations sequentially over a single connection,
		// subsequent migrations (such as 000002_create_bookings.up.sql) that create unqualified tables
		// fail with SQLSTATE 3F000 ("no schema has been selected to create in").
		// To fix this transparently in the isolated test database without modifying production migration files,
		// install a temporary DDL event trigger that ensures search_path is reset to 'public' across DDL steps.
		_ = db.Exec(`
			CREATE OR REPLACE FUNCTION reset_search_path_on_ddl()
			RETURNS event_trigger AS $$
			BEGIN
				PERFORM set_config('search_path', 'public', false);
			END;
			$$ LANGUAGE plpgsql;
		`).Error

		_ = db.Exec(`
			DROP EVENT TRIGGER IF EXISTS ensure_public_search_path;
			CREATE EVENT TRIGGER ensure_public_search_path
			ON ddl_command_end
			EXECUTE FUNCTION reset_search_path_on_ddl();
		`).Error

		migrateErr := database.Migrate(db)
		// Clean up the temporary event trigger after migrations finish
		_ = db.Exec(`DROP EVENT TRIGGER IF EXISTS ensure_public_search_path;`).Error
		_ = db.Exec(`DROP FUNCTION IF EXISTS reset_search_path_on_ddl();`).Error

		// Always restore the original working directory
		_ = os.Chdir(origWd)

		if migrateErr != nil {
			sharedTestDBErr = fmt.Errorf("database migration failed on %s: %w", currentDB, migrateErr)
			return
		}

		// Configure bounded pool safely below PostgreSQL max_connections (100)
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(15)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)

		sharedTestDB = db
	})

	if sharedTestDBErr != nil {
		t.Fatalf("Test database setup failed: %v", sharedTestDBErr)
	}

	if sharedTestDB == nil {
		t.Fatal("Test database setup failed: sharedTestDB is nil")
	}

	return sharedTestDB
}

// createTestCategory creates an active category in the isolated test database.
func createTestCategory(t *testing.T, db *gorm.DB) *models.CategoryModel {
	t.Helper()
	cat := &models.CategoryModel{
		ID:          uuid.New(),
		Name:        "Concert " + uuid.New().String()[:8],
		Description: "Concurrency testing category",
		Status:      "ACTIVE",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, db.Create(cat).Error)
	t.Cleanup(func() {
		_ = db.Delete(cat).Error
	})
	return cat
}

var userCounter int64
var counterMu sync.Mutex

func getUniqueCounter() int64 {
	counterMu.Lock()
	defer counterMu.Unlock()
	userCounter++
	return userCounter
}

// createTestUser creates an active user with the specified role.
func createTestUser(t *testing.T, db *gorm.DB, role string) *models.UserModel {
	t.Helper()
	uid := uuid.New()
	c := getUniqueCounter()
	user := &models.UserModel{
		ID:           uid,
		FullName:     fmt.Sprintf("User %s", uid.String()[:8]),
		Email:        fmt.Sprintf("test_%s@example.com", uid.String()),
		Phone:        fmt.Sprintf("+91%010d", c+1000000000),
		PasswordHash: "hashed_secret",
		Role:         role,
		Status:       "ACTIVE",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(user).Error)
	t.Cleanup(func() {
		_ = db.Delete(user).Error
	})
	return user
}

// createTestSeatedEvent creates a published physical event with a complete seat layout.
func createTestSeatedEvent(t *testing.T, db *gorm.DB, numSeats int) (*models.EventModel, *models.SeatLayoutModel, []*models.SeatModel) {
	t.Helper()

	organizer := createTestUser(t, db, "ORGANIZER")
	category := createTestCategory(t, db)

	eventID := uuid.New()
	event := &models.EventModel{
		ID:             eventID,
		OrganizerID:    organizer.ID,
		CategoryID:     category.ID,
		EventType:      domain.EventTypePhysical,
		Title:          "Concurrency Seated Event " + eventID.String()[:8],
		Description:    "Automated Concurrency Testing Event",
		AgeRestriction: 0,
		Visibility:     "PUBLIC",
		Status:         domain.EventStatusPublished,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, db.Create(event).Error)

	setting := &models.EventSettingModel{
		ID:                  uuid.New(),
		EventID:             event.ID,
		SeatLayoutType:      reservation.SeatLayoutTypeSeated,
		BookingLimitPerUser: 10,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	require.NoError(t, db.Create(setting).Error)

	layout := &models.SeatLayoutModel{
		ID:         uuid.New(),
		EventID:    event.ID,
		LayoutName: "Main Hall " + eventID.String()[:8],
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, db.Create(layout).Error)

	section := &models.SeatSectionModel{
		ID:           uuid.New(),
		SeatLayoutID: layout.ID,
		Name:         "Orchestra",
		Price:        decimal.NewFromInt(150),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(section).Error)

	row := &models.SeatRowModel{
		ID:        uuid.New(),
		SectionID: section.ID,
		RowName:   "A",
		CreatedAt: time.Now(),
	}
	require.NoError(t, db.Create(row).Error)

	seats := make([]*models.SeatModel, 0, numSeats)
	for i := 1; i <= numSeats; i++ {
		seat := &models.SeatModel{
			ID:         uuid.New(),
			RowID:      row.ID,
			GridCol:    i,
			SeatNumber: i,
			Status:     domain.SeatStatusAvailable,
			CreatedAt:  time.Now(),
		}
		require.NoError(t, db.Create(seat).Error)
		seats = append(seats, seat)
	}

	t.Cleanup(func() {
		// Clean up in reverse dependency order
		_ = db.Where("row_id = ?", row.ID).Delete(&models.SeatModel{}).Error
		_ = db.Delete(row).Error
		_ = db.Delete(section).Error
		_ = db.Delete(layout).Error
		_ = db.Where("event_id = ?", event.ID).Delete(&models.EventSettingModel{}).Error
		_ = db.Delete(event).Error
	})

	return event, layout, seats
}

// createTestGeneralEvent creates a published event with a General Admission ticket tier.
func createTestGeneralEvent(t *testing.T, db *gorm.DB, availableQty int) (*models.EventModel, *models.TicketTypeModel) {
	t.Helper()

	organizer := createTestUser(t, db, "ORGANIZER")
	category := createTestCategory(t, db)

	eventID := uuid.New()
	event := &models.EventModel{
		ID:             eventID,
		OrganizerID:    organizer.ID,
		CategoryID:     category.ID,
		EventType:      domain.EventTypePhysical,
		Title:          "Concurrency General Event " + eventID.String()[:8],
		Description:    "Automated Concurrency Testing GA Event",
		AgeRestriction: 0,
		Visibility:     "PUBLIC",
		Status:         domain.EventStatusPublished,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, db.Create(event).Error)

	setting := &models.EventSettingModel{
		ID:                  uuid.New(),
		EventID:             event.ID,
		SeatLayoutType:      reservation.SeatLayoutTypeGeneral,
		BookingLimitPerUser: 10,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	require.NoError(t, db.Create(setting).Error)

	ticketType := &models.TicketTypeModel{
		ID:                uuid.New(),
		EventID:           event.ID,
		Name:              "General Admission",
		Price:             decimal.NewFromInt(50),
		TotalQuantity:     availableQty + 5,
		AvailableQuantity: availableQty,
		Description:       "General Admission Tier",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	require.NoError(t, db.Create(ticketType).Error)

	t.Cleanup(func() {
		_ = db.Delete(ticketType).Error
		_ = db.Where("event_id = ?", event.ID).Delete(&models.EventSettingModel{}).Error
		_ = db.Delete(event).Error
	})

	return event, ticketType
}

// TestConcurrentSeatReservation_SingleSeat ensures that when multiple customers
// simultaneously attempt to reserve the exact same seat, exactly one succeeds and all others fail.
func TestConcurrentSeatReservation_SingleSeat(t *testing.T) {
	db := getIsolatedTestDB(t)

	// Build Clean Architecture reservation usecase with silent logger for clean test output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txManager := repository.NewTransactionManager(db, logger)
	usecase := reservation.NewReservationUsecase(txManager, logger)

	// Create event with 1 available seat
	event, _, seats := createTestSeatedEvent(t, db, 1)
	targetSeat := seats[0]

	const numGoroutines = 15
	users := make([]*models.UserModel, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		users[i] = createTestUser(t, db, "CUSTOMER")
	}

	type reserveResult struct {
		userIndex   int
		reservation *domain.SeatReservation
		err         error
	}

	results := make([]reserveResult, numGoroutines)
	var startBarrier sync.WaitGroup
	startBarrier.Add(1)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		userIndex := i
		userID := users[i].ID

		go func() {
			defer wg.Done()
			// Wait for barrier release so all goroutines fire concurrently
			startBarrier.Wait()

			ctx := context.Background()
			res, err := usecase.CreateReservation(ctx, reservation.CreateReservationRequest{
				UserID:  userID,
				EventID: event.ID,
				SeatIDs: []uuid.UUID{targetSeat.ID},
			})

			results[userIndex] = reserveResult{
				userIndex:   userIndex,
				reservation: res,
				err:         err,
			}
		}()
	}

	// Release all goroutines simultaneously
	startBarrier.Done()
	wg.Wait()

	// Analyze outcomes
	var successCount int
	var conflictCount int
	var otherErrors []error
	var successfulReservation *domain.SeatReservation

	for _, res := range results {
		if res.err == nil {
			successCount++
			successfulReservation = res.reservation
		} else if errors.Is(res.err, domain.ErrSeatNotAvailable) {
			conflictCount++
		} else {
			otherErrors = append(otherErrors, res.err)
		}
	}

	// 1. Exactly one reservation must succeed
	assert.Equal(t, 1, successCount, "Exactly one reservation must succeed")
	assert.Equal(t, numGoroutines-1, conflictCount, "All other attempts must fail with ErrSeatNotAvailable")
	assert.Empty(t, otherErrors, "No unexpected error types should occur")
	require.NotNil(t, successfulReservation, "Successful reservation object must be returned")

	// 2. Database verification: seat status must be RESERVED
	var dbSeat models.SeatModel
	err := db.First(&dbSeat, "id = ?", targetSeat.ID).Error
	require.NoError(t, err)
	assert.Equal(t, domain.SeatStatusReserved, dbSeat.Status, "Seat status in database must be RESERVED")

	// 3. Exactly one active reservation record in database
	var resCount int64
	err = db.Model(&models.SeatReservationModel{}).
		Where("event_id = ? AND status = ?", event.ID, domain.ReservationStatusActive).
		Count(&resCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), resCount, "Exactly one active reservation record must exist in DB")

	// 4. Exactly one reservation item in database for target seat
	var itemCount int64
	err = db.Model(&models.SeatReservationItemModel{}).
		Where("seat_id = ?", targetSeat.ID).
		Count(&itemCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), itemCount, "Exactly one reservation item must exist for the target seat")

	// Cleanup test reservation records
	t.Cleanup(func() {
		_ = db.Where("seat_id = ?", targetSeat.ID).Delete(&models.SeatReservationItemModel{}).Error
		if successfulReservation != nil {
			_ = db.Where("id = ?", successfulReservation.ID).Delete(&models.SeatReservationModel{}).Error
		}
	})
}

// TestConcurrentSeatReservation_DifferentSeats verifies that multiple customers
// concurrently reserving distinct seats for the same event all succeed without deadlocks.
func TestConcurrentSeatReservation_DifferentSeats(t *testing.T) {
	db := getIsolatedTestDB(t)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txManager := repository.NewTransactionManager(db, logger)
	usecase := reservation.NewReservationUsecase(txManager, logger)

	const numSeats = 10
	event, _, seats := createTestSeatedEvent(t, db, numSeats)

	users := make([]*models.UserModel, numSeats)
	for i := 0; i < numSeats; i++ {
		users[i] = createTestUser(t, db, "CUSTOMER")
	}

	results := make([]error, numSeats)
	var startBarrier sync.WaitGroup
	startBarrier.Add(1)

	var wg sync.WaitGroup
	wg.Add(numSeats)

	for i := 0; i < numSeats; i++ {
		idx := i
		userID := users[i].ID
		seatID := seats[i].ID

		go func() {
			defer wg.Done()
			startBarrier.Wait()

			ctx := context.Background()
			_, err := usecase.CreateReservation(ctx, reservation.CreateReservationRequest{
				UserID:  userID,
				EventID: event.ID,
				SeatIDs: []uuid.UUID{seatID},
			})
			results[idx] = err
		}()
	}

	startBarrier.Done()
	wg.Wait()

	// Every unique seat reservation must succeed
	for i, err := range results {
		assert.NoError(t, err, "Reservation for distinct seat %d must succeed", i+1)
	}

	// Verify all 10 seats are RESERVED
	var reservedCount int64
	seatIDs := make([]uuid.UUID, numSeats)
	for i, s := range seats {
		seatIDs[i] = s.ID
	}
	err := db.Model(&models.SeatModel{}).
		Where("id IN ? AND status = ?", seatIDs, domain.SeatStatusReserved).
		Count(&reservedCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(numSeats), reservedCount, "All distinct seats must be RESERVED in database")

	// Cleanup
	t.Cleanup(func() {
		_ = db.Where("seat_id IN ?", seatIDs).Delete(&models.SeatReservationItemModel{}).Error
		_ = db.Where("event_id = ?", event.ID).Delete(&models.SeatReservationModel{}).Error
	})
}

// TestConcurrentGeneralAdmission_LastTicket verifies that when multiple customers
// compete concurrently for the last available ticket in a General Admission event,
// exactly one succeeds and all others receive ErrInsufficientTicketQuantity.
func TestConcurrentGeneralAdmission_LastTicket(t *testing.T) {
	db := getIsolatedTestDB(t)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txManager := repository.NewTransactionManager(db, logger)
	usecase := reservation.NewReservationUsecase(txManager, logger)

	// Only 1 ticket available
	event, ticketType := createTestGeneralEvent(t, db, 1)

	const numGoroutines = 12
	users := make([]*models.UserModel, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		users[i] = createTestUser(t, db, "CUSTOMER")
	}

	results := make([]error, numGoroutines)
	var startBarrier sync.WaitGroup
	startBarrier.Add(1)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		idx := i
		userID := users[i].ID

		go func() {
			defer wg.Done()
			startBarrier.Wait()

			ctx := context.Background()
			_, err := usecase.CreateReservation(ctx, reservation.CreateReservationRequest{
				UserID:   userID,
				EventID:  event.ID,
				Quantity: 1,
				Tickets: []reservation.TicketSelection{
					{
						TicketTypeID: ticketType.ID,
						Quantity:     1,
					},
				},
			})
			results[idx] = err
		}()
	}

	startBarrier.Done()
	wg.Wait()

	var successCount int
	var insufficientCount int
	for _, err := range results {
		if err == nil {
			successCount++
		} else if errors.Is(err, domain.ErrInsufficientTicketQuantity) {
			insufficientCount++
		}
	}

	assert.Equal(t, 1, successCount, "Exactly 1 general admission reservation must succeed")
	assert.Equal(t, numGoroutines-1, insufficientCount, "All other attempts must receive ErrInsufficientTicketQuantity")

	// Verify available quantity in database is exactly 0 (no negative inventory)
	var updatedTier models.TicketTypeModel
	err := db.First(&updatedTier, "id = ?", ticketType.ID).Error
	require.NoError(t, err)
	assert.Equal(t, 0, updatedTier.AvailableQuantity, "Available ticket quantity must be 0")

	// Cleanup
	t.Cleanup(func() {
		_ = db.Where("ticket_type_id = ?", ticketType.ID).Delete(&models.SeatReservationItemModel{}).Error
		_ = db.Where("event_id = ?", event.ID).Delete(&models.SeatReservationModel{}).Error
	})
}

// TestFailedReservation_RollsBackAtomically verifies that when a reservation request
// contains both an available seat and an unavailable seat, the entire transaction rolls back
// and does NOT leave the available seat partially reserved.
func TestFailedReservation_RollsBackAtomically(t *testing.T) {
	db := getIsolatedTestDB(t)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txManager := repository.NewTransactionManager(db, logger)
	usecase := reservation.NewReservationUsecase(txManager, logger)

	event, _, seats := createTestSeatedEvent(t, db, 2)
	seatAvailable := seats[0]
	seatAlreadyReserved := seats[1]

	// Mark seatAlreadyReserved as RESERVED beforehand
	err := db.Model(&models.SeatModel{}).
		Where("id = ?", seatAlreadyReserved.ID).
		Update("status", domain.SeatStatusReserved).Error
	require.NoError(t, err)

	user := createTestUser(t, db, "CUSTOMER")

	// Attempt to reserve BOTH seats in one batch
	ctx := context.Background()
	_, err = usecase.CreateReservation(ctx, reservation.CreateReservationRequest{
		UserID:  user.ID,
		EventID: event.ID,
		SeatIDs: []uuid.UUID{seatAvailable.ID, seatAlreadyReserved.ID},
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrSeatNotAvailable))

	// Verify that seatAvailable was rolled back and is STILL AVAILABLE
	var checkSeatA models.SeatModel
	err = db.First(&checkSeatA, "id = ?", seatAvailable.ID).Error
	require.NoError(t, err)
	assert.Equal(t, domain.SeatStatusAvailable, checkSeatA.Status, "Seat A must remain AVAILABLE after transaction rollback")

	// Verify no reservation record was created
	var count int64
	err = db.Model(&models.SeatReservationModel{}).Where("user_id = ?", user.ID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "No reservation record must exist in DB")
}

// TestConcurrentCancellation_IdempotentRelease verifies that multiple concurrent
// cancellation attempts on the same active reservation release inventory exactly once.
func TestConcurrentCancellation_IdempotentRelease(t *testing.T) {
	db := getIsolatedTestDB(t)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txManager := repository.NewTransactionManager(db, logger)
	usecase := reservation.NewReservationUsecase(txManager, logger)

	event, _, seats := createTestSeatedEvent(t, db, 1)
	seat := seats[0]
	user := createTestUser(t, db, "CUSTOMER")

	ctx := context.Background()
	res, err := usecase.CreateReservation(ctx, reservation.CreateReservationRequest{
		UserID:  user.ID,
		EventID: event.ID,
		SeatIDs: []uuid.UUID{seat.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	// Verify seat is initially RESERVED
	var initialSeat models.SeatModel
	err = db.First(&initialSeat, "id = ?", seat.ID).Error
	require.NoError(t, err)
	require.Equal(t, domain.SeatStatusReserved, initialSeat.Status)

	// Launch 10 goroutines concurrently cancelling the exact same reservation
	const numGoroutines = 10
	results := make([]error, numGoroutines)
	var startBarrier sync.WaitGroup
	startBarrier.Add(1)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		idx := i
		go func() {
			defer wg.Done()
			startBarrier.Wait()

			results[idx] = usecase.CancelReservation(context.Background(), res.ID, user.ID)
		}()
	}

	startBarrier.Done()
	wg.Wait()

	var successCount int
	var notFoundCount int
	for _, cancelErr := range results {
		if cancelErr == nil {
			successCount++
		} else if errors.Is(cancelErr, domain.ErrSeatReservationNotFound) {
			notFoundCount++
		}
	}

	assert.Equal(t, 1, successCount, "Exactly one cancellation must succeed")
	assert.Equal(t, numGoroutines-1, notFoundCount, "All other cancellation calls must return ErrSeatReservationNotFound")

	// Verify seat in database is AVAILABLE (released exactly once)
	var finalSeat models.SeatModel
	err = db.First(&finalSeat, "id = ?", seat.ID).Error
	require.NoError(t, err)
	assert.Equal(t, domain.SeatStatusAvailable, finalSeat.Status, "Seat status must be AVAILABLE after release")

	// Verify reservation status in DB is CANCELLED
	var finalRes models.SeatReservationModel
	err = db.First(&finalRes, "id = ?", res.ID).Error
	require.NoError(t, err)
	assert.Equal(t, domain.ReservationStatusCancelled, finalRes.Status, "Reservation status must be CANCELLED")

	// Cleanup
	t.Cleanup(func() {
		_ = db.Where("id = ?", res.ID).Delete(&models.SeatReservationModel{}).Error
	})
}
