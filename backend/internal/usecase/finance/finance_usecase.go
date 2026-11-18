package finance

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

// FinanceUsecase implements domain.FinanceUsecase for recording financial transactions
// into EventHub's double-entry general ledger.
type FinanceUsecase struct {
	txManager domain.TransactionManager
	logger    *slog.Logger
}

// NewFinanceUsecase creates a new validated FinanceUsecase.
func NewFinanceUsecase(txManager domain.TransactionManager, logger *slog.Logger) (*FinanceUsecase, error) {
	if txManager == nil {
		return nil, errors.New("txManager cannot be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &FinanceUsecase{
		txManager: txManager,
		logger:    logger,
	}, nil
}

// RecordPaymentCaptureWithinTx records the financial ledger entries for a captured payment
// within an existing transaction context.
func (u *FinanceUsecase) RecordPaymentCaptureWithinTx(
	ctx context.Context,
	txRepos domain.TransactionRepositories,
	in domain.RecordPaymentCaptureInput,
) (*domainfinance.JournalTransaction, error) {
	if txRepos == nil {
		return nil, errors.New("txRepos cannot be nil")
	}

	if err := validatePaymentCaptureInput(in); err != nil {
		u.logger.WarnContext(ctx, "payment capture financial validation failed",
			slog.String("payment_id", in.PaymentID.String()),
			slog.String("booking_id", in.BookingID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	// Free Booking Bypass: Zero booking total and zero payment amount return
	// successfully without creating a financial journal or fetching ledger accounts.
	if in.Total.IsZero() && in.PaymentAmount.IsZero() {
		u.logger.InfoContext(ctx, "free booking detected; bypassing financial journal creation",
			slog.String("payment_id", in.PaymentID.String()),
			slog.String("booking_id", in.BookingID.String()),
			slog.String("event_id", in.EventID.String()),
		)
		return nil, nil
	}

	accRepo := txRepos.FinanceAccountRepository()
	if accRepo == nil {
		return nil, errors.New("FinanceAccountRepository cannot be nil")
	}
	journalRepo := txRepos.FinanceJournalRepository()
	if journalRepo == nil {
		return nil, errors.New("FinanceJournalRepository cannot be nil")
	}

	// 1. Resolve and validate Razorpay Clearing account (1010)
	clearingAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeRazorpayClearing)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to resolve razorpay clearing account",
			slog.String("code", domainfinance.AccountCodeRazorpayClearing),
			slog.Any("error", err),
		)
		return nil, err
	}
	if !clearingAcc.IsActive() {
		u.logger.ErrorContext(ctx, "razorpay clearing account is inactive",
			slog.String("code", domainfinance.AccountCodeRazorpayClearing),
		)
		return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeRazorpayClearing)
	}

	// 2. Resolve and validate Organizer Payable account (2010)
	organizerAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeOrganizerPayable)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to resolve organizer payable account",
			slog.String("code", domainfinance.AccountCodeOrganizerPayable),
			slog.Any("error", err),
		)
		return nil, err
	}
	if !organizerAcc.IsActive() {
		u.logger.ErrorContext(ctx, "organizer payable account is inactive",
			slog.String("code", domainfinance.AccountCodeOrganizerPayable),
		)
		return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeOrganizerPayable)
	}

	// 3. Resolve and validate Platform Fee Revenue account (4010) if platform fee is positive
	var platformFeeAcc *domainfinance.Account
	if in.PlatformFee.IsPositive() {
		platformFeeAcc, err = accRepo.FindByCode(ctx, domainfinance.AccountCodePlatformFeeRevenue)
		if err != nil {
			u.logger.ErrorContext(ctx, "failed to resolve platform fee revenue account",
				slog.String("code", domainfinance.AccountCodePlatformFeeRevenue),
				slog.Any("error", err),
			)
			return nil, err
		}
		if !platformFeeAcc.IsActive() {
			u.logger.ErrorContext(ctx, "platform fee revenue account is inactive",
				slog.String("code", domainfinance.AccountCodePlatformFeeRevenue),
			)
			return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodePlatformFeeRevenue)
		}
	}

	journalID := uuid.New()
	idempotencyKey := fmt.Sprintf("payment_capture:%s", in.PaymentID)
	capturedAt := in.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}

	description := fmt.Sprintf("Customer payment captured for booking %s", in.BookingID)

	journal, err := domainfinance.NewJournalTransaction(
		journalID,
		domainfinance.TransactionTypePayment,
		domainfinance.FinancialEventTicketSaleCaptured,
		domainfinance.SourceTypePayment,
		in.PaymentID.String(),
		idempotencyKey,
		domainfinance.CurrencyINR,
		description,
		nil,
		capturedAt,
	)
	if err != nil {
		return nil, err
	}

	paymentMoney, err := domainfinance.NewMoney(in.PaymentAmount, domainfinance.CurrencyINR)
	if err != nil {
		return nil, err
	}

	netTicketAmount := in.Subtotal.Sub(in.Discount)
	organizerMoney, err := domainfinance.NewMoney(netTicketAmount, domainfinance.CurrencyINR)
	if err != nil {
		return nil, err
	}

	// 1st Leg: Debit 1010 (Razorpay Clearing) for total payment amount
	debitEntry, err := domainfinance.NewDebitEntry(
		uuid.New(),
		clearingAcc.ID(),
		paymentMoney,
		nil,
		nil,
		capturedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := journal.AddEntry(debitEntry); err != nil {
		return nil, err
	}

	// 2nd Leg: Credit 2010 (Organizer Payable) for net ticket revenue (subledger: organizerID, eventID)
	creditOrgEntry, err := domainfinance.NewCreditEntry(
		uuid.New(),
		organizerAcc.ID(),
		organizerMoney,
		&in.OrganizerID,
		&in.EventID,
		capturedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := journal.AddEntry(creditOrgEntry); err != nil {
		return nil, err
	}

	// 3rd Leg: Credit 4010 (Platform Fee Revenue) for platform fee (if positive)
	if in.PlatformFee.IsPositive() {
		feeMoney, err := domainfinance.NewMoney(in.PlatformFee, domainfinance.CurrencyINR)
		if err != nil {
			return nil, err
		}
		creditFeeEntry, err := domainfinance.NewCreditEntry(
			uuid.New(),
			platformFeeAcc.ID(),
			feeMoney,
			nil,
			&in.EventID,
			capturedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := journal.AddEntry(creditFeeEntry); err != nil {
			return nil, err
		}
	}

	// Post the balanced journal
	if err := journal.Post(capturedAt); err != nil {
		u.logger.ErrorContext(ctx, "failed to post journal",
			slog.String("journal_id", journalID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	// Persist the journal header and entries atomically via repository
	savedJournal, err := journalRepo.RecordJournal(ctx, journal)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to record finance journal",
			slog.String("journal_id", journalID.String()),
			slog.String("idempotency_key", idempotencyKey),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "finance journal recorded successfully",
		slog.String("journal_id", savedJournal.ID().String()),
		slog.String("payment_id", in.PaymentID.String()),
		slog.String("booking_id", in.BookingID.String()),
		slog.String("total", in.Total.String()),
	)

	return savedJournal, nil
}

// RecordPaymentCapture executes RecordPaymentCaptureWithinTx inside a new database transaction.
func (u *FinanceUsecase) RecordPaymentCapture(
	ctx context.Context,
	in domain.RecordPaymentCaptureInput,
) (*domainfinance.JournalTransaction, error) {
	var savedJournal *domainfinance.JournalTransaction
	err := u.txManager.WithinTransaction(func(txRepos domain.TransactionRepositories) error {
		var txErr error
		savedJournal, txErr = u.RecordPaymentCaptureWithinTx(ctx, txRepos, in)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	return savedJournal, nil
}

func validatePaymentCaptureInput(in domain.RecordPaymentCaptureInput) error {
	if in.PaymentID == uuid.Nil {
		return errors.New("payment ID cannot be nil")
	}
	if in.BookingID == uuid.Nil {
		return errors.New("booking ID cannot be nil")
	}
	if in.EventID == uuid.Nil {
		return errors.New("event ID cannot be nil")
	}
	if in.OrganizerID == uuid.Nil {
		return errors.New("organizer ID cannot be nil")
	}
	if in.UserID == uuid.Nil {
		return errors.New("user ID cannot be nil")
	}

	cleanCurrency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if cleanCurrency != domainfinance.CurrencyINR {
		return domainfinance.ErrUnsupportedCurrency
	}

	if in.PaymentAmount.IsNegative() {
		return errors.New("payment amount cannot be negative")
	}
	if in.Total.IsNegative() {
		return errors.New("booking total cannot be negative")
	}
	if in.Subtotal.IsNegative() {
		return errors.New("booking subtotal cannot be negative")
	}
	if in.PlatformFee.IsNegative() {
		return errors.New("booking platform fee cannot be negative")
	}
	if in.Tax.IsNegative() {
		return errors.New("booking tax cannot be negative")
	}
	if in.Discount.IsNegative() {
		return errors.New("booking discount cannot be negative")
	}

	// Precision checks: exact paise (at most 2 decimal places)
	amounts := map[string]decimal.Decimal{
		"PaymentAmount": in.PaymentAmount,
		"Total":         in.Total,
		"Subtotal":      in.Subtotal,
		"PlatformFee":   in.PlatformFee,
		"Tax":           in.Tax,
		"Discount":      in.Discount,
	}
	for name, val := range amounts {
		if !val.Truncate(2).Equal(val) {
			return fmt.Errorf("%s has fractional paise: %s", name, val.String())
		}
	}

	// Double-entry breakdown equality: Subtotal + PlatformFee + Tax - Discount == Total
	expectedTotal := in.Subtotal.Add(in.PlatformFee).Add(in.Tax).Sub(in.Discount)
	if !expectedTotal.Equal(in.Total) {
		return fmt.Errorf("%w: breakdown sum (%s) does not equal total (%s)",
			domain.ErrInconsistentBookingAmounts, expectedTotal, in.Total)
	}

	// Payment amount must equal booking total
	if !in.PaymentAmount.Equal(in.Total) {
		return fmt.Errorf("%w: payment amount (%s) does not equal total (%s)",
			domain.ErrInconsistentBookingAmounts, in.PaymentAmount, in.Total)
	}

	// Tax check: Tax account not configured in chart of accounts
	if in.Tax.IsPositive() {
		return domain.ErrTaxAccountNotConfigured
	}

	// Discount check: Promo discount expense accounts not configured
	if in.Discount.IsPositive() {
		return domain.ErrUnsupportedDiscountAllocation
	}

	return nil
}
