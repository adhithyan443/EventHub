package domain

// DispatchTracker tracks background tasks and safely manages task registration
// during application graceful shutdown, atomically preventing new tasks from being
// scheduled once shutdown begins.
type DispatchTracker interface {
	// TryAdd attempts to register a new background task.
	// Returns true if the task was registered successfully.
	// Returns false if shutdown is active and new registrations are closed.
	TryAdd() bool

	// Done decrements the active task counter when a tracked task finishes.
	Done()
}
