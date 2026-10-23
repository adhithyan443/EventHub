package payment

import (
	"context"

	"github.com/shopspring/decimal"
)

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
}
