package domain

import (
	"context"
	"errors"
)

var ErrDuplicateWebhookEvent = errors.New("duplicate Razorpay webhook event")

type RazorpayWebhookEventRepository interface {
	Create(ctx context.Context, eventID, eventType string) error
}