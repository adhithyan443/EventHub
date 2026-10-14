package reservation_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/reservation"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type reservationTestRepository struct {
	domain.SeatReservationRepository

	expireResult bool
	expireErr    error
	items        []domain.SeatReservationItem
	itemsErr     error
	findCalls    int
}

func (m *reservationTestRepository) ExpireIfActive(
	_ context.Context,
	_ uuid.UUID,
	_ time.Time,
) (bool, error) {
	return m.expireResult, m.expireErr
}

func (m *reservationTestRepository) FindItemsByReservationID(
	_ context.Context,
	_ uuid.UUID,
) ([]domain.SeatReservationItem, error) {
	m.findCalls++
	return m.items, m.itemsErr
}

type reservationTestSeatRepository struct {
	domain.SeatRepository

	releasedIDs []uuid.UUID
	releaseErr  error
}

func (m *reservationTestSeatRepository) ReleaseReservedSeat(
	_ context.Context,
	id uuid.UUID,
) error {
	m.releasedIDs = append(m.releasedIDs, id)
	return m.releaseErr
}

type reservationTestTicketRepository struct {
	domain.TicketTypeRepository

	releasedIDs      []uuid.UUID
	releasedQuantity []int
	releaseErr       error
}

func (m *reservationTestTicketRepository) ReleaseReservedQuantity(
	_ context.Context,
	id uuid.UUID,
	quantity int,
) error {
	m.releasedIDs = append(m.releasedIDs, id)
	m.releasedQuantity = append(m.releasedQuantity, quantity)
	return m.releaseErr
}

type reservationTestRepositories struct {
	domain.TransactionRepositories

	reservations *reservationTestRepository
	seats        *reservationTestSeatRepository
	tickets      *reservationTestTicketRepository
}

func (m *reservationTestRepositories) SeatReservationRepository() domain.SeatReservationRepository {
	return m.reservations
}

func (m *reservationTestRepositories) SeatRepository() domain.SeatRepository {
	return m.seats
}

func (m *reservationTestRepositories) TicketTypeRepository() domain.TicketTypeRepository {
	return m.tickets
}

type reservationTestTransactionManager struct {
	domain.TransactionManager
	repositories domain.TransactionRepositories
	calls        int
}

func (m *reservationTestTransactionManager) WithinTransaction(
	fn func(domain.TransactionRepositories) error,
) error {
	m.calls++
	return fn(m.repositories)
}

func newReservationTestUsecase(
	repo *reservationTestRepository,
	seats *reservationTestSeatRepository,
	tickets *reservationTestTicketRepository,
) (*reservation.ReservationUsecase, *reservationTestTransactionManager) {
	repositories := &reservationTestRepositories{
		reservations: repo,
		seats:        seats,
		tickets:      tickets,
	}

	manager := &reservationTestTransactionManager{
		repositories: repositories,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	usecase := reservation.NewReservationUsecase(manager, logger)

	return usecase, manager
}

func TestExpireReservation_ReleasesReservedSeat(t *testing.T) {
	reservationID := uuid.New()
	seatID := uuid.New()

	repo := &reservationTestRepository{
		expireResult: true,
		items: []domain.SeatReservationItem{
			{SeatID: &seatID, Quantity: 1},
		},
	}
	seats := &reservationTestSeatRepository{}
	tickets := &reservationTestTicketRepository{}

	usecase, _ := newReservationTestUsecase(repo, seats, tickets)

	err := usecase.ExpireReservation(context.Background(), reservationID)

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{seatID}, seats.releasedIDs)
	assert.Empty(t, tickets.releasedIDs)
	assert.Equal(t, 1, repo.findCalls)
}

func TestExpireReservation_ReleasesGeneralAdmissionQuantity(t *testing.T) {
	reservationID := uuid.New()
	ticketTypeID := uuid.New()

	repo := &reservationTestRepository{
		expireResult: true,
		items: []domain.SeatReservationItem{
			{TicketTypeID: &ticketTypeID, Quantity: 3},
		},
	}
	seats := &reservationTestSeatRepository{}
	tickets := &reservationTestTicketRepository{}

	usecase, _ := newReservationTestUsecase(repo, seats, tickets)

	err := usecase.ExpireReservation(context.Background(), reservationID)

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{ticketTypeID}, tickets.releasedIDs)
	assert.Equal(t, []int{3}, tickets.releasedQuantity)
	assert.Empty(t, seats.releasedIDs)
}

func TestExpireReservation_NotExpired_DoesNotReleaseInventory(t *testing.T) {
	repo := &reservationTestRepository{expireResult: false}
	seats := &reservationTestSeatRepository{}
	tickets := &reservationTestTicketRepository{}

	usecase, _ := newReservationTestUsecase(repo, seats, tickets)

	err := usecase.ExpireReservation(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Equal(t, 0, repo.findCalls)
	assert.Empty(t, seats.releasedIDs)
	assert.Empty(t, tickets.releasedIDs)
}

func TestExpireReservation_InvalidID(t *testing.T) {
	repo := &reservationTestRepository{}
	seats := &reservationTestSeatRepository{}
	tickets := &reservationTestTicketRepository{}

	usecase, manager := newReservationTestUsecase(repo, seats, tickets)

	err := usecase.ExpireReservation(context.Background(), uuid.Nil)

	require.ErrorIs(t, err, domain.ErrInvalidReservationRequest)
	assert.Equal(t, 0, manager.calls)
}

func TestExpireReservation_ReleaseSeatError(t *testing.T) {
	expectedErr := errors.New("inventory release failed")
	seatID := uuid.New()

	repo := &reservationTestRepository{
		expireResult: true,
		items: []domain.SeatReservationItem{
			{SeatID: &seatID, Quantity: 1},
		},
	}
	seats := &reservationTestSeatRepository{releaseErr: expectedErr}
	tickets := &reservationTestTicketRepository{}

	usecase, _ := newReservationTestUsecase(repo, seats, tickets)

	err := usecase.ExpireReservation(context.Background(), uuid.New())

	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, []uuid.UUID{seatID}, seats.releasedIDs)
}

func TestExpireReservation_ItemWithoutInventoryReference(t *testing.T) {
	repo := &reservationTestRepository{
		expireResult: true,
		items: []domain.SeatReservationItem{
			{Quantity: 1},
		},
	}
	seats := &reservationTestSeatRepository{}
	tickets := &reservationTestTicketRepository{}

	usecase, _ := newReservationTestUsecase(repo, seats, tickets)

	err := usecase.ExpireReservation(context.Background(), uuid.New())

	require.ErrorIs(t, err, domain.ErrInvalidReservationRequest)
	assert.Empty(t, seats.releasedIDs)
	assert.Empty(t, tickets.releasedIDs)
}
