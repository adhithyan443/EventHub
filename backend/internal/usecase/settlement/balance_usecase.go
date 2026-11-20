package settlement

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrNilFinanceJournalRepository = errors.New("finance journal repository cannot be nil")
	ErrNilOrganizerRepository      = errors.New("organizer repository cannot be nil")
	ErrNilEventRepository          = errors.New("event repository cannot be nil")
	ErrNilOrganizerBankAccountRepo = errors.New("organizer bank account repository cannot be nil")
	ErrNilRefundRepository         = errors.New("refund repository cannot be nil")
	ErrInvalidUserID               = errors.New("invalid or empty user ID")
	ErrUnauthorizedOrganizer       = errors.New("organizer access required or account not active")
)

type OrganizerBalanceUsecase struct {
	journalRepo     domain.FinanceJournalRepository
	organizerRepo   domain.OrganizerRepository
	eventRepo       domain.EventRepository
	bankAccountRepo domain.OrganizerBankAccountRepository
	refundRepo      domain.RefundRepository
	settlementRepo  domain.SettlementRepository
	logger          *slog.Logger
}

func NewOrganizerBalanceUsecase(
	journalRepo domain.FinanceJournalRepository,
	organizerRepo domain.OrganizerRepository,
	eventRepo domain.EventRepository,
	bankAccountRepo domain.OrganizerBankAccountRepository,
	refundRepo domain.RefundRepository,
	logger *slog.Logger,
) (*OrganizerBalanceUsecase, error) {
	if journalRepo == nil {
		return nil, ErrNilFinanceJournalRepository
	}
	if organizerRepo == nil {
		return nil, ErrNilOrganizerRepository
	}
	if eventRepo == nil {
		return nil, ErrNilEventRepository
	}
	if bankAccountRepo == nil {
		return nil, ErrNilOrganizerBankAccountRepo
	}
	if refundRepo == nil {
		return nil, ErrNilRefundRepository
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &OrganizerBalanceUsecase{
		journalRepo:     journalRepo,
		organizerRepo:   organizerRepo,
		eventRepo:       eventRepo,
		bankAccountRepo: bankAccountRepo,
		refundRepo:      refundRepo,
		logger:          logger,
	}, nil
}

func (u *OrganizerBalanceUsecase) SetSettlementRepository(repo domain.SettlementRepository) {
	u.settlementRepo = repo
}

func (u *OrganizerBalanceUsecase) GetOrganizerBalanceProjection(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.OrganizerBalanceProjection, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}

	// Step 1: Look up authenticated organizer profile
	organizer, err := u.organizerRepo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerNotFound) {
			u.logger.WarnContext(ctx, "organizer balance access denied: organizer profile not found",
				slog.String("user_id", userID.String()),
			)
			return nil, ErrUnauthorizedOrganizer
		}
		u.logger.ErrorContext(ctx, "failed to lookup organizer by user ID",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	status := strings.ToUpper(strings.TrimSpace(organizer.Status))
	isOrganizerApproved := (status == "ACTIVE" || status == "APPROVED")

	// Step 2: Check for configured active bank account (never log full account number)
	hasActiveBankAccount := false
	bankAccount, bankErr := u.bankAccountRepo.FindByOrganizerID(organizer.ID)
	if bankErr == nil && bankAccount != nil && strings.TrimSpace(bankAccount.AccountNumberLast4) != "" {
		hasActiveBankAccount = true
	} else if bankErr != nil && !errors.Is(bankErr, domain.ErrOrganizerBankAccountNotFound) {
		u.logger.WarnContext(ctx, "failed to query organizer bank account",
			slog.String("organizer_id", organizer.ID.String()),
			slog.Any("error", bankErr),
		)
	}

	// Step 3: Query aggregate posted ledger balance for Account 2010 (Organizer Payable)
	subledger, err := u.journalRepo.GetOrganizerBalance(ctx, organizer.ID)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to query organizer subledger balance",
			slog.String("organizer_id", organizer.ID.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to query ledger balance: %w", err)
	}

	// Step 4: Query event-level subledger balances from posted ledger entries
	eventSubledgerList, err := u.journalRepo.GetOrganizerEventBalances(ctx, organizer.ID)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to query organizer event balances",
			slog.String("organizer_id", organizer.ID.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to query event ledger balances: %w", err)
	}

	eventSubledgerMap := make(map[uuid.UUID]domain.EventSubledgerBalance, len(eventSubledgerList))
	for _, eb := range eventSubledgerList {
		eventSubledgerMap[eb.EventID] = eb
	}

	// Step 5: Query organizer events for scheduling and operational lifecycle classification
	const maxFetchLimit = 10000
	organizerEvents, _, err := u.eventRepo.FindByOrganizerID(organizer.ID, 1, maxFetchLimit, "", "")
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to query organizer events for balance projection",
			slog.String("organizer_id", organizer.ID.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to query organizer events: %w", err)
	}

	now := time.Now().UTC()
	eventProjections := make([]domain.EventBalanceProjection, 0, len(organizerEvents))
	seenEvents := make(map[uuid.UUID]bool, len(organizerEvents))

	for _, event := range organizerEvents {
		seenEvents[event.ID] = true

		// Determine event lifecycle
		var lifecycle domain.EventFinancialLifecycle
		if strings.EqualFold(event.Status, domain.EventStatusCancelled) {
			lifecycle = domain.EventFinancialCancelled
		} else {
			conclusionTime := calculateEventConclusionTime(event.EventDate, event.StartTime, event.EndTime, event.IsAllDay)
			startTime := calculateEventStartTime(event.EventDate, event.StartTime)
			if now.Before(startTime) {
				lifecycle = domain.EventFinancialUpcoming
			} else if now.Before(conclusionTime) {
				lifecycle = domain.EventFinancialOngoing
			} else {
				lifecycle = domain.EventFinancialCompleted
			}
		}

		// Pending refund protection: query in-flight refunds (PENDING or PROCESSING)
		var inFlightHold decimal.Decimal = decimal.Zero
		hasInFlightRefunds := false
		refunds, rErr := u.refundRepo.FindByEventID(ctx, event.ID)
		if rErr == nil {
			for _, r := range refunds {
				if r.Status == domain.RefundStatusPending || r.Status == domain.RefundStatusProcessing {
					inFlightHold = inFlightHold.Add(r.Amount)
					hasInFlightRefunds = true
				}
			}
		} else {
			u.logger.WarnContext(ctx, "failed to inspect refunds for in-flight protection",
				slog.String("event_id", event.ID.String()),
				slog.Any("error", rErr),
			)
		}

		// Retrieve ledger subledger numbers for this event (defaults to 0 if no tickets sold)
		sub := eventSubledgerMap[event.ID]
		grossEarnings := sub.GrossEarnings
		refundsDeducted := sub.RefundsDeducted
		netPayable := sub.NetPayable

		var availableBalance decimal.Decimal = decimal.Zero
		var pendingBalance decimal.Decimal = decimal.Zero
		var ineligibilityReasons []string
		isEligible := false

		switch lifecycle {
		case domain.EventFinancialCancelled:
			// Event cancelled: funds cannot be settled while refunds remain in-flight or unposted
			if netPayable.IsPositive() {
				pendingBalance = netPayable
			}
			ineligibilityReasons = append(ineligibilityReasons, "EVENT_CANCELLED")
			if hasInFlightRefunds {
				ineligibilityReasons = append(ineligibilityReasons, "IN_FLIGHT_REFUNDS_PENDING")
			}

		case domain.EventFinancialUpcoming, domain.EventFinancialOngoing:
			// In-progress or upcoming: ticket revenues remain pending until conclusion
			if netPayable.IsPositive() {
				pendingBalance = netPayable
			}
			ineligibilityReasons = append(ineligibilityReasons, "EVENT_NOT_COMPLETED")
			if hasInFlightRefunds {
				ineligibilityReasons = append(ineligibilityReasons, "IN_FLIGHT_REFUNDS_PENDING")
			}

		case domain.EventFinancialCompleted:
			// Completed: available balance is net payable minus any in-flight refund holds and event-specific in-flight settlements
			availBeforeSettlement := netPayable.Sub(inFlightHold)
			if availBeforeSettlement.IsNegative() {
				availBeforeSettlement = decimal.Zero
			}
			var inFlightEvt decimal.Decimal = decimal.Zero
			if u.settlementRepo != nil {
				inFlightEvt, _ = u.settlementRepo.GetInFlightSettlementAmountForEvent(ctx, event.ID)
			}
			eventAvail := availBeforeSettlement.Sub(inFlightEvt)
			if eventAvail.IsNegative() {
				eventAvail = decimal.Zero
			}
			availableBalance = eventAvail
			pendingBalance = inFlightHold

			if hasInFlightRefunds {
				ineligibilityReasons = append(ineligibilityReasons, "IN_FLIGHT_REFUNDS_PENDING")
			}

			if availableBalance.IsPositive() && !hasInFlightRefunds {
				isEligible = true
			} else if availableBalance.IsZero() && len(ineligibilityReasons) == 0 {
				ineligibilityReasons = append(ineligibilityReasons, "NO_AVAILABLE_BALANCE")
			}
		}

		eventProjections = append(eventProjections, domain.EventBalanceProjection{
			EventID:              event.ID,
			EventTitle:           event.Title,
			Lifecycle:            lifecycle,
			GrossEarnings:        grossEarnings,
			RefundsDeducted:      refundsDeducted,
			NetPayable:           netPayable,
			InFlightRefundHold:   inFlightHold,
			AvailableBalance:     availableBalance,
			PendingBalance:       pendingBalance,
			Currency:             domainfinance.CurrencyINR,
			IsEligible:           isEligible,
			IneligibilityReasons: ineligibilityReasons,
		})
	}

	// Safety check: include any events that have posted ledger balances but were not in the event listing
	for _, eb := range eventSubledgerList {
		if !seenEvents[eb.EventID] {
			avail := eb.NetPayable
			if avail.IsNegative() {
				avail = decimal.Zero
			}
			eventProjections = append(eventProjections, domain.EventBalanceProjection{
				EventID:              eb.EventID,
				EventTitle:           "Archived Event",
				Lifecycle:            domain.EventFinancialCompleted,
				GrossEarnings:        eb.GrossEarnings,
				RefundsDeducted:      eb.RefundsDeducted,
				NetPayable:           eb.NetPayable,
				InFlightRefundHold:   decimal.Zero,
				AvailableBalance:     avail,
				PendingBalance:       decimal.Zero,
				Currency:             domainfinance.CurrencyINR,
				IsEligible:           avail.IsPositive(),
				IneligibilityReasons: nil,
			})
		}
	}

	// Step 6: Compute Organizer-level totals from event projections & ledger
	var totalInFlightHold decimal.Decimal = decimal.Zero
	var totalPendingBalance decimal.Decimal = decimal.Zero
	var totalAvailBeforeSettlementHold decimal.Decimal = decimal.Zero

	for _, ep := range eventProjections {
		totalInFlightHold = totalInFlightHold.Add(ep.InFlightRefundHold)
		totalPendingBalance = totalPendingBalance.Add(ep.PendingBalance)
		if ep.Lifecycle == domain.EventFinancialCompleted {
			availBefore := ep.NetPayable.Sub(ep.InFlightRefundHold)
			if availBefore.IsPositive() {
				totalAvailBeforeSettlementHold = totalAvailBeforeSettlementHold.Add(availBefore)
			}
		}
	}

	// If there were ledger-level funds not linked to any event, attribute them to available/pending
	// based on net payable
	if subledger.NetPayable.GreaterThan(totalAvailBeforeSettlementHold.Add(totalPendingBalance)) {
		diff := subledger.NetPayable.Sub(totalAvailBeforeSettlementHold.Add(totalPendingBalance))
		totalAvailBeforeSettlementHold = totalAvailBeforeSettlementHold.Add(diff)
	}

	var inFlightSettlementHold decimal.Decimal = decimal.Zero
	if u.settlementRepo != nil {
		inFlightSettlementHold, _ = u.settlementRepo.GetInFlightSettlementAmount(ctx, organizer.ID)
	}
	totalAvailableBalance := totalAvailBeforeSettlementHold.Sub(inFlightSettlementHold)
	if totalAvailableBalance.IsNegative() {
		totalAvailableBalance = decimal.Zero
	}

	// Cap total available balance by aggregate posted subledger net payable
	maxPossibleAvailable := subledger.NetPayable.Sub(totalPendingBalance).Sub(inFlightSettlementHold)
	if maxPossibleAvailable.IsNegative() {
		maxPossibleAvailable = decimal.Zero
	}
	if totalAvailableBalance.GreaterThan(maxPossibleAvailable) {
		totalAvailableBalance = maxPossibleAvailable
	}

	for i := range eventProjections {
		if eventProjections[i].AvailableBalance.GreaterThan(totalAvailableBalance) {
			eventProjections[i].AvailableBalance = totalAvailableBalance
		}
	}

	// Step 7: Evaluate overall organizer settlement eligibility
	var overallEligibility domain.SettlementEligibilityStatus
	var overallReasons []string

	if !isOrganizerApproved {
		overallReasons = append(overallReasons, "ORGANIZER_ACCOUNT_NOT_APPROVED")
	}
	if !hasActiveBankAccount {
		overallReasons = append(overallReasons, "NO_ACTIVE_BANK_ACCOUNT")
	}
	if totalInFlightHold.IsPositive() {
		overallReasons = append(overallReasons, "IN_FLIGHT_REFUNDS_PENDING")
	}

	if !isOrganizerApproved || !hasActiveBankAccount {
		overallEligibility = domain.SettlementIneligible
	} else if totalAvailableBalance.IsPositive() {
		if totalPendingBalance.IsPositive() {
			overallEligibility = domain.SettlementPartiallyEligible
		} else {
			overallEligibility = domain.SettlementEligible
		}
	} else {
		overallEligibility = domain.SettlementIneligible
		if len(overallReasons) == 0 {
			overallReasons = append(overallReasons, "NO_AVAILABLE_BALANCE")
		}
	}

	u.logger.InfoContext(ctx, "organizer_balance_projected",
		slog.String("organizer_id", organizer.ID.String()),
		slog.String("payable", subledger.NetPayable.String()),
		slog.String("available", totalAvailableBalance.String()),
		slog.String("pending", totalPendingBalance.String()),
		slog.String("eligibility_status", string(overallEligibility)),
	)

	return &domain.OrganizerBalanceProjection{
		OrganizerID:          organizer.ID,
		Currency:             domainfinance.CurrencyINR,
		GrossEarnings:        subledger.GrossEarnings,
		NetEarnings:          subledger.NetEarnings,
		OrganizerPayable:     subledger.NetPayable,
		SettledAmount:          subledger.SettledAmount,
		InFlightRefundHold:     totalInFlightHold,
		InFlightSettlementHold: inFlightSettlementHold,
		PendingBalance:         totalPendingBalance,
		AvailableBalance:     totalAvailableBalance,
		EligibilityStatus:    overallEligibility,
		EligibilityReasons:   overallReasons,
		HasActiveBankAccount: hasActiveBankAccount,
		EventBalances:        eventProjections,
	}, nil
}

func calculateEventConclusionTime(
	eventDate time.Time,
	startTime *string,
	endTime *string,
	isAllDay bool,
) time.Time {
	loc := time.UTC
	if eventDate.Location() != nil {
		loc = eventDate.Location()
	}

	if isAllDay || endTime == nil || strings.TrimSpace(*endTime) == "" {
		return time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 23, 59, 59, 999999999, loc).UTC()
	}

	tStr := strings.TrimSpace(*endTime)
	var endH, endM, endS int
	if parts := strings.Split(tStr, ":"); len(parts) >= 2 {
		endH, _ = strconv.Atoi(parts[0])
		endM, _ = strconv.Atoi(parts[1])
		if len(parts) >= 3 {
			endS, _ = strconv.Atoi(parts[2])
		}
	}

	conclusion := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), endH, endM, endS, 0, loc).UTC()

	// Overnight event support (matching event_repository.go:432)
	if startTime != nil && strings.TrimSpace(*startTime) != "" {
		startStr := strings.TrimSpace(*startTime)
		if startParts := strings.Split(startStr, ":"); len(startParts) >= 2 {
			startH, _ := strconv.Atoi(startParts[0])
			startM, _ := strconv.Atoi(startParts[1])
			if endH < startH || (endH == startH && endM < startM) {
				conclusion = conclusion.AddDate(0, 0, 1)
			}
		}
	}

	return conclusion
}

func calculateEventStartTime(
	eventDate time.Time,
	startTime *string,
) time.Time {
	loc := time.UTC
	if eventDate.Location() != nil {
		loc = eventDate.Location()
	}

	if startTime == nil || strings.TrimSpace(*startTime) == "" {
		return time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, loc).UTC()
	}

	startStr := strings.TrimSpace(*startTime)
	var startH, startM, startS int
	if parts := strings.Split(startStr, ":"); len(parts) >= 2 {
		startH, _ = strconv.Atoi(parts[0])
		startM, _ = strconv.Atoi(parts[1])
		if len(parts) >= 3 {
			startS, _ = strconv.Atoi(parts[2])
		}
	}

	return time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), startH, startM, startS, 0, loc).UTC()
}

var _ domain.OrganizerBalanceUsecase = (*OrganizerBalanceUsecase)(nil)
