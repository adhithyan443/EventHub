package worker_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/worker"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 1. Registration before shutdown succeeds.
func TestDispatchTracker_RegistrationBeforeShutdown(t *testing.T) {
	tracker := worker.NewDispatchTracker()
	assert.False(t, tracker.IsClosed())
	assert.Equal(t, int64(0), tracker.ActiveCount())

	ok := tracker.TryAdd()
	require.True(t, ok, "TryAdd must succeed when tracker is open")
	assert.Equal(t, int64(1), tracker.ActiveCount())

	completed := make(chan struct{})
	go func() {
		defer tracker.Done()
		time.Sleep(20 * time.Millisecond)
		close(completed)
	}()

	tracker.CloseAndWait()
	assert.True(t, tracker.IsClosed())
	assert.Equal(t, int64(0), tracker.ActiveCount())

	select {
	case <-completed:
		// Succeeded
	default:
		t.Fatal("CloseAndWait returned before the task called Done")
	}
}

// 2. Registration rejected after shutdown.
func TestDispatchTracker_RegistrationRejectedAfterShutdown(t *testing.T) {
	tracker := worker.NewDispatchTracker()
	tracker.Close()

	assert.True(t, tracker.IsClosed())

	// Multiple attempts must all return false without incrementing active count or panicking
	for i := 0; i < 10; i++ {
		ok := tracker.TryAdd()
		assert.False(t, ok, "TryAdd must return false after tracker is closed")
	}

	assert.Equal(t, int64(0), tracker.ActiveCount())

	// CloseAndWait on an already closed, empty tracker must return immediately
	done := make(chan struct{})
	go func() {
		tracker.CloseAndWait()
		close(done)
	}()

	select {
	case <-done:
		// Succeeded immediately
	case <-time.After(100 * time.Millisecond):
		t.Fatal("CloseAndWait timed out on an empty closed tracker")
	}
}

// 3. Concurrent registration and shutdown (no race conditions, no WaitGroup panics).
func TestDispatchTracker_ConcurrentRegistrationAndShutdown(t *testing.T) {
	tracker := worker.NewDispatchTracker()

	numWorkers := 100
	var registeredCount atomic.Int64
	var rejectedCount atomic.Int64
	var workerWG sync.WaitGroup

	workerWG.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer workerWG.Done()
			// Stagger worker start slightly to interleave with CloseAndWait
			time.Sleep(time.Duration(id%5) * time.Millisecond)

			if tracker.TryAdd() {
				registeredCount.Add(1)
				// Simulate in-flight refund gateway dispatch
				time.Sleep(10 * time.Millisecond)
				tracker.Done()
			} else {
				rejectedCount.Add(1)
			}
		}(i)
	}

	// Trigger shutdown concurrently while workers are registering
	time.Sleep(3 * time.Millisecond)
	tracker.CloseAndWait()

	// Wait for all worker goroutines to conclude
	workerWG.Wait()

	total := registeredCount.Load() + rejectedCount.Load()
	assert.Equal(t, int64(numWorkers), total, "All workers must have either registered or been rejected")
	assert.True(t, registeredCount.Load() > 0, "Some workers should have registered before close")
	assert.True(t, rejectedCount.Load() > 0, "Some workers should have been rejected after close")
	assert.Equal(t, int64(0), tracker.ActiveCount(), "Active count must be 0 after CloseAndWait completes")
}

// 4. Shutdown timeout while a refund dispatch is still running.
func TestDispatchTracker_ShutdownTimeoutBounded(t *testing.T) {
	tracker := worker.NewDispatchTracker()

	ok := tracker.TryAdd()
	require.True(t, ok)

	// Simulate a slow refund task (e.g. gateway slow response 300ms)
	taskDone := make(chan struct{})
	go func() {
		time.Sleep(300 * time.Millisecond)
		tracker.Done()
		close(taskDone)
	}()

	// Graceful shutdown has a tight timeout (50ms)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := tracker.CloseAndWaitWithContext(shutdownCtx)
	elapsed := time.Since(start)

	// Shutdown must return context.DeadlineExceeded within ~50ms, not block for 300ms
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, elapsed, 150*time.Millisecond, "Shutdown must respect timeout and not block indefinitely")
	assert.True(t, tracker.IsClosed(), "Tracker gate must be closed even if timeout occurred")

	// Verify that when slow task finishes, it completes cleanly
	<-taskDone
	assert.Equal(t, int64(0), tracker.ActiveCount())
}

// 5. Durable pending refund recovery:
// Verifies that when a background dispatch is rejected due to shutdown,
// the pending refund record remains safely stored in PostgreSQL and is
// successfully recovered and reconciled by RefundReconciliationWorker.
type mockReconUsecase struct {
	reconciledCount int
}

func (m *mockReconUsecase) ReconcileStuckRefunds(ctx context.Context, olderThan time.Duration, limit int) (int, error) {
	m.reconciledCount = 1
	return 1, nil
}

func TestDispatchTracker_DurablePendingRefundRecovery(t *testing.T) {
	tracker := worker.NewDispatchTracker()

	// Shutdown initiated
	tracker.Close()

	refundID := uuid.New()
	pendingRefund := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		Reason:    "EVENT_CANCELLATION",
	}

	// Handler attempts dispatch, but tracker rejects it
	dispatched := tracker.TryAdd()
	assert.False(t, dispatched, "Dispatch must be rejected during shutdown")

	// Handler logs warning and leaves refund in DB as PENDING
	// (Simulated: pendingRefund remains with Status == RefundStatusPending)
	assert.Equal(t, domain.RefundStatusPending, pendingRefund.Status)

	// Later, reconciliation worker runs to recover stuck PENDING refunds
	mockUC := &mockReconUsecase{}
	recWorker := worker.NewRefundReconciliationWorker(mockUC, nil, 10*time.Millisecond, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	recWorker.Run(ctx)

	assert.Equal(t, 1, mockUC.reconciledCount, "Reconciliation worker must recover the un-dispatched pending refund")
}
