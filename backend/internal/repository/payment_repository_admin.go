package repository

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type adminPaymentRow struct {
	ID                string          `gorm:"column:id"`
	BookingID         string          `gorm:"column:booking_id"`
	BookingNumber     string          `gorm:"column:booking_number"`
	UserID            *uuid.UUID      `gorm:"column:user_id"`
	UserName          *string         `gorm:"column:user_name"`
	UserEmail         *string         `gorm:"column:user_email"`
	EventID           *uuid.UUID      `gorm:"column:event_id"`
	EventTitle        *string         `gorm:"column:event_title"`
	OrganizerID       *uuid.UUID      `gorm:"column:organizer_id"`
	BusinessName      *string         `gorm:"column:business_name"`
	Amount            decimal.Decimal `gorm:"column:amount"`
	Status            string          `gorm:"column:status"`
	PaymentMethod     string          `gorm:"column:payment_method"`
	Provider          string          `gorm:"column:provider"`
	TransactionID     *string         `gorm:"column:transaction_id"`
	ProviderOrderID   *string         `gorm:"column:provider_order_id"`
	PaidAt            *time.Time      `gorm:"column:paid_at"`
	CreatedAt         time.Time       `gorm:"column:created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at"`
	HasGatewayCapture bool            `gorm:"column:has_gateway_capture"`
	AllocationStatus  string          `gorm:"column:allocation_status"`
}

// buildAllocationStatusExpression returns the canonical SQL expression to compute
// allocation_status dynamically for any payment row.
//
// Classification rules:
//   - ACCOUNTED:
//     1. Captured payment with an active, non-reversed POSTED financial journal
//     matching financial_event_type = 'TICKET_SALE_CAPTURED', transaction_type = 'PAYMENT',
//     source_type = 'PAYMENT', respecting v1 tax/discount restrictions (tax = 0, discount = 0),
//     and verifying the complete balanced double-entry distribution:
//       - Total debits = Total credits = payment total (p.amount).
//       - Debit 1010 Razorpay Clearing = payment total (p.amount).
//       - Credit 2010 Organizer Payable = booking subtotal (b.subtotal).
//       - Credit 4010 Platform Fee Revenue = platform fee (b.platform_fee).
//     2. OR legitimate free booking payment (amount = 0, status = 'SUCCESS', supported
//     by confirmed booking evidence: b.total = 0 AND b.status = 'CONFIRMED').
//   - UNALLOCATED:
//     Captured payment (has_gateway_capture = true) lacking complete active posted journal distribution.
//   - NOT_APPLICABLE:
//     Payment attempt without confirmed gateway capture evidence.
func buildAllocationStatusExpression() string {
	journalVerificationExpr := fmt.Sprintf(`EXISTS (
		SELECT 1 FROM finance_journals fj
		WHERE fj.source_type = '%s'
		  AND fj.source_id = p.id::text
		  AND fj.transaction_type = '%s'
		  AND fj.financial_event_type = '%s'
		  AND fj.status = '%s'
		  AND fj.is_reversed = FALSE
		  AND COALESCE(b.tax, 0) = 0
		  AND COALESCE(b.discount, 0) = 0
		  AND EXISTS (
			  SELECT 1 FROM finance_ledger_entries fle
			  JOIN finance_accounts fa ON fa.id = fle.account_id
			  WHERE fle.journal_id = fj.id
			  HAVING 
			      COALESCE(SUM(CASE WHEN fle.entry_type = 'DEBIT' THEN fle.amount ELSE 0 END), 0) = p.amount
			      AND COALESCE(SUM(CASE WHEN fle.entry_type = 'CREDIT' THEN fle.amount ELSE 0 END), 0) = p.amount
			      AND COALESCE(SUM(CASE WHEN fa.code = '%s' AND fle.entry_type = 'DEBIT' THEN fle.amount ELSE 0 END), 0) = p.amount
			      AND COALESCE(SUM(CASE WHEN fa.code = '%s' AND fle.entry_type = 'CREDIT' THEN fle.amount ELSE 0 END), 0) = b.subtotal
			      AND COALESCE(SUM(CASE WHEN fa.code = '%s' AND fle.entry_type = 'CREDIT' THEN fle.amount ELSE 0 END), 0) = b.platform_fee
		  )
	)`,
		finance.SourceTypePayment,
		finance.TransactionTypePayment,
		finance.FinancialEventTicketSaleCaptured,
		finance.PostingStatusPosted,
		finance.AccountCodeRazorpayClearing,
		finance.AccountCodeOrganizerPayable,
		finance.AccountCodePlatformFeeRevenue,
	)

	return fmt.Sprintf(`CASE
		WHEN (%s OR (p.status = 'SUCCESS' AND p.amount = 0 AND b.id IS NOT NULL AND b.total = 0 AND b.status = 'CONFIRMED')) THEN 'ACCOUNTED'
		WHEN (p.transaction_id IS NOT NULL AND p.paid_at IS NOT NULL) THEN 'UNALLOCATED'
		ELSE 'NOT_APPLICABLE'
	END`, journalVerificationExpr)
}

func (r *paymentRepository) buildAdminPaymentQuery(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) *gorm.DB {
	allocationExpr := buildAllocationStatusExpression()

	// Schema inspection finding on organizer_profiles join:
	// organizer_profiles.organizer_id is protected by a unique index:
	//   CREATE UNIQUE INDEX idx_organizer_profiles_organizer_id ON public.organizer_profiles (organizer_id);
	// This database constraint guarantees at most one organizer_profile row per organizer,
	// ensuring this LEFT JOIN is deterministic and can never duplicate payment records.
	query := r.db.WithContext(ctx).Table("payments AS p").
		Joins("LEFT JOIN bookings AS b ON b.id = p.booking_id").
		Joins("LEFT JOIN users AS u ON u.id = b.user_id").
		Joins("LEFT JOIN events AS e ON e.id = b.event_id").
		Joins("LEFT JOIN organizers AS o ON o.id = e.organizer_id").
		Joins("LEFT JOIN organizer_profiles AS op ON op.organizer_id = o.id")

	if filter.Status != "" && !strings.EqualFold(filter.Status, "ALL") {
		query = query.Where("p.status = ?", strings.ToUpper(filter.Status))
	}

	if filter.AllocationStatus != "" && !strings.EqualFold(filter.AllocationStatus, "ALL") {
		query = query.Where(fmt.Sprintf("(%s) = ?", allocationExpr), strings.ToUpper(filter.AllocationStatus))
	}

	if filter.From != nil {
		fromUTC := filter.From.UTC()
		query = query.Where("COALESCE(p.paid_at, p.created_at) >= ?", fromUTC)
	}

	if filter.To != nil {
		toUTC := filter.To.UTC()
		query = query.Where("COALESCE(p.paid_at, p.created_at) <= ?", toUTC)
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where(
			`(LOWER(COALESCE(p.transaction_id, '')) LIKE ? OR
			  LOWER(COALESCE(p.provider_order_id, '')) LIKE ? OR
			  LOWER(COALESCE(b.booking_number, '')) LIKE ? OR
			  LOWER(COALESCE(u.email, '')) LIKE ? OR
			  LOWER(COALESCE(u.full_name, '')) LIKE ? OR
			  p.id::text LIKE ?)`,
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm,
		)
	}

	if filter.EventID != nil {
		query = query.Where("b.event_id = ?", *filter.EventID)
	}

	return query
}

func (r *paymentRepository) FindAdminPayments(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) (*domain.AdminPaymentListResult, error) {
	const maxSafePage = 100_000
	page := filter.Page
	if page < 1 {
		page = 1
	} else if page > maxSafePage {
		page = maxSafePage
	}

	limit := filter.Limit
	if limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	// Protect against integer overflow in offset calculation
	offset64 := int64(page-1) * int64(limit)
	if offset64 < 0 || offset64 > math.MaxInt32 {
		offset64 = math.MaxInt32
	}
	offset := int(offset64)

	countQuery := r.buildAdminPaymentQuery(ctx, filter)
	var totalRecords int64
	if err := countQuery.Count(&totalRecords).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to count admin payments", slog.Any("error", err))
		return nil, err
	}

	var totalPages int
	if totalRecords > 0 && limit > 0 {
		totalPages = int((totalRecords + int64(limit) - 1) / int64(limit))
	}

	// If no records or requested offset is beyond available records, return empty result without querying
	if totalRecords == 0 || int64(offset) >= totalRecords {
		return &domain.AdminPaymentListResult{
			Payments:     make([]domain.AdminPaymentListItem, 0),
			TotalRecords: totalRecords,
			Page:         page,
			Limit:        limit,
			TotalPages:   totalPages,
		}, nil
	}

	allocationExpr := buildAllocationStatusExpression()
	selectClause := fmt.Sprintf(`
		p.id,
		p.booking_id,
		COALESCE(b.booking_number, '') AS booking_number,
		u.id AS user_id,
		u.full_name AS user_name,
		u.email AS user_email,
		e.id AS event_id,
		e.title AS event_title,
		o.id AS organizer_id,
		op.business_name AS business_name,
		p.amount,
		p.status,
		p.payment_method,
		p.provider,
		p.transaction_id,
		p.provider_order_id,
		p.paid_at,
		p.created_at,
		p.updated_at,
		(p.transaction_id IS NOT NULL AND p.paid_at IS NOT NULL) AS has_gateway_capture,
		(%s) AS allocation_status
	`, allocationExpr)

	var rows []adminPaymentRow
	listQuery := r.buildAdminPaymentQuery(ctx, filter)
	if err := listQuery.Select(selectClause).
		Order("p.created_at DESC, p.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query admin payments", slog.Any("error", err))
		return nil, err
	}

	items := make([]domain.AdminPaymentListItem, 0, len(rows))
	for _, row := range rows {
		var customerID uuid.UUID
		if row.UserID != nil {
			customerID = *row.UserID
		}
		var customerName string
		if row.UserName != nil {
			customerName = *row.UserName
		}
		var customerEmail string
		if row.UserEmail != nil {
			customerEmail = *row.UserEmail
		}
		var eventID uuid.UUID
		if row.EventID != nil {
			eventID = *row.EventID
		}
		var eventTitle string
		if row.EventTitle != nil {
			eventTitle = *row.EventTitle
		}
		var organizerID uuid.UUID
		if row.OrganizerID != nil {
			organizerID = *row.OrganizerID
		}
		var businessName string
		if row.BusinessName != nil {
			businessName = *row.BusinessName
		}

		var paidAtUTC *time.Time
		if row.PaidAt != nil {
			t := row.PaidAt.UTC()
			paidAtUTC = &t
		}

		item := domain.AdminPaymentListItem{
			ID:            row.ID,
			BookingID:     row.BookingID,
			BookingNumber: row.BookingNumber,
			Customer: domain.AdminPaymentCustomerInfo{
				ID:    customerID,
				Name:  customerName,
				Email: customerEmail,
			},
			Event: domain.AdminPaymentEventInfo{
				ID:    eventID,
				Title: eventTitle,
			},
			Organizer: domain.AdminPaymentOrganizerInfo{
				ID:           organizerID,
				BusinessName: businessName,
			},
			Amount:            row.Amount,
			Currency:          finance.CurrencyINR,
			Status:            domain.PaymentStatus(row.Status),
			HasGatewayCapture: row.HasGatewayCapture,
			AllocationStatus:  domain.AllocationStatus(row.AllocationStatus),
			Method:            domain.PaymentMethod(row.PaymentMethod),
			Provider:          domain.PaymentProvider(row.Provider),
			TransactionID:     row.TransactionID,
			ProviderOrderID:   row.ProviderOrderID,
			PaidAt:            paidAtUTC,
			CreatedAt:         row.CreatedAt.UTC(),
			UpdatedAt:         row.UpdatedAt.UTC(),
		}
		items = append(items, item)
	}

	return &domain.AdminPaymentListResult{
		Payments:     items,
		TotalRecords: totalRecords,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	}, nil
}

type adminOverallPaymentSummaryRow struct {
	TotalEvents              int64           `gorm:"column:total_events"`
	GrossCapturedAmount      decimal.Decimal `gorm:"column:gross_captured_amount"`
	RefundRequiredCount      int64           `gorm:"column:refund_required_count"`
	UnallocatedCapturedCount int64           `gorm:"column:unallocated_captured_count"`
}

type adminEventPaymentSummaryRow struct {
	EventID                            uuid.UUID       `gorm:"column:event_id"`
	EventTitle                         string          `gorm:"column:event_title"`
	OrganizerID                        uuid.UUID       `gorm:"column:organizer_id"`
	OrganizerBusinessName              string          `gorm:"column:organizer_business_name"`
	TotalPaymentsCount                 int64           `gorm:"column:total_payments_count"`
	SuccessfullyCompletedPaymentsCount int64           `gorm:"column:successfully_completed_payments_count"`
	GrossCapturedAmount                decimal.Decimal `gorm:"column:gross_captured_amount"`
	PendingPaymentsCount               int64           `gorm:"column:pending_payments_count"`
	FailedPaymentsCount                int64           `gorm:"column:failed_payments_count"`
	RefundRequiredCount                int64           `gorm:"column:refund_required_count"`
	PaymentsWithSuccessfulRefundsCount int64           `gorm:"column:payments_with_successful_refunds_count"`
	UnallocatedCapturedCount           int64           `gorm:"column:unallocated_captured_count"`
}

func (r *paymentRepository) buildAdminEventSummaryBaseQuery(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) *gorm.DB {
	query := r.db.WithContext(ctx).Table("events AS e").
		Joins("JOIN organizers AS o ON o.id = e.organizer_id").
		Joins("LEFT JOIN organizer_profiles AS op ON op.organizer_id = o.id").
		Joins("JOIN bookings AS b ON b.event_id = e.id").
		Joins("JOIN payments AS p ON p.booking_id = b.id")

	if filter.From != nil {
		fromUTC := filter.From.UTC()
		query = query.Where("COALESCE(p.paid_at, p.created_at) >= ?", fromUTC)
	}

	if filter.To != nil {
		toUTC := filter.To.UTC()
		query = query.Where("COALESCE(p.paid_at, p.created_at) <= ?", toUTC)
	}

	if filter.EventID != nil {
		query = query.Where("e.id = ?", *filter.EventID)
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where(
			`(LOWER(COALESCE(e.title, '')) LIKE ? OR
			  LOWER(COALESCE(op.business_name, '')) LIKE ? OR
			  e.id::text LIKE ?)`,
			searchTerm, searchTerm, searchTerm,
		)
	}

	return query
}

func (r *paymentRepository) GetEventPaymentSummaries(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) (*domain.AdminEventPaymentSummaryResult, error) {
	const maxSafePage = 100_000
	page := filter.Page
	if page < 1 {
		page = 1
	} else if page > maxSafePage {
		page = maxSafePage
	}

	limit := filter.Limit
	if limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	offset64 := int64(page-1) * int64(limit)
	if offset64 < 0 || offset64 > math.MaxInt32 {
		offset64 = math.MaxInt32
	}
	offset := int(offset64)

	allocationExpr := buildAllocationStatusExpression()

	// Query A: Overall Summary across all matching events (independent of pagination)
	overallSelectClause := fmt.Sprintf(`
		COUNT(DISTINCT e.id) AS total_events,
		COALESCE(SUM(CASE WHEN (p.transaction_id IS NOT NULL AND p.paid_at IS NOT NULL) THEN p.amount ELSE 0 END), 0) AS gross_captured_amount,
		COUNT(CASE WHEN p.status = 'REFUND_REQUIRED' AND p.amount > COALESCE((
			SELECT SUM(r.amount) 
			FROM refunds r 
			WHERE r.payment_id = p.id 
			  AND r.status = 'SUCCESS' 
			  AND r.gateway_refund_id IS NOT NULL 
			  AND r.processed_at IS NOT NULL
		), 0) THEN 1 END) AS refund_required_count,
		COUNT(CASE WHEN (%s) = 'UNALLOCATED' THEN 1 END) AS unallocated_captured_count
	`, allocationExpr)

	var overallRow adminOverallPaymentSummaryRow
	overallQuery := r.buildAdminEventSummaryBaseQuery(ctx, filter)
	if err := overallQuery.Select(overallSelectClause).Scan(&overallRow).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query overall admin payment summary", slog.Any("error", err))
		return nil, err
	}

	totalRecords := overallRow.TotalEvents
	var totalPages int
	if totalRecords > 0 && limit > 0 {
		totalPages = int((totalRecords + int64(limit) - 1) / int64(limit))
	}

	overallSummary := domain.AdminOverallPaymentSummary{
		TotalEvents:              overallRow.TotalEvents,
		GrossCapturedAmount:      overallRow.GrossCapturedAmount,
		Currency:                 finance.CurrencyINR,
		RefundRequiredCount:      overallRow.RefundRequiredCount,
		UnallocatedCapturedCount: overallRow.UnallocatedCapturedCount,
	}

	// Early return if no events matched or offset exceeds totalRecords
	if totalRecords == 0 || int64(offset) >= totalRecords {
		return &domain.AdminEventPaymentSummaryResult{
			Summaries:      make([]domain.AdminEventPaymentSummary, 0),
			OverallSummary: overallSummary,
			TotalRecords:   totalRecords,
			Page:           page,
			Limit:          limit,
			TotalPages:     totalPages,
		}, nil
	}

	// Query B: Paginated event-wise summaries
	eventSelectClause := fmt.Sprintf(`
		e.id AS event_id,
		COALESCE(e.title, '') AS event_title,
		o.id AS organizer_id,
		COALESCE(op.business_name, '') AS organizer_business_name,
		COUNT(p.id) AS total_payments_count,
		COUNT(CASE WHEN p.status = 'SUCCESS' AND b.status = 'CONFIRMED' THEN 1 END) AS successfully_completed_payments_count,
		COALESCE(SUM(CASE WHEN (p.transaction_id IS NOT NULL AND p.paid_at IS NOT NULL) THEN p.amount ELSE 0 END), 0) AS gross_captured_amount,
		COUNT(CASE WHEN p.status = 'PENDING' THEN 1 END) AS pending_payments_count,
		COUNT(CASE WHEN p.status = 'FAILED' THEN 1 END) AS failed_payments_count,
		COUNT(CASE WHEN p.status = 'REFUND_REQUIRED' AND p.amount > COALESCE((
			SELECT SUM(r.amount) 
			FROM refunds r 
			WHERE r.payment_id = p.id 
			  AND r.status = 'SUCCESS' 
			  AND r.gateway_refund_id IS NOT NULL 
			  AND r.processed_at IS NOT NULL
		), 0) THEN 1 END) AS refund_required_count,
		COUNT(CASE WHEN EXISTS (
			SELECT 1 
			FROM refunds r 
			WHERE r.payment_id = p.id 
			  AND r.status = 'SUCCESS' 
			  AND r.gateway_refund_id IS NOT NULL 
			  AND r.processed_at IS NOT NULL
		) THEN 1 END) AS payments_with_successful_refunds_count,
		COUNT(CASE WHEN (%s) = 'UNALLOCATED' THEN 1 END) AS unallocated_captured_count
	`, allocationExpr)

	var eventRows []adminEventPaymentSummaryRow
	eventQuery := r.buildAdminEventSummaryBaseQuery(ctx, filter)
	if err := eventQuery.Select(eventSelectClause).
		Group("e.id, e.title, o.id, op.business_name").
		Order("gross_captured_amount DESC, e.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&eventRows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query admin event payment summaries", slog.Any("error", err))
		return nil, err
	}

	summaries := make([]domain.AdminEventPaymentSummary, 0, len(eventRows))
	for _, row := range eventRows {
		item := domain.AdminEventPaymentSummary{
			EventID:                            row.EventID,
			EventTitle:                         row.EventTitle,
			OrganizerID:                        row.OrganizerID,
			OrganizerBusinessName:              row.OrganizerBusinessName,
			TotalPaymentsCount:                 row.TotalPaymentsCount,
			SuccessfullyCompletedPaymentsCount: row.SuccessfullyCompletedPaymentsCount,
			GrossCapturedAmount:                row.GrossCapturedAmount,
			Currency:                           finance.CurrencyINR,
			PendingPaymentsCount:               row.PendingPaymentsCount,
			FailedPaymentsCount:                row.FailedPaymentsCount,
			RefundRequiredCount:                row.RefundRequiredCount,
			PaymentsWithSuccessfulRefundsCount: row.PaymentsWithSuccessfulRefundsCount,
			UnallocatedCapturedCount:           row.UnallocatedCapturedCount,
		}
		summaries = append(summaries, item)
	}

	return &domain.AdminEventPaymentSummaryResult{
		Summaries:      summaries,
		OverallSummary: overallSummary,
		TotalRecords:   totalRecords,
		Page:           page,
		Limit:          limit,
		TotalPages:     totalPages,
	}, nil
}
