package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type financeJournalRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewFinanceJournalRepository creates a new PostgreSQL-backed FinanceJournalRepository.
func NewFinanceJournalRepository(db *gorm.DB, logger *slog.Logger) domain.FinanceJournalRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &financeJournalRepository{
		db:     db,
		logger: logger,
	}
}

type canonicalEntry struct {
	AccountID   string `json:"account_id"`
	EntryType   string `json:"entry_type"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	OrganizerID string `json:"organizer_id,omitempty"`
	EventID     string `json:"event_id,omitempty"`
}

type canonicalJournal struct {
	TransactionType     string           `json:"transaction_type"`
	FinancialEventType  string           `json:"financial_event_type"`
	SourceType          string           `json:"source_type"`
	SourceID            string           `json:"source_id"`
	Currency            string           `json:"currency"`
	Description         string           `json:"description"`
	ReversalReferenceID string           `json:"reversal_reference_id,omitempty"`
	Entries             []canonicalEntry `json:"entries"`
}

func computeJournalFingerprint(j *finance.JournalTransaction) (string, error) {
	if j == nil {
		return "", errors.New("cannot compute fingerprint for nil journal")
	}

	entries := j.Entries()
	canonicalEntries := make([]canonicalEntry, len(entries))
	for i, e := range entries {
		var orgID string
		if e.OrganizerID() != nil {
			orgID = e.OrganizerID().String()
		}
		var evtID string
		if e.EventID() != nil {
			evtID = e.EventID().String()
		}

		canonicalEntries[i] = canonicalEntry{
			AccountID:   e.AccountID().String(),
			EntryType:   string(e.EntryType()),
			Amount:      e.Amount().Amount().StringFixed(2),
			Currency:    e.Currency(),
			OrganizerID: orgID,
			EventID:     evtID,
		}
	}

	sort.Slice(canonicalEntries, func(i, j int) bool {
		if canonicalEntries[i].AccountID != canonicalEntries[j].AccountID {
			return canonicalEntries[i].AccountID < canonicalEntries[j].AccountID
		}
		if canonicalEntries[i].EntryType != canonicalEntries[j].EntryType {
			return canonicalEntries[i].EntryType < canonicalEntries[j].EntryType
		}
		if canonicalEntries[i].Amount != canonicalEntries[j].Amount {
			return canonicalEntries[i].Amount < canonicalEntries[j].Amount
		}
		if canonicalEntries[i].OrganizerID != canonicalEntries[j].OrganizerID {
			return canonicalEntries[i].OrganizerID < canonicalEntries[j].OrganizerID
		}
		return canonicalEntries[i].EventID < canonicalEntries[j].EventID
	})

	var revRef string
	if j.ReversalReferenceID() != nil {
		revRef = j.ReversalReferenceID().String()
	}

	canonical := canonicalJournal{
		TransactionType:     string(j.TransactionType()),
		FinancialEventType:  string(j.FinancialEventType()),
		SourceType:          string(j.SourceType()),
		SourceID:            j.SourceID(),
		Currency:            j.Currency(),
		Description:         j.Description(),
		ReversalReferenceID: revRef,
		Entries:             canonicalEntries,
	}

	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}

func (r *financeJournalRepository) RecordJournal(
	ctx context.Context,
	journal *finance.JournalTransaction,
) (*finance.JournalTransaction, error) {
	if journal == nil {
		return nil, errors.New("cannot record nil journal")
	}

	if err := journal.Validate(); err != nil {
		return nil, fmt.Errorf("domain validation failed: %w", err)
	}

	balanced, balErr := journal.IsBalanced()
	if balErr != nil {
		return nil, balErr
	}
	if !balanced {
		return nil, finance.ErrUnbalancedJournal
	}

	fingerprint, err := computeJournalFingerprint(journal)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to compute journal fingerprint", slog.Any("error", err))
		return nil, fmt.Errorf("failed to compute fingerprint: %w", err)
	}

	postedAt := time.Now().UTC()
	if postedAt.Before(journal.CreatedAt()) {
		postedAt = journal.CreatedAt()
	}

	var result *finance.JournalTransaction

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		const spIdempotency = "sp_finance_idempotency"

		if spErr := tx.SavePoint(spIdempotency).Error; spErr != nil {
			r.logger.ErrorContext(ctx, "failed to create idempotency savepoint", slog.Any("error", spErr))
			return fmt.Errorf("failed to create savepoint: %w", spErr)
		}

		idempotencyRow := models.FinanceIdempotencyKeyModel{
			IdempotencyKey: journal.IdempotencyKey(),
			Fingerprint:    fingerprint,
			JournalID:      journal.ID(),
			CreatedAt:      postedAt,
		}

		insertErr := tx.Create(&idempotencyRow).Error
		if insertErr != nil {
			var pgErr *pgconn.PgError
			if errors.As(insertErr, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "pk_finance_idempotency_keys" {
				if rbErr := tx.RollbackTo(spIdempotency).Error; rbErr != nil {
					r.logger.ErrorContext(ctx, "savepoint rollback failed",
						slog.Any("error", rbErr),
						slog.Any("original_error", insertErr),
					)
					return errors.Join(
						fmt.Errorf("savepoint rollback failed: %w", rbErr),
						fmt.Errorf("idempotency collision: %w", insertErr),
					)
				}

				var existing models.FinanceIdempotencyKeyModel
				cleanTx := tx.Session(&gorm.Session{})
				if findErr := cleanTx.Where("idempotency_key = ?", journal.IdempotencyKey()).First(&existing).Error; findErr != nil {
					r.logger.ErrorContext(ctx, "failed to retrieve existing idempotency record", slog.Any("error", findErr))
					return fmt.Errorf("failed to query existing idempotency record: %w", findErr)
				}

				if existing.Fingerprint != fingerprint {
					r.logger.WarnContext(ctx, "idempotency key conflict: payload fingerprint mismatch")
					return domain.ErrIdempotencyConflict
				}

				existingJournal, getErr := r.getJournalByIDWithinDB(ctx, cleanTx, existing.JournalID)
				if getErr != nil {
					r.logger.ErrorContext(ctx, "failed to load existing journal for replay", slog.Any("error", getErr))
					return fmt.Errorf("failed to load existing journal: %w", getErr)
				}

				result = existingJournal
				return nil
			}

			_ = tx.RollbackTo(spIdempotency)
			r.logger.ErrorContext(ctx, "unexpected error inserting idempotency key", slog.Any("error", insertErr))
			return fmt.Errorf("failed to insert idempotency key: %w", insertErr)
		}

		if relErr := tx.Exec("RELEASE SAVEPOINT " + spIdempotency).Error; relErr != nil {
			r.logger.ErrorContext(ctx, "failed to release savepoint", slog.Any("error", relErr))
			return fmt.Errorf("failed to release savepoint: %w", relErr)
		}

		// Reversal Validation: If this is a reversal, validate original journal using domain MarkReversed()
		if journal.TransactionType() == finance.TransactionTypeReversal {
			if journal.ReversalReferenceID() == nil {
				return finance.ErrInvalidReversalReference
			}
			origID := *journal.ReversalReferenceID()

			origJournal, origErr := r.getJournalByIDWithinDB(ctx, tx, origID)
			if origErr != nil {
				if errors.Is(origErr, domain.ErrFinanceJournalNotFound) {
					return finance.ErrInvalidReversalReference
				}
				return origErr
			}

			// Domain MarkReversed requires a POSTED reversal journal with postedAt set
			postedReversalSnapshot, revSnapErr := finance.ReconstituteJournalTransaction(
				journal.ID(),
				journal.TransactionType(),
				journal.FinancialEventType(),
				journal.SourceType(),
				journal.SourceID(),
				journal.IdempotencyKey(),
				journal.Currency(),
				journal.Description(),
				journal.Entries(),
				journal.ReversalReferenceID(),
				finance.PostingStatusPosted,
				journal.CreatedAt(),
				&postedAt,
				false,
				nil,
				nil,
			)
			if revSnapErr != nil {
				return revSnapErr
			}

			if markErr := origJournal.MarkReversed(postedReversalSnapshot); markErr != nil {
				r.logger.WarnContext(ctx, "reversal domain validation failed", slog.Any("error", markErr))
				return markErr
			}
		}

		// Insert Journal header as DRAFT
		journalModel := models.FinanceJournalModel{
			ID:                     journal.ID(),
			TransactionType:        string(journal.TransactionType()),
			FinancialEventType:     string(journal.FinancialEventType()),
			SourceType:             string(journal.SourceType()),
			SourceID:               journal.SourceID(),
			IdempotencyKey:         journal.IdempotencyKey(),
			IdempotencyFingerprint: fingerprint,
			Currency:               journal.Currency(),
			Description:            journal.Description(),
			Status:                 string(finance.PostingStatusDraft),
			CreatedAt:              journal.CreatedAt(),
			ReversalReferenceID:    journal.ReversalReferenceID(),
		}

		if insertJErr := tx.Create(&journalModel).Error; insertJErr != nil {
			var pgErr *pgconn.PgError
			if errors.As(insertJErr, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_finance_journals_unique_reversal_ref" {
				r.logger.WarnContext(ctx, "reversal journal already exists for original journal")
				return finance.ErrJournalAlreadyReversed
			}
			r.logger.ErrorContext(ctx, "failed to insert draft journal header", slog.Any("error", insertJErr))
			return fmt.Errorf("failed to insert draft journal: %w", insertJErr)
		}

		// Insert Ledger Entries in deterministic order
		entries := journal.Entries()
		entryModels := make([]models.FinanceLedgerEntryModel, len(entries))
		for i, e := range entries {
			entryModels[i] = models.FinanceLedgerEntryModel{
				ID:          e.ID(),
				JournalID:   journal.ID(),
				AccountID:   e.AccountID(),
				EntryType:   string(e.EntryType()),
				Amount:      e.Amount().Amount(),
				Currency:    e.Currency(),
				OrganizerID: e.OrganizerID(),
				EventID:     e.EventID(),
				CreatedAt:   e.CreatedAt(),
			}
		}

		if insertEErr := tx.Create(&entryModels).Error; insertEErr != nil {
			r.logger.ErrorContext(ctx, "failed to insert ledger entries", slog.Any("error", insertEErr))
			return fmt.Errorf("failed to insert ledger entries: %w", insertEErr)
		}

		// Transition to POSTED
		postRes := tx.Model(&models.FinanceJournalModel{}).
			Where("id = ? AND status = ?", journal.ID(), string(finance.PostingStatusDraft)).
			Updates(map[string]interface{}{
				"status":    string(finance.PostingStatusPosted),
				"posted_at": postedAt,
			})

		if postRes.Error != nil {
			r.logger.ErrorContext(ctx, "failed to transition journal to POSTED", slog.Any("error", postRes.Error))
			return fmt.Errorf("failed to post journal: %w", postRes.Error)
		}

		if postRes.RowsAffected != 1 {
			r.logger.ErrorContext(ctx, "concurrent modification: expected 1 row updated to POSTED")
			return domain.ErrConcurrentPosting
		}

		// If Reversal, atomically update original journal in DB
		if journal.TransactionType() == finance.TransactionTypeReversal {
			origID := *journal.ReversalReferenceID()
			revRes := tx.Model(&models.FinanceJournalModel{}).
				Where("id = ? AND status = ? AND is_reversed = FALSE", origID, string(finance.PostingStatusPosted)).
				Updates(map[string]interface{}{
					"is_reversed":            true,
					"reversed_by_journal_id": journal.ID(),
					"reversed_at":            postedAt,
				})

			if revRes.Error != nil {
				r.logger.ErrorContext(ctx, "failed to update original journal reversal state", slog.Any("error", revRes.Error))
				return fmt.Errorf("failed to mark original journal reversed: %w", revRes.Error)
			}

			if revRes.RowsAffected != 1 {
				r.logger.WarnContext(ctx, "original journal is already reversed or not POSTED")
				return finance.ErrJournalAlreadyReversed
			}
		}

		// Load and return the newly persisted POSTED journal
		postedJournal, getPostedErr := r.getJournalByIDWithinDB(ctx, tx, journal.ID())
		if getPostedErr != nil {
			r.logger.ErrorContext(ctx, "failed to load posted journal for return", slog.Any("error", getPostedErr))
			return fmt.Errorf("failed to load posted journal: %w", getPostedErr)
		}

		result = postedJournal
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *financeJournalRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.JournalTransaction, error) {
	if id == uuid.Nil {
		return nil, finance.ErrInvalidJournalID
	}
	return r.getJournalByIDWithinDB(ctx, r.db.WithContext(ctx), id)
}

func (r *financeJournalRepository) FindByIdempotencyKey(ctx context.Context, key string) (*finance.JournalTransaction, error) {
	cleanKey := strings.TrimSpace(key)
	if cleanKey == "" {
		return nil, finance.ErrMissingIdempotencyKey
	}

	var jm models.FinanceJournalModel
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", cleanKey).First(&jm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFinanceJournalNotFound
		}
		r.logger.ErrorContext(ctx, "failed to find finance journal by idempotency key",
			slog.String("idempotency_key", cleanKey),
			slog.Any("error", err),
		)
		return nil, err
	}

	return r.getJournalByIDWithinDB(ctx, r.db.WithContext(ctx), jm.ID)
}

func (r *financeJournalRepository) getJournalByIDWithinDB(ctx context.Context, db *gorm.DB, id uuid.UUID) (*finance.JournalTransaction, error) {
	var jm models.FinanceJournalModel
	if err := db.Where("id = ?", id).First(&jm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFinanceJournalNotFound
		}
		return nil, err
	}

	var lem []models.FinanceLedgerEntryModel
	if err := db.Where("journal_id = ?", id).Order("created_at ASC, id ASC").Find(&lem).Error; err != nil {
		return nil, err
	}

	entries := make([]finance.LedgerEntry, len(lem))
	for i, e := range lem {
		amt, err := finance.NewMoney(e.Amount, e.Currency)
		if err != nil {
			return nil, err
		}
		var entry finance.LedgerEntry
		if e.EntryType == "DEBIT" {
			entry, err = finance.NewDebitEntry(e.ID, e.AccountID, amt, e.OrganizerID, e.EventID, e.CreatedAt)
		} else {
			entry, err = finance.NewCreditEntry(e.ID, e.AccountID, amt, e.OrganizerID, e.EventID, e.CreatedAt)
		}
		if err != nil {
			return nil, err
		}
		entries[i] = entry.WithJournalID(jm.ID)
	}

	return finance.ReconstituteJournalTransaction(
		jm.ID,
		finance.TransactionType(jm.TransactionType),
		finance.FinancialEventType(jm.FinancialEventType),
		finance.SourceType(jm.SourceType),
		jm.SourceID,
		jm.IdempotencyKey,
		jm.Currency,
		jm.Description,
		entries,
		jm.ReversalReferenceID,
		finance.PostingStatus(jm.Status),
		jm.CreatedAt,
		jm.PostedAt,
		jm.IsReversed,
		jm.ReversedByJournalID,
		jm.ReversedAt,
	)
}

type subledgerAggregateRow struct {
	EventID         *uuid.UUID      `gorm:"column:event_id"`
	TotalCredits    decimal.Decimal `gorm:"column:total_credits"`
	TotalDebits     decimal.Decimal `gorm:"column:total_debits"`
	GrossEarnings   decimal.Decimal `gorm:"column:gross_earnings"`
	RefundsDeducted decimal.Decimal `gorm:"column:refunds_deducted"`
	SettledAmount   decimal.Decimal `gorm:"column:settled_amount"`
}

const subledgerSelectClause = `
	COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE 0 END), 0) AS total_credits,
	COALESCE(SUM(CASE WHEN fle.entry_type = 'DEBIT' THEN fle.amount ELSE 0 END), 0) AS total_debits,
	COALESCE(SUM(CASE 
		WHEN fle.entry_type = 'CREDIT' AND fj.financial_event_type = ? THEN fle.amount 
		WHEN fle.entry_type = 'DEBIT' AND fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type = ? THEN -fle.amount
		ELSE 0 
	END), 0) AS gross_earnings,
	COALESCE(SUM(CASE 
		WHEN fle.entry_type = 'DEBIT' AND fj.financial_event_type IN (?, ?) THEN fle.amount 
		WHEN fle.entry_type = 'CREDIT' AND fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type IN (?, ?) THEN -fle.amount
		ELSE 0 
	END), 0) AS refunds_deducted,
	COALESCE(SUM(CASE 
		WHEN fle.entry_type = 'DEBIT' AND fj.financial_event_type = ? THEN fle.amount 
		WHEN fle.entry_type = 'CREDIT' AND fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type = ? THEN -fle.amount
		ELSE 0 
	END), 0) AS settled_amount
`

// GetOrganizerBalance computes the aggregate posted Account 2010 balance for an organizer.
func (r *financeJournalRepository) GetOrganizerBalance(
	ctx context.Context,
	organizerID uuid.UUID,
) (*domain.OrganizerSubledgerBalance, error) {
	if organizerID == uuid.Nil {
		return nil, domain.ErrInvalidOrganizerID
	}

	var row subledgerAggregateRow
	query := r.db.WithContext(ctx).Table("finance_ledger_entries AS fle").
		Select(subledgerSelectClause,
			string(finance.FinancialEventTicketSaleCaptured),
			string(finance.FinancialEventTicketSaleCaptured),
			string(finance.FinancialEventCustomerRefund),
			string(finance.FinancialEventEventCancellationRefund),
			string(finance.FinancialEventCustomerRefund),
			string(finance.FinancialEventEventCancellationRefund),
			string(finance.FinancialEventOrganizerPayoutSettled),
			string(finance.FinancialEventOrganizerPayoutSettled),
		).
		Joins("JOIN finance_accounts fa ON fa.id = fle.account_id").
		Joins("JOIN finance_journals fj ON fj.id = fle.journal_id").
		Joins("LEFT JOIN finance_journals orig_fj ON orig_fj.id = fj.reversal_reference_id").
		Where("fj.status = ?", string(finance.PostingStatusPosted)).
		Where("fa.code = ?", finance.AccountCodeOrganizerPayable).
		Where("fle.organizer_id = ?", organizerID)

	if err := query.Scan(&row).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query organizer subledger balance",
			slog.String("organizer_id", organizerID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	netPayable := row.TotalCredits.Sub(row.TotalDebits)
	netEarnings := row.GrossEarnings.Sub(row.RefundsDeducted)

	return &domain.OrganizerSubledgerBalance{
		OrganizerID:     organizerID,
		GrossEarnings:   row.GrossEarnings,
		RefundsDeducted: row.RefundsDeducted,
		NetEarnings:     netEarnings,
		SettledAmount:   row.SettledAmount,
		TotalCredits:    row.TotalCredits,
		TotalDebits:     row.TotalDebits,
		NetPayable:      netPayable,
		Currency:        finance.CurrencyINR,
	}, nil
}

// GetOrganizerEventBalances computes the posted Account 2010 balances for each event owned by an organizer.
func (r *financeJournalRepository) GetOrganizerEventBalances(
	ctx context.Context,
	organizerID uuid.UUID,
) ([]domain.EventSubledgerBalance, error) {
	if organizerID == uuid.Nil {
		return nil, domain.ErrInvalidOrganizerID
	}

	var rows []subledgerAggregateRow
	selectWithEventID := "fle.event_id, " + subledgerSelectClause

	query := r.db.WithContext(ctx).Table("finance_ledger_entries AS fle").
		Select(selectWithEventID,
			string(finance.FinancialEventTicketSaleCaptured),
			string(finance.FinancialEventTicketSaleCaptured),
			string(finance.FinancialEventCustomerRefund),
			string(finance.FinancialEventEventCancellationRefund),
			string(finance.FinancialEventCustomerRefund),
			string(finance.FinancialEventEventCancellationRefund),
			string(finance.FinancialEventOrganizerPayoutSettled),
			string(finance.FinancialEventOrganizerPayoutSettled),
		).
		Joins("JOIN finance_accounts fa ON fa.id = fle.account_id").
		Joins("JOIN finance_journals fj ON fj.id = fle.journal_id").
		Joins("LEFT JOIN finance_journals orig_fj ON orig_fj.id = fj.reversal_reference_id").
		Where("fj.status = ?", string(finance.PostingStatusPosted)).
		Where("fa.code = ?", finance.AccountCodeOrganizerPayable).
		Where("fle.organizer_id = ?", organizerID).
		Where("fle.event_id IS NOT NULL").
		Group("fle.event_id").
		Order("fle.event_id ASC")

	if err := query.Scan(&rows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query organizer event subledger balances",
			slog.String("organizer_id", organizerID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	balances := make([]domain.EventSubledgerBalance, 0, len(rows))
	for _, row := range rows {
		if row.EventID == nil {
			continue
		}
		netPayable := row.TotalCredits.Sub(row.TotalDebits)
		netEarnings := row.GrossEarnings.Sub(row.RefundsDeducted)

		balances = append(balances, domain.EventSubledgerBalance{
			EventID:         *row.EventID,
			OrganizerID:     organizerID,
			GrossEarnings:   row.GrossEarnings,
			RefundsDeducted: row.RefundsDeducted,
			NetEarnings:     netEarnings,
			SettledAmount:   row.SettledAmount,
			TotalCredits:    row.TotalCredits,
			TotalDebits:     row.TotalDebits,
			NetPayable:      netPayable,
			Currency:        finance.CurrencyINR,
		})
	}

	return balances, nil
}

// GetEventBalance computes the posted Account 2010 balance for a single event owned by an organizer.
func (r *financeJournalRepository) GetEventBalance(
	ctx context.Context,
	organizerID uuid.UUID,
	eventID uuid.UUID,
) (*domain.EventSubledgerBalance, error) {
	if organizerID == uuid.Nil {
		return nil, domain.ErrInvalidOrganizerID
	}
	if eventID == uuid.Nil {
		return nil, errors.New("event ID cannot be nil")
	}

	var row subledgerAggregateRow
	query := r.db.WithContext(ctx).Table("finance_ledger_entries AS fle").
		Select(subledgerSelectClause,
			string(finance.FinancialEventTicketSaleCaptured),
			string(finance.FinancialEventTicketSaleCaptured),
			string(finance.FinancialEventCustomerRefund),
			string(finance.FinancialEventEventCancellationRefund),
			string(finance.FinancialEventCustomerRefund),
			string(finance.FinancialEventEventCancellationRefund),
			string(finance.FinancialEventOrganizerPayoutSettled),
			string(finance.FinancialEventOrganizerPayoutSettled),
		).
		Joins("JOIN finance_accounts fa ON fa.id = fle.account_id").
		Joins("JOIN finance_journals fj ON fj.id = fle.journal_id").
		Joins("LEFT JOIN finance_journals orig_fj ON orig_fj.id = fj.reversal_reference_id").
		Where("fj.status = ?", string(finance.PostingStatusPosted)).
		Where("fa.code = ?", finance.AccountCodeOrganizerPayable).
		Where("fle.organizer_id = ?", organizerID).
		Where("fle.event_id = ?", eventID)

	if err := query.Scan(&row).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query event subledger balance",
			slog.String("organizer_id", organizerID.String()),
			slog.String("event_id", eventID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	netPayable := row.TotalCredits.Sub(row.TotalDebits)
	netEarnings := row.GrossEarnings.Sub(row.RefundsDeducted)

	return &domain.EventSubledgerBalance{
		EventID:         eventID,
		OrganizerID:     organizerID,
		GrossEarnings:   row.GrossEarnings,
		RefundsDeducted: row.RefundsDeducted,
		NetEarnings:     netEarnings,
		SettledAmount:   row.SettledAmount,
		TotalCredits:    row.TotalCredits,
		TotalDebits:     row.TotalDebits,
		NetPayable:      netPayable,
		Currency:        finance.CurrencyINR,
	}, nil
}
