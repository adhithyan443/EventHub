package settlement

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrNilTransactionManager        = errors.New("transaction manager cannot be nil")
	ErrNilSettlementRepository      = errors.New("settlement repository cannot be nil")
	ErrNilOrganizerBalanceUsecase   = errors.New("organizer balance usecase cannot be nil")
	ErrNilPayoutProvider            = errors.New("payout provider cannot be nil")
)

type SettlementUsecaseImpl struct {
	txManager              domain.TransactionManager
	settlementRepo         domain.SettlementRepository
	organizerRepo          domain.OrganizerRepository
	organizerBankAccountRepo domain.OrganizerBankAccountRepository
	balanceUsecase         domain.OrganizerBalanceUsecase
	payoutProvider         domain.PayoutProvider
	logger                 *slog.Logger
}

func NewSettlementUsecase(
	txManager domain.TransactionManager,
	settlementRepo domain.SettlementRepository,
	organizerRepo domain.OrganizerRepository,
	organizerBankAccountRepo domain.OrganizerBankAccountRepository,
	balanceUsecase domain.OrganizerBalanceUsecase,
	payoutProvider domain.PayoutProvider,
	logger *slog.Logger,
) (*SettlementUsecaseImpl, error) {
	if txManager == nil {
		return nil, ErrNilTransactionManager
	}
	if settlementRepo == nil {
		return nil, ErrNilSettlementRepository
	}
	if organizerRepo == nil {
		return nil, ErrNilOrganizerRepository
	}
	if organizerBankAccountRepo == nil {
		return nil, ErrNilOrganizerBankAccountRepo
	}
	if balanceUsecase == nil {
		return nil, ErrNilOrganizerBalanceUsecase
	}
	if payoutProvider == nil {
		return nil, ErrNilPayoutProvider
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &SettlementUsecaseImpl{
		txManager:              txManager,
		settlementRepo:         settlementRepo,
		organizerRepo:          organizerRepo,
		organizerBankAccountRepo: organizerBankAccountRepo,
		balanceUsecase:         balanceUsecase,
		payoutProvider:         payoutProvider,
		logger:                 logger,
	}, nil
}

// RequestSettlement creates a new PENDING settlement request for an authenticated organizer.
func (u *SettlementUsecaseImpl) RequestSettlement(
	ctx context.Context,
	userID uuid.UUID,
	amount decimal.Decimal,
	eventID *uuid.UUID,
) (*domain.Settlement, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidSettlementAmount
	}

	// 1. Verify organizer identity and eligibility
	organizer, err := u.organizerRepo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerNotFound) {
			return nil, ErrUnauthorizedOrganizer
		}
		return nil, fmt.Errorf("failed to lookup organizer: %w", err)
	}

	status := strings.ToUpper(strings.TrimSpace(organizer.Status))
	if status != "ACTIVE" && status != "APPROVED" {
		return nil, domain.ErrOrganizerNotEligibleForSettlement
	}

	// 2. Verify active bank account presence
	bankAccount, err := u.organizerBankAccountRepo.FindByOrganizerID(organizer.ID)
	if err != nil || bankAccount == nil || strings.TrimSpace(bankAccount.AccountNumberLast4) == "" {
		return nil, domain.ErrOrganizerNotEligibleForSettlement
	}

	var createdSettlement *domain.Settlement

	// 3. Atomically evaluate available balance and create settlement within database transaction
	err = u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		// Acquire exclusive row lock on organizer to serialize concurrent settlement requests and prevent race conditions
		if _, err := tx.OrganizerRepository().FindByIDForUpdate(organizer.ID); err != nil {
			return fmt.Errorf("failed to lock organizer for settlement: %w", err)
		}

		projection, err := u.balanceUsecase.GetOrganizerBalanceProjection(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to project organizer balance: %w", err)
		}

		if eventID != nil && *eventID != uuid.Nil {
			var targetEvent *domain.EventBalanceProjection
			for i := range projection.EventBalances {
				if projection.EventBalances[i].EventID == *eventID {
					targetEvent = &projection.EventBalances[i]
					break
				}
			}

			if targetEvent == nil || !targetEvent.IsEligible {
				return domain.ErrEventNotEligibleForSettlement
			}

			if amount.GreaterThan(targetEvent.AvailableBalance) {
				return domain.ErrInsufficientAvailableBalance
			}
		} else {
			if amount.GreaterThan(projection.AvailableBalance) {
				return domain.ErrInsufficientAvailableBalance
			}
		}

		now := time.Now().UTC()
		settlement := &domain.Settlement{
			ID:          uuid.New(),
			OrganizerID: organizer.ID,
			EventID:     eventID,
			Amount:      amount,
			Currency:    domainfinance.CurrencyINR,
			Status:      domain.SettlementStatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := tx.SettlementRepository().Create(ctx, settlement); err != nil {
			return err
		}

		createdSettlement = settlement
		return nil
	})
	if err != nil {
		u.logger.WarnContext(ctx, "settlement_request_rejected",
			slog.String("user_id", userID.String()),
			slog.String("amount", amount.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "settlement_requested",
		slog.String("settlement_id", createdSettlement.ID.String()),
		slog.String("organizer_id", organizer.ID.String()),
		slog.String("amount", amount.String()),
	)

	return createdSettlement, nil
}

// ListOrganizerSettlements returns paginated settlements for an authenticated organizer.
func (u *SettlementUsecaseImpl) ListOrganizerSettlements(
	ctx context.Context,
	userID uuid.UUID,
	page, limit int,
	status *domain.SettlementStatus,
) (*domain.SettlementListResult, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}

	organizer, err := u.organizerRepo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerNotFound) {
			return nil, ErrUnauthorizedOrganizer
		}
		return nil, fmt.Errorf("failed to lookup organizer: %w", err)
	}

	filter := domain.SettlementFilter{
		OrganizerID: &organizer.ID,
		Status:      status,
		Page:        page,
		Limit:       limit,
	}

	return u.settlementRepo.List(ctx, filter)
}

// GetOrganizerSettlement returns a single settlement ensuring organizer ownership.
func (u *SettlementUsecaseImpl) GetOrganizerSettlement(
	ctx context.Context,
	userID uuid.UUID,
	settlementID uuid.UUID,
) (*domain.Settlement, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if settlementID == uuid.Nil {
		return nil, domain.ErrSettlementNotFound
	}

	organizer, err := u.organizerRepo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerNotFound) {
			return nil, ErrUnauthorizedOrganizer
		}
		return nil, fmt.Errorf("failed to lookup organizer: %w", err)
	}

	settlement, err := u.settlementRepo.FindByID(ctx, settlementID)
	if err != nil {
		return nil, err
	}

	if settlement.OrganizerID != organizer.ID {
		return nil, domain.ErrSettlementNotFound
	}

	return settlement, nil
}

// ListAdminSettlements returns paginated settlements across all organizers with optional filters.
func (u *SettlementUsecaseImpl) ListAdminSettlements(
	ctx context.Context,
	page, limit int,
	status *domain.SettlementStatus,
	organizerID *uuid.UUID,
) (*domain.SettlementListResult, error) {
	filter := domain.SettlementFilter{
		OrganizerID: organizerID,
		Status:      status,
		Page:        page,
		Limit:       limit,
	}

	return u.settlementRepo.List(ctx, filter)
}

// GetAdminSettlement retrieves detailed settlement info for admin review.
func (u *SettlementUsecaseImpl) GetAdminSettlement(
	ctx context.Context,
	settlementID uuid.UUID,
) (*domain.Settlement, error) {
	if settlementID == uuid.Nil {
		return nil, domain.ErrSettlementNotFound
	}

	return u.settlementRepo.FindByID(ctx, settlementID)
}

// ApproveSettlement transitions a PENDING settlement to APPROVED, then initiates payout.
func (u *SettlementUsecaseImpl) ApproveSettlement(
	ctx context.Context,
	adminUserID uuid.UUID,
	settlementID uuid.UUID,
) (*domain.Settlement, error) {
	if adminUserID == uuid.Nil {
		return nil, errors.New("invalid admin user ID")
	}
	if settlementID == uuid.Nil {
		return nil, domain.ErrSettlementNotFound
	}

	var approvedSettlement *domain.Settlement

	// Step 1: Transition PENDING -> APPROVED -> PROCESSING under database lock
	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		s, err := tx.SettlementRepository().FindByIDForUpdate(ctx, settlementID)
		if err != nil {
			return err
		}

		if !s.Status.CanTransitionTo(domain.SettlementStatusApproved) {
			return domain.ErrInvalidSettlementStatusTransition
		}

		now := time.Now().UTC()
		s.Status = domain.SettlementStatusApproved
		s.ApprovedBy = &adminUserID
		s.ApprovedAt = &now
		s.UpdatedAt = now

		if err := tx.SettlementRepository().Update(ctx, s); err != nil {
			return err
		}

		// Move immediately to PROCESSING for payout execution
		s.Status = domain.SettlementStatusProcessing
		s.UpdatedAt = now
		if err := tx.SettlementRepository().Update(ctx, s); err != nil {
			return err
		}

		approvedSettlement = s
		return nil
	})
	if err != nil {
		u.logger.WarnContext(ctx, "settlement_approval_failed",
			slog.String("settlement_id", settlementID.String()),
			slog.String("admin_user_id", adminUserID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "settlement_approved",
		slog.String("settlement_id", settlementID.String()),
		slog.String("admin_user_id", adminUserID.String()),
	)

	// Step 2: Execute payout via provider and post ledger journal
	return u.executePayout(ctx, approvedSettlement)
}

// RejectSettlement marks a PENDING settlement as REJECTED with an admin reason.
func (u *SettlementUsecaseImpl) RejectSettlement(
	ctx context.Context,
	adminUserID uuid.UUID,
	settlementID uuid.UUID,
	reason string,
) (*domain.Settlement, error) {
	if adminUserID == uuid.Nil {
		return nil, errors.New("invalid admin user ID")
	}
	if settlementID == uuid.Nil {
		return nil, domain.ErrSettlementNotFound
	}
	cleanReason := strings.TrimSpace(reason)
	if cleanReason == "" {
		return nil, domain.ErrRejectionReasonRequired
	}

	var rejectedSettlement *domain.Settlement

	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		s, err := tx.SettlementRepository().FindByIDForUpdate(ctx, settlementID)
		if err != nil {
			return err
		}

		if !s.Status.CanTransitionTo(domain.SettlementStatusRejected) {
			return domain.ErrInvalidSettlementStatusTransition
		}

		now := time.Now().UTC()
		s.Status = domain.SettlementStatusRejected
		s.RejectionReason = &cleanReason
		s.RejectedAt = &now
		s.UpdatedAt = now

		if err := tx.SettlementRepository().Update(ctx, s); err != nil {
			return err
		}

		rejectedSettlement = s
		return nil
	})
	if err != nil {
		u.logger.WarnContext(ctx, "settlement_rejection_failed",
			slog.String("settlement_id", settlementID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "settlement_rejected",
		slog.String("settlement_id", settlementID.String()),
		slog.String("admin_user_id", adminUserID.String()),
		slog.String("reason", cleanReason),
	)

	return rejectedSettlement, nil
}

// RetrySettlement retries payout execution for a FAILED settlement.
func (u *SettlementUsecaseImpl) RetrySettlement(
	ctx context.Context,
	adminUserID uuid.UUID,
	settlementID uuid.UUID,
) (*domain.Settlement, error) {
	if adminUserID == uuid.Nil {
		return nil, errors.New("invalid admin user ID")
	}
	if settlementID == uuid.Nil {
		return nil, domain.ErrSettlementNotFound
	}

	var retriedSettlement *domain.Settlement

	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		s, err := tx.SettlementRepository().FindByIDForUpdate(ctx, settlementID)
		if err != nil {
			return err
		}

		// Strict guard: only FAILED settlements can be retried
		if s.Status != domain.SettlementStatusFailed {
			return domain.ErrInvalidSettlementStatusTransition
		}

		if !s.Status.CanTransitionTo(domain.SettlementStatusProcessing) {
			return domain.ErrInvalidSettlementStatusTransition
		}

		// Lock organizer row to prevent concurrent race conditions
		organizer, err := tx.OrganizerRepository().FindByIDForUpdate(s.OrganizerID)
		if err != nil {
			return fmt.Errorf("failed to lock organizer for settlement retry: %w", err)
		}

		// Re-verify that available balance still covers the settlement amount
		projection, err := u.balanceUsecase.GetOrganizerBalanceProjection(ctx, organizer.UserID)
		if err != nil {
			return fmt.Errorf("failed to project organizer balance for retry: %w", err)
		}

		if s.EventID != nil && *s.EventID != uuid.Nil {
			var targetEvent *domain.EventBalanceProjection
			for i := range projection.EventBalances {
				if projection.EventBalances[i].EventID == *s.EventID {
					targetEvent = &projection.EventBalances[i]
					break
				}
			}
			if targetEvent == nil || !targetEvent.IsEligible || s.Amount.GreaterThan(targetEvent.AvailableBalance) {
				return domain.ErrInsufficientAvailableBalance
			}
		} else {
			if s.Amount.GreaterThan(projection.AvailableBalance) {
				return domain.ErrInsufficientAvailableBalance
			}
		}

		now := time.Now().UTC()
		s.Status = domain.SettlementStatusProcessing
		s.FailureReason = nil
		s.UpdatedAt = now

		if err := tx.SettlementRepository().Update(ctx, s); err != nil {
			return err
		}

		retriedSettlement = s
		return nil
	})
	if err != nil {
		u.logger.WarnContext(ctx, "settlement_retry_transition_failed",
			slog.String("settlement_id", settlementID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "settlement_retry_initiated",
		slog.String("settlement_id", settlementID.String()),
		slog.String("admin_user_id", adminUserID.String()),
	)

	return u.executePayout(ctx, retriedSettlement)
}

// executePayout dispatches the disbursement to the payout provider and posts the double-entry journal upon success.
func (u *SettlementUsecaseImpl) executePayout(
	ctx context.Context,
	settlement *domain.Settlement,
) (*domain.Settlement, error) {
	bankAccount, bErr := u.organizerBankAccountRepo.FindByOrganizerID(settlement.OrganizerID)
	accountNum := ""
	if bErr == nil && bankAccount != nil {
		accountNum = bankAccount.AccountNumberLast4
	}
	ifsc := ""
	if bErr == nil && bankAccount != nil {
		ifsc = bankAccount.IFSCCode
	}

	pReq := domain.PayoutRequest{
		SettlementID:   settlement.ID,
		OrganizerID:    settlement.OrganizerID,
		Amount:         settlement.Amount,
		Currency:       settlement.Currency,
		AccountNumber:  accountNum,
		IFSC:           ifsc,
		IdempotencyKey: fmt.Sprintf("payout:%s", settlement.ID),
	}

	payoutResp, pErr := u.payoutProvider.ProcessPayout(ctx, pReq)
	if pErr != nil || payoutResp == nil || !payoutResp.Success {
		failMsg := "payout execution failed"
		if payoutResp != nil && payoutResp.FailureReason != "" {
			failMsg = payoutResp.FailureReason
		} else if pErr != nil {
			failMsg = pErr.Error()
		}

		u.logger.ErrorContext(ctx, "settlement_payout_failed",
			slog.String("settlement_id", settlement.ID.String()),
			slog.String("failure_reason", failMsg),
			slog.Any("error", pErr),
		)

		_ = u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
			s, err := tx.SettlementRepository().FindByIDForUpdate(ctx, settlement.ID)
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			s.Status = domain.SettlementStatusFailed
			s.FailureReason = &failMsg
			s.UpdatedAt = now
			settlement = s
			return tx.SettlementRepository().Update(ctx, s)
		})

		return settlement, domain.ErrPayoutFailed
	}

	// Payout confirmed! Post double-entry financial journal and mark COMPLETED atomically
	var completedSettlement *domain.Settlement
	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		s, err := tx.SettlementRepository().FindByIDForUpdate(ctx, settlement.ID)
		if err != nil {
			return err
		}

		if !s.Status.CanTransitionTo(domain.SettlementStatusCompleted) {
			return domain.ErrInvalidSettlementStatusTransition
		}

		acc2010, err := tx.FinanceAccountRepository().FindByCode(ctx, domainfinance.AccountCodeOrganizerPayable)
		if err != nil {
			return fmt.Errorf("failed to lookup Account 2010 (Organizer Payable): %w", err)
		}
		// Account 1010 (Razorpay Clearing) represents the clearing asset pool where all customer payment
		// collections are held in the double-entry ledger. Disbursing directly from 1010 offsets this
		// clearing asset. If intermediate bank settlement is introduced in the future, a transfer journal
		// (Debit 1020 Operating Bank, Credit 1010) would precede payouts, and payouts would credit 1020.
		acc1010, err := tx.FinanceAccountRepository().FindByCode(ctx, domainfinance.AccountCodeRazorpayClearing)
		if err != nil {
			return fmt.Errorf("failed to lookup Account 1010 (Razorpay Clearing): %w", err)
		}

		now := time.Now().UTC()
		amtMoney, err := domainfinance.NewMoney(s.Amount, s.Currency)
		if err != nil {
			return err
		}

		// Debit Account 2010: reduces liability to organizer
		debitEntry, err := domainfinance.NewDebitEntry(uuid.New(), acc2010.ID(), amtMoney, &s.OrganizerID, s.EventID, now)
		if err != nil {
			return err
		}

		// Credit Account 1010: reduces asset (cash clearing)
		creditEntry, err := domainfinance.NewCreditEntry(uuid.New(), acc1010.ID(), amtMoney, nil, nil, now)
		if err != nil {
			return err
		}

		journalID := uuid.New()
		idempotencyKey := fmt.Sprintf("settlement:payout:%s", s.ID)
		desc := fmt.Sprintf("Organizer settlement payout for %s", s.OrganizerID)

		journal, err := domainfinance.NewJournalTransaction(
			journalID,
			domainfinance.TransactionTypePayout,
			domainfinance.FinancialEventOrganizerPayoutSettled,
			domainfinance.SourceTypeSettlement,
			s.ID.String(),
			idempotencyKey,
			s.Currency,
			desc,
			nil,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to construct settlement journal: %w", err)
		}

		if err := journal.AddEntry(debitEntry); err != nil {
			return err
		}
		if err := journal.AddEntry(creditEntry); err != nil {
			return err
		}

		postedJournal, err := tx.FinanceJournalRepository().RecordJournal(ctx, journal)
		if err != nil {
			u.logger.ErrorContext(ctx, "settlement_ledger_posting_failed",
				slog.String("settlement_id", s.ID.String()),
				slog.Any("error", err),
			)
			return fmt.Errorf("failed to post settlement journal: %w", err)
		}
		if postedJournal == nil || postedJournal.Status() != domainfinance.PostingStatusPosted {
			return errors.New("settlement journal was not posted")
		}

		s.Status = domain.SettlementStatusCompleted
		s.PayoutReference = &payoutResp.PayoutReference
		s.ProcessedAt = &now
		s.UpdatedAt = now

		if err := tx.SettlementRepository().Update(ctx, s); err != nil {
			return err
		}

		completedSettlement = s
		return nil
	})
	if err != nil {
		u.logger.ErrorContext(ctx, "settlement_completion_failed",
			slog.String("settlement_id", settlement.ID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "settlement_payout_completed",
		slog.String("settlement_id", completedSettlement.ID.String()),
		slog.String("organizer_id", completedSettlement.OrganizerID.String()),
		slog.String("payout_reference", *completedSettlement.PayoutReference),
		slog.String("amount", completedSettlement.Amount.String()),
	)

	return completedSettlement, nil
}

var _ domain.SettlementUsecase = (*SettlementUsecaseImpl)(nil)
