package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"

	"github.com/razorpay/razorpay-go"
	"github.com/shopspring/decimal"
)

type RazorpayProvider struct {
	client        *razorpay.Client
	secret        string
	webhookSecret string
	logger        *slog.Logger
}

func NewRazorpayProvider(
	keyID string,
	keySecret string,
	webhookSecret string,
	logger *slog.Logger,
) (*RazorpayProvider, error) {
	if strings.TrimSpace(keyID) == "" ||
		strings.TrimSpace(keySecret) == "" ||
		strings.TrimSpace(webhookSecret) == "" {
		return nil, errors.New("razorpay credentials are required")
	}

	if logger == nil {
		return nil, errors.New("logger is required")
	}

	client := razorpay.NewClient(keyID, keySecret)
	client.SetTimeout(30)

	return &RazorpayProvider{
		client:        client,
		secret:        keySecret,
		webhookSecret: webhookSecret,
		logger:        logger,
	}, nil
}

func (p *RazorpayProvider) Initiate(
	ctx context.Context,
	amount decimal.Decimal,
	currency string,
	idempotencyKey string,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if !amount.IsPositive() {
		return "", errors.New("payment amount must be positive")
	}

	if currency != "INR" {
		return "", errors.New("unsupported razorpay currency")
	}

	if strings.TrimSpace(idempotencyKey) == "" {
		return "", errors.New("payment reference is required")
	}

	// Razorpay expects paise, not rupees.
	paise := amount.Mul(decimal.NewFromInt(100))
	if !paise.Equal(paise.Truncate(0)) {
		return "", errors.New("payment amount cannot have fractional paise")
	}

	paiseInt := paise.BigInt()
	if !paiseInt.IsInt64() {
		return "", errors.New("payment amount exceeds supported range")
	}

	data := map[string]interface{}{
		"amount":          paiseInt.Int64(),
		"currency":        currency,
		"receipt":         idempotencyKey,
		"partial_payment": false,
	}

	response, err := p.client.Order.Create(data, nil)
	if err != nil {
		p.logger.ErrorContext(ctx, "razorpay order creation failed",
			slog.Any("error", err),
		)
		return "", errors.New("failed to create payment order")
	}

	orderID, ok := response["id"].(string)
	if !ok || strings.TrimSpace(orderID) == "" {
		p.logger.ErrorContext(ctx, "razorpay returned an invalid order response")
		return "", errors.New("invalid payment gateway response")
	}

	p.logger.InfoContext(ctx, "razorpay order created",
		slog.String("provider_order_id", orderID),
	)

	return orderID, nil
}

func (p *RazorpayProvider) Verify(
	ctx context.Context,
	providerOrderID string,
	paymentID string,
	signature string,
	expectedAmount decimal.Decimal,
	currency string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	if strings.TrimSpace(providerOrderID) == "" ||
		strings.TrimSpace(paymentID) == "" ||
		strings.TrimSpace(signature) == "" ||
		!expectedAmount.IsPositive() ||
		currency != "INR" {
		return false, errors.New("invalid payment verification details")
	}

	providedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false, nil
	}

	mac := hmac.New(sha256.New, []byte(p.secret))
	_, _ = mac.Write([]byte(providerOrderID + "|" + paymentID))

	if !hmac.Equal(mac.Sum(nil), providedSignature) {
		return false, nil
	}

	// A valid signature alone does not prove that payment was captured.
	response, err := p.client.Payment.Fetch(paymentID, nil, nil)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to fetch razorpay payment",
			slog.String("provider_order_id", providerOrderID),
			slog.Any("error", err),
		)
		return false, errors.New("failed to verify payment with gateway")
	}

	fetchedOrderID, _ := response["order_id"].(string)
	fetchedPaymentID, _ := response["id"].(string)
	fetchedStatus, _ := response["status"].(string)
	fetchedCurrency, _ := response["currency"].(string)
	fetchedAmount, ok := razorpayAmount(response["amount"])
	if !ok {
		return false, nil
	}

	expectedPaise := expectedAmount.Mul(decimal.NewFromInt(100))
	if !expectedPaise.Equal(expectedPaise.Truncate(0)) {
		return false, nil
	}

	expectedPaiseInt := expectedPaise.BigInt()
	if !expectedPaiseInt.IsInt64() {
		return false, nil
	}

	if fetchedPaymentID != paymentID ||
		fetchedOrderID != providerOrderID ||
		fetchedStatus != "captured" ||
		fetchedCurrency != currency ||
		fetchedAmount != expectedPaiseInt.Int64() {
		return false, nil
	}

	return true, nil
}

func (p *RazorpayProvider) VerifyWebhookSignature(
	body []byte,
	signature string,
) bool {
	if len(body) == 0 || strings.TrimSpace(signature) == "" {
		return false
	}

	providedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	_, _ = mac.Write(body)

	return hmac.Equal(mac.Sum(nil), providedSignature)
}

func razorpayAmount(value interface{}) (int64, bool) {
	switch amount := value.(type) {
	case int:
		return int64(amount), true
	case int64:
		return amount, true
	case float64:
		if amount < 0 || amount != float64(int64(amount)) {
			return 0, false
		}
		return int64(amount), true
	default:
		return 0, false
	}
}

var _ PaymentProvider = (*RazorpayProvider)(nil)
