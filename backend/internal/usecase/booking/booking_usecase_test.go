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

func (m *mockBookingRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return m.FindByID(ctx, id)
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

func (m *mockBookingRepo) FindItemsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.BookingItem, error) {
	var items []domain.BookingItem
	for _, itemList := range m.createdItems {
		for _, item := range itemList {
			if item.BookingID == bookingID {
				items = append(items, item)
			}
		}
	}
	return items, nil
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
	details       map[uuid.UUID]domain.SeatDetail
	releasedSeats []uuid.UUID
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

func (m *mockSeatRepo) ReleaseBookedSeat(ctx context.Context, seatID uuid.UUID) error {
	m.releasedSeats = append(m.releasedSeats, seatID)
	return nil
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
	updatedStatuses  map[uuid.UUID]string
}

func (m *mockTicketRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	return m.ticketsByBooking[bookingID], nil
}

func (m *mockTicketRepo) UpdateStatusByBookingID(ctx context.Context, bookingID uuid.UUID, status string) error {
	if m.updatedStatuses == nil {
		m.updatedStatuses = make(map[uuid.UUID]string)
	}
	m.updatedStatuses[bookingID] = status
	if tickets, ok := m.ticketsByBooking[bookingID]; ok {
		for i := range tickets {
			tickets[i].Status = status
		}
	}
	return nil
}

type mockTicketTypeRepo struct {
	domain.TicketTypeRepository
	types            map[uuid.UUID]*domain.TicketType
	releasedQuantity map[uuid.UUID]int
}

func (m *mockTicketTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	if t, ok := m.types[id]; ok {
		return t, nil
	}
	return nil, domain.ErrTicketTypeNotFound
}

func (m *mockTicketTypeRepo) ReleaseReservedQuantity(ctx context.Context, ticketTypeID uuid.UUID, quantity int) error {
	if m.releasedQuantity == nil {
		m.releasedQuantity = make(map[uuid.UUID]int)
	}
	m.releasedQuantity[ticketTypeID] += quantity
	if tt, ok := m.types[ticketTypeID]; ok {
		tt.AvailableQuantity += quantity
	}
	return nil
}

type mockEventCancellationRepo struct {
	domain.EventCancellationRepository
	cancellations map[uuid.UUID]*domain.EventCancellation
	findErr       error
}

func newMockEventCancellationRepo() *mockEventCancellationRepo {
	return &mockEventCancellationRepo{
		cancellations: make(map[uuid.UUID]*domain.EventCancellation),
	}
}

func (m *mockEventCancellationRepo) FindByEventID(eventID uuid.UUID) (*domain.EventCancellation, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if c, ok := m.cancellations[eventID]; ok {
		return c, nil
	}
	return nil, domain.ErrEventNotFound
}

type mockTxRepos struct {
	domain.TransactionRepositories
	bookingRepo           *mockBookingRepo
	checkoutRepo          *mockCheckoutRepo
	seatReservationRepo   *mockSeatReservationRepo
	seatRepo              *mockSeatRepo
	userRepo              *mockUserRepo
	eventRepo             *mockEventRepo
	scheduleRepo          *mockScheduleRepo
	venueRepo             *mockVenueRepo
	ticketRepo            *mockTicketRepo
	ticketTypeRepo        *mockTicketTypeRepo
	eventCancellationRepo *mockEventCancellationRepo
	paymentRepo           *mockPaymentRepo
	refundRepo            *mockRefundRepo
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
func (m *mockTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return m.eventCancellationRepo
}
func (m *mockTxRepos) PaymentRepository() domain.PaymentRepository {
	return m.paymentRepo
}
func (m *mockTxRepos) RefundRepository() domain.RefundRepository {
	return m.refundRepo
}

type mockPaymentRepo struct {
	domain.PaymentRepository
	paymentsByBooking map[string][]domain.Payment
}

func (m *mockPaymentRepo) FindByBookingID(ctx context.Context, bookingID string) ([]domain.Payment, error) {
	if m.paymentsByBooking != nil {
		return m.paymentsByBooking[bookingID], nil
	}
	return nil, nil
}

type mockRefundRepo struct {
	domain.RefundRepository
	refundsByID      map[uuid.UUID]*domain.Refund
	refundsByBooking map[uuid.UUID][]domain.Refund
	createCalls      int
	updateCalls      int
}

func newMockRefundRepo() *mockRefundRepo {
	return &mockRefundRepo{
		refundsByID:      make(map[uuid.UUID]*domain.Refund),
		refundsByBooking: make(map[uuid.UUID][]domain.Refund),
	}
}

func (m *mockRefundRepo) Create(ctx context.Context, refund *domain.Refund) error {
	m.createCalls++
	copied := *refund
	m.refundsByID[refund.ID] = &copied
	m.refundsByBooking[refund.BookingID] = append(m.refundsByBooking[refund.BookingID], copied)
	return nil
}

func (m *mockRefundRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Refund, error) {
	if r, ok := m.refundsByID[id]; ok {
		copied := *r
		return &copied, nil
	}
	return nil, domain.ErrRefundNotFound
}

func (m *mockRefundRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Refund, error) {
	return m.FindByID(ctx, id)
}

func (m *mockRefundRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Refund, error) {
	return m.refundsByBooking[bookingID], nil
}

func (m *mockRefundRepo) Update(ctx context.Context, refund *domain.Refund) error {
	m.updateCalls++
	copied := *refund
	m.refundsByID[refund.ID] = &copied
	list := m.refundsByBooking[refund.BookingID]
	for i := range list {
		if list[i].ID == refund.ID {
			list[i] = copied
			break
		}
	}
	return nil
}

type mockPaymentRefundProvider struct {
	refundResult       *domain.RefundResult
	refundErr          error
	refundCalls        int
	lastPaymentID      string
	lastAmount         decimal.Decimal
	lastIdempotencyKey string
}

func (m *mockPaymentRefundProvider) Refund(ctx context.Context, gatewayPaymentID string, amount decimal.Decimal, currency string, idempotencyKey string) (*domain.RefundResult, error) {
	m.refundCalls++
	m.lastPaymentID = gatewayPaymentID
	m.lastAmount = amount
	m.lastIdempotencyKey = idempotencyKey
	if m.refundErr != nil {
		return nil, m.refundErr
	}
	if m.refundResult != nil {
		return m.refundResult, nil
	}
	return &domain.RefundResult{
		RefundID: "rfnd_" + uuid.NewString()[:8],
		Amount:   amount,
		Status:   domain.RefundStatusSuccess,
	}, nil
}

type mockTxManager struct {
	txRepos *mockTxRepos
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.txRepos)
}

func setupBookingUsecaseWithRefund() (*bookinguc.BookingUsecase, *mockTxRepos, *mockPaymentRefundProvider) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	refundProvider := &mockPaymentRefundProvider{}
	txRepos := &mockTxRepos{
		bookingRepo:           newMockBookingRepo(),
		checkoutRepo:          newMockCheckoutRepo(),
		seatReservationRepo:   newMockSeatReservationRepo(),
		seatRepo:              newMockSeatRepo(),
		userRepo:              &mockUserRepo{users: make(map[uuid.UUID]*domain.User)},
		eventRepo:             &mockEventRepo{events: make(map[uuid.UUID]*domain.Event)},
		scheduleRepo:          &mockScheduleRepo{schedules: make(map[uuid.UUID]*domain.EventSchedule)},
		venueRepo:             &mockVenueRepo{venues: make(map[uuid.UUID]*domain.Venue)},
		ticketRepo:            &mockTicketRepo{ticketsByBooking: make(map[uuid.UUID][]domain.Ticket)},
		ticketTypeRepo:        &mockTicketTypeRepo{types: make(map[uuid.UUID]*domain.TicketType)},
		eventCancellationRepo: newMockEventCancellationRepo(),
		paymentRepo:           &mockPaymentRepo{paymentsByBooking: make(map[string][]domain.Payment)},
		refundRepo:            newMockRefundRepo(),
	}
	txManager := &mockTxManager{txRepos: txRepos}
	uc := bookinguc.NewBookingUsecase(txManager, refundProvider, logger)
	return uc, txRepos, refundProvider
}

func setupBookingUsecase() (*bookinguc.BookingUsecase, *mockTxRepos) {
	uc, repos, _ := setupBookingUsecaseWithRefund()
	return uc, repos
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

func createTestReservationAndCheckout(
	repos *mockTxRepos,
	userID, eventID uuid.UUID,
	now time.Time,
) (uuid.UUID, uuid.UUID) {
	reservationID := uuid.New()
	checkoutID := uuid.New()
	seatID := uuid.New()

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

	return reservationID, checkoutID
}

func TestBookingUsecase_CreateBooking_Snapshot_FullRefund(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	_, checkoutID := createTestReservationAndCheckout(repos, userID, eventID, now)

	repos.eventCancellationRepo.cancellations[eventID] = &domain.EventCancellation{
		ID:                        uuid.New(),
		EventID:                   eventID,
		CancellationAllowed:       true,
		CancellationDeadlineHours: 24,
		RefundPolicy:              domain.RefundPolicyFull,
		RefundPercentage:          100,
	}

	booking, err := uc.CreateBooking(ctx, bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	})
	require.NoError(t, err)
	require.NotNil(t, booking)

	assert.True(t, booking.CancellationAllowed)
	assert.Equal(t, 24, booking.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyFull, booking.RefundPolicy)
	assert.Equal(t, 100, booking.RefundPercentage)

	require.Len(t, repos.bookingRepo.createdBookings, 1)
	persisted := repos.bookingRepo.createdBookings[0]
	assert.True(t, persisted.CancellationAllowed)
	assert.Equal(t, 24, persisted.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyFull, persisted.RefundPolicy)
	assert.Equal(t, 100, persisted.RefundPercentage)
}

func TestBookingUsecase_CreateBooking_Snapshot_PartialRefund(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	_, checkoutID := createTestReservationAndCheckout(repos, userID, eventID, now)

	repos.eventCancellationRepo.cancellations[eventID] = &domain.EventCancellation{
		ID:                        uuid.New(),
		EventID:                   eventID,
		CancellationAllowed:       true,
		CancellationDeadlineHours: 48,
		RefundPolicy:              domain.RefundPolicyPartial,
		RefundPercentage:          75,
	}

	booking, err := uc.CreateBooking(ctx, bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	})
	require.NoError(t, err)
	require.NotNil(t, booking)

	assert.True(t, booking.CancellationAllowed)
	assert.Equal(t, 48, booking.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyPartial, booking.RefundPolicy)
	assert.Equal(t, 75, booking.RefundPercentage)

	require.Len(t, repos.bookingRepo.createdBookings, 1)
	persisted := repos.bookingRepo.createdBookings[0]
	assert.True(t, persisted.CancellationAllowed)
	assert.Equal(t, 48, persisted.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyPartial, persisted.RefundPolicy)
	assert.Equal(t, 75, persisted.RefundPercentage)
}

func TestBookingUsecase_CreateBooking_Snapshot_NoRefund(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	_, checkoutID := createTestReservationAndCheckout(repos, userID, eventID, now)

	repos.eventCancellationRepo.cancellations[eventID] = &domain.EventCancellation{
		ID:                        uuid.New(),
		EventID:                   eventID,
		CancellationAllowed:       true,
		CancellationDeadlineHours: 12,
		RefundPolicy:              domain.RefundPolicyNoRefund,
		RefundPercentage:          0,
	}

	booking, err := uc.CreateBooking(ctx, bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	})
	require.NoError(t, err)
	require.NotNil(t, booking)

	assert.True(t, booking.CancellationAllowed)
	assert.Equal(t, 12, booking.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyNoRefund, booking.RefundPolicy)
	assert.Equal(t, 0, booking.RefundPercentage)

	require.Len(t, repos.bookingRepo.createdBookings, 1)
	persisted := repos.bookingRepo.createdBookings[0]
	assert.True(t, persisted.CancellationAllowed)
	assert.Equal(t, 12, persisted.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyNoRefund, persisted.RefundPolicy)
	assert.Equal(t, 0, persisted.RefundPercentage)
}

func TestBookingUsecase_CreateBooking_Snapshot_CancellationDisabled(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	_, checkoutID := createTestReservationAndCheckout(repos, userID, eventID, now)

	repos.eventCancellationRepo.cancellations[eventID] = &domain.EventCancellation{
		ID:                        uuid.New(),
		EventID:                   eventID,
		CancellationAllowed:       false,
		CancellationDeadlineHours: 0,
		RefundPolicy:              domain.RefundPolicyNoRefund,
		RefundPercentage:          0,
	}

	booking, err := uc.CreateBooking(ctx, bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	})
	require.NoError(t, err)
	require.NotNil(t, booking)

	assert.False(t, booking.CancellationAllowed)
	assert.Equal(t, 0, booking.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyNoRefund, booking.RefundPolicy)
	assert.Equal(t, 0, booking.RefundPercentage)

	require.Len(t, repos.bookingRepo.createdBookings, 1)
	persisted := repos.bookingRepo.createdBookings[0]
	assert.False(t, persisted.CancellationAllowed)
	assert.Equal(t, 0, persisted.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyNoRefund, persisted.RefundPolicy)
	assert.Equal(t, 0, persisted.RefundPercentage)
}

func TestBookingUsecase_CreateBooking_Snapshot_MissingPolicy_DefaultsSafely(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	_, checkoutID := createTestReservationAndCheckout(repos, userID, eventID, now)

	// Explicitly ensure no policy exists for this event
	delete(repos.eventCancellationRepo.cancellations, eventID)

	booking, err := uc.CreateBooking(ctx, bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	})
	require.NoError(t, err)
	require.NotNil(t, booking)

	// Safe defaults must be applied when policy record is absent
	assert.False(t, booking.CancellationAllowed)
	assert.Equal(t, 0, booking.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyNoRefund, booking.RefundPolicy)
	assert.Equal(t, 0, booking.RefundPercentage)

	require.Len(t, repos.bookingRepo.createdBookings, 1)
	persisted := repos.bookingRepo.createdBookings[0]
	assert.False(t, persisted.CancellationAllowed)
	assert.Equal(t, 0, persisted.CancellationDeadlineHours)
	assert.Equal(t, domain.RefundPolicyNoRefund, persisted.RefundPolicy)
	assert.Equal(t, 0, persisted.RefundPercentage)
}

func TestBookingUsecase_CreateBooking_Snapshot_RepositoryError_AbortsTransaction(t *testing.T) {
	uc, repos := setupBookingUsecase()
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	_, checkoutID := createTestReservationAndCheckout(repos, userID, eventID, now)

	// Unexpected database failure
	dbErr := errors.New("db connection failure")
	repos.eventCancellationRepo.findErr = dbErr

	booking, err := uc.CreateBooking(ctx, bookinguc.CreateBookingRequest{
		CheckoutID: checkoutID,
		UserID:     userID,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Nil(t, booking)

	// No bookings should have been created due to transaction rollback
	assert.Empty(t, repos.bookingRepo.createdBookings)
}

// --- Customer-Initiated Cancellation Tests ---

func createCancellableBookingFixture(
	repos *mockTxRepos,
	userID uuid.UUID,
	total decimal.Decimal,
	cancellationAllowed bool,
	deadlineHours int,
	policy string,
	percentage int,
	eventDateFromNow time.Duration,
) (*domain.Booking, *domain.Payment, []domain.BookingItem, []domain.Ticket) {
	now := time.Now()
	eventID := uuid.New()
	eventDate := now.Add(eventDateFromNow)

	repos.scheduleRepo.schedules[eventID] = &domain.EventSchedule{
		EventID:   eventID,
		EventDate: eventDate,
		StartTime: eventDate,
	}

	bookingID := uuid.New()
	booking := &domain.Booking{
		ID:                        bookingID,
		BookingNumber:             "BK-" + bookingID.String()[:8],
		UserID:                    userID,
		EventID:                   eventID,
		Total:                     total,
		Status:                    domain.BookingStatusConfirmed,
		CancellationAllowed:       cancellationAllowed,
		CancellationDeadlineHours: deadlineHours,
		RefundPolicy:              policy,
		RefundPercentage:          percentage,
		CreatedAt:                 now.Add(-2 * time.Hour),
		UpdatedAt:                 now.Add(-2 * time.Hour),
	}
	repos.bookingRepo.bookingsByID[bookingID] = booking

	ticketTypeID := uuid.New()
	repos.ticketTypeRepo.types[ticketTypeID] = &domain.TicketType{
		ID:                ticketTypeID,
		EventID:           eventID,
		Name:              "General Admission",
		TotalQuantity:     100,
		AvailableQuantity: 50,
		Price:             total,
	}

	item := domain.BookingItem{
		ID:           uuid.New(),
		BookingID:    bookingID,
		TicketTypeID: &ticketTypeID,
		Quantity:     2,
		UnitPrice:    total.Div(decimal.NewFromInt(2)),
		TotalPrice:   total,
	}
	repos.bookingRepo.createdItems = append(repos.bookingRepo.createdItems, []domain.BookingItem{item})

	ticket1 := domain.Ticket{
		ID:           uuid.New(),
		BookingID:    bookingID,
		UserID:       userID,
		EventID:      eventID,
		TicketTypeID: &ticketTypeID,
		Status:       domain.TicketStatusValid,
	}
	ticket2 := domain.Ticket{
		ID:           uuid.New(),
		BookingID:    bookingID,
		UserID:       userID,
		EventID:      eventID,
		TicketTypeID: &ticketTypeID,
		Status:       domain.TicketStatusValid,
	}
	tickets := []domain.Ticket{ticket1, ticket2}
	repos.ticketRepo.ticketsByBooking[bookingID] = tickets

	var payment *domain.Payment
	if total.IsPositive() {
		payID := uuid.New().String()
		txID := "pay_test_" + payID[:8]
		payment = &domain.Payment{
			ID:            payID,
			BookingID:     bookingID.String(),
			TransactionID: &txID,
			Amount:        total,
			Status:        domain.PaymentStatusSuccess,
			PaidAt:        &now,
		}
		repos.paymentRepo.paymentsByBooking[bookingID.String()] = []domain.Payment{*payment}
	}

	return booking, payment, []domain.BookingItem{item}, tickets
}

func TestBookingUsecase_CancelBooking_FullRefundSuccess(t *testing.T) {
	uc, repos, provider := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, payment, items, tickets := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.NoError(t, err)
	require.NotNil(t, out)

	assert.Equal(t, booking.ID, out.BookingID)
	assert.Equal(t, domain.BookingStatusCancelled, out.Status)
	assert.True(t, out.RefundAmount.Equal(total), "expected 100% refund of 500, got %s", out.RefundAmount)
	assert.Equal(t, domain.RefundStatusSuccess, out.RefundStatus)
	assert.NotEmpty(t, out.RefundID)

	// Booking status in repository updated to CANCELLED
	assert.Equal(t, domain.BookingStatusCancelled, repos.bookingRepo.bookingsByID[booking.ID].Status)

	// Tickets status updated to CANCELLED
	for _, tk := range repos.ticketRepo.ticketsByBooking[booking.ID] {
		assert.Equal(t, domain.TicketStatusCancelled, tk.Status)
	}

	// GA inventory restored
	ticketTypeID := *items[0].TicketTypeID
	assert.Equal(t, 2, repos.ticketTypeRepo.releasedQuantity[ticketTypeID])
	assert.Equal(t, 52, repos.ticketTypeRepo.types[ticketTypeID].AvailableQuantity)

	// Gateway invoked with correct payment ID and amount
	assert.Equal(t, 1, provider.refundCalls)
	assert.Equal(t, *payment.TransactionID, provider.lastPaymentID)
	assert.True(t, provider.lastAmount.Equal(total))

	// Refund record persisted with SUCCESS
	refunds := repos.refundRepo.refundsByBooking[booking.ID]
	require.Len(t, refunds, 1)
	assert.Equal(t, domain.RefundStatusSuccess, refunds[0].Status)
	assert.True(t, refunds[0].Amount.Equal(total))
	assert.Equal(t, out.RefundID, *refunds[0].GatewayRefundID)
	assert.NotNil(t, refunds[0].ProcessedAt)

	_ = tickets
}

func TestBookingUsecase_CancelBooking_PartialRefundSuccess(t *testing.T) {
	uc, repos, provider := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(300)

	booking, payment, _, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyPartial, 50, 48*time.Hour,
	)

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.NoError(t, err)
	require.NotNil(t, out)

	expectedRefund := decimal.NewFromInt(150)
	assert.True(t, out.RefundAmount.Equal(expectedRefund), "expected 50% refund of 300 = 150, got %s", out.RefundAmount)
	assert.Equal(t, domain.RefundStatusSuccess, out.RefundStatus)

	// Gateway invoked with 150
	assert.Equal(t, 1, provider.refundCalls)
	assert.True(t, provider.lastAmount.Equal(expectedRefund))
	assert.Equal(t, *payment.TransactionID, provider.lastPaymentID)
}

func TestBookingUsecase_CancelBooking_NoRefundPolicy(t *testing.T) {
	uc, repos, provider := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, _, items, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyNoRefund, 0, 48*time.Hour,
	)

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.NoError(t, err)
	require.NotNil(t, out)

	assert.Equal(t, domain.BookingStatusCancelled, out.Status)
	assert.True(t, out.RefundAmount.IsZero())
	assert.Equal(t, "NO_REFUND", out.RefundStatus)

	// Gateway should NOT be called
	assert.Equal(t, 0, provider.refundCalls)

	// Booking and tickets cancelled, inventory released
	assert.Equal(t, domain.BookingStatusCancelled, repos.bookingRepo.bookingsByID[booking.ID].Status)
	for _, tk := range repos.ticketRepo.ticketsByBooking[booking.ID] {
		assert.Equal(t, domain.TicketStatusCancelled, tk.Status)
	}
	assert.Equal(t, 2, repos.ticketTypeRepo.releasedQuantity[*items[0].TicketTypeID])
}

func TestBookingUsecase_CancelBooking_FreeBooking(t *testing.T) {
	uc, repos, provider := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.Zero

	booking, _, items, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.NoError(t, err)
	require.NotNil(t, out)

	assert.Equal(t, domain.BookingStatusCancelled, out.Status)
	assert.True(t, out.RefundAmount.IsZero())

	// Free bookings must not trigger gateway call
	assert.Equal(t, 0, provider.refundCalls)

	// Inventory restored
	assert.Equal(t, 2, repos.ticketTypeRepo.releasedQuantity[*items[0].TicketTypeID])
}

func TestBookingUsecase_CancelBooking_CancellationDisabled(t *testing.T) {
	uc, repos, provider := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, _, _, _ := createCancellableBookingFixture(
		repos, userID, total, false, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCancellationNotAllowed)
	assert.Nil(t, out)

	// Booking remains confirmed, gateway untouched
	assert.Equal(t, domain.BookingStatusConfirmed, repos.bookingRepo.bookingsByID[booking.ID].Status)
	assert.Equal(t, 0, provider.refundCalls)
}

func TestBookingUsecase_CancelBooking_DeadlinePassed(t *testing.T) {
	uc, repos, provider := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	// Event is in 10 hours, but deadline is 24 hours before event (deadline passed 14h ago)
	booking, _, _, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyFull, 100, 10*time.Hour,
	)

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCancellationDeadlinePassed)
	assert.Nil(t, out)

	// Booking remains confirmed
	assert.Equal(t, domain.BookingStatusConfirmed, repos.bookingRepo.bookingsByID[booking.ID].Status)
	assert.Equal(t, 0, provider.refundCalls)
}

func TestBookingUsecase_CancelBooking_TicketAlreadyUsed(t *testing.T) {
	uc, repos, provider := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, _, _, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)

	// Mark one ticket as USED
	repos.ticketRepo.ticketsByBooking[booking.ID][0].Status = domain.TicketStatusUsed

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTicketAlreadyUsed)
	assert.Nil(t, out)

	assert.Equal(t, domain.BookingStatusConfirmed, repos.bookingRepo.bookingsByID[booking.ID].Status)
	assert.Equal(t, 0, provider.refundCalls)
}

func TestBookingUsecase_CancelBooking_CrossCustomerUnauthorized(t *testing.T) {
	uc, repos, _ := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	ownerID := uuid.New()
	otherUserID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, _, _, _ := createCancellableBookingFixture(
		repos, ownerID, total, true, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)

	out, err := uc.CancelBooking(ctx, otherUserID, booking.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrBookingNotFound)
	assert.Nil(t, out)
}

func TestBookingUsecase_CancelBooking_AlreadyCancelled(t *testing.T) {
	uc, repos, _ := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, _, _, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)
	booking.Status = domain.BookingStatusCancelled

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrBookingAlreadyCancelled)
	assert.Nil(t, out)
}

func TestBookingUsecase_CancelBooking_BookingNotConfirmed(t *testing.T) {
	uc, repos, _ := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, _, _, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)
	booking.Status = domain.BookingStatusPending

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrBookingCannotBeCancelled)
	assert.Nil(t, out)
}

func TestBookingUsecase_CancelBooking_GatewayFailureAndRetry(t *testing.T) {
	uc, repos, provider := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, _, items, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)

	// Attempt 1: Gateway fails
	provider.refundErr = errors.New("gateway connection timeout")

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrGatewayRefundFailed)
	assert.Nil(t, out)

	// Refund record must be marked FAILED
	refunds := repos.refundRepo.refundsByBooking[booking.ID]
	require.Len(t, refunds, 1)
	assert.Equal(t, domain.RefundStatusFailed, refunds[0].Status)

	// Booking remains CONFIRMED, tickets remain VALID, inventory NOT restored
	assert.Equal(t, domain.BookingStatusConfirmed, repos.bookingRepo.bookingsByID[booking.ID].Status)
	for _, tk := range repos.ticketRepo.ticketsByBooking[booking.ID] {
		assert.Equal(t, domain.TicketStatusValid, tk.Status)
	}
	assert.Equal(t, 0, repos.ticketTypeRepo.releasedQuantity[*items[0].TicketTypeID])

	// Attempt 2: Gateway succeeds on retry
	provider.refundErr = nil
	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_retry_success",
		Amount:   total,
		Status:   domain.RefundStatusSuccess,
	}

	out2, err2 := uc.CancelBooking(ctx, userID, booking.ID)
	require.NoError(t, err2)
	require.NotNil(t, out2)

	assert.Equal(t, domain.BookingStatusCancelled, out2.Status)
	assert.Equal(t, "rfnd_retry_success", out2.RefundID)
	assert.Equal(t, domain.RefundStatusSuccess, out2.RefundStatus)

	// Now booking is CANCELLED, tickets CANCELLED, inventory restored
	assert.Equal(t, domain.BookingStatusCancelled, repos.bookingRepo.bookingsByID[booking.ID].Status)
	for _, tk := range repos.ticketRepo.ticketsByBooking[booking.ID] {
		assert.Equal(t, domain.TicketStatusCancelled, tk.Status)
	}
	assert.Equal(t, 2, repos.ticketTypeRepo.releasedQuantity[*items[0].TicketTypeID])

	// Total 2 refund records: 1 FAILED, 1 SUCCESS
	refundsAfterRetry := repos.refundRepo.refundsByBooking[booking.ID]
	require.Len(t, refundsAfterRetry, 2)
	assert.Equal(t, domain.RefundStatusFailed, refundsAfterRetry[0].Status)
	assert.Equal(t, domain.RefundStatusSuccess, refundsAfterRetry[1].Status)
}

func TestBookingUsecase_CancelBooking_ReservedSeatRelease(t *testing.T) {
	uc, repos, _ := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, _, _, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)

	// Replace items with a reserved seat item
	seatID := uuid.New()
	repos.bookingRepo.createdItems = [][]domain.BookingItem{
		{
			{
				ID:         uuid.New(),
				BookingID:  booking.ID,
				SeatID:     &seatID,
				Quantity:   1,
				UnitPrice:  total,
				TotalPrice: total,
			},
		},
	}

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.NoError(t, err)
	require.NotNil(t, out)

	// Verify seat was released
	require.Len(t, repos.seatRepo.releasedSeats, 1)
	assert.Equal(t, seatID, repos.seatRepo.releasedSeats[0])
}

func TestBookingUsecase_CancelBooking_RefundInProgressConflict(t *testing.T) {
	uc, repos, provider := setupBookingUsecaseWithRefund()
	ctx := context.Background()
	userID := uuid.New()
	total := decimal.NewFromInt(500)

	booking, payment, _, _ := createCancellableBookingFixture(
		repos, userID, total, true, 24, domain.RefundPolicyFull, 100, 48*time.Hour,
	)

	// Inject active pending refund within the 2-minute in-flight window
	payUUID, _ := uuid.Parse(payment.ID)
	repos.refundRepo.refundsByBooking[booking.ID] = []domain.Refund{
		{
			ID:        uuid.New(),
			BookingID: booking.ID,
			PaymentID: payUUID,
			Amount:    total,
			Status:    domain.RefundStatusPending,
			CreatedAt: time.Now().Add(-30 * time.Second),
		},
	}

	out, err := uc.CancelBooking(ctx, userID, booking.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRefundInProgress)
	assert.Nil(t, out)
	assert.Equal(t, 0, provider.refundCalls)
}
