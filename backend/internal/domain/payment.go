package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrPaymentNotFound            = errors.New("payment not found")
	ErrInvalidPayment             = errors.New("invalid payment")
	ErrPaymentAlreadyComplete     = errors.New("payment already completed")
	ErrPaymentAmountMismatch      = errors.New("payment amount mismatch")
	ErrUnsupportedPaymentProvider = errors.New("unsupported payment provider")
	ErrUnsupportedPaymentMethod   = errors.New("unsupported payment method")
	ErrCheckoutNotPayable         = errors.New("checkout is not payable")
	ErrPaymentInProgress          = errors.New("payment attempt is already in progress")
	ErrReservationExpired         = errors.New("reservation has expired")
	ErrPaymentRefundPending       = errors.New("previous payment requires refund resolution before retrying")
	ErrPaymentRefundRequired      = errors.New("Payment received, but booking could not be completed. Your refund is pending processing.")
	ErrPaymentProviderUnavailable = errors.New("payment provider is temporarily unavailable")
	ErrInvalidOrganizerID         = errors.New("invalid or empty organizer ID")
)

type PaymentStatus string

const (
	PaymentStatusPending        PaymentStatus = "PENDING"
	PaymentStatusSuccess        PaymentStatus = "SUCCESS"
	PaymentStatusFailed         PaymentStatus = "FAILED"
	PaymentStatusRefundRequired PaymentStatus = "REFUND_REQUIRED"
)

type PaymentMethod string

const (
	PaymentMethodUPI        PaymentMethod = "UPI"
	PaymentMethodCard       PaymentMethod = "CARD"
	PaymentMethodNetBanking PaymentMethod = "NET_BANKING"
	PaymentMethodWallet     PaymentMethod = "WALLET"
)

type PaymentProvider string

const (
	PaymentProviderRazorpay PaymentProvider = "RAZORPAY"
	PaymentProviderPayPal   PaymentProvider = "PAYPAL"
	PaymentProviderMock     PaymentProvider = "MOCK"
)

type AllocationStatus string

const (
	AllocationStatusAccounted     AllocationStatus = "ACCOUNTED"
	AllocationStatusUnallocated   AllocationStatus = "UNALLOCATED"
	AllocationStatusNotApplicable AllocationStatus = "NOT_APPLICABLE"
)

type PaymentRefundStatus = string

const (
	PaymentRefundStatusNone    PaymentRefundStatus = "NONE"
	PaymentRefundStatusPartial PaymentRefundStatus = "PARTIAL"
	PaymentRefundStatusFull    PaymentRefundStatus = "FULL"
	PaymentRefundStatusPending PaymentRefundStatus = "PENDING"
	PaymentRefundStatusFailed  PaymentRefundStatus = "FAILED"
)

type PaymentRefundItem struct {
	ID              uuid.UUID       `json:"id"`
	GatewayRefundID *string         `json:"gateway_refund_id,omitempty"`
	Amount          decimal.Decimal `json:"amount"`
	Status          string          `json:"status"`
	Reason          string          `json:"reason"`
	CreatedAt       time.Time       `json:"created_at"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty"`
}

type AdminPaymentFilter struct {
	Page             int
	Limit            int
	Status           string
	AllocationStatus string
	From             *time.Time
	To               *time.Time
	Search           string
	EventID          *uuid.UUID
}

type AdminEventPaymentSummary struct {
	EventID                            uuid.UUID       `json:"event_id"`
	EventTitle                         string          `json:"event_title"`
	OrganizerID                        uuid.UUID       `json:"organizer_id"`
	OrganizerBusinessName              string          `json:"organizer_business_name"`
	TotalPaymentsCount                 int64           `json:"total_payments_count"`
	SuccessfullyCompletedPaymentsCount int64           `json:"successfully_completed_payments_count"`
	GrossCapturedAmount                decimal.Decimal `json:"gross_captured_amount"`
	Currency                           string          `json:"currency"`
	PendingPaymentsCount               int64           `json:"pending_payments_count"`
	FailedPaymentsCount                int64           `json:"failed_payments_count"`
	RefundRequiredCount                int64           `json:"refund_required_count"`
	PaymentsWithSuccessfulRefundsCount int64           `json:"payments_with_successful_refunds_count"`
	UnallocatedCapturedCount           int64           `json:"unallocated_captured_count"`
}

type AdminOverallPaymentSummary struct {
	TotalEvents              int64           `json:"total_events"`
	GrossCapturedAmount      decimal.Decimal `json:"gross_captured_amount"`
	Currency                 string          `json:"currency"`
	RefundRequiredCount      int64           `json:"refund_required_count"`
	UnallocatedCapturedCount int64           `json:"unallocated_captured_count"`
}

type AdminEventPaymentSummaryResult struct {
	Summaries      []AdminEventPaymentSummary `json:"summaries"`
	OverallSummary AdminOverallPaymentSummary `json:"overall_summary"`
	TotalRecords   int64                      `json:"total_records"`
	Page           int                        `json:"page"`
	Limit          int                        `json:"limit"`
	TotalPages     int                        `json:"total_pages"`
}

type AdminPaymentCustomerInfo struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type AdminPaymentEventInfo struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

type AdminPaymentOrganizerInfo struct {
	ID           uuid.UUID `json:"id"`
	BusinessName string    `json:"business_name"`
}

type AdminPaymentListItem struct {
	ID                  string                    `json:"id"`
	BookingID           string                    `json:"booking_id"`
	BookingNumber       string                    `json:"booking_number"`
	Customer            AdminPaymentCustomerInfo  `json:"customer"`
	Event               AdminPaymentEventInfo     `json:"event"`
	Organizer           AdminPaymentOrganizerInfo `json:"organizer"`
	Amount              decimal.Decimal           `json:"amount"`
	Currency            string                    `json:"currency"`
	Status              PaymentStatus             `json:"status"`
	HasGatewayCapture   bool                      `json:"has_gateway_capture"`
	AllocationStatus    AllocationStatus          `json:"allocation_status"`
	RefundedAmount      decimal.Decimal           `json:"refunded_amount"`
	PendingRefundAmount decimal.Decimal           `json:"pending_refund_amount"`
	RefundStatus        PaymentRefundStatus       `json:"refund_status"`
	NetAmount           decimal.Decimal           `json:"net_amount"`
	RefundCount         int                       `json:"refund_count"`
	Refunds             []PaymentRefundItem       `json:"refunds"`
	Method              PaymentMethod             `json:"method"`
	Provider            PaymentProvider           `json:"provider"`
	TransactionID       *string                   `json:"transaction_id,omitempty"`
	ProviderOrderID     *string                   `json:"provider_order_id,omitempty"`
	PaidAt              *time.Time                `json:"paid_at,omitempty"`
	CreatedAt           time.Time                 `json:"created_at"`
	UpdatedAt           time.Time                 `json:"updated_at"`
}

type AdminPaymentListResult struct {
	Payments     []AdminPaymentListItem `json:"payments"`
	TotalRecords int64                  `json:"total_records"`
	Page         int                    `json:"page"`
	Limit        int                    `json:"limit"`
	TotalPages   int                    `json:"total_pages"`
}

type OrganizerPaymentFilter struct {
	OrganizerID      uuid.UUID
	Page             int
	Limit            int
	Status           string
	AllocationStatus string
	From             *time.Time
	To               *time.Time
	Search           string
	EventID          *uuid.UUID
}

type OrganizerPaymentCustomerInfo struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type OrganizerPaymentEventInfo struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

type OrganizerPaymentListItem struct {
	ID                  string                       `json:"id"`
	BookingID           string                       `json:"booking_id"`
	BookingNumber       string                       `json:"booking_number"`
	Customer            OrganizerPaymentCustomerInfo `json:"customer"`
	Event               OrganizerPaymentEventInfo    `json:"event"`
	TicketQuantity      int                          `json:"ticket_quantity"`
	Amount              decimal.Decimal              `json:"amount"`
	Currency            string                       `json:"currency"`
	Status              PaymentStatus                `json:"status"`
	HasGatewayCapture   bool                         `json:"has_gateway_capture"`
	AllocationStatus    AllocationStatus             `json:"allocation_status"`
	RefundedAmount      decimal.Decimal              `json:"refunded_amount"`
	PendingRefundAmount decimal.Decimal              `json:"pending_refund_amount"`
	RefundStatus        PaymentRefundStatus          `json:"refund_status"`
	NetAmount           decimal.Decimal              `json:"net_amount"`
	RefundCount         int                          `json:"refund_count"`
	Refunds             []PaymentRefundItem          `json:"refunds"`
	Method              PaymentMethod                `json:"method"`
	Provider            PaymentProvider              `json:"provider"`
	TransactionID       *string                      `json:"transaction_id,omitempty"`
	ProviderOrderID     *string                      `json:"provider_order_id,omitempty"`
	PaidAt              *time.Time                   `json:"paid_at,omitempty"`
	CreatedAt           time.Time                    `json:"created_at"`
	UpdatedAt           time.Time                    `json:"updated_at"`
}

type OrganizerPaymentListResult struct {
	Payments     []OrganizerPaymentListItem `json:"payments"`
	TotalRecords int64                      `json:"total_records"`
	Page         int                        `json:"page"`
	Limit        int                        `json:"limit"`
	TotalPages   int                        `json:"total_pages"`
}

type OrganizerEventPaymentSummary struct {
	EventID                            uuid.UUID       `json:"event_id"`
	EventTitle                         string          `json:"event_title"`
	TicketsSold                        int64           `json:"tickets_sold"`
	GrossCapturedAmount                decimal.Decimal `json:"gross_captured_amount"`
	PlatformFees                       decimal.Decimal `json:"platform_fees"`
	OrganizerEarnings                  decimal.Decimal `json:"organizer_earnings"`
	Currency                           string          `json:"currency"`
	TotalPaymentsCount                 int64           `json:"total_payments_count"`
	SuccessfullyCompletedPaymentsCount int64           `json:"successfully_completed_payments_count"`
	PendingPaymentsCount               int64           `json:"pending_payments_count"`
	FailedPaymentsCount                int64           `json:"failed_payments_count"`
	RefundRequiredCount                int64           `json:"refund_required_count"`
	PaymentsWithSuccessfulRefundsCount int64           `json:"payments_with_successful_refunds_count"`
	UnallocatedCapturedCount           int64           `json:"unallocated_captured_count"`
}

type OrganizerPaymentOverview struct {
	GrossCapturedAmount                decimal.Decimal `json:"gross_captured_amount"`
	TotalTicketsSold                   int64           `json:"total_tickets_sold"`
	TotalTicketSalesAmount             decimal.Decimal `json:"total_ticket_sales_amount"`
	TotalPlatformFees                  decimal.Decimal `json:"total_platform_fees"`
	OrganizerEarnings                  decimal.Decimal `json:"organizer_earnings"`
	Currency                           string          `json:"currency"`
	// TotalEvents represents the count of distinct events belonging to the organizer
	// that have payment transactions matching the applied filters.
	TotalEvents                        int64           `json:"total_events"`
	TotalPaymentsCount                 int64           `json:"total_payments_count"`
	SuccessfullyCompletedPaymentsCount int64           `json:"successfully_completed_payments_count"`
	RefundRequiredCount                int64           `json:"refund_required_count"`
	UnallocatedCapturedCount           int64           `json:"unallocated_captured_count"`
}

type OrganizerEventPaymentSummaryResult struct {
	Summaries    []OrganizerEventPaymentSummary `json:"summaries"`
	Overview     OrganizerPaymentOverview       `json:"overview"`
	TotalRecords int64                          `json:"total_records"`
	Page         int                            `json:"page"`
	Limit        int                            `json:"limit"`
	TotalPages   int                            `json:"total_pages"`
}

type Payment struct {
	ID              string
	BookingID       string
	TransactionID   *string
	Provider        PaymentProvider
	ProviderOrderID *string
	Method          PaymentMethod
	Amount          decimal.Decimal
	Status          PaymentStatus
	PaidAt          *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	FindByID(ctx context.Context, id string) (*Payment, error)
	FindByIDForUpdate(ctx context.Context, id string) (*Payment, error)
	FindByBookingID(ctx context.Context, bookingID string) ([]Payment, error)
	Update(ctx context.Context, payment *Payment) error
	FindByProviderOrderID(ctx context.Context, providerOrderID string) (*Payment, error)
	MarkSuccessful(
		ctx context.Context,
		paymentID string,
		transactionID string,
		paidAt time.Time,
	) error
	MarkFailed(
		ctx context.Context,
		paymentID string,
	) error
	MarkRefundRequired(
		ctx context.Context,
		paymentID string,
		transactionID string,
		paidAt time.Time,
	) error

	FindByProviderOrderIDForUpdate(
		ctx context.Context,
		providerOrderID string,
	) (*Payment, error)

	UpdateProviderOrderID(
		ctx context.Context,
		paymentID string,
		providerOrderID string,
	) error

	FindAdminPayments(
		ctx context.Context,
		filter AdminPaymentFilter,
	) (*AdminPaymentListResult, error)

	GetEventPaymentSummaries(
		ctx context.Context,
		filter AdminPaymentFilter,
	) (*AdminEventPaymentSummaryResult, error)

	FindOrganizerPayments(
		ctx context.Context,
		filter OrganizerPaymentFilter,
	) (*OrganizerPaymentListResult, error)

	GetOrganizerEventPaymentSummaries(
		ctx context.Context,
		filter OrganizerPaymentFilter,
	) (*OrganizerEventPaymentSummaryResult, error)

	GetOrganizerPaymentOverview(
		ctx context.Context,
		filter OrganizerPaymentFilter,
	) (*OrganizerPaymentOverview, error)
}
