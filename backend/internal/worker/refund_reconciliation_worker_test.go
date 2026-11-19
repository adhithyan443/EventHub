package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/worker"
	"github.com/stretchr/testify/assert"
)

type mockRefundReconciliationUsecase struct {
	mu        sync.Mutex
	calls     int
	olderThan time.Duration
	limit     int
	returnErr error
}

func (m *mockRefundReconciliationUsecase) ReconcileStuckRefunds(
	ctx context.Context,
	olderThan time.Duration,
	limit int,
) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.olderThan = olderThan
	m.limit = limit
	if m.returnErr != nil {
		return 0, m.returnErr
	}
	return 2, nil
}

func TestRefundReconciliationWorker_RunsAndStops(t *testing.T) {
	mockUC := &mockRefundReconciliationUsecase{}
	w := worker.NewRefundReconciliationWorker(
		mockUC,
		slog.Default(),
		20*time.Millisecond,
		1*time.Minute,
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		w.Run(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Clean stop
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not stop gracefully")
	}

	mockUC.mu.Lock()
	assert.GreaterOrEqual(t, mockUC.calls, 1)
	assert.Equal(t, 1*time.Minute, mockUC.olderThan)
	assert.Equal(t, 50, mockUC.limit)
	mockUC.mu.Unlock()
}

func TestRefundReconciliationWorker_HandlesErrorGracefully(t *testing.T) {
	mockUC := &mockRefundReconciliationUsecase{
		returnErr: errors.New("db error"),
	}
	w := worker.NewRefundReconciliationWorker(
		mockUC,
		slog.Default(),
		15*time.Millisecond,
		2*time.Minute,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	w.Run(ctx)

	mockUC.mu.Lock()
	assert.GreaterOrEqual(t, mockUC.calls, 1)
	mockUC.mu.Unlock()
}
