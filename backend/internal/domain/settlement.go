package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrNilFinanceJournalRepository       = errors.New("finance journal repository cannot be nil")
	ErrNilOrganizerBankAccountRepo       = errors.New("organizer bank account repository cannot be nil")
	ErrSettlementNotFound                = errors.New("settlement not found")
	ErrInvalidSettlementStatus           = errors.New("invalid settlement status")
	ErrInvalidSettlementStatusTransition = errors.New("invalid settlement status transition")
	ErrInvalidSettlementAmount           = errors.New("settlement amount must be positive")
	ErrInsufficientAvailableBalance      = errors.New("settlement amount exceeds available balance")
	ErrOrganizerNotEligibleForSettlement  = errors.New("organizer is not eligible for settlement")
	ErrEventNotEligibleForSettlement     = errors.New("event is not eligible for settlement")
	ErrSettlementAlreadyProcessed        = errors.New("settlement has already been processed")
	ErrRejectionReasonRequired           = errors.New("rejection reason is required")
	ErrPayoutFailed                      = errors.New("payout execution failed")
)

// EventFinancialLifecycle classifies the operational and scheduling status of an event
// for settlement eligibility.
type EventFinancialLifecycle string

const (
	EventFinancialUpcoming  EventFinancialLifecycle = "UPCOMING"
	EventFinancialOngoing   EventFinancialLifecycle = "ONGOING"
	EventFinancialCompleted EventFinancialLifecycle = "COMPLETED"
	EventFinancialCancelled EventFinancialLifecycle = "CANCELLED"
)

// SettlementEligibilityStatus classifies whether an organizer's balance can be disbursed.
type SettlementEligibilityStatus string

const (
	SettlementEligible          SettlementEligibilityStatus = "ELIGIBLE"
	SettlementIneligible        SettlementEligibilityStatus = "INELIGIBLE"
	SettlementPartiallyEligible SettlementEligibilityStatus = "PARTIALLY_ELIGIBLE"
)

// SettlementStatus represents the lifecycle state of a settlement request.
type SettlementStatus string

const (
	SettlementStatusPending    SettlementStatus = "PENDING"
	SettlementStatusApproved   SettlementStatus = "APPROVED"
	SettlementStatusProcessing SettlementStatus = "PROCESSING"
	SettlementStatusCompleted  SettlementStatus = "COMPLETED"
	SettlementStatusRejected   SettlementStatus = "REJECTED"
	SettlementStatusFailed     SettlementStatus = "FAILED"
)

// CanTransitionTo enforces the permitted settlement lifecycle transitions.
// PENDING -> APPROVED -> PROCESSING -> COMPLETED
// PENDING -> REJECTED
// PROCESSING -> FAILED
// FAILED -> PROCESSING (retry)
func (s SettlementStatus) CanTransitionTo(next SettlementStatus) bool {
	switch s {
	case SettlementStatusPending:
		return next == SettlementStatusApproved || next == SettlementStatusRejected
	case SettlementStatusApproved:
		return next == SettlementStatusProcessing
	case SettlementStatusProcessing:
		return next == SettlementStatusCompleted || next == SettlementStatusFailed
	case SettlementStatusFailed:
		return next == SettlementStatusProcessing || next == SettlementStatusRejected
	default:
		return false
	}
}

// Settlement represents an organizer payout disbursement record.
type Settlement struct {
	ID              uuid.UUID        `json:"id"`
	OrganizerID     uuid.UUID        `json:"organizer_id"`
	EventID         *uuid.UUID       `json:"event_id,omitempty"`
	Amount          decimal.Decimal  `json:"amount"`
	Currency        string           `json:"currency"`
	Status          SettlementStatus `json:"status"`
	PayoutReference *string          `json:"payout_reference,omitempty"`
	FailureReason   *string          `json:"failure_reason,omitempty"`
	RejectionReason *string          `json:"rejection_reason,omitempty"`
	ApprovedBy      *uuid.UUID       `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time       `json:"approved_at,omitempty"`
	RejectedAt      *time.Time       `json:"rejected_at,omitempty"`
	ProcessedAt     *time.Time       `json:"processed_at,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// SettlementFilter specifies filtering and pagination criteria for settlements.
type SettlementFilter struct {
	OrganizerID *uuid.UUID        `json:"organizer_id,omitempty"`
	EventID     *uuid.UUID        `json:"event_id,omitempty"`
	Status      *SettlementStatus `json:"status,omitempty"`
	Page        int               `json:"page"`
	Limit       int               `json:"limit"`
}

// SettlementListResult encapsulates paginated settlement results.
type SettlementListResult struct {
	Settlements []*Settlement `json:"settlements"`
	Total       int64         `json:"total"`
	Page        int           `json:"page"`
	Limit       int           `json:"limit"`
}

// SettlementRepository defines the persistence contract for settlement records.
type SettlementRepository interface {
	Create(ctx context.Context, settlement *Settlement) error
	Update(ctx context.Context, settlement *Settlement) error
	FindByID(ctx context.Context, id uuid.UUID) (*Settlement, error)
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*Settlement, error)
	List(ctx context.Context, filter SettlementFilter) (*SettlementListResult, error)
	GetInFlightSettlementAmount(ctx context.Context, organizerID uuid.UUID) (decimal.Decimal, error)
	GetInFlightSettlementAmountForEvent(ctx context.Context, eventID uuid.UUID) (decimal.Decimal, error)
}

// PayoutRequest encapsulates the payload passed to a payout provider.
type PayoutRequest struct {
	SettlementID   uuid.UUID
	OrganizerID    uuid.UUID
	Amount         decimal.Decimal
	Currency       string
	AccountNumber  string
	IFSC           string
	IdempotencyKey string
}

// PayoutResponse encapsulates the payout execution result from the provider.
type PayoutResponse struct {
	PayoutReference string
	Success         bool
	FailureReason   string
}

// PayoutProvider defines the contract for external/mock payout gateways.
type PayoutProvider interface {
	ProcessPayout(ctx context.Context, req PayoutRequest) (*PayoutResponse, error)
}

// EventBalanceProjection represents the financial ledger balance and settlement eligibility
// for an individual event.
type EventBalanceProjection struct {
	EventID              uuid.UUID               `json:"event_id"`
	EventTitle           string                  `json:"event_title"`
	Lifecycle            EventFinancialLifecycle `json:"lifecycle"`
	GrossEarnings        decimal.Decimal         `json:"gross_earnings"`
	RefundsDeducted      decimal.Decimal         `json:"refunds_deducted"`
	NetPayable           decimal.Decimal         `json:"net_payable"`
	InFlightRefundHold   decimal.Decimal         `json:"in_flight_refund_hold"`
	AvailableBalance     decimal.Decimal         `json:"available_balance"`
	PendingBalance       decimal.Decimal         `json:"pending_balance"`
	Currency             string                  `json:"currency"`
	IsEligible           bool                    `json:"is_eligible"`
	IneligibilityReasons []string                `json:"ineligibility_reasons,omitempty"`
}

// OrganizerBalanceProjection represents the complete financial ledger balance,
// pending holds, and settlement eligibility for an authenticated organizer.
type OrganizerBalanceProjection struct {
	OrganizerID          uuid.UUID                   `json:"organizer_id"`
	Currency             string                      `json:"currency"`
	GrossEarnings        decimal.Decimal             `json:"gross_earnings"`
	RefundsDeducted      decimal.Decimal             `json:"refunds_deducted"`
	NetEarnings          decimal.Decimal             `json:"net_earnings"`
	OrganizerPayable     decimal.Decimal             `json:"organizer_payable"`
	SettledAmount        decimal.Decimal             `json:"settled_amount"`
	InFlightRefundHold   decimal.Decimal             `json:"in_flight_refund_hold"`
	InFlightSettlementHold decimal.Decimal           `json:"in_flight_settlement_hold"`
	PendingBalance       decimal.Decimal             `json:"pending_balance"`
	AvailableBalance     decimal.Decimal             `json:"available_balance"`
	EligibilityStatus    SettlementEligibilityStatus `json:"eligibility_status"`
	EligibilityReasons   []string                    `json:"eligibility_reasons"`
	HasActiveBankAccount bool                        `json:"has_active_bank_account"`
	EventBalances        []EventBalanceProjection    `json:"event_balances"`
}

// OrganizerBalanceUsecase defines the usecase interface for projecting organizer financial balances.
type OrganizerBalanceUsecase interface {
	GetOrganizerBalanceProjection(
		ctx context.Context,
		userID uuid.UUID,
	) (*OrganizerBalanceProjection, error)
}

// SettlementUsecase defines the operations for requesting, reviewing, and processing settlements.
type SettlementUsecase interface {
	RequestSettlement(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, eventID *uuid.UUID) (*Settlement, error)
	ListOrganizerSettlements(ctx context.Context, userID uuid.UUID, page, limit int, status *SettlementStatus) (*SettlementListResult, error)
	GetOrganizerSettlement(ctx context.Context, userID uuid.UUID, settlementID uuid.UUID) (*Settlement, error)

	ListAdminSettlements(ctx context.Context, page, limit int, status *SettlementStatus, organizerID *uuid.UUID) (*SettlementListResult, error)
	GetAdminSettlement(ctx context.Context, settlementID uuid.UUID) (*Settlement, error)
	ApproveSettlement(ctx context.Context, adminUserID uuid.UUID, settlementID uuid.UUID) (*Settlement, error)
	RejectSettlement(ctx context.Context, adminUserID uuid.UUID, settlementID uuid.UUID, reason string) (*Settlement, error)
	RetrySettlement(ctx context.Context, adminUserID uuid.UUID, settlementID uuid.UUID) (*Settlement, error)
}
