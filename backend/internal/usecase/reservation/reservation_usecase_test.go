package reservation

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEventRepo struct {
	domain.EventRepository
	event *domain.Event
	err   error
}

func (m *mockEventRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.event, nil
}

type mockEventSettingRepo struct {
	domain.EventSettingRepository
	setting *domain.EventSetting
	err     error
}

func (m *mockEventSettingRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSetting, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.setting, nil
}

type mockSeatReservationRepo struct {
	domain.SeatReservationRepository
	createdReservation *domain.SeatReservation
	createdItems       []domain.SeatReservationItem
	itemsByResID       map[uuid.UUID][]domain.SeatReservationItem
	cancelSuccess      bool
	expireSuccess      bool
	err                error
}

func (m *mockSeatReservationRepo) Create(ctx context.Context, reservation *domain.SeatReservation) error {
	if m.err != nil {
		return m.err
	}
	m.createdReservation = reservation
	return nil
}

func (m *mockSeatReservationRepo) CreateItem(ctx context.Context, item *domain.SeatReservationItem) error {
	if m.err != nil {
		return m.err
	}
	m.createdItems = append(m.createdItems, *item)
	if m.itemsByResID == nil {
		m.itemsByResID = make(map[uuid.UUID][]domain.SeatReservationItem)
	}
	m.itemsByResID[item.ReservationID] = append(m.itemsByResID[item.ReservationID], *item)
	return nil
}

func (m *mockSeatReservationRepo) FindItemsByReservationID(ctx context.Context, reservationID uuid.UUID) ([]domain.SeatReservationItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.itemsByResID[reservationID], nil
}

func (m *mockSeatReservationRepo) CancelIfActive(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.cancelSuccess, nil
}

func (m *mockSeatReservationRepo) ExpireIfActive(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.expireSuccess, nil
}

type mockTicketTypeRepo struct {
	domain.TicketTypeRepository
	ticketTypes    map[uuid.UUID]*domain.TicketType
	reservedQty    map[uuid.UUID]int
	releasedQty    map[uuid.UUID]int
	reserveFailFor map[uuid.UUID]bool
	err            error
}

func (m *mockTicketTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	if m.err != nil {
		return nil, m.err
	}
	if tt, ok := m.ticketTypes[id]; ok {
		return tt, nil
	}
	return nil, domain.ErrTicketTypeNotFound
}

func (m *mockTicketTypeRepo) ReserveQuantityIfAvailable(ctx context.Context, ticketTypeID uuid.UUID, quantity int) error {
	if m.reserveFailFor != nil && m.reserveFailFor[ticketTypeID] {
		return domain.ErrInsufficientTicketQuantity
	}
	if m.reservedQty == nil {
		m.reservedQty = make(map[uuid.UUID]int)
	}
	m.reservedQty[ticketTypeID] += quantity
	return nil
}

func (m *mockTicketTypeRepo) ReleaseReservedQuantity(ctx context.Context, ticketTypeID uuid.UUID, quantity int) error {
	if m.releasedQty == nil {
		m.releasedQty = make(map[uuid.UUID]int)
	}
	m.releasedQty[ticketTypeID] += quantity
	return nil
}

type mockCheckoutRepo struct {
	domain.CheckoutRepository
	expiredByResID []uuid.UUID
}

func (m *mockCheckoutRepo) ExpireActiveByReservationID(ctx context.Context, reservationID uuid.UUID) (bool, error) {
	m.expiredByResID = append(m.expiredByResID, reservationID)
	return true, nil
}

type testTxRepos struct {
	domain.TransactionRepositories
	eventRepo        *mockEventRepo
	eventSettingRepo *mockEventSettingRepo
	seatResRepo      *mockSeatReservationRepo
	ticketTypeRepo   *mockTicketTypeRepo
	checkoutRepo     *mockCheckoutRepo
}

func (r *testTxRepos) EventRepository() domain.EventRepository {
	return r.eventRepo
}
func (r *testTxRepos) EventSettingRepository() domain.EventSettingRepository {
	return r.eventSettingRepo
}
func (r *testTxRepos) SeatReservationRepository() domain.SeatReservationRepository {
	return r.seatResRepo
}
func (r *testTxRepos) TicketTypeRepository() domain.TicketTypeRepository {
	return r.ticketTypeRepo
}
func (r *testTxRepos) CheckoutRepository() domain.CheckoutRepository {
	return r.checkoutRepo
}

type testTxManager struct {
	repos *testTxRepos
}

func (m *testTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

func setupReservationUsecase(repos *testTxRepos) *ReservationUsecase {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewReservationUsecase(&testTxManager{repos: repos}, logger)
}

func TestReservationUsecase_CreateReservation_MultipleTiers_Success(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.New()
	userID := uuid.New()
	tier1ID := uuid.New()
	tier2ID := uuid.New()

	event := &domain.Event{
		ID:        eventID,
		EventType: domain.EventTypePhysical,
		Status:    domain.EventStatusPublished,
	}

	setting := &domain.EventSetting{
		EventID:            eventID,
		SeatLayoutType:     SeatLayoutTypeGeneral,
		BookingLimitPerUser: 10,
	}

	ticketTypes := map[uuid.UUID]*domain.TicketType{
		tier1ID: {
			ID:      tier1ID,
			EventID: eventID,
			Name:    "Test Pass 01",
			Price:   decimal.NewFromInt(100),
		},
		tier2ID: {
			ID:      tier2ID,
			EventID: eventID,
			Name:    "Test Pass 2",
			Price:   decimal.NewFromInt(200),
		},
	}

	repos := &testTxRepos{
		eventRepo:        &mockEventRepo{event: event},
		eventSettingRepo: &mockEventSettingRepo{setting: setting},
		seatResRepo:      &mockSeatReservationRepo{},
		ticketTypeRepo:   &mockTicketTypeRepo{ticketTypes: ticketTypes},
		checkoutRepo:     &mockCheckoutRepo{},
	}

	uc := setupReservationUsecase(repos)

	req := CreateReservationRequest{
		UserID:  userID,
		EventID: eventID,
		Tickets: []TicketSelection{
			{TicketTypeID: tier1ID, Quantity: 1},
			{TicketTypeID: tier2ID, Quantity: 2},
		},
	}

	res, err := uc.CreateReservation(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, domain.ReservationStatusActive, res.Status)
	assert.Len(t, res.Items, 2)
	assert.Equal(t, 1, repos.ticketTypeRepo.reservedQty[tier1ID])
	assert.Equal(t, 2, repos.ticketTypeRepo.reservedQty[tier2ID])
}

func TestReservationUsecase_CreateReservation_AggregateBookingLimit_Exceeded(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.New()
	userID := uuid.New()
	tier1ID := uuid.New()
	tier2ID := uuid.New()

	event := &domain.Event{
		ID:        eventID,
		EventType: domain.EventTypePhysical,
		Status:    domain.EventStatusPublished,
	}

	setting := &domain.EventSetting{
		EventID:            eventID,
		SeatLayoutType:     SeatLayoutTypeGeneral,
		BookingLimitPerUser: 4, // Limit is 4
	}

	ticketTypes := map[uuid.UUID]*domain.TicketType{
		tier1ID: {ID: tier1ID, EventID: eventID, Price: decimal.NewFromInt(100)},
		tier2ID: {ID: tier2ID, EventID: eventID, Price: decimal.NewFromInt(200)},
	}

	repos := &testTxRepos{
		eventRepo:        &mockEventRepo{event: event},
		eventSettingRepo: &mockEventSettingRepo{setting: setting},
		seatResRepo:      &mockSeatReservationRepo{},
		ticketTypeRepo:   &mockTicketTypeRepo{ticketTypes: ticketTypes},
		checkoutRepo:     &mockCheckoutRepo{},
	}

	uc := setupReservationUsecase(repos)

	// 3 + 2 = 5 > 4 (booking limit exceeded!)
	req := CreateReservationRequest{
		UserID:  userID,
		EventID: eventID,
		Tickets: []TicketSelection{
			{TicketTypeID: tier1ID, Quantity: 3},
			{TicketTypeID: tier2ID, Quantity: 2},
		},
	}

	_, err := uc.CreateReservation(ctx, req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidReservationRequest))
}

func TestReservationUsecase_CreateReservation_PartialInventoryFailure_Rollback(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.New()
	userID := uuid.New()
	tier1ID := uuid.New()
	tier2ID := uuid.New()

	event := &domain.Event{
		ID:        eventID,
		EventType: domain.EventTypePhysical,
		Status:    domain.EventStatusPublished,
	}

	setting := &domain.EventSetting{
		EventID:            eventID,
		SeatLayoutType:     SeatLayoutTypeGeneral,
		BookingLimitPerUser: 10,
	}

	ticketTypes := map[uuid.UUID]*domain.TicketType{
		tier1ID: {ID: tier1ID, EventID: eventID, Price: decimal.NewFromInt(100)},
		tier2ID: {ID: tier2ID, EventID: eventID, Price: decimal.NewFromInt(200)},
	}

	repos := &testTxRepos{
		eventRepo:        &mockEventRepo{event: event},
		eventSettingRepo: &mockEventSettingRepo{setting: setting},
		seatResRepo:      &mockSeatReservationRepo{},
		ticketTypeRepo: &mockTicketTypeRepo{
			ticketTypes: ticketTypes,
			reserveFailFor: map[uuid.UUID]bool{
				tier2ID: true, // Tier 2 fails inventory
			},
		},
		checkoutRepo: &mockCheckoutRepo{},
	}

	uc := setupReservationUsecase(repos)

	req := CreateReservationRequest{
		UserID:  userID,
		EventID: eventID,
		Tickets: []TicketSelection{
			{TicketTypeID: tier1ID, Quantity: 1},
			{TicketTypeID: tier2ID, Quantity: 2},
		},
	}

	_, err := uc.CreateReservation(ctx, req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInsufficientTicketQuantity))
}

func TestReservationUsecase_CreateReservation_CrossEventMismatch(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.New()
	otherEventID := uuid.New()
	userID := uuid.New()
	tier1ID := uuid.New()

	event := &domain.Event{
		ID:        eventID,
		EventType: domain.EventTypePhysical,
		Status:    domain.EventStatusPublished,
	}

	setting := &domain.EventSetting{
		EventID:            eventID,
		SeatLayoutType:     SeatLayoutTypeGeneral,
		BookingLimitPerUser: 10,
	}

	ticketTypes := map[uuid.UUID]*domain.TicketType{
		tier1ID: {
			ID:      tier1ID,
			EventID: otherEventID, // Belongs to different event!
			Price:   decimal.NewFromInt(100),
		},
	}

	repos := &testTxRepos{
		eventRepo:        &mockEventRepo{event: event},
		eventSettingRepo: &mockEventSettingRepo{setting: setting},
		seatResRepo:      &mockSeatReservationRepo{},
		ticketTypeRepo:   &mockTicketTypeRepo{ticketTypes: ticketTypes},
		checkoutRepo:     &mockCheckoutRepo{},
	}

	uc := setupReservationUsecase(repos)

	req := CreateReservationRequest{
		UserID:  userID,
		EventID: eventID,
		Tickets: []TicketSelection{
			{TicketTypeID: tier1ID, Quantity: 1},
		},
	}

	_, err := uc.CreateReservation(ctx, req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrTicketTypeEventMismatch))
}

func TestReservationUsecase_CreateReservation_LegacySingleCategory_Compatibility(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.New()
	userID := uuid.New()
	tier1ID := uuid.New()

	event := &domain.Event{
		ID:        eventID,
		EventType: domain.EventTypePhysical,
		Status:    domain.EventStatusPublished,
	}

	setting := &domain.EventSetting{
		EventID:            eventID,
		SeatLayoutType:     SeatLayoutTypeGeneral,
		BookingLimitPerUser: 10,
	}

	ticketTypes := map[uuid.UUID]*domain.TicketType{
		tier1ID: {
			ID:      tier1ID,
			EventID: eventID,
			Name:    "Legacy Ticket",
			Price:   decimal.NewFromInt(150),
		},
	}

	repos := &testTxRepos{
		eventRepo:        &mockEventRepo{event: event},
		eventSettingRepo: &mockEventSettingRepo{setting: setting},
		seatResRepo:      &mockSeatReservationRepo{},
		ticketTypeRepo:   &mockTicketTypeRepo{ticketTypes: ticketTypes},
		checkoutRepo:     &mockCheckoutRepo{},
	}

	uc := setupReservationUsecase(repos)

	// Legacy request supplying TicketTypeID and Quantity instead of Tickets array
	req := CreateReservationRequest{
		UserID:       userID,
		EventID:      eventID,
		TicketTypeID: &tier1ID,
		Quantity:     2,
	}

	res, err := uc.CreateReservation(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, domain.ReservationStatusActive, res.Status)
	assert.Len(t, res.Items, 1)
	assert.Equal(t, 2, repos.ticketTypeRepo.reservedQty[tier1ID])
}

func TestReservationUsecase_ExpireReservation_RestoresAllTiers(t *testing.T) {
	ctx := context.Background()
	resID := uuid.New()
	tier1ID := uuid.New()
	tier2ID := uuid.New()

	items := []domain.SeatReservationItem{
		{ReservationID: resID, TicketTypeID: &tier1ID, Quantity: 2},
		{ReservationID: resID, TicketTypeID: &tier2ID, Quantity: 3},
	}

	repos := &testTxRepos{
		seatResRepo: &mockSeatReservationRepo{
			expireSuccess: true,
			itemsByResID: map[uuid.UUID][]domain.SeatReservationItem{
				resID: items,
			},
		},
		ticketTypeRepo: &mockTicketTypeRepo{},
		checkoutRepo:   &mockCheckoutRepo{},
	}

	uc := setupReservationUsecase(repos)

	err := uc.ExpireReservation(ctx, resID)
	require.NoError(t, err)
	assert.Equal(t, 2, repos.ticketTypeRepo.releasedQty[tier1ID])
	assert.Equal(t, 3, repos.ticketTypeRepo.releasedQty[tier2ID])
	assert.Contains(t, repos.checkoutRepo.expiredByResID, resID)
}

func TestReservationUsecase_CancelReservation_RestoresAllTiersAndExpiresCheckout(t *testing.T) {
	ctx := context.Background()
	resID := uuid.New()
	userID := uuid.New()
	tier1ID := uuid.New()
	tier2ID := uuid.New()

	items := []domain.SeatReservationItem{
		{ReservationID: resID, TicketTypeID: &tier1ID, Quantity: 1},
		{ReservationID: resID, TicketTypeID: &tier2ID, Quantity: 4},
	}

	repos := &testTxRepos{
		seatResRepo: &mockSeatReservationRepo{
			cancelSuccess: true,
			itemsByResID: map[uuid.UUID][]domain.SeatReservationItem{
				resID: items,
			},
		},
		ticketTypeRepo: &mockTicketTypeRepo{},
		checkoutRepo:   &mockCheckoutRepo{},
	}

	uc := setupReservationUsecase(repos)

	err := uc.CancelReservation(ctx, resID, userID)
	require.NoError(t, err)
	assert.Equal(t, 1, repos.ticketTypeRepo.releasedQty[tier1ID])
	assert.Equal(t, 4, repos.ticketTypeRepo.releasedQty[tier2ID])
	assert.Contains(t, repos.checkoutRepo.expiredByResID, resID)
}
