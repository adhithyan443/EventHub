package worker

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

var _ domain.DispatchTracker = (*DispatchTracker)(nil)

// DispatchTracker implements domain.DispatchTracker using a closed-gate pattern
// with sync.RWMutex and sync.WaitGroup to safely prevent registration races during shutdown.
// It ensures that sync.WaitGroup Add() cannot be called concurrently with Wait() when the counter is zero.
type DispatchTracker struct {
	mu     sync.RWMutex
	wg     sync.WaitGroup
	closed bool
	active atomic.Int64
}

// NewDispatchTracker creates a new thread-safe DispatchTracker with gate open.
func NewDispatchTracker() *DispatchTracker {
	return &DispatchTracker{}
}

// TryAdd attempts to register a new background task.
// Returns true if the task was registered successfully.
// Returns false if the tracker has been closed for shutdown, atomically rejecting new registrations.
func (t *DispatchTracker) TryAdd() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return false
	}

	t.active.Add(1)
	t.wg.Add(1)
	return true
}

// Done signals the completion of a tracked background task.
func (t *DispatchTracker) Done() {
	t.active.Add(-1)
	t.wg.Done()
}

// Close closes the registration gate. Subsequent TryAdd calls will atomically return false.
// It is idempotent and safe to call multiple times.
func (t *DispatchTracker) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
}

// Wait blocks until all in-flight registered tasks have called Done.
func (t *DispatchTracker) Wait() {
	t.wg.Wait()
}

// CloseAndWait closes the gate and waits for all active tasks to complete.
func (t *DispatchTracker) CloseAndWait() {
	t.Close()
	t.Wait()
}

// WaitWithContext waits for all active tasks to complete or returns ctx.Err() if the context expires.
func (t *DispatchTracker) WaitWithContext(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CloseAndWaitWithContext closes the registration gate and waits for all active tasks to complete,
// bounded by the provided context.
func (t *DispatchTracker) CloseAndWaitWithContext(ctx context.Context) error {
	t.Close()
	return t.WaitWithContext(ctx)
}

// ActiveCount returns the number of currently active registered tasks.
func (t *DispatchTracker) ActiveCount() int64 {
	return t.active.Load()
}

// IsClosed returns whether the tracker registration gate has been closed.
func (t *DispatchTracker) IsClosed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.closed
}
