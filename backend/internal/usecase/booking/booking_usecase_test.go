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

type mockTxRepos struct {
	domain.TransactionRepositories
	bookingRepo         *mockBookingRepo
	checkoutRepo        *mockCheckoutRepo
	seatReservationRepo *mockSeatReservationRepo
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
