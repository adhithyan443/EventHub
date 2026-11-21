package payout

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

// MockPayoutProvider implements domain.PayoutProvider for testing and MVP simulation.
type MockPayoutProvider struct {
	mu                   sync.Mutex
	idempotentResponses  map[string]*domain.PayoutResponse
	failureSimulations   map[uuid.UUID]string
	shouldFailGlobal     bool
	globalFailureReason  string
}

func NewMockPayoutProvider() *MockPayoutProvider {
	return &MockPayoutProvider{
		idempotentResponses: make(map[string]*domain.PayoutResponse),
		failureSimulations:  make(map[uuid.UUID]string),
	}
}

// ProcessPayout simulates an asynchronous/synchronous payout execution.
// It guarantees idempotency by caching responses by IdempotencyKey.
func (p *MockPayoutProvider) ProcessPayout(
	ctx context.Context,
	req domain.PayoutRequest,
) (*domain.PayoutResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check idempotency cache first
	if cached, exists := p.idempotentResponses[req.IdempotencyKey]; exists {
		return cached, nil
	}

	// Check simulated failure for specific settlement
	if reason, shouldFail := p.failureSimulations[req.SettlementID]; shouldFail {
		resp := &domain.PayoutResponse{
			PayoutReference: "",
			Success:         false,
			FailureReason:   reason,
		}
		p.idempotentResponses[req.IdempotencyKey] = resp
		return resp, nil
	}

	// Check global simulated failure
	if p.shouldFailGlobal {
		reason := p.globalFailureReason
		if reason == "" {
			reason = "provider temporarily unavailable"
		}
		resp := &domain.PayoutResponse{
			PayoutReference: "",
			Success:         false,
			FailureReason:   reason,
		}
		p.idempotentResponses[req.IdempotencyKey] = resp
		return resp, nil
	}

	// Generate realistic unique payout reference
	randomBytes := make([]byte, 6)
	_, _ = rand.Read(randomBytes)
	ref := fmt.Sprintf("pout_mock_%s_%s", req.SettlementID.String()[:8], hex.EncodeToString(randomBytes))

	resp := &domain.PayoutResponse{
		PayoutReference: ref,
		Success:         true,
		FailureReason:   "",
	}
	p.idempotentResponses[req.IdempotencyKey] = resp

	return resp, nil
}

// SetFailure configures the mock to fail a specific settlement with a custom reason.
func (p *MockPayoutProvider) SetFailure(settlementID uuid.UUID, reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failureSimulations[settlementID] = reason
}

// ClearFailure removes any simulated failure for a specific settlement.
func (p *MockPayoutProvider) ClearFailure(settlementID uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.failureSimulations, settlementID)
	// Clear any cached failed idempotency key specifically for this settlement ID
	targetKey := fmt.Sprintf("payout:%s", settlementID)
	if resp, exists := p.idempotentResponses[targetKey]; exists && !resp.Success {
		delete(p.idempotentResponses, targetKey)
	}
}

// SetGlobalFailure configures the mock to fail all payouts.
func (p *MockPayoutProvider) SetGlobalFailure(fail bool, reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shouldFailGlobal = fail
	p.globalFailureReason = reason
}

var _ domain.PayoutProvider = (*MockPayoutProvider)(nil)
