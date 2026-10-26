package booking_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	bookinguc "github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBookingRepo struct {
	domain.BookingRepository
	bookingsByReservation map[uuid.UUID]*domain.Booking
	bookingsByID          map[uuid.UUID]*domain.Booking
	createErr             error
	createdBookings       []*domain.Booking
	createdItems          [][]domain.BookingItem
}

func newMockBookingRepo() *mockBookingRepo {
	return &mockBookingRepo{
		bookingsByReservation: make(map[uuid.UUID]*domain.Booking),
		bookingsByID:          make(map[uuid.UUID]*domain.Booking),
	}
}

func (m *mockBookingRepo) Create(ctx context.Context, booking *domain.Booking, items []domain.BookingItem) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createdBookings = append(m.createdBookings, booking)
	m.createdItems = append(m.createdItems, items)
	m.bookingsByID[booking.ID] = booking
	if booking.ReservationID != nil {
		m.bookingsByReservation[*booking.ReservationID] = booking
	}
	return nil
}

func (m *mockBookingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	if b, ok := m.bookingsByID[id]; ok {
		return b, nil
	}
	return nil, domain.ErrBookingNotFound
}

func (m *mockBookingRepo) FindByBookingNumber(ctx context.Context, bookingNumber string) (*domain.Booking, error) {
	for _, b := range m.bookingsByID {
		if b.BookingNumber == bookingNumber {
			return b, nil
		}
	}
	return nil, domain.ErrBookingNotFound
}

func (m *mockBookingRepo) FindByReservationID(ctx context.Context, reservationID uuid.UUID) (*domain.Booking, error) {
	if b, ok := m.bookingsByReservation[reservationID]; ok {
		return b, nil
	}
	return nil, domain.ErrBookingNotFound
}

func (m *mockBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if b, ok := m.bookingsByID[id]; ok {
		b.Status = status
		return nil
	}
	return domain.ErrBookingNotFound
}

func (m *mockBookingRepo) ConfirmPending(ctx context.Context, id uuid.UUID) error {
	if b, ok := m.bookingsByID[id]; ok {
		if b.Status == domain.BookingStatusPending {
			b.Status = domain.BookingStatusConfirmed
			return nil
		}
	}
	return domain.ErrBookingNotFound
}

func (m *mockBookingRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	var result []domain.Booking
	for _, b := range m.bookingsByID {
		if b.UserID == userID {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (m *mockBookingRepo) FindWithItemsByID(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
	b, ok := m.bookingsByID[id]
	if !ok {
		return nil, nil, domain.ErrBookingNotFound
	}
	var items []domain.BookingItem
	for _, itemList := range m.createdItems {
		for _, item := range itemList {
			if item.BookingID == id {
				items = append(items, item)
			}
		}
	}
	return b, items, nil
}

func (m *mockBookingRepo) FindItemsByBookingIDs(ctx context.Context, bookingIDs []uuid.UUID) (map[uuid.UUID][]domain.BookingItem, error) {
	result := make(map[uuid.UUID][]domain.BookingItem)
	for _, id := range bookingIDs {
		for _, itemList := range m.createdItems {
			for _, item := range itemList {
				if item.BookingID == id {
					result[id] = append(result[id], item)
				}
			}
		}
	}
	return result, nil
}

type mockCheckoutRepo struct {
	domain.CheckoutRepository
	checkouts map[uuid.UUID]*domain.Checkout
}

func newMockCheckoutRepo() *mockCheckoutRepo {
	return &mockCheckoutRepo{
		checkouts: make(map[uuid.UUID]*domain.Checkout),
	}
}

func (m *mockCheckoutRepo) Create(ctx context.Context, checkout *domain.Checkout) error {
	m.checkouts[checkout.ID] = checkout
	return nil
}

func (m *mockCheckoutRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Checkout, error) {
	if c, ok := m.checkouts[id]; ok {
		return c, nil
	}
	return nil, domain.ErrCheckoutNotFound
}

func (m *mockCheckoutRepo) FindByReservationID(ctx context.Context, reservationID uuid.UUID) (*domain.Checkout, error) {
	for _, c := range m.checkouts {
		if c.ReservationID == reservationID {
			return c, nil
		}
	}
	return nil, domain.ErrCheckoutNotFound
}

func (m *mockCheckoutRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if c, ok := m.checkouts[id]; ok {
		c.Status = status
		return nil
	}
	return domain.ErrCheckoutNotFound
}

func (m *mockCheckoutRepo) ExpireActiveByReservationID(ctx context.Context, resID uuid.UUID) (bool, error) {
	for _, c := range m.checkouts {
		if c.ReservationID == resID && c.Status == domain.CheckoutStatusActive {
			c.Status = domain.CheckoutStatusExpired
			return true, nil
		}
	}
	return false, nil
}

type mockSeatReservationRepo struct {
	domain.SeatReservationRepository
	reservations map[uuid.UUID]*domain.SeatReservation
	items        map[uuid.UUID][]domain.SeatReservationItem
}

func newMockSeatReservationRepo() *mockSeatReservationRepo {
	return &mockSeatReservationRepo{
		reservations: make(map[uuid.UUID]*domain.SeatReservation),
		items:        make(map[uuid.UUID][]domain.SeatReservationItem),
	}
}

func (m *mockSeatReservationRepo) Create(ctx context.Context, reservation *domain.SeatReservation) error {
	m.reservations[reservation.ID] = reservation
	return nil
}

func (m *mockSeatReservationRepo) CreateItem(ctx context.Context, item *domain.SeatReservationItem) error {
	m.items[item.ReservationID] = append(m.items[item.ReservationID], *item)
	return nil
}

func (m *mockSeatReservationRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	if r, ok := m.reservations[id]; ok {
		return r, nil
	}
	return nil, domain.ErrSeatReservationNotFound
}

func (m *mockSeatReservationRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	return m.FindByID(ctx, id)
}

func (m *mockSeatReservationRepo) FindItemsByReservationID(ctx context.Context, reservationID uuid.UUID) ([]domain.SeatReservationItem, error) {
	return m.items[reservationID], nil
}

func (m *mockSeatReservationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if r, ok := m.reservations[id]; ok {
		r.Status = status
		return nil
	}
	return domain.ErrSeatReservationNotFound
}

type mockSeatRepo struct {
	domain.SeatRepository
	details map[uuid.UUID]domain.SeatDetail
}

func newMockSeatRepo() *mockSeatRepo {
	return &mockSeatRepo{
		details: make(map[uuid.UUID]domain.SeatDetail),
	}
}

func (m *mockSeatRepo) FindDetailsBySeatIDs(ctx context.Context, seatIDs []uuid.UUID) (map[uuid.UUID]domain.SeatDetail, error) {
	res := make(map[uuid.UUID]domain.SeatDetail)
	for _, id := range seatIDs {
		if d, ok := m.details[id]; ok {
			res[id] = d
		}
	}
	return res, nil
}

type mockUserRepo struct {
	domain.UserRepository
	users map[uuid.UUID]*domain.User
}

func (m *mockUserRepo) FindByID(id uuid.UUID) (*domain.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

type mockEventRepo struct {
	domain.EventRepository
	events map[uuid.UUID]*domain.Event
}

func (m *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if e, ok := m.events[id]; ok {
		return e, nil
	}
	return nil, domain.ErrEventNotFound
}

type mockScheduleRepo struct {
	domain.EventScheduleRepository
	schedules map[uuid.UUID]*domain.EventSchedule
}

func (m *mockScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	if s, ok := m.schedules[eventID]; ok {
		return s, nil
	}
	return nil, domain.ErrEventNotFound
}

type mockVenueRepo struct {
	domain.VenueRepository
	venues map[uuid.UUID]*domain.Venue
}

func (m *mockVenueRepo) FindByID(id uuid.UUID) (*domain.Venue, error) {
	if v, ok := m.venues[id]; ok {
		return v, nil
	}
	return nil, domain.ErrVenueNotFound
}

type mockTicketRepo struct {
	domain.TicketRepository
	ticketsByBooking map[uuid.UUID][]domain.Ticket
}

func (m *mockTicketRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	return m.ticketsByBooking[bookingID], nil
}

type mockTicketTypeRepo struct {
	domain.TicketTypeRepository
	types map[uuid.UUID]*domain.TicketType
}

func (m *mockTicketTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	if t, ok := m.types[id]; ok {
		return t, nil
	}
	return nil, domain.ErrTicketTypeNotFound
}

type mockTxRepos struct {
	domain.TransactionRepositories
	bookingRepo         *mockBookingRepo
	checkoutRepo        *mockCheckoutRepo
	seatReservationRepo *mockSeatReservationRepo
	seatRepo            *mockSeatRepo
	userRepo            *mockUserRepo
	eventRepo           *mockEventRepo
	scheduleRepo        *mockScheduleRepo
	venueRepo           *mockVenueRepo
	ticketRepo          *mockTicketRepo
	ticketTypeRepo      *mockTicketTypeRepo
}

func (m *mockTxRepos) SeatReservationRepository() domain.SeatReservationRepository {
	return m.seatReservationRepo
}
func (m *mockTxRepos) CheckoutRepository() domain.CheckoutRepository {
	return m.checkoutRepo
}
func (m *mockTxRepos) BookingRepository() domain.BookingRepository {
	return m.bookingRepo
}
func (m *mockTxRepos) SeatRepository() domain.SeatRepository {
	return m.seatRepo
}
func (m *mockTxRepos) UserRepository() domain.UserRepository {
	return m.userRepo
}
func (m *mockTxRepos) EventRepository() domain.EventRepository {
	return m.eventRepo
}
func (m *mockTxRepos) EventScheduleRepository() domain.EventScheduleRepository {
	return m.scheduleRepo
}
func (m *mockTxRepos) VenueRepository() domain.VenueRepository {
	return m.venueRepo
}
func (m *mockTxRepos) TicketRepository() domain.TicketRepository {
	return m.ticketRepo
}
func (m *mockTxRepos) TicketTypeRepository() domain.TicketTypeRepository {
	return m.ticketTypeRepo
}

type mockTxManager struct {
	txRepos *mockTxRepos
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.txRepos)
}

func setupBookingUsecase() (*bookinguc.BookingUsecase, *mockTxRepos) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txRepos := &mockTxRepos{
		bookingRepo:         newMockBookingRepo(),
		checkoutRepo:        newMockCheckoutRepo(),
		seatReservationRepo: newMockSeatReservationRepo(),
		seatRepo:            newMockSeatRepo(),
		userRepo:            &mockUserRepo{users: make(map[uuid.UUID]*domain.User)},
		eventRepo:           &mockEventRepo{events: make(map[uuid.UUID]*domain.Event)},
		scheduleRepo:        &mockScheduleRepo{schedules: make(map[uuid.UUID]*domain.EventSchedule)},
		venueRepo:           &mockVenueRepo{venues: make(map[uuid.UUID]*domain.Venue)},
		ticketRepo:          &mockTicketRepo{ticketsByBooking: make(map[uuid.UUID][]domain.Ticket)},
		ticketTypeRepo:      &mockTicketTypeRepo{types: make(map[uuid.UUID]*domain.TicketType)},
	}
	txManager := &mockTxManager{txRepos: txRepos}
	uc := bookinguc.NewBookingUsecase(txManager, logger)
	return uc, txRepos
}

func TestBookingUsecase_CreateBooking_Success(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	reservationID := uuid.New()
	seatID := uuid.New()
	now := time.Now()

	repos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:        reservationID,
		UserID:    userID,
		EventID:   eventID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(15 * time.Minute),
	}

	repos.seatReservationRepo.items[reservationID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: reservationID,
			SeatID:        &seatID,
			Quantity:      1,
			Price:         decimal.NewFromInt(500),
		},
	}

	checkoutID := uuid.New()
	repos.checkoutRepo.checkouts[checkoutID] = &domain.Checkout{
		ID:             checkoutID,
		ReservationID:  reservationID,
		UserID:         userID,
		Subtotal:       decimal.NewFromInt(500),
		PlatformFee:    decimal.NewFromInt(25),
		TaxAmount:      decimal.NewFromInt(90),
		DiscountAmount: decimal.Zero,
		TotalAmount:    decimal.NewFromInt(615),
		Status:         domain.CheckoutStatusActive,
		ExpiresAt:      now.Add(15 * time.Minute),
	}

	req := bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	}

	booking, err := uc.CreateBooking(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, booking)

	// Verify consistent monetary amounts transferred from checkout snapshot to booking
	assert.Equal(t, userID, booking.UserID)
	assert.Equal(t, eventID, booking.EventID)
	assert.Equal(t, reservationID, *booking.ReservationID)
	assert.True(t, booking.Subtotal.Equal(decimal.NewFromInt(500)))
	assert.True(t, booking.PlatformFee.Equal(decimal.NewFromInt(25)))
	assert.True(t, booking.Tax.Equal(decimal.NewFromInt(90)), "Booking.Tax must equal Checkout.TaxAmount")
	assert.True(t, booking.Discount.Equal(decimal.Zero))
	assert.True(t, booking.Total.Equal(decimal.NewFromInt(615)), "Booking.Total must equal Checkout.TotalAmount")
	assert.Equal(t, domain.BookingStatusPending, booking.Status)

	// Verify repository interactions
	require.Len(t, repos.bookingRepo.createdBookings, 1)
	require.Len(t, repos.bookingRepo.createdItems, 1)
	assert.Equal(t, booking.ID, repos.bookingRepo.createdBookings[0].ID)
	assert.Len(t, repos.bookingRepo.createdItems[0], 1)
}

func TestBookingUsecase_CreateBooking_Idempotency_ReturnsExisting(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	reservationID := uuid.New()
	checkoutID := uuid.New()
	now := time.Now()

	existingBooking := &domain.Booking{
		ID:            uuid.New(),
		BookingNumber: "EVH-EXISTING123",
		UserID:        userID,
		EventID:       eventID,
		ReservationID: &reservationID,
		Subtotal:      decimal.NewFromInt(500),
		PlatformFee:   decimal.Zero,
		Tax:           decimal.Zero,
		Discount:      decimal.Zero,
		Total:         decimal.NewFromInt(500),
		Status:        domain.BookingStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	repos.bookingRepo.bookingsByReservation[reservationID] = existingBooking
	repos.bookingRepo.bookingsByID[existingBooking.ID] = existingBooking

	repos.checkoutRepo.checkouts[checkoutID] = &domain.Checkout{
		ID:            checkoutID,
		ReservationID: reservationID,
		UserID:        userID,
		Status:        domain.CheckoutStatusActive,
		ExpiresAt:     now.Add(10 * time.Minute),
	}

	booking, err := uc.CreateBooking(ctx, bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	})

	require.NoError(t, err)
	assert.Equal(t, existingBooking.ID, booking.ID)
	// No new booking created
	assert.Empty(t, repos.bookingRepo.createdBookings)
}

func TestBookingUsecase_CreateBooking_ExpiredCheckout_Rejected(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	reservationID := uuid.New()
	checkoutID := uuid.New()

	repos.checkoutRepo.checkouts[checkoutID] = &domain.Checkout{
		ID:            checkoutID,
		ReservationID: reservationID,
		UserID:        userID,
		Status:        domain.CheckoutStatusActive,
		ExpiresAt:     time.Now().Add(-5 * time.Minute), // expired
	}

	_, err := uc.CreateBooking(ctx, bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidReservationRequest)
}

func TestBookingUsecase_CreateBooking_RepoError_RollsBack(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	reservationID := uuid.New()
	now := time.Now()

	repos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:        reservationID,
		UserID:    userID,
		EventID:   eventID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(10 * time.Minute),
	}
	repos.seatReservationRepo.items[reservationID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: reservationID,
			Quantity:      1,
			Price:         decimal.NewFromInt(100),
		},
	}

	checkoutID := uuid.New()
	repos.checkoutRepo.checkouts[checkoutID] = &domain.Checkout{
		ID:            checkoutID,
		ReservationID: reservationID,
		UserID:        userID,
		Subtotal:      decimal.NewFromInt(100),
		TotalAmount:   decimal.NewFromInt(100),
		Status:        domain.CheckoutStatusActive,
		ExpiresAt:     now.Add(10 * time.Minute),
	}

	repos.bookingRepo.createErr = errors.New("db insert failure")

	_, err := uc.CreateBooking(ctx, bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	})

	assert.EqualError(t, err, "db insert failure")
}

func TestBookingUsecase_GetMyBookings_Success(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	b1 := &domain.Booking{
		ID:            uuid.New(),
		BookingNumber: "EVH-111111111111",
		UserID:        userID,
		Total:         decimal.NewFromInt(150),
		Status:        domain.BookingStatusConfirmed,
	}
	b2 := &domain.Booking{
		ID:            uuid.New(),
		BookingNumber: "EVH-222222222222",
		UserID:        userID,
		Total:         decimal.NewFromInt(300),
		Status:        domain.BookingStatusPending,
	}
	bOther := &domain.Booking{
		ID:            uuid.New(),
		BookingNumber: "EVH-333333333333",
		UserID:        uuid.New(),
		Total:         decimal.NewFromInt(500),
		Status:        domain.BookingStatusConfirmed,
	}

	repos.bookingRepo.bookingsByID[b1.ID] = b1
	repos.bookingRepo.bookingsByID[b2.ID] = b2
	repos.bookingRepo.bookingsByID[bOther.ID] = bOther

	bookings, err := uc.GetMyBookings(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, bookings, 2)
}

func TestBookingUsecase_GetMyBookings_Empty(t *testing.T) {
	uc, _ := setupBookingUsecase()
	ctx := context.Background()

	bookings, err := uc.GetMyBookings(ctx, uuid.New())
	require.NoError(t, err)
	assert.NotNil(t, bookings)
	assert.Empty(t, bookings)
}

func TestBookingUsecase_GetMyBookings_NilUserID(t *testing.T) {
	uc, _ := setupBookingUsecase()
	ctx := context.Background()

	_, err := uc.GetMyBookings(ctx, uuid.Nil)
	assert.ErrorIs(t, err, domain.ErrBookingNotFound)
}

func TestBookingUsecase_GetBookingByID_Success(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	bookingID := uuid.New()
	booking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "EVH-BOOKING-001",
		UserID:        userID,
		Subtotal:      decimal.NewFromInt(200),
		PlatformFee:   decimal.Zero,
		Tax:           decimal.Zero,
		Discount:      decimal.Zero,
		Total:         decimal.NewFromInt(200),
		Status:        domain.BookingStatusConfirmed,
	}
	items := []domain.BookingItem{
		{
			ID:         uuid.New(),
			BookingID:  bookingID,
			Quantity:   2,
			UnitPrice:  decimal.NewFromInt(100),
			TotalPrice: decimal.NewFromInt(200),
		},
	}

	repos.bookingRepo.bookingsByID[bookingID] = booking
	repos.bookingRepo.createdItems = append(repos.bookingRepo.createdItems, items)

	result, err := uc.GetBookingByID(ctx, userID, bookingID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, bookingID, result.ID)
	assert.Equal(t, "EVH-BOOKING-001", result.BookingNumber)
	assert.True(t, decimal.NewFromInt(200).Equal(result.Total))
	assert.Len(t, result.Items, 1)
	assert.Equal(t, 2, result.Items[0].Quantity)
}

func TestBookingUsecase_GetBookingByID_NotFound(t *testing.T) {
	uc, _ := setupBookingUsecase()
	ctx := context.Background()

	_, err := uc.GetBookingByID(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrBookingNotFound)
}

func TestBookingUsecase_GetBookingByID_CrossCustomerAccess_FailsClosed(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	ownerID := uuid.New()
	callerID := uuid.New()
	bookingID := uuid.New()

	booking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "EVH-PRIVATE-001",
		UserID:        ownerID,
		Status:        domain.BookingStatusConfirmed,
	}
	repos.bookingRepo.bookingsByID[bookingID] = booking

	_, err := uc.GetBookingByID(ctx, callerID, bookingID)
	assert.ErrorIs(t, err, domain.ErrBookingNotFound)
}

func TestBookingUsecase_GetBookingByID_NilIDs(t *testing.T) {
	uc, _ := setupBookingUsecase()
	ctx := context.Background()

	_, err := uc.GetBookingByID(ctx, uuid.Nil, uuid.New())
	assert.ErrorIs(t, err, domain.ErrBookingNotFound)

	_, err = uc.GetBookingByID(ctx, uuid.New(), uuid.Nil)
	assert.ErrorIs(t, err, domain.ErrBookingNotFound)
}

func TestBookingUsecase_GetBookingByID_WithAssignedSeats(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	bookingID := uuid.New()
	seatID1 := uuid.New()
	seatID2 := uuid.New()

	booking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "EVH-SEATS-001",
		UserID:        userID,
		Status:        domain.BookingStatusConfirmed,
	}
	items := []domain.BookingItem{
		{
			ID:        uuid.New(),
			BookingID: bookingID,
			SeatID:    &seatID1,
			Quantity:  1,
		},
		{
			ID:        uuid.New(),
			BookingID: bookingID,
			SeatID:    &seatID2,
			Quantity:  1,
		},
	}
	repos.bookingRepo.bookingsByID[bookingID] = booking
	repos.bookingRepo.createdItems = append(repos.bookingRepo.createdItems, items)

	repos.seatRepo.details[seatID1] = domain.SeatDetail{
		SeatID:      seatID1,
		SeatNumber:  1,
		RowName:     "A",
		SectionName: "VIP",
		Label:       "VIP • Row A, Seat 1",
	}
	repos.seatRepo.details[seatID2] = domain.SeatDetail{
		SeatID:      seatID2,
		SeatNumber:  2,
		RowName:     "A",
		SectionName: "VIP",
		Label:       "VIP • Row A, Seat 2",
	}

	result, err := uc.GetBookingByID(ctx, userID, bookingID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Seats, 2)
	assert.Equal(t, "VIP • Row A, Seat 1", result.Seats[0].Label)
	assert.Equal(t, "VIP • Row A, Seat 2", result.Seats[1].Label)
	require.NotNil(t, result.Items[0].Seat)
	assert.Equal(t, 1, result.Items[0].Seat.SeatNumber)
}

func TestBookingUsecase_GetBookingByID_GeneralAdmission(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	bookingID := uuid.New()
	ticketTypeID := uuid.New()

	booking := &domain.Booking{
		ID:            bookingID,
		BookingNumber: "EVH-GA-001",
		UserID:        userID,
		Status:        domain.BookingStatusConfirmed,
	}
	items := []domain.BookingItem{
		{
			ID:           uuid.New(),
			BookingID:    bookingID,
			SeatID:       nil, // GA has no seat ID
			TicketTypeID: &ticketTypeID,
			Quantity:     3,
		},
	}
	repos.bookingRepo.bookingsByID[bookingID] = booking
	repos.bookingRepo.createdItems = append(repos.bookingRepo.createdItems, items)

	result, err := uc.GetBookingByID(ctx, userID, bookingID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Seats, "GA bookings must have empty seats list")
	assert.Nil(t, result.Items[0].Seat, "GA booking items must have nil Seat")
}

func TestBookingUsecase_GetMyBookings_WithSeatsAndGA(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	bookingID1 := uuid.New()
	bookingID2 := uuid.New()
	seatID1 := uuid.New()

	booking1 := domain.Booking{
		ID:            bookingID1,
		BookingNumber: "EVH-RESERVED",
		UserID:        userID,
		Status:        domain.BookingStatusConfirmed,
	}
	booking2 := domain.Booking{
		ID:            bookingID2,
		BookingNumber: "EVH-GA",
		UserID:        userID,
		Status:        domain.BookingStatusConfirmed,
	}
	repos.bookingRepo.bookingsByID[bookingID1] = &booking1
	repos.bookingRepo.bookingsByID[bookingID2] = &booking2

	repos.bookingRepo.createdItems = append(repos.bookingRepo.createdItems,
		[]domain.BookingItem{
			{ID: uuid.New(), BookingID: bookingID1, SeatID: &seatID1, Quantity: 1},
		},
		[]domain.BookingItem{
			{ID: uuid.New(), BookingID: bookingID2, SeatID: nil, Quantity: 2},
		},
	)

	repos.seatRepo.details[seatID1] = domain.SeatDetail{
		SeatID:      seatID1,
		SeatNumber:  5,
		RowName:     "B",
		SectionName: "Balcony",
		Label:       "Balcony • Row B, Seat 5",
	}

	bookings, err := uc.GetMyBookings(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, bookings, 2)

	for _, b := range bookings {
		if b.ID == bookingID1 {
			require.Len(t, b.Seats, 1)
			assert.Equal(t, "Balcony • Row B, Seat 5", b.Seats[0].Label)
		} else if b.ID == bookingID2 {
			assert.Empty(t, b.Seats)
		}
	}
}

func TestBookingUsecase_GetBookingPDFData_Success_MultipleSeats(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	venueID := uuid.New()
	bookingID := uuid.New()
	seatID1 := uuid.New()
	seatID2 := uuid.New()
	typeID := uuid.New()

	repos.userRepo.users[userID] = &domain.User{
		ID:       userID,
		FullName: "Alice Johnson",
	}
	repos.eventRepo.events[eventID] = &domain.Event{
		ID:        eventID,
		Title:     "Symphony Night",
		EventType: "IN_PERSON",
		VenueID:   &venueID,
	}
	repos.venueRepo.venues[venueID] = &domain.Venue{
		ID:      venueID,
		Name:    "Concert Hall",
		City:    "Mumbai",
		Address: "Marine Drive",
	}
	repos.scheduleRepo.schedules[eventID] = &domain.EventSchedule{
		EventID:   eventID,
		EventDate: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
		StartTime: time.Date(0, 1, 1, 19, 30, 0, 0, time.UTC),
	}
	repos.ticketTypeRepo.types[typeID] = &domain.TicketType{
		ID:   typeID,
		Name: "Orchestra Premium",
	}

	repos.bookingRepo.bookingsByID[bookingID] = &domain.Booking{
		ID:            bookingID,
		BookingNumber: "EVH-ORCH-001",
		UserID:        userID,
		EventID:       eventID,
		Status:        domain.BookingStatusConfirmed,
	}
	items := []domain.BookingItem{
		{ID: uuid.New(), BookingID: bookingID, SeatID: &seatID1, TicketTypeID: &typeID, Quantity: 1},
		{ID: uuid.New(), BookingID: bookingID, SeatID: &seatID2, TicketTypeID: &typeID, Quantity: 1},
	}
	repos.bookingRepo.createdItems = append(repos.bookingRepo.createdItems, items)

	repos.seatRepo.details[seatID1] = domain.SeatDetail{
		SeatID:      seatID1,
		SeatNumber:  1,
		RowName:     "O",
		SectionName: "Orchestra",
		Label:       "Orchestra • Row O, Seat 1",
	}
	repos.seatRepo.details[seatID2] = domain.SeatDetail{
		SeatID:      seatID2,
		SeatNumber:  2,
		RowName:     "O",
		SectionName: "Orchestra",
		Label:       "Orchestra • Row O, Seat 2",
	}

	pdfData, err := uc.GetBookingPDFData(ctx, userID, bookingID)
	require.NoError(t, err)
	require.NotNil(t, pdfData)
	assert.Equal(t, bookingID, pdfData.BookingID)
	assert.Equal(t, "EVH-ORCH-001", pdfData.BookingNumber)
	assert.Equal(t, "Symphony Night", pdfData.EventName)
	assert.Equal(t, "Alice Johnson", pdfData.CustomerName)
	assert.Equal(t, "Orchestra Premium", pdfData.TicketType)
	assert.Equal(t, 2, pdfData.TicketCount)
	assert.Equal(t, []string{"Orchestra • Row O, Seat 1", "Orchestra • Row O, Seat 2"}, pdfData.Seats)
	assert.Equal(t, "Concert Hall", pdfData.VenueName)
}

func TestBookingUsecase_GetBookingPDFData_CrossCustomer_FailsClosed(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	ownerID := uuid.New()
	callerID := uuid.New()
	bookingID := uuid.New()

	repos.bookingRepo.bookingsByID[bookingID] = &domain.Booking{
		ID:            bookingID,
		BookingNumber: "EVH-PRIVATE-002",
		UserID:        ownerID,
		Status:        domain.BookingStatusConfirmed,
	}

	_, err := uc.GetBookingPDFData(ctx, callerID, bookingID)
	assert.ErrorIs(t, err, domain.ErrBookingNotFound)
}
