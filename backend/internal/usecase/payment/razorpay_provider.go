package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
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
		// If Razorpay rejected duplicate receipt, attempt immediate recovery by receipt.
		if isDuplicateReceiptError(err) {
			existingOrder, findErr := p.FindOrderByReceipt(ctx, idempotencyKey, amount, currency)
			if findErr == nil && existingOrder != nil && existingOrder.ID != "" {
				p.logger.InfoContext(ctx, "recovered existing razorpay order after duplicate receipt conflict",
					slog.String("provider_order_id", existingOrder.ID),
					slog.String("receipt", idempotencyKey),
				)
				return existingOrder.ID, nil
			}
		}

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

func (p *RazorpayProvider) FindOrderByReceipt(
	ctx context.Context,
	receipt string,
	expectedAmount decimal.Decimal,
	currency string,
) (*ProviderOrder, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	trimmedReceipt := strings.TrimSpace(receipt)
	if trimmedReceipt == "" {
		return nil, errors.New("receipt is required")
	}
	if len(trimmedReceipt) > 40 {
		return nil, errors.New("receipt exceeds maximum supported length")
	}
	if !expectedAmount.IsPositive() {
		return nil, errors.New("expected amount must be positive")
	}
	if currency != "INR" {
		return nil, errors.New("unsupported currency")
	}

	params := map[string]interface{}{
		"receipt": trimmedReceipt,
	}

	response, err := p.client.Order.All(params, nil)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to query razorpay orders by receipt",
			slog.String("receipt", trimmedReceipt),
			slog.Any("error", err),
		)
		return nil, errors.New("failed to query orders from gateway")
	}

	items, ok := response["items"].([]interface{})
	if !ok || len(items) == 0 {
		return nil, nil // Zero orders found (absent)
	}

	if len(items) > 1 {
		p.logger.ErrorContext(ctx, "ambiguous gateway state: multiple orders found for receipt",
			slog.String("receipt", trimmedReceipt),
			slog.Int("order_count", len(items)),
		)
		return nil, errors.New("ambiguous gateway response: multiple orders returned for receipt")
	}

	orderMap, ok := items[0].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid order item response from gateway")
	}

	orderID, _ := orderMap["id"].(string)
	if strings.TrimSpace(orderID) == "" {
		return nil, errors.New("gateway returned order with missing ID")
	}

	orderReceipt, _ := orderMap["receipt"].(string)
	if orderReceipt != trimmedReceipt {
		p.logger.ErrorContext(ctx, "receipt mismatch on gateway order",
			slog.String("expected_receipt", trimmedReceipt),
			slog.String("actual_receipt", orderReceipt),
		)
		return nil, errors.New("gateway returned order with mismatched receipt")
	}

	orderCurrency, _ := orderMap["currency"].(string)
	if orderCurrency != currency {
		p.logger.ErrorContext(ctx, "currency mismatch on gateway order",
			slog.String("expected_currency", currency),
			slog.String("actual_currency", orderCurrency),
		)
		return nil, errors.New("gateway returned order with mismatched currency")
	}

	amountPaise, ok := razorpayAmount(orderMap["amount"])
	if !ok {
		return nil, errors.New("invalid order amount from gateway")
	}

	actualAmount := decimal.NewFromInt(amountPaise).Div(decimal.NewFromInt(100))
	if actualAmount.Cmp(expectedAmount) != 0 {
		p.logger.ErrorContext(ctx, "amount mismatch on gateway order",
			slog.String("expected_amount", expectedAmount.String()),
			slog.String("actual_amount", actualAmount.String()),
		)
		return nil, errors.New("gateway returned order with mismatched amount")
	}

	statusStr, _ := orderMap["status"].(string)
	orderStatus := OrderStatusUsable
	if statusStr == "paid" {
		orderStatus = OrderStatusPaid
	} else if statusStr == "cancelled" || statusStr == "expired" {
		orderStatus = OrderStatusUnusable
	}

	return &ProviderOrder{
		ID:       orderID,
		Status:   orderStatus,
		Amount:   actualAmount,
		Currency: orderCurrency,
	}, nil
}

func (p *RazorpayProvider) CheckOrderStatus(
	ctx context.Context,
	providerOrderID string,
	expectedAmount decimal.Decimal,
	currency string,
) (*ProviderOrder, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(providerOrderID) == "" {
		return nil, errors.New("provider order ID is required")
	}

	response, err := p.client.Order.Fetch(providerOrderID, nil, nil)
	if err != nil {
		if isOrderNotFoundError(err) {
			p.logger.WarnContext(ctx, "razorpay order not found on gateway",
				slog.String("provider_order_id", providerOrderID),
			)
			return &ProviderOrder{
				ID:     providerOrderID,
				Status: OrderStatusUnusable,
			}, nil
		}

		p.logger.ErrorContext(ctx, "failed to fetch razorpay order",
			slog.String("provider_order_id", providerOrderID),
			slog.Any("error", err),
		)
		return nil, errors.New("failed to fetch order from gateway")
	}

	orderStatus, _ := response["status"].(string)
	orderCurrency, _ := response["currency"].(string)
	amountPaise, ok := razorpayAmount(response["amount"])
	if !ok {
		return nil, errors.New("invalid order amount from gateway")
	}

	actualAmount := decimal.NewFromInt(amountPaise).Div(decimal.NewFromInt(100))

	// Currency or amount mismatch means this order cannot be reused
	if orderCurrency != currency || actualAmount.Cmp(expectedAmount) != 0 {
		return &ProviderOrder{
			ID:       providerOrderID,
			Status:   OrderStatusUnusable,
			Amount:   actualAmount,
			Currency: orderCurrency,
		}, nil
	}

	if orderStatus == "paid" {
		return &ProviderOrder{
			ID:       providerOrderID,
			Status:   OrderStatusPaid,
			Amount:   actualAmount,
			Currency: orderCurrency,
		}, nil
	}

	if orderStatus == "cancelled" || orderStatus == "expired" {
		return &ProviderOrder{
			ID:       providerOrderID,
			Status:   OrderStatusUnusable,
			Amount:   actualAmount,
			Currency: orderCurrency,
		}, nil
	}

	// For attempted orders: inspect payment attempts and PROPAGATE lookup failures
	if orderStatus == "attempted" {
		paymentsResp, paymentsErr := p.client.Order.Payments(providerOrderID, nil, nil)
		if paymentsErr != nil {
			p.logger.ErrorContext(ctx, "failed to fetch payments for attempted order",
				slog.String("provider_order_id", providerOrderID),
				slog.Any("error", paymentsErr),
			)
			return nil, fmt.Errorf("failed to verify payments for attempted order: %w", paymentsErr)
		}

		if paymentsResp == nil {
			return nil, errors.New("nil payments response received from gateway")
		}

		items, ok := paymentsResp["items"].([]interface{})
		if !ok {
			return nil, errors.New("invalid payments collection response from gateway")
		}

		hasCaptured := false
		hasInFlight := false
		allFailed := true

		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				return nil, errors.New("invalid payment item response from gateway")
			}
			paymentStatus, _ := itemMap["status"].(string)
			switch paymentStatus {
			case "captured":
				hasCaptured = true
				allFailed = false
			case "authorized", "created":
				hasInFlight = true
				allFailed = false
			case "failed":
				// definitively failed
			default:
				p.logger.WarnContext(ctx, "unrecognized payment attempt status on order",
					slog.String("provider_order_id", providerOrderID),
					slog.String("payment_status", paymentStatus),
				)
				allFailed = false
				hasInFlight = true
			}
		}

		if hasCaptured {
			return &ProviderOrder{
				ID:       providerOrderID,
				Status:   OrderStatusPaid,
				Amount:   actualAmount,
				Currency: orderCurrency,
			}, nil
		}

		if hasInFlight {
			return &ProviderOrder{
				ID:       providerOrderID,
				Status:   OrderStatusInFlight,
				Amount:   actualAmount,
				Currency: orderCurrency,
			}, nil
		}

		if !allFailed && len(items) > 0 {
			return nil, errors.New("ambiguous payment attempts state for order")
		}
	} else if orderStatus != "created" {
		// Never classify an unknown order status as usable
		p.logger.WarnContext(ctx, "unrecognized gateway order status",
			slog.String("provider_order_id", providerOrderID),
			slog.String("status", orderStatus),
		)
		return nil, fmt.Errorf("unrecognized gateway order status: %s", orderStatus)
	}

	return &ProviderOrder{
		ID:       providerOrderID,
		Status:   OrderStatusUsable,
		Amount:   actualAmount,
		Currency: orderCurrency,
	}, nil
}
func isOrderNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "does not exist") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "404")
}

func isDuplicateReceiptError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "receipt") &&
		(strings.Contains(errStr, "already") || strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "exist"))
}

func (p *RazorpayProvider) Refund(
	ctx context.Context,
	gatewayPaymentID string,
	amount decimal.Decimal,
	currency string,
	idempotencyKey string,
) (*domain.RefundResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	trimmedPaymentID := strings.TrimSpace(gatewayPaymentID)
	if trimmedPaymentID == "" {
		return nil, errors.New("gateway payment ID is required")
	}

	if !amount.IsPositive() {
		return nil, errors.New("refund amount must be positive")
	}

	if currency != "INR" {
		return nil, errors.New("unsupported razorpay currency")
	}

	trimmedIdempotencyKey := strings.TrimSpace(idempotencyKey)
	if trimmedIdempotencyKey == "" {
		return nil, errors.New("idempotency key is required")
	}

	paise := amount.Mul(decimal.NewFromInt(100))
	if !paise.Equal(paise.Truncate(0)) {
		return nil, errors.New("refund amount cannot have fractional paise")
	}

	paiseInt := paise.BigInt()
	if !paiseInt.IsInt64() {
		return nil, errors.New("refund amount exceeds supported range")
	}

	data := map[string]interface{}{
		"amount":  paiseInt.Int64(),
		"receipt": trimmedIdempotencyKey,
		"notes": map[string]interface{}{
			"idempotency_key": trimmedIdempotencyKey,
		},
	}

	extraHeaders := map[string]string{
		"X-Razorpay-Idempotency-Key": trimmedIdempotencyKey,
	}

	response, err := p.client.Payment.Refund(trimmedPaymentID, int(paiseInt.Int64()), data, extraHeaders)
	if err != nil {
		p.logger.ErrorContext(ctx, "razorpay_refund_failed",
			slog.String("gateway_payment_id", trimmedPaymentID),
			slog.String("amount", amount.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("gateway refund failed: %w", err)
	}

	refundID, _ := response["id"].(string)
	statusStr, _ := response["status"].(string)

	var refundStatus string
	switch strings.ToLower(statusStr) {
	case "processed", "success":
		refundStatus = domain.RefundStatusSuccess
	case "pending":
		refundStatus = domain.RefundStatusPending
	case "failed":
		refundStatus = domain.RefundStatusFailed
	default:
		if refundID != "" {
			refundStatus = domain.RefundStatusSuccess
		} else {
			refundStatus = domain.RefundStatusFailed
		}
	}

	actualAmount := amount
	if amtVal, ok := razorpayAmount(response["amount"]); ok && amtVal > 0 {
		actualAmount = decimal.NewFromInt(amtVal).Div(decimal.NewFromInt(100))
	}

	p.logger.InfoContext(ctx, "razorpay_refund_processed",
		slog.String("refund_id", refundID),
		slog.String("gateway_payment_id", trimmedPaymentID),
		slog.String("status", string(refundStatus)),
		slog.String("amount", actualAmount.String()),
	)

	return &domain.RefundResult{
		RefundID: refundID,
		Amount:   actualAmount,
		Status:   refundStatus,
	}, nil
}

var _ PaymentProvider = (*RazorpayProvider)(nil)
var _ domain.PaymentRefundProvider = (*RazorpayProvider)(nil)
