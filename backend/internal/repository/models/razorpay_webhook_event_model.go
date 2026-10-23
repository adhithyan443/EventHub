package models

import "time"

type RazorpayWebhookEventModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventID   string    `gorm:"type:varchar(100);not null;uniqueIndex"`
	EventType string    `gorm:"type:varchar(100);not null"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (RazorpayWebhookEventModel) TableName() string {
	return "razorpay_webhook_events"
}
