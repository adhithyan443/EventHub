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

type AdminPaymentFilter struct {
	Page             int
	Limit            int
	Status           string
	AllocationStatus string
	From             *time.Time
	To               *time.Time
	Search           string
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
	ID                string                    `json:"id"`
	BookingID         string                    `json:"booking_id"`
	BookingNumber     string                    `json:"booking_number"`
	Customer          AdminPaymentCustomerInfo  `json:"customer"`
	Event             AdminPaymentEventInfo     `json:"event"`
	Organizer         AdminPaymentOrganizerInfo `json:"organizer"`
	Amount            decimal.Decimal           `json:"amount"`
	Currency          string                    `json:"currency"`
	Status            PaymentStatus             `json:"status"`
	HasGatewayCapture bool                      `json:"has_gateway_capture"`
	AllocationStatus  AllocationStatus          `json:"allocation_status"`
	Method            PaymentMethod             `json:"method"`
	Provider          PaymentProvider           `json:"provider"`
	TransactionID     *string                   `json:"transaction_id,omitempty"`
	ProviderOrderID   *string                   `json:"provider_order_id,omitempty"`
	PaidAt            *time.Time                `json:"paid_at,omitempty"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

type AdminPaymentListResult struct {
	Payments     []AdminPaymentListItem `json:"payments"`
	TotalRecords int64                  `json:"total_records"`
	Page         int                    `json:"page"`
	Limit        int                    `json:"limit"`
	TotalPages   int                    `json:"total_pages"`
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
}
