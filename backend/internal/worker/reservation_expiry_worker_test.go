package worker

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type reservationExpiryUsecaseMock struct {
	mu sync.Mutex

	processCalls int
	processLimit int
	processErr   error

	called chan struct{}
}

func (m *reservationExpiryUsecaseMock) ExpireReservation(
	_ context.Context,
	_ uuid.UUID,
) error {
	return nil
}

func (m *reservationExpiryUsecaseMock) ProcessExpiredReservations(
	_ context.Context,
	limit int,
) error {
	m.mu.Lock()
	m.processCalls++
	m.processLimit = limit
	m.mu.Unlock()

	if m.called != nil {
		select {
		case m.called <- struct{}{}:
		default:
		}
	}

	return m.processErr
}

func (m *reservationExpiryUsecaseMock) getProcessCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.processCalls
}

func (m *reservationExpiryUsecaseMock) getProcessLimit() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.processLimit
}

func newTestLogger(output *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(output, nil))
}

func TestReservationExpiryWorker_InvalidInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
	}{
		{
			name:     "zero interval",
			interval: 0,
		},
		{
			name:     "negative interval",
			interval: -time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecase := &reservationExpiryUsecaseMock{}
			logger := newTestLogger(&bytes.Buffer{})

			worker := NewReservationExpiryWorker(
				usecase,
				logger,
				tt.interval,
			)

			worker.Run(context.Background())

			require.Zero(t, usecase.getProcessCalls())
		})
	}
}

func TestReservationExpiryWorker_ProcessesExpiredReservations(t *testing.T) {
	usecase := &reservationExpiryUsecaseMock{
		called: make(chan struct{}, 1),
	}
	logger := newTestLogger(&bytes.Buffer{})

	worker := NewReservationExpiryWorker(
		usecase,
		logger,
		10*time.Millisecond,
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		worker.Run(ctx)
	}()

	select {
	case <-usecase.called:
		// The worker processed an expiry batch.
	case <-time.After(time.Second):
		cancel()
		<-done
		t.Fatal("worker did not process expired reservations")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}

	require.GreaterOrEqual(t, usecase.getProcessCalls(), 1)
	require.Equal(t, 100, usecase.getProcessLimit())
}

func TestReservationExpiryWorker_LogsProcessingError(t *testing.T) {
	output := &bytes.Buffer{}
	usecase := &reservationExpiryUsecaseMock{
		called:     make(chan struct{}, 1),
		processErr: errors.New("database unavailable"),
	}
	logger := newTestLogger(output)

	worker := NewReservationExpiryWorker(
		usecase,
		logger,
		10*time.Millisecond,
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		worker.Run(ctx)
	}()

	select {
	case <-usecase.called:
		// Wait until the worker has finished logging the error.
	case <-time.After(time.Second):
		cancel()
		<-done
		t.Fatal("worker did not attempt to process expired reservations")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}

	require.Contains(t, output.String(), "reservation expiry batch processing failed")
	require.Contains(t, output.String(), "database unavailable")
	require.True(t, strings.Contains(output.String(), "ERROR"))
}
