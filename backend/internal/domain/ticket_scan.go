package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	TicketScanStatusSuccess = "SUCCESS"
)

type TicketScan struct {
	ID        uuid.UUID
	TicketID  uuid.UUID
	ScannedBy uuid.UUID
	Status    string
	ScannedAt time.Time
}

type TicketScanRepository interface {
	Create(ctx context.Context, scan *TicketScan) error
}
