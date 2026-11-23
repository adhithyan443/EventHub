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

	if filter.EventID != nil {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM finance_ledger_entries sub_fle WHERE sub_fle.journal_id = fj.id AND sub_fle.event_id = ?)")
		args = append(args, *filter.EventID)
	}

	if filter.OrganizerID != nil {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM finance_ledger_entries sub_fle WHERE sub_fle.journal_id = fj.id AND sub_fle.organizer_id = ?)")
		args = append(args, *filter.OrganizerID)
	}

	if filter.SettlementStatus != "" && strings.ToUpper(filter.SettlementStatus) != "ALL" {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM settlements s WHERE s.id::text = fj.source_id AND s.status = ?)")
		args = append(args, strings.ToUpper(filter.SettlementStatus))
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

type ledgerAccountAggregateRow struct {
	Code        string          `gorm:"column:code"`
	NetCredit   decimal.Decimal `gorm:"column:net_credit"`
	NetDebit    decimal.Decimal `gorm:"column:net_debit"`
	TotalCredit decimal.Decimal `gorm:"column:total_credit"`
	TotalDebit  decimal.Decimal `gorm:"column:total_debit"`
}

type eventBreakdownSQLRow struct {
	EventID              uuid.UUID       `gorm:"column:event_id"`
	EventTitle           string          `gorm:"column:event_title"`
	OrganizerID          uuid.UUID       `gorm:"column:organizer_id"`
	OrganizerName        string          `gorm:"column:organizer_name"`
	Lifecycle            string          `gorm:"column:lifecycle"`
	GrossCollections     decimal.Decimal `gorm:"column:gross_collections"`
	Refunds              decimal.Decimal `gorm:"column:refunds"`
	OrganizerEarnings    decimal.Decimal `gorm:"column:organizer_earnings"`
	PlatformRevenue      decimal.Decimal `gorm:"column:platform_revenue"`
	TaxLiability         decimal.Decimal `gorm:"column:tax_liability"`
	CompletedSettlements decimal.Decimal `gorm:"column:completed_settlements"`
	PendingSettlements   decimal.Decimal `gorm:"column:pending_settlements"`
}

type organizerBreakdownSQLRow struct {
	OrganizerID          uuid.UUID       `gorm:"column:organizer_id"`
	OrganizerName        string          `gorm:"column:organizer_name"`
	Email                string          `gorm:"column:email"`
	TotalEvents          int64           `gorm:"column:total_events"`
	GrossCollections     decimal.Decimal `gorm:"column:gross_collections"`
	Refunds              decimal.Decimal `gorm:"column:refunds"`
	NetEarnings          decimal.Decimal `gorm:"column:net_earnings"`
	CompletedSettlements decimal.Decimal `gorm:"column:completed_settlements"`
	PendingSettlements   decimal.Decimal `gorm:"column:pending_settlements"`
}

func (r *adminLedgerRepository) GetFinancialOverview(
	ctx context.Context,
	filter domain.AdminFinancialFilter,
) (*domain.AdminFinancialOverviewResponse, error) {
	var fromUTC, toUTC *time.Time
	if filter.From != nil && !filter.From.IsZero() {
		t := filter.From.UTC()
		fromUTC = &t
	}
	if filter.To != nil && !filter.To.IsZero() {
		t := filter.To.UTC()
		toUTC = &t
	}

	// 1. Query Payments Gross Captured
	grossQuery := r.db.WithContext(ctx).Table("payments").
		Select("COALESCE(SUM(amount), 0)").
		Where("status = 'SUCCESS' AND paid_at IS NOT NULL")
	if fromUTC != nil {
		grossQuery = grossQuery.Where("COALESCE(paid_at, created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		grossQuery = grossQuery.Where("COALESCE(paid_at, created_at) <= ?", *toUTC)
	}
	if filter.EventID != nil {
		grossQuery = grossQuery.Where("event_id = ?", *filter.EventID)
	}
	if filter.OrganizerID != nil {
		grossQuery = grossQuery.Where("event_id IN (SELECT id FROM events WHERE organizer_id = ?)", *filter.OrganizerID)
	}
	var grossCaptured decimal.Decimal
	_ = grossQuery.Scan(&grossCaptured)

	// 2. Query Refunds
	refundQuery := r.db.WithContext(ctx).Table("refunds").
		Select("COALESCE(SUM(amount), 0)").
		Where("status = 'SUCCESS'")
	if fromUTC != nil {
		refundQuery = refundQuery.Where("COALESCE(processed_at, created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		refundQuery = refundQuery.Where("COALESCE(processed_at, created_at) <= ?", *toUTC)
	}
	if filter.EventID != nil {
		refundQuery = refundQuery.Where("booking_id IN (SELECT id FROM bookings WHERE event_id = ?)", *filter.EventID)
	}
	if filter.OrganizerID != nil {
		refundQuery = refundQuery.Where("booking_id IN (SELECT b.id FROM bookings b JOIN events e ON e.id = b.event_id WHERE e.organizer_id = ?)", *filter.OrganizerID)
	}
	var totalRefunds decimal.Decimal
	_ = refundQuery.Scan(&totalRefunds)

	// 3. Query Completed & Pending Settlements
	completedSettlementsQuery := r.db.WithContext(ctx).Table("settlements").
		Select("COALESCE(SUM(amount), 0)").
		Where("status = 'COMPLETED'")
	if fromUTC != nil {
		completedSettlementsQuery = completedSettlementsQuery.Where("COALESCE(processed_at, created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		completedSettlementsQuery = completedSettlementsQuery.Where("COALESCE(processed_at, created_at) <= ?", *toUTC)
	}
	if filter.EventID != nil {
		completedSettlementsQuery = completedSettlementsQuery.Where("event_id = ?", *filter.EventID)
	}
	if filter.OrganizerID != nil {
		completedSettlementsQuery = completedSettlementsQuery.Where("organizer_id = ?", *filter.OrganizerID)
	}
	if filter.SettlementStatus != "" && strings.ToUpper(filter.SettlementStatus) != "ALL" {
		completedSettlementsQuery = completedSettlementsQuery.Where("status = ?", strings.ToUpper(filter.SettlementStatus))
	}
	var completedSettlements decimal.Decimal
	_ = completedSettlementsQuery.Scan(&completedSettlements)

	pendingSettlementsQuery := r.db.WithContext(ctx).Table("settlements").
		Select("COALESCE(SUM(amount), 0)").
		Where("status IN ('PENDING', 'APPROVED', 'PROCESSING')")
	if fromUTC != nil {
		pendingSettlementsQuery = pendingSettlementsQuery.Where("created_at >= ?", *fromUTC)
	}
	if toUTC != nil {
		pendingSettlementsQuery = pendingSettlementsQuery.Where("created_at <= ?", *toUTC)
	}
	if filter.EventID != nil {
		pendingSettlementsQuery = pendingSettlementsQuery.Where("event_id = ?", *filter.EventID)
	}
	if filter.OrganizerID != nil {
		pendingSettlementsQuery = pendingSettlementsQuery.Where("organizer_id = ?", *filter.OrganizerID)
	}
	if filter.SettlementStatus != "" && strings.ToUpper(filter.SettlementStatus) != "ALL" {
		pendingSettlementsQuery = pendingSettlementsQuery.Where("status = ?", strings.ToUpper(filter.SettlementStatus))
	}
	var pendingSettlements decimal.Decimal
	_ = pendingSettlementsQuery.Scan(&pendingSettlements)

	// 4. Authoritative Ledger Balances from Chart of Accounts
	ledgerQuery := r.db.WithContext(ctx).Table("finance_ledger_entries AS fle").
		Select(`
			fa.code AS code,
			COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE -fle.amount END), 0) AS net_credit,
			COALESCE(SUM(CASE WHEN fle.entry_type = 'DEBIT' THEN fle.amount ELSE -fle.amount END), 0) AS net_debit,
			COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE 0 END), 0) AS total_credit,
			COALESCE(SUM(CASE WHEN fle.entry_type = 'DEBIT' THEN fle.amount ELSE 0 END), 0) AS total_debit
		`).
		Joins("JOIN finance_accounts fa ON fa.id = fle.account_id").
		Joins("JOIN finance_journals fj ON fj.id = fle.journal_id").
		Where("fj.status = 'POSTED'").
		Where("fa.code IN ('1010', '2010', '2030', '4010', '5010')")

	if fromUTC != nil {
		ledgerQuery = ledgerQuery.Where("COALESCE(fj.posted_at, fj.created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		ledgerQuery = ledgerQuery.Where("COALESCE(fj.posted_at, fj.created_at) <= ?", *toUTC)
	}
	if filter.EventID != nil {
		ledgerQuery = ledgerQuery.Where("fle.event_id = ?", *filter.EventID)
	}
	if filter.OrganizerID != nil {
		ledgerQuery = ledgerQuery.Where("fle.organizer_id = ?", *filter.OrganizerID)
	}
	if filter.TransactionType != "" && strings.ToUpper(filter.TransactionType) != "ALL" {
		ledgerQuery = ledgerQuery.Where("fj.transaction_type = ?", strings.ToUpper(filter.TransactionType))
	}

	var rows []ledgerAccountAggregateRow
	if err := ledgerQuery.Group("fa.code").Scan(&rows).Error; err != nil {
		r.logger.WarnContext(ctx, "failed to query ledger financial balances", slog.Any("error", err))
	}

	platformFeeRevenue := decimal.Zero
	organizerPayable := decimal.Zero
	taxLiability := decimal.Zero
	gatewayFees := decimal.Zero
	var debit1010, credit1010 decimal.Decimal

	for _, row := range rows {
		switch row.Code {
		case "4010":
			platformFeeRevenue = row.NetCredit
		case "2010":
			organizerPayable = row.NetCredit
		case "2030":
			taxLiability = row.NetCredit
		case "5010":
			gatewayFees = row.NetDebit
		case "1010":
			debit1010 = row.TotalDebit
			credit1010 = row.TotalCredit
		}
	}

	// Fallback/Reconciliation: if payments table had no records but ledger has 1010 entries
	if grossCaptured.IsZero() && debit1010.IsPositive() {
		grossCaptured = debit1010
	}
	if totalRefunds.IsZero() && credit1010.IsPositive() {
		totalRefunds = credit1010
	}

	netCollections := grossCaptured.Sub(totalRefunds)
	if netCollections.IsNegative() {
		netCollections = decimal.Zero
	}

	// Total organizer earnings is net ticket revenue allocated (unsettled liability + already settled payouts)
	totalOrganizerEarnings := organizerPayable.Add(completedSettlements)
	if totalOrganizerEarnings.IsNegative() {
		totalOrganizerEarnings = decimal.Zero
	}

	availablePlatformBalance := platformFeeRevenue.Sub(gatewayFees)

	metrics := domain.AdminFinancialMetrics{
		TotalGrossCollections:    grossCaptured,
		TotalRefunds:             totalRefunds,
		NetCollections:           netCollections,
		PlatformRevenue:          platformFeeRevenue,
		TaxLiability:             taxLiability,
		OrganizerEarnings:        totalOrganizerEarnings,
		OrganizerPayouts:         completedSettlements,
		PendingSettlements:       pendingSettlements,
		PaymentGatewayFees:       gatewayFees,
		AvailablePlatformBalance: availablePlatformBalance,
		Currency:                 "INR",
	}

	// 5. Build Event Breakdown
	var eventBreakdowns []domain.AdminFinancialEventBreakdownItem
	var eventRows []eventBreakdownSQLRow

	eventSQL := `
		SELECT 
			e.id AS event_id,
			e.title AS event_title,
			e.organizer_id AS organizer_id,
			COALESCE(op.business_name, u.name, 'Unknown Organizer') AS organizer_name,
			e.status AS lifecycle,
			COALESCE(p_agg.gross, 0) AS gross_collections,
			COALESCE(r_agg.refunds, 0) AS refunds,
			COALESCE(fle_org.net_earnings, 0) AS organizer_earnings,
			COALESCE(fle_fee.net_revenue, 0) AS platform_revenue,
			COALESCE(fle_tax.net_tax, 0) AS tax_liability,
			COALESCE(s_comp.settled, 0) AS completed_settlements,
			COALESCE(s_pend.pending, 0) AS pending_settlements
		FROM events e
		JOIN organizers o ON o.id = e.organizer_id
		LEFT JOIN organizer_profiles op ON op.organizer_id = o.id
		LEFT JOIN users u ON u.id = o.user_id
		LEFT JOIN (
			SELECT event_id, COALESCE(SUM(amount), 0) AS gross
			FROM payments
			WHERE status = 'SUCCESS' AND paid_at IS NOT NULL
			GROUP BY event_id
		) p_agg ON p_agg.event_id = e.id
		LEFT JOIN (
			SELECT b.event_id, COALESCE(SUM(r.amount), 0) AS refunds
			FROM refunds r
			JOIN bookings b ON b.id = r.booking_id
			WHERE r.status = 'SUCCESS'
			GROUP BY b.event_id
		) r_agg ON r_agg.event_id = e.id
		LEFT JOIN (
			SELECT fle.event_id, COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE -fle.amount END), 0) AS net_earnings
			FROM finance_ledger_entries fle
			JOIN finance_accounts fa ON fa.id = fle.account_id AND fa.code = '2010'
			JOIN finance_journals fj ON fj.id = fle.journal_id AND fj.status = 'POSTED'
			WHERE fle.event_id IS NOT NULL
			GROUP BY fle.event_id
		) fle_org ON fle_org.event_id = e.id
		LEFT JOIN (
			SELECT fle.event_id, COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE -fle.amount END), 0) AS net_revenue
			FROM finance_ledger_entries fle
			JOIN finance_accounts fa ON fa.id = fle.account_id AND fa.code = '4010'
			JOIN finance_journals fj ON fj.id = fle.journal_id AND fj.status = 'POSTED'
			WHERE fle.event_id IS NOT NULL
			GROUP BY fle.event_id
		) fle_fee ON fle_fee.event_id = e.id
		LEFT JOIN (
			SELECT fle.event_id, COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE -fle.amount END), 0) AS net_tax
			FROM finance_ledger_entries fle
			JOIN finance_accounts fa ON fa.id = fle.account_id AND fa.code = '2030'
			JOIN finance_journals fj ON fj.id = fle.journal_id AND fj.status = 'POSTED'
			WHERE fle.event_id IS NOT NULL
			GROUP BY fle.event_id
		) fle_tax ON fle_tax.event_id = e.id
		LEFT JOIN (
			SELECT event_id, COALESCE(SUM(amount), 0) AS settled
			FROM settlements
			WHERE status = 'COMPLETED' AND event_id IS NOT NULL
			GROUP BY event_id
		) s_comp ON s_comp.event_id = e.id
		LEFT JOIN (
			SELECT event_id, COALESCE(SUM(amount), 0) AS pending
			FROM settlements
			WHERE status IN ('PENDING', 'APPROVED', 'PROCESSING') AND event_id IS NOT NULL
			GROUP BY event_id
		) s_pend ON s_pend.event_id = e.id
		WHERE 1=1
	`
	var eventArgs []interface{}
	if filter.EventID != nil {
		eventSQL += " AND e.id = ?"
		eventArgs = append(eventArgs, *filter.EventID)
	}
	if filter.OrganizerID != nil {
		eventSQL += " AND e.organizer_id = ?"
		eventArgs = append(eventArgs, *filter.OrganizerID)
	}
	eventSQL += " ORDER BY e.created_at DESC LIMIT 50"

	if err := r.db.WithContext(ctx).Raw(eventSQL, eventArgs...).Scan(&eventRows).Error; err == nil {
		for _, er := range eventRows {
			netColl := er.GrossCollections.Sub(er.Refunds)
			if netColl.IsNegative() {
				netColl = decimal.Zero
			}
			avail := er.OrganizerEarnings.Sub(er.CompletedSettlements).Sub(er.PendingSettlements)
			if avail.IsNegative() {
				avail = decimal.Zero
			}

			eventBreakdowns = append(eventBreakdowns, domain.AdminFinancialEventBreakdownItem{
				EventID:              er.EventID,
				EventTitle:           er.EventTitle,
				OrganizerID:          er.OrganizerID,
				OrganizerName:        er.OrganizerName,
				Lifecycle:            er.Lifecycle,
				GrossCollections:     er.GrossCollections,
				Refunds:              er.Refunds,
				NetCollections:       netColl,
				OrganizerEarnings:    er.OrganizerEarnings,
				PlatformRevenue:      er.PlatformRevenue,
				TaxLiability:         er.TaxLiability,
				CompletedSettlements: er.CompletedSettlements,
				PendingSettlements:   er.PendingSettlements,
				AvailableBalance:     avail,
			})
		}
	}

	// 6. Build Organizer Breakdown
	var organizerBreakdowns []domain.AdminFinancialOrganizerBreakdownItem
	var orgRows []organizerBreakdownSQLRow

	orgSQL := `
		SELECT
			o.id AS organizer_id,
			COALESCE(op.business_name, u.name, 'Unknown Organizer') AS organizer_name,
			COALESCE(op.email, u.email, '') AS email,
			COUNT(DISTINCT e.id) AS total_events,
			COALESCE(p_agg.gross, 0) AS gross_collections,
			COALESCE(r_agg.refunds, 0) AS refunds,
			COALESCE(fle_org.net_earnings, 0) AS net_earnings,
			COALESCE(s_comp.settled, 0) AS completed_settlements,
			COALESCE(s_pend.pending, 0) AS pending_settlements
		FROM organizers o
		LEFT JOIN organizer_profiles op ON op.organizer_id = o.id
		LEFT JOIN users u ON u.id = o.user_id
		LEFT JOIN events e ON e.organizer_id = o.id
		LEFT JOIN (
			SELECT e2.organizer_id, COALESCE(SUM(p.amount), 0) AS gross
			FROM payments p
			JOIN events e2 ON e2.id = p.event_id
			WHERE p.status = 'SUCCESS' AND p.paid_at IS NOT NULL
			GROUP BY e2.organizer_id
		) p_agg ON p_agg.organizer_id = o.id
		LEFT JOIN (
			SELECT e3.organizer_id, COALESCE(SUM(r.amount), 0) AS refunds
			FROM refunds r
			JOIN bookings b ON b.id = r.booking_id
			JOIN events e3 ON e3.id = b.event_id
			WHERE r.status = 'SUCCESS'
			GROUP BY e3.organizer_id
		) r_agg ON r_agg.organizer_id = o.id
		LEFT JOIN (
			SELECT fle.organizer_id, COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE -fle.amount END), 0) AS net_earnings
			FROM finance_ledger_entries fle
			JOIN finance_accounts fa ON fa.id = fle.account_id AND fa.code = '2010'
			JOIN finance_journals fj ON fj.id = fle.journal_id AND fj.status = 'POSTED'
			WHERE fle.organizer_id IS NOT NULL
			GROUP BY fle.organizer_id
		) fle_org ON fle_org.organizer_id = o.id
		LEFT JOIN (
			SELECT organizer_id, COALESCE(SUM(amount), 0) AS settled
			FROM settlements
			WHERE status = 'COMPLETED'
			GROUP BY organizer_id
		) s_comp ON s_comp.organizer_id = o.id
		LEFT JOIN (
			SELECT organizer_id, COALESCE(SUM(amount), 0) AS pending
			FROM settlements
			WHERE status IN ('PENDING', 'APPROVED', 'PROCESSING')
			GROUP BY organizer_id
		) s_pend ON s_pend.organizer_id = o.id
		WHERE 1=1
	`
	var orgArgs []interface{}
	if filter.OrganizerID != nil {
		orgSQL += " AND o.id = ?"
		orgArgs = append(orgArgs, *filter.OrganizerID)
	}
	orgSQL += " GROUP BY o.id, op.business_name, u.name, op.email, u.email, p_agg.gross, r_agg.refunds, fle_org.net_earnings, s_comp.settled, s_pend.pending ORDER BY net_earnings DESC LIMIT 50"

	if err := r.db.WithContext(ctx).Raw(orgSQL, orgArgs...).Scan(&orgRows).Error; err == nil {
		for _, org := range orgRows {
			avail := org.NetEarnings.Sub(org.CompletedSettlements).Sub(org.PendingSettlements)
			if avail.IsNegative() {
				avail = decimal.Zero
			}
			organizerBreakdowns = append(organizerBreakdowns, domain.AdminFinancialOrganizerBreakdownItem{
				OrganizerID:          org.OrganizerID,
				OrganizerName:        org.OrganizerName,
				Email:                org.Email,
				TotalEvents:          org.TotalEvents,
				GrossCollections:     org.GrossCollections,
				Refunds:              org.Refunds,
				NetEarnings:          org.NetEarnings,
				CompletedSettlements: org.CompletedSettlements,
				PendingSettlements:   org.PendingSettlements,
				AvailableBalance:     avail,
			})
		}
	}

	return &domain.AdminFinancialOverviewResponse{
		Metrics:            metrics,
		EventBreakdown:     eventBreakdowns,
		OrganizerBreakdown: organizerBreakdowns,
		FilterApplied:      filter,
	}, nil
}
