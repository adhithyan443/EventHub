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

type organizerBookingRow struct {
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
	TicketCount          int             `gorm:"column:ticket_count"`
	OriginalBookingTotal decimal.Decimal `gorm:"column:original_booking_total"`
	BookingStatus        string          `gorm:"column:booking_status"`
	PaymentStatus        string          `gorm:"column:payment_status"`
	PaymentMethod        string          `gorm:"column:payment_method"`
	RefundStatus         string          `gorm:"column:refund_status"`
}

type organizerBookingStatsRow struct {
	TotalBookings              int64           `gorm:"column:total_bookings"`
	ConfirmedBookings          int64           `gorm:"column:confirmed_bookings"`
	CancelledBookings          int64           `gorm:"column:cancelled_bookings"`
	PartiallyCancelledBookings int64           `gorm:"column:partially_cancelled_bookings"`
	TotalTicketsSold           int64           `gorm:"column:total_tickets_sold"`
	GrossCapturedAmount        decimal.Decimal `gorm:"column:gross_captured_amount"`
	TotalRefundedAmount        decimal.Decimal `gorm:"column:total_refunded_amount"`
}

type organizerBookingDetailRow struct {
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

type bookingItemDetailRow struct {
	ID             uuid.UUID       `gorm:"column:id"`
	TicketTypeID   *uuid.UUID      `gorm:"column:ticket_type_id"`
	TicketTypeName string          `gorm:"column:ticket_type_name"`
	Quantity       int             `gorm:"column:quantity"`
	UnitPrice      decimal.Decimal `gorm:"column:unit_price"`
	TotalPrice     decimal.Decimal `gorm:"column:total_price"`
	SeatID         *uuid.UUID      `gorm:"column:seat_id"`
	SeatLabel      string          `gorm:"column:seat_label"`
}

type ticketAdmissionDetailRow struct {
	ID              uuid.UUID       `gorm:"column:id"`
	TicketCode      string          `gorm:"column:ticket_code"`
	AdmissionNumber int             `gorm:"column:admission_number"`
	TicketTypeName  string          `gorm:"column:ticket_type_name"`
	SeatLabel       string          `gorm:"column:seat_label"`
	Price           decimal.Decimal `gorm:"column:price"`
	Status          string          `gorm:"column:status"`
	IssuedAt        time.Time       `gorm:"column:issued_at"`
	UsedAt          *time.Time      `gorm:"column:used_at"`
}

type paymentSummaryRow struct {
	ID              string          `gorm:"column:id"`
	TransactionID   *string         `gorm:"column:transaction_id"`
	Provider        string          `gorm:"column:provider"`
	ProviderOrderID *string         `gorm:"column:provider_order_id"`
	PaymentMethod   string          `gorm:"column:payment_method"`
	Amount          decimal.Decimal `gorm:"column:amount"`
	Status          string          `gorm:"column:status"`
	PaidAt          *time.Time      `gorm:"column:paid_at"`
	CreatedAt       time.Time       `gorm:"column:created_at"`
}

type refundSummaryRow struct {
	ID              uuid.UUID       `gorm:"column:id"`
	PaymentID       uuid.UUID       `gorm:"column:payment_id"`
	GatewayRefundID *string         `gorm:"column:gateway_refund_id"`
	Amount          decimal.Decimal `gorm:"column:amount"`
	Status          string          `gorm:"column:status"`
	Reason          string          `gorm:"column:reason"`
	CreatedAt       time.Time       `gorm:"column:created_at"`
	ProcessedAt     *time.Time      `gorm:"column:processed_at"`
}

// computeOrganizerNetCaptured calculates net captured revenue without clamping.
func computeOrganizerNetCaptured(grossCaptured, totalRefunded decimal.Decimal) decimal.Decimal {
	return grossCaptured.Sub(totalRefunded)
}

// computeActiveTickets determines active sold tickets, ensuring that when ticket records exist,
// only VALID or USED admissions are counted, and cancelled/refunded tickets never fall back to original quantities.
func computeActiveTickets(ticketRecordsExist bool, activeValidOrUsedTickets int64, fallbackQuantity int64) int64 {
	if ticketRecordsExist {
		return activeValidOrUsedTickets
	}
	return fallbackQuantity
}

// resolveBookingRefundStatus determines the authoritative refund status for a booking.
// Semantics:
// - "" (empty string / NONE): No refund records exist.
// - "FULL": All captured amount has been successfully refunded.
// - "PENDING": An in-flight refund is awaiting gateway or administrative completion (never hidden by past partial refunds).
// - "PARTIAL": A partial refund succeeded and no subsequent refund is in-flight.
// - "FAILED": Refund attempts were made and all failed, with none succeeding or pending.
func resolveBookingRefundStatus(grossCaptured, totalRefunded decimal.Decimal, hasPendingRefund, hasFailedRefund bool, refundCount int) string {
	if refundCount == 0 {
		return ""
	}
	if grossCaptured.IsPositive() && totalRefunded.GreaterThanOrEqual(grossCaptured) && totalRefunded.IsPositive() {
		return "FULL"
	}
	if hasPendingRefund {
		return "PENDING"
	}
	if totalRefunded.IsPositive() {
		return "PARTIAL"
	}
	if hasFailedRefund {
		return "FAILED"
	}
	return ""
}

// matchesBookingRefundFilter evaluates whether a booking's financial and refund state
// satisfies a refund-related filter status ("REFUNDED", "FULL", "PARTIAL", "PENDING_REFUND").
func matchesBookingRefundFilter(
	filterStatus string,
	grossCaptured, totalSuccessfulRefund decimal.Decimal,
	hasSuccessfulRefund, hasPendingRefund, hasFailedRefund bool,
) bool {
	switch strings.ToUpper(strings.TrimSpace(filterStatus)) {
	case "REFUNDED":
		return hasSuccessfulRefund
	case "FULL":
		return grossCaptured.IsPositive() &&
			totalSuccessfulRefund.GreaterThanOrEqual(grossCaptured) &&
			totalSuccessfulRefund.IsPositive()
	case "PARTIAL":
		return totalSuccessfulRefund.IsPositive() &&
			totalSuccessfulRefund.LessThan(grossCaptured)
	case "PENDING_REFUND":
		return hasPendingRefund
	default:
		return false
	}
}

// computeAuthoritativeBookingCapture determines the legitimate successful capture amount for a booking,
// prioritizing the primary successful captured payment and avoiding double-counting duplicate payment attempts.
func computeAuthoritativeBookingCapture(paymentRows []paymentSummaryRow) decimal.Decimal {
	for i := range paymentRows {
		if paymentRows[i].Status == "SUCCESS" && paymentRows[i].PaidAt != nil {
			return paymentRows[i].Amount
		}
	}
	return decimal.Zero
}

func (r *bookingRepository) buildOrganizerBookingBaseQuery(
	ctx context.Context,
	filter domain.OrganizerBookingFilter,
) *gorm.DB {
	query := r.db.WithContext(ctx).Table("bookings AS b").
		Joins("JOIN events AS e ON e.id = b.event_id").
		Joins("LEFT JOIN users AS u ON u.id = b.user_id").
		Where("e.organizer_id = ?", filter.OrganizerID)

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

// FindOrganizerBookings returns a paginated list of bookings owned by the organizer.
func (r *bookingRepository) FindOrganizerBookings(
	ctx context.Context,
	filter domain.OrganizerBookingFilter,
) (*domain.OrganizerBookingListResult, error) {
	if filter.OrganizerID == uuid.Nil {
		return nil, domain.ErrOrganizerNotFound
	}

	baseQuery := r.buildOrganizerBookingBaseQuery(ctx, filter)

	var totalRecords int64
	if err := baseQuery.Count(&totalRecords).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to count organizer bookings",
			slog.String("organizer_id", filter.OrganizerID.String()),
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
		return &domain.OrganizerBookingListResult{
			Bookings:     make([]domain.OrganizerBookingListItem, 0),
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

	var rows []organizerBookingRow
	err := r.buildOrganizerBookingBaseQuery(ctx, filter).
		Joins("LEFT JOIN venues AS v ON v.id = e.venue_id").
		Select(selectClause).
		Order("b.created_at DESC, b.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to query organizer bookings",
			slog.String("organizer_id", filter.OrganizerID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	bookings := make([]domain.OrganizerBookingListItem, len(rows))
	for i, row := range rows {
		bookings[i] = domain.OrganizerBookingListItem{
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
			TicketCount:          row.TicketCount,
			OriginalBookingTotal: row.OriginalBookingTotal,
			Currency:             "INR",
			BookingStatus:        row.BookingStatus,
			PaymentStatus:        row.PaymentStatus,
			PaymentMethod:        row.PaymentMethod,
			RefundStatus:         row.RefundStatus,
		}
	}

	return &domain.OrganizerBookingListResult{
		Bookings:     bookings,
		TotalRecords: totalRecords,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	}, nil
}

// GetOrganizerBookingStats computes summary totals and captured financial figures for the filtered scope.
func (r *bookingRepository) GetOrganizerBookingStats(
	ctx context.Context,
	filter domain.OrganizerBookingFilter,
) (*domain.OrganizerBookingStats, error) {
	if filter.OrganizerID == uuid.Nil {
		return nil, domain.ErrOrganizerNotFound
	}

	// Authoritative gross captured amount uses the primary successful payment capture per booking,
	// strictly preventing duplicate payment attempts from double-counting captured revenue.
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
	err := r.buildOrganizerBookingBaseQuery(ctx, filter).
		Select(statsSelectClause).
		Scan(&row).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to query organizer booking stats",
			slog.String("organizer_id", filter.OrganizerID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	netCaptured := computeOrganizerNetCaptured(row.GrossCapturedAmount, row.TotalRefundedAmount)

	return &domain.OrganizerBookingStats{
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

// FindOrganizerBookingDetail loads the complete order breakdown with ownership verification.
func (r *bookingRepository) FindOrganizerBookingDetail(
	ctx context.Context,
	organizerID uuid.UUID,
	bookingID uuid.UUID,
) (*domain.OrganizerBookingDetail, error) {
	if organizerID == uuid.Nil {
		return nil, domain.ErrOrganizerNotFound
	}
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
		WHERE b.id = ? AND e.organizer_id = ?
	`

	var bRow organizerBookingDetailRow
	if err := r.db.WithContext(ctx).Raw(bookingDetailQuery, bookingID, organizerID).First(&bRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBookingNotFound
		}
		r.logger.ErrorContext(ctx, "failed to query organizer booking detail",
			slog.String("booking_id", bookingID.String()),
			slog.String("organizer_id", organizerID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	// 1. Load Items
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
		r.logger.ErrorContext(ctx, "failed to query booking items for detail",
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

	// 2. Load Tickets
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
		r.logger.ErrorContext(ctx, "failed to query tickets for detail",
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

	// 3. Load Payments
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
		r.logger.ErrorContext(ctx, "failed to query payments for detail",
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

	// Authoritative captured payment for this booking (prevents duplicate capture double-counting)
	successfulCapturedAmount := computeAuthoritativeBookingCapture(paymentRows)

	// 4. Load Refunds
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
		r.logger.ErrorContext(ctx, "failed to query refunds for detail",
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

	detail := &domain.OrganizerBookingDetail{
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
