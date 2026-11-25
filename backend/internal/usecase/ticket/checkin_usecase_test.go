package ticket

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// In-memory mock repositories and transaction manager
type mockTransactionManager struct {
	txRepos domain.TransactionRepositories
}

func (m *mockTransactionManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	// Create an isolated snapshot of state for rollback simulation
	txCopy := m.txRepos.(*mockTxRepos).clone()
	err := fn(txCopy)
	if err != nil {
		// Rollback: do not commit clone to parent
		return err
	}
	// Commit: apply changes to parent
	m.txRepos.(*mockTxRepos).apply(txCopy)
	return nil
}

type mockTxRepos struct {
	mu           sync.Mutex
	organizers   map[uuid.UUID]*domain.Organizer
	events       map[uuid.UUID]*domain.Event
	bookings     map[uuid.UUID]*domain.Booking
	tickets      map[uuid.UUID]*domain.Ticket
	scans        []domain.TicketScan
	failScanRepo bool
	scanRepoErr  error
}

func newMockTxRepos() *mockTxRepos {
	return &mockTxRepos{
		organizers: make(map[uuid.UUID]*domain.Organizer),
		events:     make(map[uuid.UUID]*domain.Event),
		bookings:   make(map[uuid.UUID]*domain.Booking),
		tickets:    make(map[uuid.UUID]*domain.Ticket),
		scans:      make([]domain.TicketScan, 0),
	}
}

func (m *mockTxRepos) clone() *mockTxRepos {
	m.mu.Lock()
	defer m.mu.Unlock()

	c := newMockTxRepos()
	c.failScanRepo = m.failScanRepo
	c.scanRepoErr = m.scanRepoErr

	for k, v := range m.organizers {
		val := *v
		c.organizers[k] = &val
	}
	for k, v := range m.events {
		val := *v
		c.events[k] = &val
	}
	for k, v := range m.bookings {
		val := *v
		c.bookings[k] = &val
	}
	for k, v := range m.tickets {
		val := *v
		c.tickets[k] = &val
	}
	c.scans = append(c.scans, m.scans...)
	return c
}

func (m *mockTxRepos) apply(from *mockTxRepos) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.organizers = from.organizers
	m.events = from.events
	m.bookings = from.bookings
	m.tickets = from.tickets
	m.scans = from.scans
}

// Unused TransactionRepositories methods implemented as stubs
func (m *mockTxRepos) ExecuteWithSavepoint(name domain.SavepointName, fn func() error) (domain.SavepointResult, error) {
	return domain.SavepointResult{Status: domain.SavepointStatusCommitted}, fn()
}
func (m *mockTxRepos) UserRepository() domain.UserRepository                                 { return nil }
func (m *mockTxRepos) PendingRegistrationRepository() domain.PendingRegistrationRepository   { return nil }
func (m *mockTxRepos) EmailChangeRequestRepository() domain.EmailChangeRequestRepository     { return nil }
func (m *mockTxRepos) AcquireEmailAdvisoryLock(email string) error                           { return nil }
func (m *mockTxRepos) PasswordResetTokenRepository() domain.PasswordResetTokenRepository     { return nil }
func (m *mockTxRepos) RefreshTokenRepository() domain.RefreshTokenRepository                 { return nil }
func (m *mockTxRepos) OrganizerApplicationRepository() domain.OrganizerApplicationRepository { return nil }
func (m *mockTxRepos) OrganizerProfileRepository() domain.OrganizerProfileRepository         { return nil }
func (m *mockTxRepos) OrganizerAddressRepository() domain.OrganizerAddressRepository         { return nil }
func (m *mockTxRepos) OrganizerBankAccountRepository() domain.OrganizerBankAccountRepository { return nil }
func (m *mockTxRepos) CategoryRepository() domain.CategoryRepository                         { return nil }
func (m *mockTxRepos) VenueRepository() domain.VenueRepository                               { return nil }
func (m *mockTxRepos) EventScheduleRepository() domain.EventScheduleRepository               { return nil }
func (m *mockTxRepos) EventSettingRepository() domain.EventSettingRepository                 { return nil }
func (m *mockTxRepos) EventCancellationRepository() domain.EventCancellationRepository       { return nil }
func (m *mockTxRepos) EventContactRepository() domain.EventContactRepository                 { return nil }
func (m *mockTxRepos) SeatLayoutRepository() domain.SeatLayoutRepository                     { return nil }
func (m *mockTxRepos) SeatColumnRepository() domain.SeatColumnRepository                     { return nil }
func (m *mockTxRepos) SeatSectionRepository() domain.SeatSectionRepository                   { return nil }
func (m *mockTxRepos) SeatRowRepository() domain.SeatRowRepository                           { return nil }
func (m *mockTxRepos) SeatRepository() domain.SeatRepository                                 { return nil }
func (m *mockTxRepos) TicketTypeRepository() domain.TicketTypeRepository                     { return nil }
func (m *mockTxRepos) SeatReservationRepository() domain.SeatReservationRepository           { return nil }
func (m *mockTxRepos) CheckoutRepository() domain.CheckoutRepository                         { return nil }
func (m *mockTxRepos) PaymentRepository() domain.PaymentRepository                           { return nil }
func (m *mockTxRepos) RefundRepository() domain.RefundRepository                             { return nil }
func (m *mockTxRepos) RazorpayWebhookEventRepository() domain.RazorpayWebhookEventRepository { return nil }
func (m *mockTxRepos) PricingTaxConfigurationRepository() domain.PricingTaxConfigurationRepository {
	return nil
}
func (m *mockTxRepos) InvoiceRepository() domain.InvoiceRepository               { return nil }
func (m *mockTxRepos) FinanceAccountRepository() domain.FinanceAccountRepository { return nil }
func (m *mockTxRepos) FinanceJournalRepository() domain.FinanceJournalRepository { return nil }
func (m *mockTxRepos) SettlementRepository() domain.SettlementRepository         { return nil }

func (m *mockTxRepos) OrganizerRepository() domain.OrganizerRepository {
	return &mockOrganizerRepo{m: m}
}
func (m *mockTxRepos) EventRepository() domain.EventRepository {
	return &mockEventRepo{m: m}
}
func (m *mockTxRepos) BookingRepository() domain.BookingRepository {
	return &mockBookingRepo{m: m}
}
func (m *mockTxRepos) TicketRepository() domain.TicketRepository {
	return &mockTicketRepo{m: m}
}
func (m *mockTxRepos) TicketScanRepository() domain.TicketScanRepository {
	return &mockTicketScanRepo{m: m}
}

// Mock Organizer Repository
type mockOrganizerRepo struct {
	m *mockTxRepos
}

func (r *mockOrganizerRepo) Create(organizer *domain.Organizer) error { return nil }
func (r *mockOrganizerRepo) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	for _, o := range r.m.organizers {
		if o.ID == id {
			return o, nil
		}
	}
	return nil, domain.ErrOrganizerNotFound
}
func (r *mockOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	o, ok := r.m.organizers[userID]
	if !ok {
		return nil, domain.ErrOrganizerNotFound
	}
	return o, nil
}
func (r *mockOrganizerRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Organizer, error) {
	return r.FindByID(id)
}
func (r *mockOrganizerRepo) List(filter domain.AdminOrganizerListFilter) (*domain.AdminOrganizerList, error) {
	return nil, nil
}

// Mock Event Repository
type mockEventRepo struct {
	m *mockTxRepos
}

func (r *mockEventRepo) Create(event *domain.Event) error { return nil }
func (r *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	ev, ok := r.m.events[id]
	if !ok {
		return nil, domain.ErrEventNotFound
	}
	return ev, nil
}
func (r *mockEventRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) {
	return r.FindByID(id)
}
func (r *mockEventRepo) Update(event *domain.Event) error             { return nil }
func (r *mockEventRepo) Delete(id uuid.UUID) error                   { return nil }
func (r *mockEventRepo) FindByOrganizerID(organizerID uuid.UUID, page, limit int, status, search string) ([]*domain.OrganizerEvent, int64, error) {
	return nil, 0, nil
}
func (r *mockEventRepo) FindPublicEvents(filter domain.PublicEventFilter) ([]*domain.PublicEvent, int64, error) {
	return nil, 0, nil
}
func (r *mockEventRepo) FindAllEvents(filter domain.AdminEventFilter) ([]*domain.AdminEventListItem, int64, error) {
	return nil, 0, nil
}
func (r *mockEventRepo) GetAdminStatusCounts() (domain.AdminEventStatusCounts, error) {
	return domain.AdminEventStatusCounts{}, nil
}

// Mock Booking Repository
type mockBookingRepo struct {
	m *mockTxRepos
}

func (r *mockBookingRepo) Create(ctx context.Context, booking *domain.Booking, items []domain.BookingItem) error {
	return nil
}
func (r *mockBookingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	b, ok := r.m.bookings[id]
	if !ok {
		return nil, domain.ErrBookingNotFound
	}
	return b, nil
}
func (r *mockBookingRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return r.FindByID(ctx, id)
}
func (r *mockBookingRepo) FindByBookingNumber(ctx context.Context, bookingNumber string) (*domain.Booking, error) {
	return nil, nil
}
func (r *mockBookingRepo) FindByReservationID(ctx context.Context, reservationID uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}
func (r *mockBookingRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	return nil, nil
}
func (r *mockBookingRepo) FindByUserIDPaginated(ctx context.Context, userID uuid.UUID, filter domain.CustomerBookingFilter) ([]domain.Booking, int64, error) {
	return nil, 0, nil
}
func (r *mockBookingRepo) FindWithItemsByID(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
	return nil, nil, nil
}
func (r *mockBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (r *mockBookingRepo) ConfirmPending(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (r *mockBookingRepo) UpdateCancellationSnapshot(ctx context.Context, id uuid.UUID, allowed bool, deadlineHours int, policy string, percentage int) error {
	return nil
}
func (r *mockBookingRepo) FindItemsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.BookingItem, error) {
	return nil, nil
}
func (r *mockBookingRepo) FindItemsByBookingIDs(ctx context.Context, bookingIDs []uuid.UUID) (map[uuid.UUID][]domain.BookingItem, error) {
	return nil, nil
}
func (r *mockBookingRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Booking, error) {
	return nil, nil
}
func (r *mockBookingRepo) CountActiveByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *mockBookingRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *mockBookingRepo) CountConfirmedByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *mockBookingRepo) GetBookingStatsByEventID(ctx context.Context, eventID uuid.UUID) (domain.AdminEventBookingStats, error) {
	return domain.AdminEventBookingStats{}, nil
}
func (r *mockBookingRepo) GetAdminEventBookingStats(ctx context.Context, eventID uuid.UUID) (domain.AdminEventBookingStats, error) {
	return domain.AdminEventBookingStats{}, nil
}

// Mock Ticket Repository
type mockTicketRepo struct {
	m *mockTxRepos
}

func (r *mockTicketRepo) Create(ctx context.Context, ticket *domain.Ticket) error { return nil }
func (r *mockTicketRepo) CreateMany(ctx context.Context, tickets []domain.Ticket) error {
	return nil
}
func (r *mockTicketRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	t, ok := r.m.tickets[id]
	if !ok {
		return nil, domain.ErrTicketNotFound
	}
	return t, nil
}
func (r *mockTicketRepo) FindByCode(ctx context.Context, code string) (*domain.Ticket, error) {
	return nil, nil
}
func (r *mockTicketRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	var res []domain.Ticket
	for _, t := range r.m.tickets {
		if t.BookingID == bookingID {
			res = append(res, *t)
		}
	}
	return res, nil
}
func (r *mockTicketRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (r *mockTicketRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (r *mockTicketRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (r *mockTicketRepo) UpdateStatusForTicketIDs(ctx context.Context, ticketIDs []uuid.UUID, status string) error {
	return nil
}
func (r *mockTicketRepo) UpdateStatusByBookingID(ctx context.Context, bookingID uuid.UUID, status string) error {
	return nil
}
func (r *mockTicketRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *mockTicketRepo) MarkAsUsedIfValid(ctx context.Context, ticketID, eventID uuid.UUID, usedAt time.Time) (bool, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	t, ok := r.m.tickets[ticketID]
	if !ok || t.EventID != eventID || t.Status != domain.TicketStatusValid {
		return false, nil
	}
	t.Status = domain.TicketStatusUsed
	t.UsedAt = &usedAt
	return true, nil
}

// Mock TicketScan Repository
type mockTicketScanRepo struct {
	m *mockTxRepos
}

func (r *mockTicketScanRepo) Create(ctx context.Context, scan *domain.TicketScan) error {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	if r.m.failScanRepo {
		if r.m.scanRepoErr != nil {
			return r.m.scanRepoErr
		}
		return errors.New(`ERROR: relation "ticket_scans" does not exist (SQLSTATE 42P01)`)
	}
	r.m.scans = append(r.m.scans, *scan)
	return nil
}

// Helper to set up test environment
func setupTestContext(t *testing.T) (*CheckInUseCase, *mockTxRepos, *QRService, uuid.UUID, uuid.UUID, uuid.UUID) {
	secret := "test-secret-at-least-32-bytes-long-12345"
	qrService, err := NewQRService(secret)
	if err != nil {
		t.Fatalf("failed to create QRService: %v", err)
	}

	repos := newMockTxRepos()
	txManager := &mockTransactionManager{txRepos: repos}
	logger := slog.Default()

	useCase := NewCheckInUseCase(txManager, qrService, logger)

	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()

	repos.organizers[userID] = &domain.Organizer{
		ID:     organizerID,
		UserID: userID,
		Status: "APPROVED",
	}

	repos.events[eventID] = &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		Title:       "Test Concert",
		Status:      "PUBLISHED",
	}

	return useCase, repos, qrService, userID, organizerID, eventID
}

// 1. Valid individual ticket QR
func TestCheckIn_ValidIndividualTicketQR(t *testing.T) {
	uc, repos, qrSvc, userID, _, eventID := setupTestContext(t)
	ticketID := uuid.New()
	bookingID := uuid.New()

	repos.tickets[ticketID] = &domain.Ticket{
		ID:         ticketID,
		TicketCode: "TCK-101",
		BookingID:  bookingID,
		EventID:    eventID,
		Status:     domain.TicketStatusValid,
		Price:      decimal.NewFromInt(100),
	}

	payload, err := qrSvc.GeneratePayload(ticketID)
	if err != nil {
		t.Fatalf("failed to generate payload: %v", err)
	}

	res, err := uc.CheckIn(context.Background(), userID, eventID, payload)
	if err != nil {
		t.Fatalf("expected check-in to succeed, got %v", err)
	}

	if res.Type != CheckInTypeIndividual {
		t.Errorf("expected type INDIVIDUAL, got %s", res.Type)
	}
	if res.TotalCount != 1 {
		t.Errorf("expected TotalCount 1, got %d", res.TotalCount)
	}
	if res.Ticket.Status != domain.TicketStatusUsed {
		t.Errorf("expected ticket status USED, got %s", res.Ticket.Status)
	}
	if len(repos.scans) != 1 {
		t.Fatalf("expected 1 scan record, got %d", len(repos.scans))
	}
	if repos.scans[0].TicketID != ticketID || repos.scans[0].Status != domain.TicketScanStatusSuccess {
		t.Errorf("unexpected scan record: %+v", repos.scans[0])
	}
}

// 2. Valid booking QR with multiple tickets
func TestCheckIn_ValidBookingQR_MultipleTickets(t *testing.T) {
	uc, repos, qrSvc, userID, _, eventID := setupTestContext(t)
	bookingID := uuid.New()

	repos.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		BookingNumber: "BK-1000",
		EventID:       eventID,
		Status:        domain.BookingStatusConfirmed,
	}

	t1 := uuid.New()
	t2 := uuid.New()
	t3 := uuid.New()

	for i, tid := range []uuid.UUID{t1, t2, t3} {
		repos.tickets[tid] = &domain.Ticket{
			ID:         tid,
			TicketCode: fmt.Sprintf("TCK-MULT-%d", i+1),
			BookingID:  bookingID,
			EventID:    eventID,
			Status:     domain.TicketStatusValid,
			Price:      decimal.NewFromInt(150),
		}
	}

	payload, err := qrSvc.GenerateBookingPayload(bookingID)
	if err != nil {
		t.Fatalf("failed to generate booking payload: %v", err)
	}

	res, err := uc.CheckIn(context.Background(), userID, eventID, payload)
	if err != nil {
		t.Fatalf("expected booking check-in to succeed, got %v", err)
	}

	if res.Type != CheckInTypeBooking {
		t.Errorf("expected type BOOKING, got %s", res.Type)
	}
	if res.TotalCount != 3 {
		t.Errorf("expected TotalCount 3, got %d", res.TotalCount)
	}
	if len(res.Tickets) != 3 {
		t.Errorf("expected 3 tickets in result, got %d", len(res.Tickets))
	}
	for _, tck := range res.Tickets {
		if tck.Status != domain.TicketStatusUsed {
			t.Errorf("expected ticket %s to be USED, got %s", tck.ID, tck.Status)
		}
	}
	if len(repos.scans) != 3 {
		t.Fatalf("expected 3 scan records, got %d", len(repos.scans))
	}
}

// 3. Invalid booking signature
func TestCheckIn_InvalidBookingSignature(t *testing.T) {
	uc, _, _, userID, _, eventID := setupTestContext(t)
	tamperedPayload := "EVH:BK:a0000000-0000-0000-0000-000000000001.deadbeef12345678"

	_, err := uc.CheckIn(context.Background(), userID, eventID, tamperedPayload)
	if !errors.Is(err, ErrInvalidQRPayload) {
		t.Fatalf("expected ErrInvalidQRPayload, got %v", err)
	}
}

// 4. Booking from another event
func TestCheckIn_BookingFromAnotherEvent(t *testing.T) {
	uc, repos, qrSvc, userID, _, eventID := setupTestContext(t)
	otherEventID := uuid.New()
	bookingID := uuid.New()

	repos.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		BookingNumber: "BK-OTHER-EVENT",
		EventID:       otherEventID,
		Status:        domain.BookingStatusConfirmed,
	}

	payload, _ := qrSvc.GenerateBookingPayload(bookingID)

	_, err := uc.CheckIn(context.Background(), userID, eventID, payload)
	if !errors.Is(err, domain.ErrTicketNotValid) {
		t.Fatalf("expected ErrTicketNotValid, got %v", err)
	}
}

// 5. Unauthorized organizer
func TestCheckIn_UnauthorizedOrganizer(t *testing.T) {
	uc, repos, qrSvc, _, _, eventID := setupTestContext(t)
	ticketID := uuid.New()
	repos.tickets[ticketID] = &domain.Ticket{
		ID:      ticketID,
		EventID: eventID,
		Status:  domain.TicketStatusValid,
	}

	unauthorizedUser := uuid.New()
	unauthorizedOrg := uuid.New()
	repos.organizers[unauthorizedUser] = &domain.Organizer{
		ID:     unauthorizedOrg,
		UserID: unauthorizedUser,
		Status: "APPROVED",
	}

	payload, _ := qrSvc.GeneratePayload(ticketID)

	_, err := uc.CheckIn(context.Background(), unauthorizedUser, eventID, payload)
	if !errors.Is(err, ErrOrganizerNotAuthorized) {
		t.Fatalf("expected ErrOrganizerNotAuthorized, got %v", err)
	}
}

// 6. Already-used ticket
func TestCheckIn_AlreadyUsedTicket(t *testing.T) {
	uc, repos, qrSvc, userID, _, eventID := setupTestContext(t)
	ticketID := uuid.New()
	repos.tickets[ticketID] = &domain.Ticket{
		ID:      ticketID,
		EventID: eventID,
		Status:  domain.TicketStatusUsed,
	}

	payload, _ := qrSvc.GeneratePayload(ticketID)

	_, err := uc.CheckIn(context.Background(), userID, eventID, payload)
	if !errors.Is(err, domain.ErrTicketNotValid) {
		t.Fatalf("expected ErrTicketNotValid, got %v", err)
	}
}

// 7. Repeated booking QR scan
func TestCheckIn_RepeatedBookingQRScan(t *testing.T) {
	uc, repos, qrSvc, userID, _, eventID := setupTestContext(t)
	bookingID := uuid.New()

	repos.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		BookingNumber: "BK-REPEAT",
		EventID:       eventID,
		Status:        domain.BookingStatusConfirmed,
	}

	t1 := uuid.New()
	repos.tickets[t1] = &domain.Ticket{
		ID:        t1,
		BookingID: bookingID,
		EventID:   eventID,
		Status:    domain.TicketStatusValid,
	}

	payload, _ := qrSvc.GenerateBookingPayload(bookingID)

	// First scan succeeds
	res1, err := uc.CheckIn(context.Background(), userID, eventID, payload)
	if err != nil {
		t.Fatalf("first scan failed: %v", err)
	}
	if res1.TotalCount != 1 {
		t.Errorf("expected 1 ticket checked in, got %d", res1.TotalCount)
	}

	// Repeated scan must fail
	_, err = uc.CheckIn(context.Background(), userID, eventID, payload)
	if !errors.Is(err, domain.ErrTicketNotValid) {
		t.Fatalf("expected ErrTicketNotValid on repeated scan, got %v", err)
	}

	// Verify scans count did not duplicate
	if len(repos.scans) != 1 {
		t.Errorf("expected exactly 1 scan recorded, got %d", len(repos.scans))
	}
}

// 8. Rollback when TicketScan insertion fails
func TestCheckIn_RollbackWhenTicketScanInsertionFails(t *testing.T) {
	uc, repos, qrSvc, userID, _, eventID := setupTestContext(t)
	ticketID := uuid.New()
	repos.tickets[ticketID] = &domain.Ticket{
		ID:      ticketID,
		EventID: eventID,
		Status:  domain.TicketStatusValid,
	}

	repos.failScanRepo = true
	repos.scanRepoErr = errors.New("simulated scan insert failure")

	payload, _ := qrSvc.GeneratePayload(ticketID)

	_, err := uc.CheckIn(context.Background(), userID, eventID, payload)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify rollback: ticket remains VALID
	if repos.tickets[ticketID].Status != domain.TicketStatusValid {
		t.Errorf("expected ticket to remain VALID after rollback, got %s", repos.tickets[ticketID].Status)
	}
}

// 9. Missing scan table behavior
func TestCheckIn_MissingScanTableBehavior(t *testing.T) {
	uc, repos, qrSvc, userID, _, eventID := setupTestContext(t)
	ticketID := uuid.New()
	repos.tickets[ticketID] = &domain.Ticket{
		ID:      ticketID,
		EventID: eventID,
		Status:  domain.TicketStatusValid,
	}

	repos.failScanRepo = true
	repos.scanRepoErr = errors.New(`ERROR: relation "ticket_scans" does not exist (SQLSTATE 42P01)`)

	payload, _ := qrSvc.GeneratePayload(ticketID)

	_, err := uc.CheckIn(context.Background(), userID, eventID, payload)
	if err == nil {
		t.Fatal("expected error when scan table missing, got nil")
	}

	if repos.tickets[ticketID].Status != domain.TicketStatusValid {
		t.Errorf("expected ticket status to remain VALID, got %s", repos.tickets[ticketID].Status)
	}
}

// 10. Concurrent attempts to check in the same ticket
func TestCheckIn_ConcurrentAttempts(t *testing.T) {
	uc, repos, qrSvc, userID, _, eventID := setupTestContext(t)
	ticketID := uuid.New()
	repos.tickets[ticketID] = &domain.Ticket{
		ID:      ticketID,
		EventID: eventID,
		Status:  domain.TicketStatusValid,
	}

	payload, _ := qrSvc.GeneratePayload(ticketID)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	successCount := 0
	conflictCount := 0
	var countMu sync.Mutex

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := uc.CheckIn(context.Background(), userID, eventID, payload)
			countMu.Lock()
			defer countMu.Unlock()
			if err == nil {
				successCount++
			} else if errors.Is(err, domain.ErrTicketNotValid) {
				conflictCount++
			}
		}()
	}

	wg.Wait()

	if successCount != 1 {
		t.Errorf("expected exactly 1 successful check-in, got %d", successCount)
	}
	if conflictCount != goroutines-1 {
		t.Errorf("expected %d conflict rejections, got %d", goroutines-1, conflictCount)
	}
}
