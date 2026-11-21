package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidDateRange = errors.New("from date cannot be after to date")
)

// AdminPlatformOverview represents the platform-wide volume metrics.
type AdminPlatformOverview struct {
	TotalUsers      int64 `json:"total_users"`
	TotalOrganizers int64 `json:"total_organizers"`
	TotalEvents     int64 `json:"total_events"`
	ActiveEvents    int64 `json:"active_events"`
	TotalBookings   int64 `json:"total_bookings"`
	TicketsSold     int64 `json:"tickets_sold"`
}

// AdminFinancialOverview represents the consolidated financial position derived from the general ledger and payment records.
type AdminFinancialOverview struct {
	GrossCapturedAmount       decimal.Decimal `json:"gross_captured_amount"`
	TotalRefunds              decimal.Decimal `json:"total_refunds"`
	NetCapturedAmount         decimal.Decimal `json:"net_captured_amount"`
	PlatformFeeRevenue        decimal.Decimal `json:"platform_fee_revenue"`
	OrganizerPayableLiability decimal.Decimal `json:"organizer_payable_liability"`
	CompletedSettlements      decimal.Decimal `json:"completed_settlements"`
	PendingSettlements        decimal.Decimal `json:"pending_settlements"`
	TaxLiability              decimal.Decimal `json:"tax_liability"`
	Currency                  string          `json:"currency"`
}

// AdminRevenueTrendPoint represents time-bucketed revenue performance.
type AdminRevenueTrendPoint struct {
	Period       string          `json:"period"`
	GrossVolume  decimal.Decimal `json:"gross_volume"`
	PlatformFees decimal.Decimal `json:"platform_fees"`
}

// AdminBookingTrendPoint represents time-bucketed booking volume.
type AdminBookingTrendPoint struct {
	Period       string `json:"period"`
	BookingCount int64  `json:"booking_count"`
	TicketsSold  int64  `json:"tickets_sold"`
}

// AdminRefundTrendPoint represents time-bucketed refund volume.
type AdminRefundTrendPoint struct {
	Period       string          `json:"period"`
	RefundAmount decimal.Decimal `json:"refund_amount"`
	RefundCount  int64           `json:"refund_count"`
}

// AdminDashboardTrends groups the time-bucketed performance metrics.
type AdminDashboardTrends struct {
	RevenueTrends []AdminRevenueTrendPoint `json:"revenue_trends"`
	BookingTrends []AdminBookingTrendPoint `json:"booking_trends"`
	RefundTrends  []AdminRefundTrendPoint  `json:"refund_trends"`
}

// AdminPaymentStatusBreakdownItem represents payment count and amount by payment status.
type AdminPaymentStatusBreakdownItem struct {
	Status string          `json:"status"`
	Count  int64           `json:"count"`
	Amount decimal.Decimal `json:"amount"`
}

// AdminRecentBookingItem represents the safe summary of a recent booking for the admin dashboard.
type AdminRecentBookingItem struct {
	ID            uuid.UUID       `json:"id"`
	BookingNumber string          `json:"booking_number"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	CustomerName  string          `json:"customer_name"`
	CustomerEmail string          `json:"customer_email"`
	EventID       uuid.UUID       `json:"event_id"`
	EventTitle    string          `json:"event_title"`
	Amount        decimal.Decimal `json:"amount"`
	Status        string          `json:"status"`
	PaymentStatus string          `json:"payment_status"`
	CreatedAt     time.Time       `json:"created_at"`
}

// AdminDashboardData combines all admin dashboard sections into a single response payload.
type AdminDashboardData struct {
	PlatformOverview       AdminPlatformOverview             `json:"platform_overview"`
	FinancialOverview      AdminFinancialOverview            `json:"financial_overview"`
	Trends                 AdminDashboardTrends              `json:"trends"`
	PaymentStatusBreakdown []AdminPaymentStatusBreakdownItem `json:"payment_status_breakdown"`
	RecentBookings         []AdminRecentBookingItem          `json:"recent_bookings"`
}

// AdminDashboardFilter holds optional date boundaries for dashboard queries.
type AdminDashboardFilter struct {
	From *time.Time
	To   *time.Time
}

// AdminDashboardRepository provides database queries for the Admin Dashboard.
type AdminDashboardRepository interface {
	GetPlatformOverview(ctx context.Context) (*AdminPlatformOverview, error)
	GetFinancialOverview(ctx context.Context, filter AdminDashboardFilter) (*AdminFinancialOverview, error)
	GetTrends(ctx context.Context, filter AdminDashboardFilter) (*AdminDashboardTrends, error)
	GetPaymentStatusBreakdown(ctx context.Context, filter AdminDashboardFilter) ([]AdminPaymentStatusBreakdownItem, error)
	GetRecentBookings(ctx context.Context, limit int) ([]AdminRecentBookingItem, error)
}

// AdminDashboardUsecase orchestrates retrieving the complete dashboard view.
type AdminDashboardUsecase interface {
	GetDashboardData(ctx context.Context, filter AdminDashboardFilter) (*AdminDashboardData, error)
}
