package repository

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type adminBookingRow struct {
	ID                   uuid.UUID       `gorm:"column:id"`
	BookingNumber        string          `gorm:"column:booking_number"`
	CreatedAt            time.Time       `gorm:"column:created_at"`
	UserID               uuid.UUID       `gorm:"column:user_id"`
	UserName             string          `gorm:"column:user_name"`
	UserEmail            string          `gorm:"column:user_email"`
	EventID              uuid.UUID       `gorm:"column:event_id"`
	EventTitle           string          `gorm:"column:event_title"`
	EventType            string          `gorm:"column:event_type"`
	BannerURL            string          `gorm:"column:banner_url"`
	VenueName            string          `gorm:"column:venue_name"`
	OrganizerID          uuid.UUID       `gorm:"column:organizer_id"`
	OrganizerBusiness    string          `gorm:"column:organizer_business_name"`
	OrganizerEmail       string          `gorm:"column:organizer_email"`
	TicketCount          int             `gorm:"column:ticket_count"`
	OriginalBookingTotal decimal.Decimal `gorm:"column:original_booking_total"`
	BookingStatus        string          `gorm:"column:booking_status"`
	PaymentStatus        string          `gorm:"column:payment_status"`
	PaymentMethod        string          `gorm:"column:payment_method"`
	RefundStatus         string          `gorm:"column:refund_status"`
}

type adminBookingDetailRow struct {
	ID                        uuid.UUID       `gorm:"column:id"`
	BookingNumber             string          `gorm:"column:booking_number"`
	CreatedAt                 time.Time       `gorm:"column:created_at"`
	UpdatedAt                 time.Time       `gorm:"column:updated_at"`
	BookingStatus             string          `gorm:"column:booking_status"`
	UserID                    uuid.UUID       `gorm:"column:user_id"`
	UserName                  string          `gorm:"column:user_name"`
	UserEmail                 string          `gorm:"column:user_email"`
	EventID                   uuid.UUID       `gorm:"column:event_id"`
	EventTitle                string          `gorm:"column:event_title"`
	EventType                 string          `gorm:"column:event_type"`
	BannerURL                 string          `gorm:"column:banner_url"`
	VenueName                 string          `gorm:"column:venue_name"`
	OrganizerID               uuid.UUID       `gorm:"column:organizer_id"`
	OrganizerBusiness         string          `gorm:"column:organizer_business_name"`
	OrganizerEmail            string          `gorm:"column:organizer_email"`
	Subtotal                  decimal.Decimal `gorm:"column:subtotal"`
	PlatformFee               decimal.Decimal `gorm:"column:platform_fee"`
	Tax                       decimal.Decimal `gorm:"column:tax"`
	Discount                  decimal.Decimal `gorm:"column:discount"`
	Total                     decimal.Decimal `gorm:"column:total"`
	CancellationAllowed       bool            `gorm:"column:cancellation_allowed"`
	CancellationDeadlineHours int             `gorm:"column:cancellation_deadline_hours"`
	RefundPolicy              string          `gorm:"column:cancellation_refund_policy"`
	RefundPercentage          int             `gorm:"column:cancellation_refund_percentage"`
}

func (r *bookingRepository) buildAdminBookingBaseQuery(
	ctx context.Context,
	filter domain.AdminBookingFilter,
) *gorm.DB {
	query := r.db.WithContext(ctx).Table("bookings AS b").
		Joins("JOIN events AS e ON e.id = b.event_id").
		Joins("LEFT JOIN users AS u ON u.id = b.user_id").
		Joins("LEFT JOIN organizers AS o ON o.id = e.organizer_id").
		Joins("LEFT JOIN organizer_profiles AS op ON op.organizer_id = o.id").
		Joins("LEFT JOIN users AS ou ON ou.id = o.user_id")

	if filter.OrganizerID != nil && *filter.OrganizerID != uuid.Nil {
		query = query.Where("e.organizer_id = ?", *filter.OrganizerID)
	}

	if filter.EventID != nil && *filter.EventID != uuid.Nil {
		query = query.Where("b.event_id = ?", *filter.EventID)
	}

	if filter.BookingStatus != "" && !strings.EqualFold(filter.BookingStatus, "ALL") {
		query = query.Where("b.status = ?", strings.ToUpper(strings.TrimSpace(filter.BookingStatus)))
	}

	if filter.PaymentStatus != "" && !strings.EqualFold(filter.PaymentStatus, "ALL") {
		status := strings.ToUpper(strings.TrimSpace(filter.PaymentStatus))
		switch status {
		case "REFUNDED":
			query = query.Where(`EXISTS (
				SELECT 1 FROM refunds r 
				WHERE r.booking_id = b.id AND r.status = 'SUCCESS'
			)`)
		case "FULL":
			query = query.Where(`(
				SELECT COALESCE(SUM(p.amount), 0) 
				FROM payments p 
				WHERE p.booking_id = b.id AND p.status = 'SUCCESS' AND p.paid_at IS NOT NULL
			) > 0 AND (
				SELECT COALESCE(SUM(r.amount), 0) 
				FROM refunds r 
				WHERE r.booking_id = b.id AND r.status = 'SUCCESS'
			) >= (
				SELECT COALESCE(SUM(p.amount), 0) 
				FROM payments p 
				WHERE p.booking_id = b.id AND p.status = 'SUCCESS' AND p.paid_at IS NOT NULL
			) AND (
				SELECT COALESCE(SUM(r.amount), 0) 
				FROM refunds r 
				WHERE r.booking_id = b.id AND r.status = 'SUCCESS'
			) > 0`)
		case "PARTIAL":
			query = query.Where(`(
				SELECT COALESCE(SUM(r.amount), 0) 
				FROM refunds r 
				WHERE r.booking_id = b.id AND r.status = 'SUCCESS'
			) > 0 AND (
				SELECT COALESCE(SUM(r.amount), 0) 
				FROM refunds r 
				WHERE r.booking_id = b.id AND r.status = 'SUCCESS'
			) < (
				SELECT COALESCE(SUM(p.amount), 0) 
				FROM payments p 
				WHERE p.booking_id = b.id AND p.status = 'SUCCESS' AND p.paid_at IS NOT NULL
			)`)
		case "PENDING_REFUND":
			query = query.Where(`EXISTS (
				SELECT 1 FROM refunds r 
				WHERE r.booking_id = b.id AND r.status IN ('PENDING', 'PROCESSING')
			)`)
		default:
			query = query.Where(`COALESCE((
				SELECT p.status 
				FROM payments p 
				WHERE p.booking_id = b.id 
				ORDER BY (p.status = 'SUCCESS') DESC, p.created_at DESC, p.id DESC 
				LIMIT 1
			), 'PENDING') = ?`, status)
		}
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where(
			"(LOWER(b.booking_number) LIKE ? OR LOWER(u.full_name) LIKE ? OR LOWER(u.email) LIKE ?)",
			searchPattern, searchPattern, searchPattern,
		)
	}

	fromUTC, toUTC := normalizeDateRange(filter.From, filter.To)
	if fromUTC != nil {
		query = query.Where("b.created_at >= ?", *fromUTC)
	}
	if toUTC != nil {
		query = query.Where("b.created_at <= ?", *toUTC)
	}

	return query
}

// FindAdminBookings returns a paginated list of bookings across the platform.
func (r *bookingRepository) FindAdminBookings(
	ctx context.Context,
	filter domain.AdminBookingFilter,
) (*domain.AdminBookingListResult, error) {
	baseQuery := r.buildAdminBookingBaseQuery(ctx, filter)

	var totalRecords int64
	if err := baseQuery.Count(&totalRecords).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to count admin bookings",
			slog.Any("error", err),
		)
		return nil, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	totalPages := 0
	if totalRecords > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(limit)))
	}

	if totalRecords == 0 || offset >= int(totalRecords) {
		return &domain.AdminBookingListResult{
			Bookings:     make([]domain.AdminBookingListItem, 0),
			TotalRecords: totalRecords,
			Page:         page,
			Limit:        limit,
			TotalPages:   totalPages,
		}, nil
	}

	selectClause := `
		b.id,
		b.booking_number,
		b.created_at,
		b.status AS booking_status,
		b.total AS original_booking_total,
		u.id AS user_id,
		COALESCE(u.full_name, '') AS user_name,
		COALESCE(u.email, '') AS user_email,
		e.id AS event_id,
		COALESCE(e.title, '') AS event_title,
		COALESCE(e.event_type, '') AS event_type,
		COALESCE(e.banner_url, '') AS banner_url,
		COALESCE(v.name, '') AS venue_name,
		o.id AS organizer_id,
		COALESCE(NULLIF(op.business_name, ''), NULLIF(ou.full_name, ''), 'Unknown Organizer') AS organizer_business_name,
		COALESCE(ou.email, '') AS organizer_email,
		CASE 
			WHEN EXISTS (SELECT 1 FROM tickets t WHERE t.booking_id = b.id) 
			THEN (SELECT COUNT(t.id) FROM tickets t WHERE t.booking_id = b.id)
			ELSE (SELECT COALESCE(SUM(bi.quantity), 0) FROM booking_items bi WHERE bi.booking_id = b.id)
		END AS ticket_count,
		COALESCE((
			SELECT p.status 
			FROM payments p 
			WHERE p.booking_id = b.id 
			ORDER BY (p.status = 'SUCCESS') DESC, p.created_at DESC, p.id DESC 
			LIMIT 1
		), 'PENDING') AS payment_status,
		COALESCE((
			SELECT p.payment_method 
			FROM payments p 
			WHERE p.booking_id = b.id 
			ORDER BY (p.status = 'SUCCESS') DESC, p.created_at DESC, p.id DESC 
			LIMIT 1
		), '') AS payment_method,
		COALESCE((
			SELECT 
				CASE 
					WHEN NOT EXISTS (SELECT 1 FROM refunds r WHERE r.booking_id = b.id) THEN ''
					WHEN (
						SELECT COALESCE(SUM(p.amount), 0) 
						FROM payments p 
						WHERE p.booking_id = b.id AND p.status = 'SUCCESS' AND p.paid_at IS NOT NULL
					) > 0 
					AND COALESCE(SUM(CASE WHEN r.status = 'SUCCESS' THEN r.amount ELSE 0 END), 0) >= (
						SELECT COALESCE(SUM(p.amount), 0) 
						FROM payments p 
						WHERE p.booking_id = b.id AND p.status = 'SUCCESS' AND p.paid_at IS NOT NULL
					)
					AND COALESCE(SUM(CASE WHEN r.status = 'SUCCESS' THEN r.amount ELSE 0 END), 0) > 0
						THEN 'FULL'
					WHEN bool_or(r.status IN ('PENDING', 'PROCESSING')) THEN 'PENDING'
					WHEN COALESCE(SUM(CASE WHEN r.status = 'SUCCESS' THEN r.amount ELSE 0 END), 0) > 0 THEN 'PARTIAL'
					WHEN bool_or(r.status = 'FAILED') THEN 'FAILED'
					ELSE ''
				END
			FROM refunds r
			WHERE r.booking_id = b.id
		), '') AS refund_status
	`

	var rows []adminBookingRow
	err := r.buildAdminBookingBaseQuery(ctx, filter).
		Joins("LEFT JOIN venues AS v ON v.id = e.venue_id").
		Select(selectClause).
		Order("b.created_at DESC, b.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to query admin bookings",
			slog.Any("error", err),
		)
		return nil, err
	}

	bookings := make([]domain.AdminBookingListItem, len(rows))
	for i, row := range rows {
		bookings[i] = domain.AdminBookingListItem{
			ID:            row.ID,
			BookingNumber: row.BookingNumber,
			CreatedAt:     row.CreatedAt,
			Customer: domain.OrganizerCustomerSummary{
				ID:       row.UserID,
				FullName: row.UserName,
				Email:    row.UserEmail,
			},
			Event: domain.OrganizerEventSummary{
				ID:        row.EventID,
				Title:     row.EventTitle,
				EventType: row.EventType,
				BannerURL: row.BannerURL,
				VenueName: row.VenueName,
			},
			Organizer: domain.AdminOrganizerBrief{
				ID:           row.OrganizerID,
				BusinessName: row.OrganizerBusiness,
				Email:        row.OrganizerEmail,
			},
			TicketCount:          row.TicketCount,
			OriginalBookingTotal: row.OriginalBookingTotal,
			Currency:             "INR",
			BookingStatus:        row.BookingStatus,
			PaymentStatus:        row.PaymentStatus,
			PaymentMethod:        row.PaymentMethod,
			RefundStatus:         row.RefundStatus,
		}
	}

	return &domain.AdminBookingListResult{
		Bookings:     bookings,
		TotalRecords: totalRecords,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	}, nil
}

// GetAdminBookingStats computes platform-wide aggregate booking metrics matching the filter.
func (r *bookingRepository) GetAdminBookingStats(
	ctx context.Context,
	filter domain.AdminBookingFilter,
) (*domain.AdminBookingStats, error) {
	statsSelectClause := `
		COUNT(b.id) AS total_bookings,
		COUNT(CASE WHEN b.status = 'CONFIRMED' THEN 1 END) AS confirmed_bookings,
		COUNT(CASE WHEN b.status = 'CANCELLED' THEN 1 END) AS cancelled_bookings,
		COUNT(CASE WHEN b.status = 'PARTIALLY_CANCELLED' THEN 1 END) AS partially_cancelled_bookings,
		COALESCE(SUM(
			CASE 
				WHEN b.status IN ('CONFIRMED', 'PARTIALLY_CANCELLED') 
				THEN 
					CASE 
						WHEN EXISTS (SELECT 1 FROM tickets t WHERE t.booking_id = b.id)
						THEN (SELECT COUNT(t.id) FROM tickets t WHERE t.booking_id = b.id AND t.status IN ('VALID', 'USED'))
						ELSE (SELECT COALESCE(SUM(bi.quantity), 0) FROM booking_items bi WHERE bi.booking_id = b.id)
					END
				ELSE 0 
			END
		), 0) AS total_tickets_sold,
		COALESCE(SUM((
			SELECT p.amount
			FROM payments p
			WHERE p.booking_id = b.id
			  AND p.status = 'SUCCESS'
			  AND p.paid_at IS NOT NULL
			ORDER BY p.paid_at DESC NULLS LAST, p.created_at DESC, p.id DESC
			LIMIT 1
		)), 0) AS gross_captured_amount,
		COALESCE(SUM((
			SELECT COALESCE(SUM(r.amount), 0)
			FROM refunds r
			WHERE r.booking_id = b.id
			  AND r.status = 'SUCCESS'
		)), 0) AS total_refunded_amount
	`

	var row organizerBookingStatsRow
	err := r.buildAdminBookingBaseQuery(ctx, filter).
		Select(statsSelectClause).
		Scan(&row).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to query admin booking stats",
			slog.Any("error", err),
		)
		return nil, err
	}

	netCaptured := computeOrganizerNetCaptured(row.GrossCapturedAmount, row.TotalRefundedAmount)

	return &domain.AdminBookingStats{
		TotalBookings:              row.TotalBookings,
		ConfirmedBookings:          row.ConfirmedBookings,
		CancelledBookings:          row.CancelledBookings,
		PartiallyCancelledBookings: row.PartiallyCancelledBookings,
		TotalTicketsSold:           row.TotalTicketsSold,
		GrossCapturedAmount:        row.GrossCapturedAmount,
		TotalRefundedAmount:        row.TotalRefundedAmount,
		NetCapturedAmount:          netCaptured,
		Currency:                   "INR",
	}, nil
}

// FindAdminBookingDetail loads the full order breakdown for any booking across the platform.
func (r *bookingRepository) FindAdminBookingDetail(
	ctx context.Context,
	bookingID uuid.UUID,
) (*domain.AdminBookingDetail, error) {
	if bookingID == uuid.Nil {
		return nil, domain.ErrBookingNotFound
	}

	bookingDetailQuery := `
		SELECT 
			b.id,
			b.booking_number,
			b.created_at,
			b.updated_at,
			b.status AS booking_status,
			u.id AS user_id,
			COALESCE(u.full_name, '') AS user_name,
			COALESCE(u.email, '') AS user_email,
			e.id AS event_id,
			COALESCE(e.title, '') AS event_title,
			COALESCE(e.event_type, '') AS event_type,
			COALESCE(e.banner_url, '') AS banner_url,
			COALESCE(v.name, '') AS venue_name,
			o.id AS organizer_id,
			COALESCE(NULLIF(op.business_name, ''), NULLIF(ou.full_name, ''), 'Unknown Organizer') AS organizer_business_name,
			COALESCE(ou.email, '') AS organizer_email,
			b.subtotal,
			b.platform_fee,
			b.tax,
			b.discount,
			b.total,
			b.cancellation_allowed,
			b.cancellation_deadline_hours,
			b.cancellation_refund_policy,
			b.cancellation_refund_percentage
		FROM bookings b
		JOIN events e ON e.id = b.event_id
		LEFT JOIN users u ON u.id = b.user_id
		LEFT JOIN venues v ON v.id = e.venue_id
		LEFT JOIN organizers o ON o.id = e.organizer_id
		LEFT JOIN organizer_profiles op ON op.organizer_id = o.id
		LEFT JOIN users ou ON ou.id = o.user_id
		WHERE b.id = ?
	`

	var bRow adminBookingDetailRow
	if err := r.db.WithContext(ctx).Raw(bookingDetailQuery, bookingID).First(&bRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBookingNotFound
		}
		r.logger.ErrorContext(ctx, "failed to query admin booking detail",
			slog.String("booking_id", bookingID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	// 1. Items
	itemsQuery := `
		SELECT 
			bi.id,
			bi.ticket_type_id,
			COALESCE(tt.name, 'General Admission') AS ticket_type_name,
			bi.quantity,
			bi.unit_price,
			bi.total_price,
			bi.seat_id,
			COALESCE(
				CASE 
					WHEN ss.name IS NOT NULL AND ss.name != '' THEN ss.name || ' • Row ' || sr.row_name || ', Seat ' || CAST(s.seat_number AS text)
					ELSE 'Row ' || sr.row_name || ', Seat ' || CAST(s.seat_number AS text)
				END, ''
			) AS seat_label
		FROM booking_items bi
		LEFT JOIN ticket_types tt ON tt.id = bi.ticket_type_id
		LEFT JOIN seats s ON s.id = bi.seat_id
		LEFT JOIN seat_rows sr ON sr.id = s.row_id
		LEFT JOIN seat_sections ss ON ss.id = sr.section_id
		WHERE bi.booking_id = ?
		ORDER BY bi.created_at ASC, bi.id ASC
	`
	var itemRows []bookingItemDetailRow
	if err := r.db.WithContext(ctx).Raw(itemsQuery, bookingID).Scan(&itemRows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query booking items for admin detail",
			slog.String("booking_id", bookingID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	items := make([]domain.OrganizerBookingItemDetail, len(itemRows))
	for i, ir := range itemRows {
		items[i] = domain.OrganizerBookingItemDetail{
			ID:             ir.ID,
			TicketTypeID:   ir.TicketTypeID,
			TicketTypeName: ir.TicketTypeName,
			Quantity:       ir.Quantity,
			UnitPrice:      ir.UnitPrice,
			TotalPrice:     ir.TotalPrice,
			SeatID:         ir.SeatID,
			SeatLabel:      ir.SeatLabel,
		}
	}

	// 2. Tickets
	ticketsQuery := `
		SELECT 
			t.id,
			t.ticket_code,
			t.admission_number,
			COALESCE(tt.name, 'Admission') AS ticket_type_name,
			COALESCE(
				CASE 
					WHEN ss.name IS NOT NULL AND ss.name != '' THEN ss.name || ' • Row ' || sr.row_name || ', Seat ' || CAST(s.seat_number AS text)
					ELSE 'Row ' || sr.row_name || ', Seat ' || CAST(s.seat_number AS text)
				END, ''
			) AS seat_label,
			t.price,
			t.status,
			t.issued_at,
			t.used_at
		FROM tickets t
		LEFT JOIN ticket_types tt ON tt.id = t.ticket_type_id
		LEFT JOIN seats s ON s.id = t.seat_id
		LEFT JOIN seat_rows sr ON sr.id = s.row_id
		LEFT JOIN seat_sections ss ON ss.id = sr.section_id
		WHERE t.booking_id = ?
		ORDER BY t.admission_number ASC, t.id ASC
	`
	var ticketRows []ticketAdmissionDetailRow
	if err := r.db.WithContext(ctx).Raw(ticketsQuery, bookingID).Scan(&ticketRows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query tickets for admin detail",
			slog.String("booking_id", bookingID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	tickets := make([]domain.OrganizerTicketAdmissionDetail, len(ticketRows))
	for i, tr := range ticketRows {
		tickets[i] = domain.OrganizerTicketAdmissionDetail{
			ID:              tr.ID,
			TicketCode:      tr.TicketCode,
			AdmissionNumber: tr.AdmissionNumber,
			TicketTypeName:  tr.TicketTypeName,
			SeatLabel:       tr.SeatLabel,
			Price:           tr.Price,
			Status:          tr.Status,
			IssuedAt:        tr.IssuedAt,
			UsedAt:          tr.UsedAt,
		}
	}

	// 3. Payments
	paymentsQuery := `
		SELECT 
			p.id,
			p.transaction_id,
			p.provider,
			p.provider_order_id,
			p.payment_method,
			p.amount,
			p.status,
			p.paid_at,
			p.created_at
		FROM payments p
		WHERE p.booking_id = ?
		ORDER BY (p.status = 'SUCCESS') DESC, p.paid_at DESC NULLS LAST, p.created_at DESC, p.id DESC
	`
	var paymentRows []paymentSummaryRow
	if err := r.db.WithContext(ctx).Raw(paymentsQuery, bookingID.String()).Scan(&paymentRows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query payments for admin detail",
			slog.String("booking_id", bookingID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	payments := make([]domain.OrganizerPaymentSummary, len(paymentRows))
	for i, pr := range paymentRows {
		payments[i] = domain.OrganizerPaymentSummary{
			ID:              pr.ID,
			TransactionID:   pr.TransactionID,
			Provider:        pr.Provider,
			ProviderOrderID: pr.ProviderOrderID,
			PaymentMethod:   pr.PaymentMethod,
			Amount:          pr.Amount,
			Status:          pr.Status,
			PaidAt:          pr.PaidAt,
			CreatedAt:       pr.CreatedAt,
		}
	}

	successfulCapturedAmount := computeAuthoritativeBookingCapture(paymentRows)

	// 4. Refunds
	refundsQuery := `
		SELECT 
			r.id,
			r.payment_id,
			r.gateway_refund_id,
			r.amount,
			r.status,
			r.reason,
			r.created_at,
			r.processed_at
		FROM refunds r
		WHERE r.booking_id = ?
		ORDER BY r.created_at DESC, r.id DESC
	`
	var refundRows []refundSummaryRow
	if err := r.db.WithContext(ctx).Raw(refundsQuery, bookingID).Scan(&refundRows).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query refunds for admin detail",
			slog.String("booking_id", bookingID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	totalRefundedAmount := decimal.Zero
	refunds := make([]domain.OrganizerRefundSummary, len(refundRows))
	for i, rr := range refundRows {
		if rr.Status == "SUCCESS" {
			totalRefundedAmount = totalRefundedAmount.Add(rr.Amount)
		}
		refunds[i] = domain.OrganizerRefundSummary{
			ID:              rr.ID,
			PaymentID:       rr.PaymentID,
			GatewayRefundID: rr.GatewayRefundID,
			Amount:          rr.Amount,
			Status:          rr.Status,
			Reason:          rr.Reason,
			CreatedAt:       rr.CreatedAt,
			ProcessedAt:     rr.ProcessedAt,
		}
	}

	netCapturedAfterRefunds := computeOrganizerNetCaptured(successfulCapturedAmount, totalRefundedAmount)

	detail := &domain.AdminBookingDetail{
		ID:            bRow.ID,
		BookingNumber: bRow.BookingNumber,
		CreatedAt:     bRow.CreatedAt,
		UpdatedAt:     bRow.UpdatedAt,
		BookingStatus: bRow.BookingStatus,
		Customer: domain.OrganizerCustomerSummary{
			ID:       bRow.UserID,
			FullName: bRow.UserName,
			Email:    bRow.UserEmail,
		},
		Event: domain.OrganizerEventSummary{
			ID:        bRow.EventID,
			Title:     bRow.EventTitle,
			EventType: bRow.EventType,
			BannerURL: bRow.BannerURL,
			VenueName: bRow.VenueName,
		},
		Organizer: domain.AdminOrganizerBrief{
			ID:           bRow.OrganizerID,
			BusinessName: bRow.OrganizerBusiness,
			Email:        bRow.OrganizerEmail,
		},
		Financials: domain.OrganizerBookingFinancials{
			Subtotal:                 bRow.Subtotal,
			PlatformFee:              bRow.PlatformFee,
			Tax:                      bRow.Tax,
			Discount:                 bRow.Discount,
			OriginalBookingTotal:     bRow.Total,
			SuccessfulCapturedAmount: successfulCapturedAmount,
			TotalRefundedAmount:      totalRefundedAmount,
			NetCapturedAfterRefunds:  netCapturedAfterRefunds,
			Currency:                 "INR",
		},
		Items:    items,
		Tickets:  tickets,
		Payments: payments,
		Refunds:  refunds,
		CancellationPolicy: domain.OrganizerCancellationPolicy{
			CancellationAllowed:       bRow.CancellationAllowed,
			CancellationDeadlineHours: bRow.CancellationDeadlineHours,
			RefundPolicy:              bRow.RefundPolicy,
			RefundPercentage:          bRow.RefundPercentage,
		},
	}

	return detail, nil
}
