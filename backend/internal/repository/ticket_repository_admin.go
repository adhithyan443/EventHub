package repository

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type adminAttendeeRow struct {
	TicketID          uuid.UUID       `gorm:"column:ticket_id"`
	TicketCode        string          `gorm:"column:ticket_code"`
	BookingID         uuid.UUID       `gorm:"column:booking_id"`
	BookingNumber     string          `gorm:"column:booking_number"`
	UserID            uuid.UUID       `gorm:"column:user_id"`
	UserName          string          `gorm:"column:user_name"`
	UserEmail         string          `gorm:"column:user_email"`
	EventID           uuid.UUID       `gorm:"column:event_id"`
	EventTitle        string          `gorm:"column:event_title"`
	EventType         string          `gorm:"column:event_type"`
	BannerURL         string          `gorm:"column:banner_url"`
	VenueName         string          `gorm:"column:venue_name"`
	OrganizerID       uuid.UUID       `gorm:"column:organizer_id"`
	OrganizerBusiness string          `gorm:"column:organizer_business_name"`
	OrganizerEmail    string          `gorm:"column:organizer_email"`
	TicketTypeName    string          `gorm:"column:ticket_type_name"`
	AdmissionNumber   int             `gorm:"column:admission_number"`
	SeatLabel         string          `gorm:"column:seat_label"`
	Price             decimal.Decimal `gorm:"column:price"`
	Status            string          `gorm:"column:status"`
	IssuedAt          time.Time       `gorm:"column:issued_at"`
	UsedAt            *time.Time      `gorm:"column:used_at"`
}

func (r *ticketRepository) buildAdminAttendeeBaseQuery(
	ctx context.Context,
	filter domain.AdminAttendeeFilter,
) *gorm.DB {
	query := r.db.WithContext(ctx).Table("tickets AS t").
		Joins("JOIN events AS e ON e.id = t.event_id").
		Joins("JOIN bookings AS b ON b.id = t.booking_id").
		Joins("LEFT JOIN users AS u ON u.id = b.user_id").
		Joins("LEFT JOIN organizers AS o ON o.id = e.organizer_id").
		Joins("LEFT JOIN organizer_profiles AS op ON op.organizer_id = o.id").
		Joins("LEFT JOIN users AS ou ON ou.id = o.user_id")

	if filter.OrganizerID != nil && *filter.OrganizerID != uuid.Nil {
		query = query.Where("e.organizer_id = ?", *filter.OrganizerID)
	}

	if filter.EventID != nil && *filter.EventID != uuid.Nil {
		query = query.Where("t.event_id = ?", *filter.EventID)
	}

	if filter.CheckInStatus != "" && !strings.EqualFold(filter.CheckInStatus, "ALL") {
		switch strings.ToUpper(strings.TrimSpace(filter.CheckInStatus)) {
		case "USED", "CHECKED_IN":
			query = query.Where("t.status = 'USED'")
		case "VALID", "PENDING":
			query = query.Where("t.status = 'VALID'")
		case "CANCELLED":
			query = query.Where("t.status = 'CANCELLED'")
		case "REFUNDED":
			query = query.Where("t.status = 'REFUNDED'")
		case "CANCELLED_REFUNDED":
			query = query.Where("t.status IN ('CANCELLED', 'REFUNDED')")
		default:
			query = query.Where("t.status = ?", strings.ToUpper(strings.TrimSpace(filter.CheckInStatus)))
		}
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where(
			"(LOWER(t.ticket_code) LIKE ? OR LOWER(b.booking_number) LIKE ? OR LOWER(u.full_name) LIKE ? OR LOWER(u.email) LIKE ?)",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	return query
}

// FindAdminAttendees returns a paginated list of attendee passes platform-wide.
func (r *ticketRepository) FindAdminAttendees(
	ctx context.Context,
	filter domain.AdminAttendeeFilter,
) (*domain.AdminAttendeeListResult, error) {
	baseQuery := r.buildAdminAttendeeBaseQuery(ctx, filter)

	var totalRecords int64
	if err := baseQuery.Count(&totalRecords).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to count admin attendees",
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
		return &domain.AdminAttendeeListResult{
			Attendees:    make([]domain.AdminAttendeeListItem, 0),
			TotalRecords: totalRecords,
			Page:         page,
			Limit:        limit,
			TotalPages:   totalPages,
		}, nil
	}

	selectClause := `
		t.id AS ticket_id,
		t.ticket_code,
		b.id AS booking_id,
		b.booking_number,
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
		COALESCE(tt.name, 'Admission') AS ticket_type_name,
		t.admission_number,
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
	`

	var rows []adminAttendeeRow
	err := r.buildAdminAttendeeBaseQuery(ctx, filter).
		Joins("LEFT JOIN venues AS v ON v.id = e.venue_id").
		Joins("LEFT JOIN ticket_types AS tt ON tt.id = t.ticket_type_id").
		Joins("LEFT JOIN seats AS s ON s.id = t.seat_id").
		Joins("LEFT JOIN seat_rows AS sr ON sr.id = s.row_id").
		Joins("LEFT JOIN seat_sections AS ss ON ss.id = sr.section_id").
		Select(selectClause).
		Order("t.issued_at DESC, t.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to query admin attendees",
			slog.Any("error", err),
		)
		return nil, err
	}

	attendees := make([]domain.AdminAttendeeListItem, len(rows))
	for i, row := range rows {
		attendees[i] = domain.AdminAttendeeListItem{
			TicketID:      row.TicketID,
			TicketCode:    row.TicketCode,
			BookingID:     row.BookingID,
			BookingNumber: row.BookingNumber,
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
			TicketTypeName:  row.TicketTypeName,
			AdmissionNumber: row.AdmissionNumber,
			SeatLabel:       row.SeatLabel,
			Price:           row.Price,
			Status:          row.Status,
			IssuedAt:        row.IssuedAt,
			UsedAt:          row.UsedAt,
		}
	}

	return &domain.AdminAttendeeListResult{
		Attendees:    attendees,
		TotalRecords: totalRecords,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	}, nil
}

// GetAdminAttendeeStats aggregates platform-wide admission counts and calculates gate check-in rates.
func (r *ticketRepository) GetAdminAttendeeStats(
	ctx context.Context,
	filter domain.AdminAttendeeFilter,
) (*domain.AdminAttendeeStats, error) {
	query := r.db.WithContext(ctx).Table("tickets AS t").
		Joins("JOIN events AS e ON e.id = t.event_id").
		Joins("JOIN bookings AS b ON b.id = t.booking_id")

	if filter.OrganizerID != nil && *filter.OrganizerID != uuid.Nil {
		query = query.Where("e.organizer_id = ?", *filter.OrganizerID)
	}

	if filter.EventID != nil && *filter.EventID != uuid.Nil {
		query = query.Where("t.event_id = ?", *filter.EventID)
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Joins("LEFT JOIN users AS u ON u.id = b.user_id").
			Where(
				"(LOWER(t.ticket_code) LIKE ? OR LOWER(b.booking_number) LIKE ? OR LOWER(u.full_name) LIKE ? OR LOWER(u.email) LIKE ?)",
				searchPattern, searchPattern, searchPattern, searchPattern,
			)
	}

	statsSelectClause := `
		COUNT(t.id) AS total_issued_tickets,
		COUNT(CASE WHEN t.status IN ('VALID', 'USED') THEN 1 END) AS active_eligible_admissions,
		COUNT(CASE WHEN t.status = 'USED' THEN 1 END) AS checked_in_tickets,
		COUNT(CASE WHEN t.status = 'VALID' THEN 1 END) AS pending_check_in_tickets,
		COUNT(CASE WHEN t.status IN ('CANCELLED', 'REFUNDED') THEN 1 END) AS cancelled_tickets
	`

	var row organizerAttendeeStatsRow
	if err := query.Select(statsSelectClause).Scan(&row).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to query admin attendee stats",
			slog.Any("error", err),
		)
		return nil, err
	}

	checkInRate := decimal.Zero
	if row.ActiveEligibleAdmissions > 0 {
		checkedInDec := decimal.NewFromInt(row.CheckedInTickets)
		eligibleDec := decimal.NewFromInt(row.ActiveEligibleAdmissions)
		checkInRate = checkedInDec.Div(eligibleDec).Mul(decimal.NewFromInt(100)).Round(2)
	}

	return &domain.AdminAttendeeStats{
		TotalIssuedTickets:       row.TotalIssuedTickets,
		ActiveEligibleAdmissions: row.ActiveEligibleAdmissions,
		CheckedInTickets:         row.CheckedInTickets,
		PendingCheckInTickets:    row.PendingCheckInTickets,
		CancelledTickets:         row.CancelledTickets,
		CheckInRate:              checkInRate,
	}, nil
}
