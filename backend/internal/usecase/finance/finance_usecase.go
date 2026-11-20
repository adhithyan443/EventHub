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

	// 4. Resolve and validate Tax Payable account (2030) if tax is positive
	var taxAcc *domainfinance.Account
	if in.Tax.IsPositive() {
		taxAcc, err = accRepo.FindByCode(ctx, domainfinance.AccountCodeTaxPayable)
		if err != nil {
			if errors.Is(err, domain.ErrFinanceAccountNotFound) {
				return nil, domain.ErrTaxAccountNotConfigured
			}
			u.logger.ErrorContext(ctx, "failed to resolve tax payable account",
				slog.String("code", domainfinance.AccountCodeTaxPayable),
				slog.Any("error", err),
			)
			return nil, err
		}
		if !taxAcc.IsActive() {
			u.logger.ErrorContext(ctx, "tax payable account is inactive",
				slog.String("code", domainfinance.AccountCodeTaxPayable),
			)
			return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeTaxPayable)
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

	// 4th Leg: Credit 2030 (Tax Payable) for tax (if positive)
	if in.Tax.IsPositive() {
		taxMoney, err := domainfinance.NewMoney(in.Tax, domainfinance.CurrencyINR)
		if err != nil {
			return nil, err
		}
		creditTaxEntry, err := domainfinance.NewCreditEntry(
			uuid.New(),
			taxAcc.ID(),
			taxMoney,
			nil,
			&in.EventID,
			capturedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := journal.AddEntry(creditTaxEntry); err != nil {
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

// RecordRefundWithinTx records double-entry financial ledger entries for a refund
// within an existing transaction context.
func (u *FinanceUsecase) RecordRefundWithinTx(
	ctx context.Context,
	txRepos domain.TransactionRepositories,
	in domain.RecordRefundInput,
) (*domainfinance.JournalTransaction, error) {
	if txRepos == nil {
		return nil, errors.New("txRepos cannot be nil")
	}

	if err := validateRefundInput(in); err != nil {
		u.logger.WarnContext(ctx, "refund financial validation failed",
			slog.String("refund_id", in.RefundID.String()),
			slog.String("booking_id", in.BookingID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	// Zero Refund Amount: Free booking or zero refund bypasses journal creation.
	if in.RefundAmount.IsZero() {
		u.logger.InfoContext(ctx, "zero refund amount detected; bypassing financial journal creation",
			slog.String("refund_id", in.RefundID.String()),
			slog.String("booking_id", in.BookingID.String()),
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

	// Always need Razorpay Clearing account (1010)
	clearingAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeRazorpayClearing)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to resolve razorpay clearing account",
			slog.String("code", domainfinance.AccountCodeRazorpayClearing),
			slog.Any("error", err),
		)
		return nil, err
	}
	if !clearingAcc.IsActive() {
		return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeRazorpayClearing)
	}

	journalID := uuid.New()
	processedAt := in.ProcessedAt
	if processedAt.IsZero() {
		processedAt = time.Now().UTC()
	}

	refundMoney, err := domainfinance.NewMoney(in.RefundAmount, domainfinance.CurrencyINR)
	if err != nil {
		return nil, err
	}

	// Credit leg is always 1010 (Razorpay Clearing) for the total refund amount
	creditClearingEntry, err := domainfinance.NewCreditEntry(
		uuid.New(),
		clearingAcc.ID(),
		refundMoney,
		nil,
		nil,
		processedAt,
	)
	if err != nil {
		return nil, err
	}

	var journal *domainfinance.JournalTransaction

	switch {
	case in.IsEscrowRefund:
		// Scenario 3: Escrow refund for captured payment on cancelled booking
		customerRefundAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeCustomerRefundPayable)
		if err != nil {
			u.logger.ErrorContext(ctx, "failed to resolve customer refund payable account",
				slog.String("code", domainfinance.AccountCodeCustomerRefundPayable),
				slog.Any("error", err),
			)
			return nil, err
		}
		if !customerRefundAcc.IsActive() {
			return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeCustomerRefundPayable)
		}

		idempotencyKey := fmt.Sprintf("refund:%s", in.RefundID)
		description := fmt.Sprintf("Escrow refund for cancelled booking payment %s", in.PaymentID)

		journal, err = domainfinance.NewJournalTransaction(
			journalID,
			domainfinance.TransactionTypeRefund,
			domainfinance.FinancialEventEscrowRefund,
			domainfinance.SourceTypeRefund,
			in.RefundID.String(),
			idempotencyKey,
			domainfinance.CurrencyINR,
			description,
			nil,
			processedAt,
		)
		if err != nil {
			return nil, err
		}

		// Debit: 2020 Customer Refund Payable
		debitEntry, err := domainfinance.NewDebitEntry(
			uuid.New(),
			customerRefundAcc.ID(),
			refundMoney,
			nil,
			nil,
			processedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := journal.AddEntry(debitEntry); err != nil {
			return nil, err
		}
		if err := journal.AddEntry(creditClearingEntry); err != nil {
			return nil, err
		}

	case in.IsEventCancellation:
		// Scenario 2: Event cancellation bulk refund (ticket refund + platform fee refund + tax refund)
		organizerPortion := in.RefundAmount.Sub(in.PlatformFeeRefund).Sub(in.TaxRefund)
		idempotencyKey := fmt.Sprintf("refund:%s", in.RefundID)
		description := fmt.Sprintf("Event cancellation refund for booking %s", in.BookingID)

		journal, err = domainfinance.NewJournalTransaction(
			journalID,
			domainfinance.TransactionTypeRefund,
			domainfinance.FinancialEventEventCancellationRefund,
			domainfinance.SourceTypeRefund,
			in.RefundID.String(),
			idempotencyKey,
			domainfinance.CurrencyINR,
			description,
			nil,
			processedAt,
		)
		if err != nil {
			return nil, err
		}

		// Debit Organizer Payable (2010) for ticket portion if positive
		if organizerPortion.IsPositive() {
			organizerAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeOrganizerPayable)
			if err != nil {
				return nil, err
			}
			if !organizerAcc.IsActive() {
				return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeOrganizerPayable)
			}
			orgMoney, err := domainfinance.NewMoney(organizerPortion, domainfinance.CurrencyINR)
			if err != nil {
				return nil, err
			}
			debitOrgEntry, err := domainfinance.NewDebitEntry(
				uuid.New(),
				organizerAcc.ID(),
				orgMoney,
				&in.OrganizerID,
				&in.EventID,
				processedAt,
			)
			if err != nil {
				return nil, err
			}
			if err := journal.AddEntry(debitOrgEntry); err != nil {
				return nil, err
			}
		}

		// Debit Platform Fee Revenue (4010) for platform fee portion if positive
		if in.PlatformFeeRefund.IsPositive() {
			platformFeeAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodePlatformFeeRevenue)
			if err != nil {
				return nil, err
			}
			if !platformFeeAcc.IsActive() {
				return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodePlatformFeeRevenue)
			}
			feeMoney, err := domainfinance.NewMoney(in.PlatformFeeRefund, domainfinance.CurrencyINR)
			if err != nil {
				return nil, err
			}
			debitFeeEntry, err := domainfinance.NewDebitEntry(
				uuid.New(),
				platformFeeAcc.ID(),
				feeMoney,
				nil,
				&in.EventID,
				processedAt,
			)
			if err != nil {
				return nil, err
			}
			if err := journal.AddEntry(debitFeeEntry); err != nil {
				return nil, err
			}
		}

		// Debit Tax Payable (2030) for tax portion if positive
		if in.TaxRefund.IsPositive() {
			taxAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeTaxPayable)
			if err != nil {
				return nil, err
			}
			if !taxAcc.IsActive() {
				return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeTaxPayable)
			}
			taxMoney, err := domainfinance.NewMoney(in.TaxRefund, domainfinance.CurrencyINR)
			if err != nil {
				return nil, err
			}
			debitTaxEntry, err := domainfinance.NewDebitEntry(
				uuid.New(),
				taxAcc.ID(),
				taxMoney,
				nil,
				&in.EventID,
				processedAt,
			)
			if err != nil {
				return nil, err
			}
			if err := journal.AddEntry(debitTaxEntry); err != nil {
				return nil, err
			}
		}

		if err := journal.AddEntry(creditClearingEntry); err != nil {
			return nil, err
		}

	default:
		// Scenario 1: Customer-initiated cancellation refund (proportional ticket, platform fee, and tax)
		organizerPortion := in.RefundAmount.Sub(in.PlatformFeeRefund).Sub(in.TaxRefund)
		idempotencyKey := fmt.Sprintf("refund:%s", in.RefundID)
		description := fmt.Sprintf("Customer cancellation refund for booking %s", in.BookingID)

		journal, err = domainfinance.NewJournalTransaction(
			journalID,
			domainfinance.TransactionTypeRefund,
			domainfinance.FinancialEventCustomerRefund,
			domainfinance.SourceTypeRefund,
			in.RefundID.String(),
			idempotencyKey,
			domainfinance.CurrencyINR,
			description,
			nil,
			processedAt,
		)
		if err != nil {
			return nil, err
		}

		// Debit Organizer Payable (2010) for ticket portion if positive
		if organizerPortion.IsPositive() {
			organizerAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeOrganizerPayable)
			if err != nil {
				return nil, err
			}
			if !organizerAcc.IsActive() {
				return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeOrganizerPayable)
			}
			orgMoney, err := domainfinance.NewMoney(organizerPortion, domainfinance.CurrencyINR)
			if err != nil {
				return nil, err
			}
			debitOrgEntry, err := domainfinance.NewDebitEntry(
				uuid.New(),
				organizerAcc.ID(),
				orgMoney,
				&in.OrganizerID,
				&in.EventID,
				processedAt,
			)
			if err != nil {
				return nil, err
			}
			if err := journal.AddEntry(debitOrgEntry); err != nil {
				return nil, err
			}
		}

		// Debit Platform Fee Revenue (4010) for platform fee portion if positive
		if in.PlatformFeeRefund.IsPositive() {
			platformFeeAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodePlatformFeeRevenue)
			if err != nil {
				return nil, err
			}
			if !platformFeeAcc.IsActive() {
				return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodePlatformFeeRevenue)
			}
			feeMoney, err := domainfinance.NewMoney(in.PlatformFeeRefund, domainfinance.CurrencyINR)
			if err != nil {
				return nil, err
			}
			debitFeeEntry, err := domainfinance.NewDebitEntry(
				uuid.New(),
				platformFeeAcc.ID(),
				feeMoney,
				nil,
				&in.EventID,
				processedAt,
			)
			if err != nil {
				return nil, err
			}
			if err := journal.AddEntry(debitFeeEntry); err != nil {
				return nil, err
			}
		}

		// Debit Tax Payable (2030) for tax portion if positive
		if in.TaxRefund.IsPositive() {
			taxAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeTaxPayable)
			if err != nil {
				return nil, err
			}
			if !taxAcc.IsActive() {
				return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeTaxPayable)
			}
			taxMoney, err := domainfinance.NewMoney(in.TaxRefund, domainfinance.CurrencyINR)
			if err != nil {
				return nil, err
			}
			debitTaxEntry, err := domainfinance.NewDebitEntry(
				uuid.New(),
				taxAcc.ID(),
				taxMoney,
				nil,
				&in.EventID,
				processedAt,
			)
			if err != nil {
				return nil, err
			}
			if err := journal.AddEntry(debitTaxEntry); err != nil {
				return nil, err
			}
		}

		if err := journal.AddEntry(creditClearingEntry); err != nil {
			return nil, err
		}
	}

	// Post the balanced journal
	if err := journal.Post(processedAt); err != nil {
		u.logger.ErrorContext(ctx, "failed to post refund journal",
			slog.String("journal_id", journalID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	// Persist the journal header and entries atomically via repository
	savedJournal, err := journalRepo.RecordJournal(ctx, journal)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to record finance refund journal",
			slog.String("journal_id", journalID.String()),
			slog.String("refund_id", in.RefundID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "finance refund journal recorded successfully",
		slog.String("journal_id", savedJournal.ID().String()),
		slog.String("refund_id", in.RefundID.String()),
		slog.String("amount", in.RefundAmount.String()),
	)

	return savedJournal, nil
}

// RecordRefund executes RecordRefundWithinTx inside a new database transaction.
func (u *FinanceUsecase) RecordRefund(
	ctx context.Context,
	in domain.RecordRefundInput,
) (*domainfinance.JournalTransaction, error) {
	var savedJournal *domainfinance.JournalTransaction
	err := u.txManager.WithinTransaction(func(txRepos domain.TransactionRepositories) error {
		var txErr error
		savedJournal, txErr = u.RecordRefundWithinTx(ctx, txRepos, in)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	return savedJournal, nil
}

// RecordEscrowCaptureWithinTx records double-entry financial ledger entries for incoming funds
// captured for a cancelled booking into escrow liability.
func (u *FinanceUsecase) RecordEscrowCaptureWithinTx(
	ctx context.Context,
	txRepos domain.TransactionRepositories,
	in domain.RecordEscrowCaptureInput,
) (*domainfinance.JournalTransaction, error) {
	if txRepos == nil {
		return nil, errors.New("txRepos cannot be nil")
	}

	if in.PaymentID == uuid.Nil {
		return nil, errors.New("payment ID cannot be nil")
	}
	if in.BookingID == uuid.Nil {
		return nil, errors.New("booking ID cannot be nil")
	}
	if in.PaymentAmount.IsNegative() {
		return nil, errors.New("payment amount cannot be negative")
	}
	if !in.PaymentAmount.Truncate(2).Equal(in.PaymentAmount) {
		return nil, fmt.Errorf("payment amount has fractional paise: %s", in.PaymentAmount.String())
	}
	cleanCurrency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if cleanCurrency == "" {
		cleanCurrency = domainfinance.CurrencyINR
	}
	if cleanCurrency != domainfinance.CurrencyINR {
		return nil, domainfinance.ErrUnsupportedCurrency
	}

	if in.PaymentAmount.IsZero() {
		u.logger.InfoContext(ctx, "zero payment amount detected for escrow capture; bypassing financial journal creation",
			slog.String("payment_id", in.PaymentID.String()),
			slog.String("booking_id", in.BookingID.String()),
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

	clearingAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeRazorpayClearing)
	if err != nil {
		return nil, err
	}
	if !clearingAcc.IsActive() {
		return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeRazorpayClearing)
	}

	customerRefundAcc, err := accRepo.FindByCode(ctx, domainfinance.AccountCodeCustomerRefundPayable)
	if err != nil {
		return nil, err
	}
	if !customerRefundAcc.IsActive() {
		return nil, fmt.Errorf("%w: account %s is inactive", domain.ErrFinanceAccountInactive, domainfinance.AccountCodeCustomerRefundPayable)
	}

	journalID := uuid.New()
	capturedAt := in.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}

	idempotencyKey := fmt.Sprintf("escrow_capture:%s", in.PaymentID)
	description := fmt.Sprintf("Escrow capture for cancelled booking payment %s", in.PaymentID)

	journal, err := domainfinance.NewJournalTransaction(
		journalID,
		domainfinance.TransactionTypePayment,
		domainfinance.FinancialEventEscrowCapture,
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

	money, err := domainfinance.NewMoney(in.PaymentAmount, domainfinance.CurrencyINR)
	if err != nil {
		return nil, err
	}

	// 1st Leg: Debit 1010 (Razorpay Clearing)
	debitEntry, err := domainfinance.NewDebitEntry(
		uuid.New(),
		clearingAcc.ID(),
		money,
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

	// 2nd Leg: Credit 2020 (Customer Refund Payable)
	creditEntry, err := domainfinance.NewCreditEntry(
		uuid.New(),
		customerRefundAcc.ID(),
		money,
		nil,
		nil,
		capturedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := journal.AddEntry(creditEntry); err != nil {
		return nil, err
	}

	// Post the balanced journal
	if err := journal.Post(capturedAt); err != nil {
		u.logger.ErrorContext(ctx, "failed to post escrow capture journal",
			slog.String("journal_id", journalID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	savedJournal, err := journalRepo.RecordJournal(ctx, journal)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to record finance escrow capture journal",
			slog.String("journal_id", journalID.String()),
			slog.String("payment_id", in.PaymentID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "finance escrow capture journal recorded successfully",
		slog.String("journal_id", savedJournal.ID().String()),
		slog.String("payment_id", in.PaymentID.String()),
		slog.String("amount", in.PaymentAmount.String()),
	)

	return savedJournal, nil
}

// RecordEscrowCapture executes RecordEscrowCaptureWithinTx inside a new database transaction.
func (u *FinanceUsecase) RecordEscrowCapture(
	ctx context.Context,
	in domain.RecordEscrowCaptureInput,
) (*domainfinance.JournalTransaction, error) {
	var savedJournal *domainfinance.JournalTransaction
	err := u.txManager.WithinTransaction(func(txRepos domain.TransactionRepositories) error {
		var txErr error
		savedJournal, txErr = u.RecordEscrowCaptureWithinTx(ctx, txRepos, in)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	return savedJournal, nil
}

func validateRefundInput(in domain.RecordRefundInput) error {
	if in.RefundID == uuid.Nil {
		return errors.New("refund ID cannot be nil")
	}
	if in.PaymentID == uuid.Nil {
		return errors.New("payment ID cannot be nil")
	}
	if in.BookingID == uuid.Nil {
		return errors.New("booking ID cannot be nil")
	}

	cleanCurrency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if cleanCurrency != "" && cleanCurrency != domainfinance.CurrencyINR {
		return domainfinance.ErrUnsupportedCurrency
	}

	if in.RefundAmount.IsNegative() {
		return domain.ErrInvalidRefundAmountInput
	}
	if in.PlatformFeeRefund.IsNegative() {
		return domain.ErrInvalidPlatformFeeRefundAmount
	}
	if in.PlatformFeeRefund.GreaterThan(in.RefundAmount) {
		return domain.ErrInvalidPlatformFeeRefundAmount
	}
	if in.TaxRefund.IsNegative() {
		return domain.ErrInvalidTaxRefundAmount
	}
	if in.PlatformFeeRefund.Add(in.TaxRefund).GreaterThan(in.RefundAmount) {
		return domain.ErrInvalidRefundAmountInput
	}

	if !in.RefundAmount.Truncate(2).Equal(in.RefundAmount) {
		return fmt.Errorf("refund amount has fractional paise: %s", in.RefundAmount.String())
	}
	if !in.PlatformFeeRefund.Truncate(2).Equal(in.PlatformFeeRefund) {
		return fmt.Errorf("platform fee refund has fractional paise: %s", in.PlatformFeeRefund.String())
	}
	if !in.TaxRefund.Truncate(2).Equal(in.TaxRefund) {
		return fmt.Errorf("tax refund has fractional paise: %s", in.TaxRefund.String())
	}

	if !in.IsEscrowRefund {
		if in.EventID == uuid.Nil {
			return errors.New("event ID cannot be nil")
		}
		if in.OrganizerID == uuid.Nil {
			return errors.New("organizer ID cannot be nil")
		}
	}

	return nil
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

	// Discount check: Promo discount expense accounts not configured
	if in.Discount.IsPositive() {
		return domain.ErrUnsupportedDiscountAllocation
	}

	return nil
}
