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
