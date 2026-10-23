package domain

import (
	"context"
	"errors"
	"time"

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
}
