package payment

import (
	"context"

	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	OrderStatusUsable   OrderStatus = "USABLE"
	OrderStatusInFlight OrderStatus = "IN_FLIGHT"
	OrderStatusPaid     OrderStatus = "PAID"
	OrderStatusUnusable OrderStatus = "UNUSABLE"
)

type ProviderOrder struct {
	ID       string
	Status   OrderStatus
	Amount   decimal.Decimal
	Currency string
}

type PaymentProvider interface {
	Initiate(
		ctx context.Context,
		amount decimal.Decimal,
		currency string,
		idempotencyKey string,
	) (string, error)

	Verify(
		ctx context.Context,
		providerOrderID string,
		paymentID string,
		signature string,
		expectedAmount decimal.Decimal,
		currency string,
	) (bool, error)

	VerifyWebhookSignature(body []byte, signature string) bool

	FindOrderByReceipt(
		ctx context.Context,
		receipt string,
		expectedAmount decimal.Decimal,
		currency string,
	) (*ProviderOrder, error)

	CheckOrderStatus(
		ctx context.Context,
		providerOrderID string,
		expectedAmount decimal.Decimal,
		currency string,
	) (*ProviderOrder, error)
}
