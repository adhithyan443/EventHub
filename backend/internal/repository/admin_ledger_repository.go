package repository

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type adminLedgerRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewAdminLedgerRepository creates a new PostgreSQL-backed AdminLedgerRepository.
func NewAdminLedgerRepository(db *gorm.DB, logger *slog.Logger) domain.AdminLedgerRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &adminLedgerRepository{
		db:     db,
		logger: logger,
	}
}

type journalListRow struct {
	ID                  uuid.UUID       `gorm:"column:id"`
	TransactionType     string          `gorm:"column:transaction_type"`
	FinancialEventType  string          `gorm:"column:financial_event_type"`
	SourceType          string          `gorm:"column:source_type"`
	SourceID            string          `gorm:"column:source_id"`
	IdempotencyKey      string          `gorm:"column:idempotency_key"`
	Currency            string          `gorm:"column:currency"`
	Description         string          `gorm:"column:description"`
	Status              string          `gorm:"column:status"`
	TotalDebit          decimal.Decimal `gorm:"column:total_debit"`
	TotalCredit         decimal.Decimal `gorm:"column:total_credit"`
	IsReversed          bool            `gorm:"column:is_reversed"`
	ReversalReferenceID *uuid.UUID      `gorm:"column:reversal_reference_id"`
	ReversedByJournalID *uuid.UUID      `gorm:"column:reversed_by_journal_id"`
	ReversedAt          *time.Time      `gorm:"column:reversed_at"`
	CreatedAt           time.Time       `gorm:"column:created_at"`
	PostedAt            *time.Time      `gorm:"column:posted_at"`
}

type journalHeaderRow struct {
	ID                  uuid.UUID  `gorm:"column:id"`
	TransactionType     string     `gorm:"column:transaction_type"`
	FinancialEventType  string     `gorm:"column:financial_event_type"`
	SourceType          string     `gorm:"column:source_type"`
	SourceID            string     `gorm:"column:source_id"`
	IdempotencyKey      string     `gorm:"column:idempotency_key"`
	Currency            string     `gorm:"column:currency"`
	Description         string     `gorm:"column:description"`
	Status              string     `gorm:"column:status"`
	IsReversed          bool       `gorm:"column:is_reversed"`
	ReversalReferenceID *uuid.UUID `gorm:"column:reversal_reference_id"`
	ReversedByJournalID *uuid.UUID `gorm:"column:reversed_by_journal_id"`
	ReversedAt          *time.Time `gorm:"column:reversed_at"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	PostedAt            *time.Time `gorm:"column:posted_at"`
}

type ledgerEntryRow struct {
	ID          uuid.UUID       `gorm:"column:id"`
	JournalID   uuid.UUID       `gorm:"column:journal_id"`
	AccountID   uuid.UUID       `gorm:"column:account_id"`
	AccountCode string          `gorm:"column:account_code"`
	AccountName string          `gorm:"column:account_name"`
	AccountType string          `gorm:"column:account_type"`
	EntryType   string          `gorm:"column:entry_type"`
	Amount      decimal.Decimal `gorm:"column:amount"`
	Currency    string          `gorm:"column:currency"`
	OrganizerID *uuid.UUID      `gorm:"column:organizer_id"`
	EventID     *uuid.UUID      `gorm:"column:event_id"`
	CreatedAt   time.Time       `gorm:"column:created_at"`
}

func (r *adminLedgerRepository) ListJournals(ctx context.Context, filter domain.AdminLedgerFilter) (*domain.AdminLedgerListResult, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}

	if filter.TransactionType != "" && strings.ToUpper(filter.TransactionType) != "ALL" {
		conditions = append(conditions, "fj.transaction_type = ?")
		args = append(args, strings.ToUpper(filter.TransactionType))
	}

	if filter.Status != "" && strings.ToUpper(filter.Status) != "ALL" {
		conditions = append(conditions, "fj.status = ?")
		args = append(args, strings.ToUpper(filter.Status))
	}

	if filter.From != nil {
		conditions = append(conditions, "COALESCE(fj.posted_at, fj.created_at) >= ?")
		args = append(args, *filter.From)
	}

	if filter.To != nil {
		conditions = append(conditions, "COALESCE(fj.posted_at, fj.created_at) <= ?")
		args = append(args, *filter.To)
	}

	if search := strings.TrimSpace(filter.Search); search != "" {
		pattern := "%" + search + "%"
		conditions = append(conditions, "(fj.source_id ILIKE ? OR fj.idempotency_key ILIKE ? OR fj.id::text ILIKE ? OR fj.description ILIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM finance_journals fj %s", whereClause)
	var totalRecords int64
	if err := r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&totalRecords).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to count admin ledger journals", slog.Any("error", err))
		return nil, fmt.Errorf("failed to count ledger journals: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT 
			fj.id,
			fj.transaction_type,
			fj.financial_event_type,
			fj.source_type,
			fj.source_id,
			fj.idempotency_key,
			fj.currency,
			fj.description,
			fj.status,
			COALESCE(SUM(CASE WHEN fle.entry_type = 'DEBIT' THEN fle.amount ELSE 0 END), 0) AS total_debit,
			COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE 0 END), 0) AS total_credit,
			fj.is_reversed,
			fj.reversal_reference_id,
			fj.reversed_by_journal_id,
			fj.reversed_at,
			fj.created_at,
			fj.posted_at
		FROM finance_journals fj
		LEFT JOIN finance_ledger_entries fle ON fle.journal_id = fj.id
		%s
		GROUP BY fj.id
		ORDER BY COALESCE(fj.posted_at, fj.created_at) DESC, fj.created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	listArgs := make([]interface{}, len(args), len(args)+2)
	copy(listArgs, args)
	listArgs = append(listArgs, limit, offset)

	var rows []journalListRow
	if err := r.db.WithContext(ctx).Raw(listQuery, listArgs...).Scan(&rows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to list admin ledger journals", slog.Any("error", err))
		return nil, fmt.Errorf("failed to list ledger journals: %w", err)
	}

	journals := make([]domain.AdminLedgerJournalListItem, 0, len(rows))
	for _, row := range rows {
		journals = append(journals, domain.AdminLedgerJournalListItem{
			ID:                  row.ID,
			TransactionType:     row.TransactionType,
			FinancialEventType:  row.FinancialEventType,
			SourceType:          row.SourceType,
			SourceID:            row.SourceID,
			IdempotencyKey:      row.IdempotencyKey,
			Currency:            row.Currency,
			Description:         row.Description,
			Status:              row.Status,
			TotalDebit:          row.TotalDebit,
			TotalCredit:         row.TotalCredit,
			IsBalanced:          row.TotalDebit.Equal(row.TotalCredit) && row.TotalDebit.GreaterThan(decimal.Zero),
			IsReversed:          row.IsReversed,
			ReversalReferenceID: row.ReversalReferenceID,
			ReversedByJournalID: row.ReversedByJournalID,
			ReversedAt:          row.ReversedAt,
			CreatedAt:           row.CreatedAt,
			PostedAt:            row.PostedAt,
		})
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.AdminLedgerListResult{
		Journals:     journals,
		Page:         page,
		Limit:        limit,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
	}, nil
}

func (r *adminLedgerRepository) GetJournalByID(ctx context.Context, id uuid.UUID) (*domain.AdminLedgerJournalDetail, error) {
	const headerQuery = `
		SELECT 
			id,
			transaction_type,
			financial_event_type,
			source_type,
			source_id,
			idempotency_key,
			currency,
			description,
			status,
			is_reversed,
			reversal_reference_id,
			reversed_by_journal_id,
			reversed_at,
			created_at,
			posted_at
		FROM finance_journals
		WHERE id = ?
	`

	var header journalHeaderRow
	if err := r.db.WithContext(ctx).Raw(headerQuery, id).Scan(&header).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query admin ledger journal header", slog.Any("error", err), slog.String("journal_id", id.String()))
		return nil, fmt.Errorf("failed to query ledger journal header: %w", err)
	}

	if header.ID == uuid.Nil {
		return nil, domain.ErrLedgerJournalNotFound
	}

	const entriesQuery = `
		SELECT 
			fle.id,
			fle.journal_id,
			fle.account_id,
			COALESCE(fa.code, '') AS account_code,
			COALESCE(fa.name, '') AS account_name,
			COALESCE(fa.type, '') AS account_type,
			fle.entry_type,
			fle.amount,
			fle.currency,
			fle.organizer_id,
			fle.event_id,
			fle.created_at
		FROM finance_ledger_entries fle
		LEFT JOIN finance_accounts fa ON fa.id = fle.account_id
		WHERE fle.journal_id = ?
		ORDER BY fle.entry_type DESC, fle.created_at ASC
	`

	var entryRows []ledgerEntryRow
	if err := r.db.WithContext(ctx).Raw(entriesQuery, id).Scan(&entryRows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query admin ledger entries", slog.Any("error", err), slog.String("journal_id", id.String()))
		return nil, fmt.Errorf("failed to query ledger entries: %w", err)
	}

	totalDebit := decimal.Zero
	totalCredit := decimal.Zero
	entries := make([]domain.AdminLedgerEntryItem, 0, len(entryRows))

	for _, er := range entryRows {
		if er.EntryType == "DEBIT" {
			totalDebit = totalDebit.Add(er.Amount)
		} else if er.EntryType == "CREDIT" {
			totalCredit = totalCredit.Add(er.Amount)
		}

		entries = append(entries, domain.AdminLedgerEntryItem{
			ID:          er.ID,
			JournalID:   er.JournalID,
			AccountID:   er.AccountID,
			AccountCode: er.AccountCode,
			AccountName: er.AccountName,
			AccountType: er.AccountType,
			EntryType:   er.EntryType,
			Amount:      er.Amount,
			Currency:    er.Currency,
			OrganizerID: er.OrganizerID,
			EventID:     er.EventID,
			CreatedAt:   er.CreatedAt,
		})
	}

	isBalanced := totalDebit.Equal(totalCredit) && (len(entries) == 0 || totalDebit.GreaterThan(decimal.Zero))

	return &domain.AdminLedgerJournalDetail{
		ID:                  header.ID,
		TransactionType:     header.TransactionType,
		FinancialEventType:  header.FinancialEventType,
		SourceType:          header.SourceType,
		SourceID:            header.SourceID,
		IdempotencyKey:      header.IdempotencyKey,
		Currency:            header.Currency,
		Description:         header.Description,
		Status:              header.Status,
		TotalDebit:          totalDebit,
		TotalCredit:         totalCredit,
		IsBalanced:          isBalanced,
		IsReversed:          header.IsReversed,
		ReversalReferenceID: header.ReversalReferenceID,
		ReversedByJournalID: header.ReversedByJournalID,
		ReversedAt:          header.ReversedAt,
		CreatedAt:           header.CreatedAt,
		PostedAt:            header.PostedAt,
		Entries:             entries,
	}, nil
}
