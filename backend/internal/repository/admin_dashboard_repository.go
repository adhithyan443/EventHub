package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type adminDashboardRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewAdminDashboardRepository constructs a new PostgreSQL-backed AdminDashboardRepository.
func NewAdminDashboardRepository(db *gorm.DB, logger *slog.Logger) domain.AdminDashboardRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &adminDashboardRepository{
		db:     db,
		logger: logger,
	}
}

type platformOverviewRow struct {
	TotalUsers      int64 `gorm:"column:total_users"`
	TotalOrganizers int64 `gorm:"column:total_organizers"`
	TotalEvents     int64 `gorm:"column:total_events"`
	ActiveEvents    int64 `gorm:"column:active_events"`
	TotalBookings   int64 `gorm:"column:total_bookings"`
	TicketsSold     int64 `gorm:"column:tickets_sold"`
}

func (r *adminDashboardRepository) GetPlatformOverview(ctx context.Context) (*domain.AdminPlatformOverview, error) {
	const queryStr = `
		SELECT
			(SELECT COUNT(*) FROM users) AS total_users,
			(SELECT COUNT(*) FROM organizers) AS total_organizers,
			(SELECT COUNT(*) FROM events) AS total_events,
			(SELECT COUNT(*) FROM events WHERE status IN ('PUBLISHED', 'ONGOING')) AS active_events,
			(SELECT COUNT(*) FROM bookings) AS total_bookings,
			(SELECT COALESCE(SUM(bi.quantity), 0) 
			 FROM booking_items bi 
			 JOIN bookings b ON b.id = bi.booking_id 
			 WHERE b.status = 'CONFIRMED') AS tickets_sold
	`

	var row platformOverviewRow
	if err := r.db.WithContext(ctx).Raw(queryStr).Scan(&row).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query admin platform overview", slog.Any("error", err))
		return nil, fmt.Errorf("failed to query platform overview: %w", err)
	}

	return &domain.AdminPlatformOverview{
		TotalUsers:      row.TotalUsers,
		TotalOrganizers: row.TotalOrganizers,
		TotalEvents:     row.TotalEvents,
		ActiveEvents:    row.ActiveEvents,
		TotalBookings:   row.TotalBookings,
		TicketsSold:     row.TicketsSold,
	}, nil
}

type ledgerAccountRow struct {
	Code      string          `gorm:"column:code"`
	NetCredit decimal.Decimal `gorm:"column:net_credit"`
}

func (r *adminDashboardRepository) GetFinancialOverview(
	ctx context.Context,
	filter domain.AdminDashboardFilter,
) (*domain.AdminFinancialOverview, error) {
	fromUTC, toUTC := normalizeDateRange(filter.From, filter.To)

	// 1. Gross Captured Amount from payments
	grossQuery := r.db.WithContext(ctx).Table("payments").
		Select("COALESCE(SUM(amount), 0)").
		Where("status = 'SUCCESS' AND paid_at IS NOT NULL")
	if fromUTC != nil {
		grossQuery = grossQuery.Where("COALESCE(paid_at, created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		grossQuery = grossQuery.Where("COALESCE(paid_at, created_at) <= ?", *toUTC)
	}

	var grossCaptured decimal.Decimal
	if err := grossQuery.Scan(&grossCaptured).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query gross captured amount", slog.Any("error", err))
		return nil, fmt.Errorf("failed to query gross captured amount: %w", err)
	}

	// 2. Total Successful Refunds
	refundQuery := r.db.WithContext(ctx).Table("refunds").
		Select("COALESCE(SUM(amount), 0)").
		Where("status = 'SUCCESS'")
	if fromUTC != nil {
		refundQuery = refundQuery.Where("COALESCE(processed_at, created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		refundQuery = refundQuery.Where("COALESCE(processed_at, created_at) <= ?", *toUTC)
	}

	var totalRefunds decimal.Decimal
	if err := refundQuery.Scan(&totalRefunds).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query total refunds", slog.Any("error", err))
		return nil, fmt.Errorf("failed to query total refunds: %w", err)
	}

	// 3. Completed Settlements
	completedQuery := r.db.WithContext(ctx).Table("settlements").
		Select("COALESCE(SUM(amount), 0)").
		Where("status = 'COMPLETED'")
	if fromUTC != nil {
		completedQuery = completedQuery.Where("COALESCE(processed_at, created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		completedQuery = completedQuery.Where("COALESCE(processed_at, created_at) <= ?", *toUTC)
	}

	var completedSettlements decimal.Decimal
	if err := completedQuery.Scan(&completedSettlements).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query completed settlements", slog.Any("error", err))
		return nil, fmt.Errorf("failed to query completed settlements: %w", err)
	}

	// 4. In-Flight Pending Settlements
	pendingQuery := r.db.WithContext(ctx).Table("settlements").
		Select("COALESCE(SUM(amount), 0)").
		Where("status IN ('PENDING', 'APPROVED', 'PROCESSING')")
	if fromUTC != nil {
		pendingQuery = pendingQuery.Where("created_at >= ?", *fromUTC)
	}
	if toUTC != nil {
		pendingQuery = pendingQuery.Where("created_at <= ?", *toUTC)
	}

	var pendingSettlements decimal.Decimal
	if err := pendingQuery.Scan(&pendingSettlements).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query pending settlements", slog.Any("error", err))
		return nil, fmt.Errorf("failed to query pending settlements: %w", err)
	}

	// 5. Authoritative Ledger Balances from Chart of Accounts
	// Account 4010: Platform Fee Revenue (Net Credit: Credits - Debits)
	// Account 2010: Organizer Payable Liability (Net Credit: Credits - Debits)
	// Account 2030: Tax Payable Liability (Net Credit: Credits - Debits)
	ledgerQuery := r.db.WithContext(ctx).Table("finance_ledger_entries AS fle").
		Select(`
			fa.code AS code,
			COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE -fle.amount END), 0) AS net_credit
		`).
		Joins("JOIN finance_accounts fa ON fa.id = fle.account_id").
		Joins("JOIN finance_journals fj ON fj.id = fle.journal_id").
		Joins("LEFT JOIN finance_journals orig_fj ON orig_fj.id = fj.reversal_reference_id").
		Where("fj.status = ?", string(finance.PostingStatusPosted)).
		Where("fa.code IN (?, ?, ?)",
			finance.AccountCodePlatformFeeRevenue,
			finance.AccountCodeOrganizerPayable,
			finance.AccountCodeTaxPayable,
		)

	if fromUTC != nil {
		ledgerQuery = ledgerQuery.Where("COALESCE(fj.posted_at, fj.created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		ledgerQuery = ledgerQuery.Where("COALESCE(fj.posted_at, fj.created_at) <= ?", *toUTC)
	}

	var ledgerRows []ledgerAccountRow
	if err := ledgerQuery.Group("fa.code").Scan(&ledgerRows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query ledger financial balances", slog.Any("error", err))
		return nil, fmt.Errorf("failed to query ledger balances: %w", err)
	}

	platformFeeRevenue := decimal.Zero
	organizerPayable := decimal.Zero
	taxLiability := decimal.Zero

	for _, row := range ledgerRows {
		switch row.Code {
		case finance.AccountCodePlatformFeeRevenue:
			platformFeeRevenue = row.NetCredit
		case finance.AccountCodeOrganizerPayable:
			organizerPayable = row.NetCredit
		case finance.AccountCodeTaxPayable:
			taxLiability = row.NetCredit
		}
	}

	netCaptured := grossCaptured.Sub(totalRefunds)
	if netCaptured.IsNegative() {
		netCaptured = decimal.Zero
	}

	return &domain.AdminFinancialOverview{
		GrossCapturedAmount:       grossCaptured,
		TotalRefunds:              totalRefunds,
		NetCapturedAmount:         netCaptured,
		PlatformFeeRevenue:        platformFeeRevenue,
		OrganizerPayableLiability: organizerPayable,
		CompletedSettlements:      completedSettlements,
		PendingSettlements:        pendingSettlements,
		TaxLiability:              taxLiability,
		Currency:                  finance.CurrencyINR,
	}, nil
}

type trendPeriodGross struct {
	Period      string          `gorm:"column:period"`
	GrossVolume decimal.Decimal `gorm:"column:gross_volume"`
}

type trendPeriodFees struct {
	Period       string          `gorm:"column:period"`
	PlatformFees decimal.Decimal `gorm:"column:platform_fees"`
}

type trendPeriodBookings struct {
	Period       string `gorm:"column:period"`
	BookingCount int64  `gorm:"column:booking_count"`
	TicketsSold  int64  `gorm:"column:tickets_sold"`
}

type trendPeriodRefunds struct {
	Period       string          `gorm:"column:period"`
	RefundAmount decimal.Decimal `gorm:"column:refund_amount"`
	RefundCount  int64           `gorm:"column:refund_count"`
}

func generateMonthPeriods(from, to time.Time) []string {
	var periods []string
	curr := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)

	for !curr.After(end) {
		periods = append(periods, curr.Format("2006-01"))
		curr = curr.AddDate(0, 1, 0)
	}
	return periods
}

func (r *adminDashboardRepository) GetTrends(
	ctx context.Context,
	filter domain.AdminDashboardFilter,
) (*domain.AdminDashboardTrends, error) {
	var fromTime, toTime time.Time
	if filter.To != nil {
		toTime = filter.To.UTC()
	} else {
		toTime = time.Now().UTC()
	}

	if filter.From != nil {
		fromTime = filter.From.UTC()
	} else {
		// Default to past 12 calendar months
		fromTime = toTime.AddDate(0, -11, 0)
	}

	fromUTC, toUTC := normalizeDateRange(&fromTime, &toTime)
	periods := generateMonthPeriods(*fromUTC, *toUTC)

	// 1. Revenue Trends: Gross payment capture volume
	var grossRows []trendPeriodGross
	grossErr := r.db.WithContext(ctx).Table("payments").
		Select(`
			TO_CHAR(DATE_TRUNC('month', COALESCE(paid_at, created_at)), 'YYYY-MM') AS period,
			COALESCE(SUM(amount), 0) AS gross_volume
		`).
		Where("status = 'SUCCESS' AND paid_at IS NOT NULL").
		Where("COALESCE(paid_at, created_at) >= ? AND COALESCE(paid_at, created_at) <= ?", *fromUTC, *toUTC).
		Group("DATE_TRUNC('month', COALESCE(paid_at, created_at))").
		Order("period ASC").
		Scan(&grossRows).Error
	if grossErr != nil {
		r.logger.ErrorContext(ctx, "failed to query gross revenue trends", slog.Any("error", grossErr))
		return nil, fmt.Errorf("failed to query gross revenue trends: %w", grossErr)
	}

	grossMap := make(map[string]decimal.Decimal, len(grossRows))
	for _, row := range grossRows {
		grossMap[row.Period] = row.GrossVolume
	}

	// 2. Revenue Trends: Platform fee revenue from Account 4010 ledger entries
	var feeRows []trendPeriodFees
	feeErr := r.db.WithContext(ctx).Table("finance_ledger_entries AS fle").
		Select(`
			TO_CHAR(DATE_TRUNC('month', COALESCE(fj.posted_at, fj.created_at)), 'YYYY-MM') AS period,
			COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE -fle.amount END), 0) AS platform_fees
		`).
		Joins("JOIN finance_accounts fa ON fa.id = fle.account_id").
		Joins("JOIN finance_journals fj ON fj.id = fle.journal_id").
		Joins("LEFT JOIN finance_journals orig_fj ON orig_fj.id = fj.reversal_reference_id").
		Where("fj.status = ?", string(finance.PostingStatusPosted)).
		Where("fa.code = ?", finance.AccountCodePlatformFeeRevenue).
		Where("COALESCE(fj.posted_at, fj.created_at) >= ? AND COALESCE(fj.posted_at, fj.created_at) <= ?", *fromUTC, *toUTC).
		Group("DATE_TRUNC('month', COALESCE(fj.posted_at, fj.created_at))").
		Order("period ASC").
		Scan(&feeRows).Error
	if feeErr != nil {
		r.logger.ErrorContext(ctx, "failed to query platform fee revenue trends", slog.Any("error", feeErr))
		return nil, fmt.Errorf("failed to query platform fee revenue trends: %w", feeErr)
	}

	feeMap := make(map[string]decimal.Decimal, len(feeRows))
	for _, row := range feeRows {
		feeMap[row.Period] = row.PlatformFees
	}

	revenueTrends := make([]domain.AdminRevenueTrendPoint, 0, len(periods))
	for _, p := range periods {
		revenueTrends = append(revenueTrends, domain.AdminRevenueTrendPoint{
			Period:       p,
			GrossVolume:  grossMap[p],
			PlatformFees: feeMap[p],
		})
	}

	// 3. Booking Trends
	var bookingRows []trendPeriodBookings
	bookingErr := r.db.WithContext(ctx).Table("bookings AS b").
		Select(`
			TO_CHAR(DATE_TRUNC('month', b.created_at), 'YYYY-MM') AS period,
			COUNT(b.id) AS booking_count,
			COALESCE(SUM(CASE WHEN b.status = 'CONFIRMED' THEN (
				SELECT COALESCE(SUM(bi.quantity), 0) FROM booking_items bi WHERE bi.booking_id = b.id
			) ELSE 0 END), 0) AS tickets_sold
		`).
		Where("b.created_at >= ? AND b.created_at <= ?", *fromUTC, *toUTC).
		Group("DATE_TRUNC('month', b.created_at)").
		Order("period ASC").
		Scan(&bookingRows).Error
	if bookingErr != nil {
		r.logger.ErrorContext(ctx, "failed to query booking trends", slog.Any("error", bookingErr))
		return nil, fmt.Errorf("failed to query booking trends: %w", bookingErr)
	}

	bookingMap := make(map[string]trendPeriodBookings, len(bookingRows))
	for _, row := range bookingRows {
		bookingMap[row.Period] = row
	}

	bookingTrends := make([]domain.AdminBookingTrendPoint, 0, len(periods))
	for _, p := range periods {
		item := bookingMap[p]
		bookingTrends = append(bookingTrends, domain.AdminBookingTrendPoint{
			Period:       p,
			BookingCount: item.BookingCount,
			TicketsSold:  item.TicketsSold,
		})
	}

	// 4. Refund Trends
	var refundRows []trendPeriodRefunds
	refundErr := r.db.WithContext(ctx).Table("refunds AS r").
		Select(`
			TO_CHAR(DATE_TRUNC('month', COALESCE(r.processed_at, r.created_at)), 'YYYY-MM') AS period,
			COALESCE(SUM(r.amount), 0) AS refund_amount,
			COUNT(r.id) AS refund_count
		`).
		Where("r.status = 'SUCCESS'").
		Where("COALESCE(r.processed_at, r.created_at) >= ? AND COALESCE(r.processed_at, r.created_at) <= ?", *fromUTC, *toUTC).
		Group("DATE_TRUNC('month', COALESCE(r.processed_at, r.created_at))").
		Order("period ASC").
		Scan(&refundRows).Error
	if refundErr != nil {
		r.logger.ErrorContext(ctx, "failed to query refund trends", slog.Any("error", refundErr))
		return nil, fmt.Errorf("failed to query refund trends: %w", refundErr)
	}

	refundMap := make(map[string]trendPeriodRefunds, len(refundRows))
	for _, row := range refundRows {
		refundMap[row.Period] = row
	}

	refundTrends := make([]domain.AdminRefundTrendPoint, 0, len(periods))
	for _, p := range periods {
		item := refundMap[p]
		refundTrends = append(refundTrends, domain.AdminRefundTrendPoint{
			Period:       p,
			RefundAmount: item.RefundAmount,
			RefundCount:  item.RefundCount,
		})
	}

	return &domain.AdminDashboardTrends{
		RevenueTrends: revenueTrends,
		BookingTrends: bookingTrends,
		RefundTrends:  refundTrends,
	}, nil
}

func (r *adminDashboardRepository) GetPaymentStatusBreakdown(
	ctx context.Context,
	filter domain.AdminDashboardFilter,
) ([]domain.AdminPaymentStatusBreakdownItem, error) {
	fromUTC, toUTC := normalizeDateRange(filter.From, filter.To)

	query := r.db.WithContext(ctx).Table("payments").
		Select(`
			status,
			COUNT(id) AS count,
			COALESCE(SUM(amount), 0) AS amount
		`)

	if fromUTC != nil {
		query = query.Where("COALESCE(paid_at, created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		query = query.Where("COALESCE(paid_at, created_at) <= ?", *toUTC)
	}

	var rows []domain.AdminPaymentStatusBreakdownItem
	if err := query.Group("status").Order("count DESC").Scan(&rows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query payment status breakdown", slog.Any("error", err))
		return nil, fmt.Errorf("failed to query payment status breakdown: %w", err)
	}

	// Ensure core statuses are accounted for even if count is 0
	standardStatuses := []string{
		string(domain.PaymentStatusSuccess),
		string(domain.PaymentStatusPending),
		string(domain.PaymentStatusFailed),
		string(domain.PaymentStatusRefundRequired),
	}

	existingMap := make(map[string]domain.AdminPaymentStatusBreakdownItem, len(rows))
	for _, r := range rows {
		existingMap[strings.ToUpper(r.Status)] = r
	}

	result := make([]domain.AdminPaymentStatusBreakdownItem, 0, len(rows)+len(standardStatuses))
	added := make(map[string]bool)

	// First append standard statuses in clean order
	for _, s := range standardStatuses {
		if item, ok := existingMap[s]; ok {
			result = append(result, item)
		} else {
			result = append(result, domain.AdminPaymentStatusBreakdownItem{
				Status: s,
				Count:  0,
				Amount: decimal.Zero,
			})
		}
		added[s] = true
	}

	// Then append any non-standard statuses found in DB
	for _, r := range rows {
		up := strings.ToUpper(r.Status)
		if !added[up] {
			result = append(result, r)
			added[up] = true
		}
	}

	return result, nil
}

func (r *adminDashboardRepository) GetRecentBookings(
	ctx context.Context,
	limit int,
) ([]domain.AdminRecentBookingItem, error) {
	if limit <= 0 {
		limit = 5
	}

	const queryStr = `
		SELECT
			b.id,
			b.booking_number,
			b.user_id AS customer_id,
			COALESCE(u.full_name, '') AS customer_name,
			COALESCE(u.email, '') AS customer_email,
			b.event_id,
			COALESCE(e.title, '') AS event_title,
			b.total AS amount,
			b.status,
			COALESCE((
				SELECT p.status 
				FROM payments p 
				WHERE p.booking_id = b.id 
				ORDER BY p.created_at DESC 
				LIMIT 1
			), 'UNKNOWN') AS payment_status,
			b.created_at
		FROM bookings b
		LEFT JOIN users u ON u.id = b.user_id
		LEFT JOIN events e ON e.id = b.event_id
		ORDER BY b.created_at DESC
		LIMIT ?
	`

	var items []domain.AdminRecentBookingItem
	if err := r.db.WithContext(ctx).Raw(queryStr, limit).Scan(&items).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query recent bookings", slog.Any("error", err))
		return nil, fmt.Errorf("failed to query recent bookings: %w", err)
	}

	if items == nil {
		items = []domain.AdminRecentBookingItem{}
	}

	return items, nil
}
