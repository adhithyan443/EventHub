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

type organizerPaymentRow struct {
	ID                  string          `gorm:"column:id"`
	BookingID           string          `gorm:"column:booking_id"`
	BookingNumber       string          `gorm:"column:booking_number"`
	UserID              *uuid.UUID      `gorm:"column:user_id"`
	UserName            *string         `gorm:"column:user_name"`
	UserEmail           *string         `gorm:"column:user_email"`
	EventID             *uuid.UUID      `gorm:"column:event_id"`
	EventTitle          *string         `gorm:"column:event_title"`
	TicketQuantity      int             `gorm:"column:ticket_quantity"`
	Amount              decimal.Decimal `gorm:"column:amount"`
	Status              string          `gorm:"column:status"`
	PaymentMethod       string          `gorm:"column:payment_method"`
	Provider            string          `gorm:"column:provider"`
	TransactionID       *string         `gorm:"column:transaction_id"`
	ProviderOrderID     *string         `gorm:"column:provider_order_id"`
	PaidAt              *time.Time      `gorm:"column:paid_at"`
	CreatedAt           time.Time       `gorm:"column:created_at"`
	UpdatedAt           time.Time       `gorm:"column:updated_at"`
	HasGatewayCapture   bool            `gorm:"column:has_gateway_capture"`
	AllocationStatus    string          `gorm:"column:allocation_status"`
	RefundedAmount      decimal.Decimal `gorm:"column:refunded_amount"`
	PendingRefundAmount decimal.Decimal `gorm:"column:pending_refund_amount"`
	RefundCount         int             `gorm:"column:refund_count"`
}

type organizerPaymentOverviewRow struct {
	TotalEvents                        int64           `gorm:"column:total_events"`
	TotalPaymentsCount                 int64           `gorm:"column:total_payments_count"`
	SuccessfullyCompletedPaymentsCount int64           `gorm:"column:successfully_completed_payments_count"`
	GrossCapturedAmount                decimal.Decimal `gorm:"column:gross_captured_amount"`
	TotalTicketsSold                   int64           `gorm:"column:total_tickets_sold"`
	TotalTicketSalesAmount             decimal.Decimal `gorm:"column:total_ticket_sales_amount"`
	TotalPlatformFees                  decimal.Decimal `gorm:"column:total_platform_fees"`
	OrganizerEarnings                  decimal.Decimal `gorm:"column:organizer_earnings"`
	RefundRequiredCount                int64           `gorm:"column:refund_required_count"`
	UnallocatedCapturedCount           int64           `gorm:"column:unallocated_captured_count"`
}

type organizerEventPaymentSummaryRow struct {
	EventID                            uuid.UUID       `gorm:"column:event_id"`
	EventTitle                         string          `gorm:"column:event_title"`
	TicketsSold                        int64           `gorm:"column:tickets_sold"`
	GrossCapturedAmount                decimal.Decimal `gorm:"column:gross_captured_amount"`
	PlatformFees                       decimal.Decimal `gorm:"column:platform_fees"`
	OrganizerEarnings                  decimal.Decimal `gorm:"column:organizer_earnings"`
	TotalPaymentsCount                 int64           `gorm:"column:total_payments_count"`
	SuccessfullyCompletedPaymentsCount int64           `gorm:"column:successfully_completed_payments_count"`
	PendingPaymentsCount               int64           `gorm:"column:pending_payments_count"`
	FailedPaymentsCount                int64           `gorm:"column:failed_payments_count"`
	RefundRequiredCount                int64           `gorm:"column:refund_required_count"`
	PaymentsWithSuccessfulRefundsCount int64           `gorm:"column:payments_with_successful_refunds_count"`
	UnallocatedCapturedCount           int64           `gorm:"column:unallocated_captured_count"`
}

const primarySuccessfulPaymentCondition = `(p.status = 'SUCCESS' AND b.status = 'CONFIRMED' AND p.id = (
	SELECT p_primary.id
	FROM payments p_primary
	WHERE p_primary.booking_id = b.id
	  AND p_primary.status = 'SUCCESS'
	ORDER BY p_primary.paid_at DESC NULLS LAST, p_primary.created_at DESC, p_primary.id DESC
	LIMIT 1
))`

func normalizeDateRange(from, to *time.Time) (*time.Time, *time.Time) {
	var fromUTC, toUTC *time.Time
	if from != nil {
		t := from.UTC()
		fromUTC = &t
	}
	if to != nil {
		// If calendar date (midnight with 0 hour, minute, second, nanosecond in its given location),
		// include the complete end date up to 23:59:59.999999999.
		if to.Hour() == 0 && to.Minute() == 0 && to.Second() == 0 && to.Nanosecond() == 0 {
			endOfDay := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999_999_999, to.Location()).UTC()
			toUTC = &endOfDay
		} else {
			t := to.UTC()
			toUTC = &t
		}
	}
	return fromUTC, toUTC
}

func (r *paymentRepository) buildOrganizerPaymentQuery(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) *gorm.DB {
	allocationExpr := buildAllocationStatusExpression()

	query := r.db.WithContext(ctx).Table("payments AS p").
		Joins("JOIN bookings AS b ON b.id = p.booking_id").
		Joins("JOIN events AS e ON e.id = b.event_id").
		Joins("LEFT JOIN users AS u ON u.id = b.user_id").
		Where("e.organizer_id = ?", filter.OrganizerID)

	if filter.Status != "" && !strings.EqualFold(filter.Status, "ALL") {
		query = query.Where("p.status = ?", strings.ToUpper(filter.Status))
	}

	if filter.AllocationStatus != "" && !strings.EqualFold(filter.AllocationStatus, "ALL") {
		query = query.Where(fmt.Sprintf("(%s) = ?", allocationExpr), strings.ToUpper(filter.AllocationStatus))
	}

	fromUTC, toUTC := normalizeDateRange(filter.From, filter.To)
	if fromUTC != nil {
		query = query.Where("COALESCE(p.paid_at, p.created_at) >= ?", *fromUTC)
	}

	if toUTC != nil {
		query = query.Where("COALESCE(p.paid_at, p.created_at) <= ?", *toUTC)
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

func (r *paymentRepository) FindOrganizerPayments(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerPaymentListResult, error) {
	if filter.OrganizerID == uuid.Nil {
		return nil, domain.ErrInvalidOrganizerID
	}

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

	countQuery := r.buildOrganizerPaymentQuery(ctx, filter)
	var totalRecords int64
	if err := countQuery.Count(&totalRecords).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to count organizer payments", slog.Any("error", err), slog.String("organizer_id", filter.OrganizerID.String()))
		return nil, err
	}

	var totalPages int
	if totalRecords > 0 && limit > 0 {
		totalPages = int((totalRecords + int64(limit) - 1) / int64(limit))
	}

	if totalRecords == 0 || int64(offset) >= totalRecords {
		return &domain.OrganizerPaymentListResult{
			Payments:     make([]domain.OrganizerPaymentListItem, 0),
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
		COALESCE((SELECT SUM(bi.quantity) FROM booking_items bi WHERE bi.booking_id = b.id), 0) AS ticket_quantity,
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
		(%s) AS allocation_status,
		COALESCE((
			SELECT SUM(r.amount) 
			FROM refunds r 
			WHERE r.payment_id = p.id AND r.status = 'SUCCESS'
		), 0) AS refunded_amount,
		COALESCE((
			SELECT SUM(r.amount) 
			FROM refunds r 
			WHERE r.payment_id = p.id AND r.status IN ('PENDING', 'PROCESSING')
		), 0) AS pending_refund_amount,
		COALESCE((
			SELECT COUNT(r.id) 
			FROM refunds r 
			WHERE r.payment_id = p.id
		), 0) AS refund_count
	`, allocationExpr)

	var rows []organizerPaymentRow
	listQuery := r.buildOrganizerPaymentQuery(ctx, filter)
	if err := listQuery.Select(selectClause).
		Order("p.created_at DESC, p.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query organizer payments", slog.Any("error", err), slog.String("organizer_id", filter.OrganizerID.String()))
		return nil, err
	}

	paymentUUIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if id, err := uuid.Parse(row.ID); err == nil {
			paymentUUIDs = append(paymentUUIDs, id)
		}
	}

	type paymentRefundRow struct {
		ID              uuid.UUID       `gorm:"column:id"`
		PaymentID       uuid.UUID       `gorm:"column:payment_id"`
		GatewayRefundID *string         `gorm:"column:gateway_refund_id"`
		Amount          decimal.Decimal `gorm:"column:amount"`
		Status          string          `gorm:"column:status"`
		Reason          string          `gorm:"column:reason"`
		CreatedAt       time.Time       `gorm:"column:created_at"`
		ProcessedAt     *time.Time      `gorm:"column:processed_at"`
	}

	refundsByPaymentID := make(map[uuid.UUID][]domain.PaymentRefundItem)
	if len(paymentUUIDs) > 0 {
		var refundRows []paymentRefundRow
		if err := r.db.WithContext(ctx).Table("refunds").
			Select("id, payment_id, gateway_refund_id, amount, status, reason, created_at, processed_at").
			Where("payment_id IN ?", paymentUUIDs).
			Order("created_at DESC, id DESC").
			Scan(&refundRows).Error; err != nil {
			r.logger.ErrorContext(ctx, "failed to query refunds for organizer payments", slog.Any("error", err), slog.String("organizer_id", filter.OrganizerID.String()))
			return nil, err
		}
		for _, rf := range refundRows {
			var procAt *time.Time
			if rf.ProcessedAt != nil {
				t := rf.ProcessedAt.UTC()
				procAt = &t
			}
			refundsByPaymentID[rf.PaymentID] = append(refundsByPaymentID[rf.PaymentID], domain.PaymentRefundItem{
				ID:              rf.ID,
				GatewayRefundID: rf.GatewayRefundID,
				Amount:          rf.Amount,
				Status:          rf.Status,
				Reason:          rf.Reason,
				CreatedAt:       rf.CreatedAt.UTC(),
				ProcessedAt:     procAt,
			})
		}
	}

	items := make([]domain.OrganizerPaymentListItem, 0, len(rows))
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

		var paidAtUTC *time.Time
		if row.PaidAt != nil {
			t := row.PaidAt.UTC()
			paidAtUTC = &t
		}

		var payUUID uuid.UUID
		if parsed, err := uuid.Parse(row.ID); err == nil {
			payUUID = parsed
		}

		paymentRefunds := refundsByPaymentID[payUUID]
		if paymentRefunds == nil {
			paymentRefunds = []domain.PaymentRefundItem{}
		}

		netAmount, refundStatus := computePaymentRefundDetails(
			row.Amount,
			row.RefundedAmount,
			row.PendingRefundAmount,
			row.RefundCount,
		)

		item := domain.OrganizerPaymentListItem{
			ID:            row.ID,
			BookingID:     row.BookingID,
			BookingNumber: row.BookingNumber,
			Customer: domain.OrganizerPaymentCustomerInfo{
				ID:    customerID,
				Name:  customerName,
				Email: customerEmail,
			},
			Event: domain.OrganizerPaymentEventInfo{
				ID:    eventID,
				Title: eventTitle,
			},
			TicketQuantity:      row.TicketQuantity,
			Amount:              row.Amount,
			Currency:            finance.CurrencyINR,
			Status:              domain.PaymentStatus(row.Status),
			HasGatewayCapture:   row.HasGatewayCapture,
			AllocationStatus:    domain.AllocationStatus(row.AllocationStatus),
			RefundedAmount:      row.RefundedAmount,
			PendingRefundAmount: row.PendingRefundAmount,
			RefundStatus:        refundStatus,
			NetAmount:           netAmount,
			RefundCount:         row.RefundCount,
			Refunds:             paymentRefunds,
			Method:              domain.PaymentMethod(row.PaymentMethod),
			Provider:            domain.PaymentProvider(row.Provider),
			TransactionID:       row.TransactionID,
			ProviderOrderID:     row.ProviderOrderID,
			PaidAt:              paidAtUTC,
			CreatedAt:           row.CreatedAt.UTC(),
			UpdatedAt:           row.UpdatedAt.UTC(),
		}
		items = append(items, item)
	}

	return &domain.OrganizerPaymentListResult{
		Payments:     items,
		TotalRecords: totalRecords,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	}, nil
}

func (r *paymentRepository) buildOrganizerEventSummaryBaseQuery(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) *gorm.DB {
	query := r.db.WithContext(ctx).Table("events AS e").
		Joins("JOIN bookings AS b ON b.event_id = e.id").
		Joins("JOIN payments AS p ON p.booking_id = b.id").
		Where("e.organizer_id = ?", filter.OrganizerID)

	if filter.Status != "" && !strings.EqualFold(filter.Status, "ALL") {
		query = query.Where("p.status = ?", strings.ToUpper(filter.Status))
	}

	if filter.AllocationStatus != "" && !strings.EqualFold(filter.AllocationStatus, "ALL") {
		allocationExpr := buildAllocationStatusExpression()
		query = query.Where(fmt.Sprintf("(%s) = ?", allocationExpr), strings.ToUpper(filter.AllocationStatus))
	}

	fromUTC, toUTC := normalizeDateRange(filter.From, filter.To)
	if fromUTC != nil {
		query = query.Where("COALESCE(p.paid_at, p.created_at) >= ?", *fromUTC)
	}

	if toUTC != nil {
		query = query.Where("COALESCE(p.paid_at, p.created_at) <= ?", *toUTC)
	}

	if filter.EventID != nil {
		query = query.Where("e.id = ?", *filter.EventID)
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where(
			`(LOWER(COALESCE(e.title, '')) LIKE ? OR e.id::text LIKE ?)`,
			searchTerm, searchTerm,
		)
	}

	return query
}

func (r *paymentRepository) getOrganizerLedgerEarnings(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (decimal.Decimal, decimal.Decimal, error) {
	// If the filter targets unallocated payments or non-successful statuses,
	// by financial definition there are zero accounted ledger earnings.
	if strings.EqualFold(filter.AllocationStatus, "UNALLOCATED") ||
		strings.EqualFold(filter.Status, "FAILED") ||
		strings.EqualFold(filter.Status, "PENDING") ||
		strings.EqualFold(filter.Status, "REFUND_REQUIRED") {
		return decimal.Zero, decimal.Zero, nil
	}

	type ledgerSumRow struct {
		OrganizerEarnings decimal.Decimal `gorm:"column:organizer_earnings"`
		PlatformFees      decimal.Decimal `gorm:"column:platform_fees"`
	}

	query := r.db.WithContext(ctx).Table("finance_ledger_entries AS fle").
		Select(`
			COALESCE(SUM(CASE 
				WHEN fa.code = ? AND fle.entry_type = 'CREDIT' AND fj.financial_event_type = ? THEN fle.amount
				WHEN fa.code = ? AND fle.entry_type = 'DEBIT' AND (
					fj.financial_event_type IN (?, ?) OR (fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type = ?)
				) THEN -fle.amount
				WHEN fa.code = ? AND fle.entry_type = 'CREDIT' AND fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type IN (?, ?) THEN fle.amount
				ELSE 0
			END), 0) AS organizer_earnings,
			COALESCE(SUM(CASE 
				WHEN fa.code = ? AND fle.entry_type = 'CREDIT' AND fj.financial_event_type = ? THEN fle.amount
				WHEN fa.code = ? AND fle.entry_type = 'DEBIT' AND (
					fj.financial_event_type IN (?, ?) OR (fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type = ?)
				) THEN -fle.amount
				WHEN fa.code = ? AND fle.entry_type = 'CREDIT' AND fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type IN (?, ?) THEN fle.amount
				ELSE 0
			END), 0) AS platform_fees
		`,
			finance.AccountCodeOrganizerPayable, string(finance.FinancialEventTicketSaleCaptured),
			finance.AccountCodeOrganizerPayable, string(finance.FinancialEventCustomerRefund), string(finance.FinancialEventEventCancellationRefund), string(finance.FinancialEventTicketSaleCaptured),
			finance.AccountCodeOrganizerPayable, string(finance.FinancialEventCustomerRefund), string(finance.FinancialEventEventCancellationRefund),

			finance.AccountCodePlatformFeeRevenue, string(finance.FinancialEventTicketSaleCaptured),
			finance.AccountCodePlatformFeeRevenue, string(finance.FinancialEventCustomerRefund), string(finance.FinancialEventEventCancellationRefund), string(finance.FinancialEventTicketSaleCaptured),
			finance.AccountCodePlatformFeeRevenue, string(finance.FinancialEventCustomerRefund), string(finance.FinancialEventEventCancellationRefund),
		).
		Joins("JOIN finance_journals fj ON fj.id = fle.journal_id").
		Joins("LEFT JOIN finance_journals orig_fj ON orig_fj.id = fj.reversal_reference_id").
		Joins("JOIN finance_accounts fa ON fa.id = fle.account_id").
		Joins("JOIN events e ON e.id = fle.event_id").
		Where("fa.code IN (?, ?)", finance.AccountCodeOrganizerPayable, finance.AccountCodePlatformFeeRevenue).
		Where("fj.status = ?", string(finance.PostingStatusPosted)).
		Where("e.organizer_id = ?", filter.OrganizerID)

	if filter.EventID != nil {
		query = query.Where("fle.event_id = ?", *filter.EventID)
	}

	fromUTC, toUTC := normalizeDateRange(filter.From, filter.To)
	if fromUTC != nil {
		query = query.Where("COALESCE(fj.posted_at, fj.created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		query = query.Where("COALESCE(fj.posted_at, fj.created_at) <= ?", *toUTC)
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where(
			`(LOWER(COALESCE(e.title, '')) LIKE ? OR e.id::text LIKE ?)`,
			searchTerm, searchTerm,
		)
	}

	var row ledgerSumRow
	if err := query.Scan(&row).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query organizer ledger earnings",
			slog.Any("error", err),
			slog.String("organizer_id", filter.OrganizerID.String()),
		)
		return decimal.Zero, decimal.Zero, err
	}

	return row.OrganizerEarnings, row.PlatformFees, nil
}

type eventLedgerAmounts struct {
	OrganizerEarnings decimal.Decimal
	PlatformFees      decimal.Decimal
}

func (r *paymentRepository) getOrganizerEventLedgerEarnings(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (map[uuid.UUID]eventLedgerAmounts, error) {
	result := make(map[uuid.UUID]eventLedgerAmounts)
	if strings.EqualFold(filter.AllocationStatus, "UNALLOCATED") ||
		strings.EqualFold(filter.Status, "FAILED") ||
		strings.EqualFold(filter.Status, "PENDING") ||
		strings.EqualFold(filter.Status, "REFUND_REQUIRED") {
		return result, nil
	}

	type eventLedgerRow struct {
		EventID           uuid.UUID       `gorm:"column:event_id"`
		OrganizerEarnings decimal.Decimal `gorm:"column:organizer_earnings"`
		PlatformFees      decimal.Decimal `gorm:"column:platform_fees"`
	}

	query := r.db.WithContext(ctx).Table("finance_ledger_entries AS fle").
		Select(`
			fle.event_id,
			COALESCE(SUM(CASE 
				WHEN fa.code = ? AND fle.entry_type = 'CREDIT' AND fj.financial_event_type = ? THEN fle.amount
				WHEN fa.code = ? AND fle.entry_type = 'DEBIT' AND (
					fj.financial_event_type IN (?, ?) OR (fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type = ?)
				) THEN -fle.amount
				WHEN fa.code = ? AND fle.entry_type = 'CREDIT' AND fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type IN (?, ?) THEN fle.amount
				ELSE 0
			END), 0) AS organizer_earnings,
			COALESCE(SUM(CASE 
				WHEN fa.code = ? AND fle.entry_type = 'CREDIT' AND fj.financial_event_type = ? THEN fle.amount
				WHEN fa.code = ? AND fle.entry_type = 'DEBIT' AND (
					fj.financial_event_type IN (?, ?) OR (fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type = ?)
				) THEN -fle.amount
				WHEN fa.code = ? AND fle.entry_type = 'CREDIT' AND fj.transaction_type = 'REVERSAL' AND orig_fj.financial_event_type IN (?, ?) THEN fle.amount
				ELSE 0
			END), 0) AS platform_fees
		`,
			finance.AccountCodeOrganizerPayable, string(finance.FinancialEventTicketSaleCaptured),
			finance.AccountCodeOrganizerPayable, string(finance.FinancialEventCustomerRefund), string(finance.FinancialEventEventCancellationRefund), string(finance.FinancialEventTicketSaleCaptured),
			finance.AccountCodeOrganizerPayable, string(finance.FinancialEventCustomerRefund), string(finance.FinancialEventEventCancellationRefund),

			finance.AccountCodePlatformFeeRevenue, string(finance.FinancialEventTicketSaleCaptured),
			finance.AccountCodePlatformFeeRevenue, string(finance.FinancialEventCustomerRefund), string(finance.FinancialEventEventCancellationRefund), string(finance.FinancialEventTicketSaleCaptured),
			finance.AccountCodePlatformFeeRevenue, string(finance.FinancialEventCustomerRefund), string(finance.FinancialEventEventCancellationRefund),
		).
		Joins("JOIN finance_journals fj ON fj.id = fle.journal_id").
		Joins("LEFT JOIN finance_journals orig_fj ON orig_fj.id = fj.reversal_reference_id").
		Joins("JOIN finance_accounts fa ON fa.id = fle.account_id").
		Joins("JOIN events e ON e.id = fle.event_id").
		Where("fa.code IN (?, ?)", finance.AccountCodeOrganizerPayable, finance.AccountCodePlatformFeeRevenue).
		Where("fj.status = ?", string(finance.PostingStatusPosted)).
		Where("e.organizer_id = ?", filter.OrganizerID).
		Where("fle.event_id IS NOT NULL")

	if filter.EventID != nil {
		query = query.Where("fle.event_id = ?", *filter.EventID)
	}

	fromUTC, toUTC := normalizeDateRange(filter.From, filter.To)
	if fromUTC != nil {
		query = query.Where("COALESCE(fj.posted_at, fj.created_at) >= ?", *fromUTC)
	}
	if toUTC != nil {
		query = query.Where("COALESCE(fj.posted_at, fj.created_at) <= ?", *toUTC)
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where(
			`(LOWER(COALESCE(e.title, '')) LIKE ? OR e.id::text LIKE ?)`,
			searchTerm, searchTerm,
		)
	}

	var rows []eventLedgerRow
	if err := query.Group("fle.event_id").Scan(&rows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query event ledger earnings",
			slog.Any("error", err),
			slog.String("organizer_id", filter.OrganizerID.String()),
		)
		return nil, err
	}

	for _, row := range rows {
		result[row.EventID] = eventLedgerAmounts{
			OrganizerEarnings: row.OrganizerEarnings,
			PlatformFees:      row.PlatformFees,
		}
	}

	return result, nil
}

func (r *paymentRepository) GetOrganizerPaymentOverview(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerPaymentOverview, error) {
	if filter.OrganizerID == uuid.Nil {
		return nil, domain.ErrInvalidOrganizerID
	}

	allocationExpr := buildAllocationStatusExpression()
	overviewSelectClause := fmt.Sprintf(`
		COUNT(DISTINCT e.id) AS total_events,
		COUNT(p.id) AS total_payments_count,
		COUNT(CASE WHEN p.status = 'SUCCESS' AND b.status = 'CONFIRMED' THEN 1 END) AS successfully_completed_payments_count,
		COALESCE(SUM(CASE WHEN (p.transaction_id IS NOT NULL AND p.paid_at IS NOT NULL) THEN p.amount ELSE 0 END), 0) AS gross_captured_amount,
		COALESCE(SUM(CASE WHEN %s THEN (SELECT COALESCE(SUM(bi.quantity), 0) FROM booking_items bi WHERE bi.booking_id = b.id) ELSE 0 END), 0) AS total_tickets_sold,
		COUNT(CASE WHEN p.status = 'REFUND_REQUIRED' AND p.amount > COALESCE((
			SELECT SUM(r.amount) 
			FROM refunds r 
			WHERE r.payment_id = p.id 
			  AND r.status = 'SUCCESS' 
			  AND r.gateway_refund_id IS NOT NULL 
			  AND r.processed_at IS NOT NULL
		), 0) THEN 1 END) AS refund_required_count,
		COUNT(CASE WHEN (%s) = 'UNALLOCATED' THEN 1 END) AS unallocated_captured_count
	`, primarySuccessfulPaymentCondition, allocationExpr)

	var row organizerPaymentOverviewRow
	query := r.buildOrganizerEventSummaryBaseQuery(ctx, filter)
	if err := query.Select(overviewSelectClause).Scan(&row).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query organizer payment overview", slog.Any("error", err), slog.String("organizer_id", filter.OrganizerID.String()))
		return nil, err
	}

	// Authoritative organizer earnings and platform fees derived from Account 2010 and 4010 ledger entries
	ledgerEarnings, ledgerPlatformFees, err := r.getOrganizerLedgerEarnings(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &domain.OrganizerPaymentOverview{
		GrossCapturedAmount:                row.GrossCapturedAmount,
		TotalTicketsSold:                   row.TotalTicketsSold,
		TotalTicketSalesAmount:             ledgerEarnings,
		TotalPlatformFees:                  ledgerPlatformFees,
		OrganizerEarnings:                  ledgerEarnings,
		Currency:                           finance.CurrencyINR,
		TotalEvents:                        row.TotalEvents,
		TotalPaymentsCount:                 row.TotalPaymentsCount,
		SuccessfullyCompletedPaymentsCount: row.SuccessfullyCompletedPaymentsCount,
		RefundRequiredCount:                row.RefundRequiredCount,
		UnallocatedCapturedCount:           row.UnallocatedCapturedCount,
	}, nil
}

func (r *paymentRepository) GetOrganizerEventPaymentSummaries(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerEventPaymentSummaryResult, error) {
	if filter.OrganizerID == uuid.Nil {
		return nil, domain.ErrInvalidOrganizerID
	}

	overview, err := r.GetOrganizerPaymentOverview(ctx, filter)
	if err != nil {
		return nil, err
	}

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

	totalRecords := overview.TotalEvents
	var totalPages int
	if totalRecords > 0 && limit > 0 {
		totalPages = int((totalRecords + int64(limit) - 1) / int64(limit))
	}

	if totalRecords == 0 || int64(offset) >= totalRecords {
		return &domain.OrganizerEventPaymentSummaryResult{
			Summaries:    make([]domain.OrganizerEventPaymentSummary, 0),
			Overview:     *overview,
			TotalRecords: totalRecords,
			Page:         page,
			Limit:        limit,
			TotalPages:   totalPages,
		}, nil
	}

	allocationExpr := buildAllocationStatusExpression()
	eventSelectClause := fmt.Sprintf(`
		e.id AS event_id,
		COALESCE(e.title, '') AS event_title,
		COALESCE(SUM(CASE WHEN %s THEN (SELECT COALESCE(SUM(bi.quantity), 0) FROM booking_items bi WHERE bi.booking_id = b.id) ELSE 0 END), 0) AS tickets_sold,
		COALESCE(SUM(CASE WHEN (p.transaction_id IS NOT NULL AND p.paid_at IS NOT NULL) THEN p.amount ELSE 0 END), 0) AS gross_captured_amount,
		COUNT(p.id) AS total_payments_count,
		COUNT(CASE WHEN p.status = 'SUCCESS' AND b.status = 'CONFIRMED' THEN 1 END) AS successfully_completed_payments_count,
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
	`, primarySuccessfulPaymentCondition, allocationExpr)

	var eventRows []organizerEventPaymentSummaryRow
	eventQuery := r.buildOrganizerEventSummaryBaseQuery(ctx, filter)
	if err := eventQuery.Select(eventSelectClause).
		Group("e.id, e.title").
		Order("gross_captured_amount DESC, e.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&eventRows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query organizer event payment summaries", slog.Any("error", err), slog.String("organizer_id", filter.OrganizerID.String()))
		return nil, err
	}

	eventLedgerMap, err := r.getOrganizerEventLedgerEarnings(ctx, filter)
	if err != nil {
		return nil, err
	}

	summaries := make([]domain.OrganizerEventPaymentSummary, 0, len(eventRows))
	for _, row := range eventRows {
		eventAmounts := eventLedgerMap[row.EventID]
		item := domain.OrganizerEventPaymentSummary{
			EventID:                            row.EventID,
			EventTitle:                         row.EventTitle,
			TicketsSold:                        row.TicketsSold,
			GrossCapturedAmount:                row.GrossCapturedAmount,
			PlatformFees:                       eventAmounts.PlatformFees,
			OrganizerEarnings:                  eventAmounts.OrganizerEarnings,
			Currency:                           finance.CurrencyINR,
			TotalPaymentsCount:                 row.TotalPaymentsCount,
			SuccessfullyCompletedPaymentsCount: row.SuccessfullyCompletedPaymentsCount,
			PendingPaymentsCount:               row.PendingPaymentsCount,
			FailedPaymentsCount:                row.FailedPaymentsCount,
			RefundRequiredCount:                row.RefundRequiredCount,
			PaymentsWithSuccessfulRefundsCount: row.PaymentsWithSuccessfulRefundsCount,
			UnallocatedCapturedCount:           row.UnallocatedCapturedCount,
		}
		summaries = append(summaries, item)
	}

	return &domain.OrganizerEventPaymentSummaryResult{
		Summaries:    summaries,
		Overview:     *overview,
		TotalRecords: totalRecords,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	}, nil
}
