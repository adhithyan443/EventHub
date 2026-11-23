package email_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/smtp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/service/email"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyNetError struct {
	msg     string
	timeout bool
}

func (e *dummyNetError) Error() string   { return e.msg }
func (e *dummyNetError) Timeout() bool   { return e.timeout }
func (e *dummyNetError) Temporary() bool { return true }

func newTestSMTPEmailService() (*email.SMTPEmailService, *slog.Logger) {
	svc := email.NewSMTPEmailService(
		"smtp.example.com",
		"587",
		"user@example.com",
		"secret",
		"noreply@eventhub.com",
		"http://localhost:3000",
	).(*email.SMTPEmailService)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc.SetLogger(logger)
	return svc, logger
}

func TestSMTPEmailService_NotifyCustomer_Success(t *testing.T) {
	svc, _ := newTestSMTPEmailService()

	var sentAddr string
	var sentFrom string
	var sentTo []string
	var sentMsg []byte
	var mu sync.Mutex

	svc.SetSendMailFn(func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		mu.Lock()
		defer mu.Unlock()
		sentAddr = addr
		sentFrom = from
		sentTo = to
		sentMsg = msg
		return nil
	})

	evID := uuid.New()
	custID := uuid.New()
	notice := domain.CustomerCancellationNotice{
		CustomerID:         custID,
		CustomerName:       "Jane Customer",
		CustomerEmail:      "customer@example.com",
		EventID:            evID,
		EventTitle:         "Annual Tech Expo 2026",
		EventDate:          "2026-12-15 10:00 UTC",
		CancellationReason: "Unforeseen venue structural repairs required.",
		BookingReferences:  []string{"BK-11111", "BK-22222"},
		TicketsCancelled:   3,
		RefundAmount:       decimal.NewFromFloat(225.00),
		RefundStatus:       "INITIATED",
		RefundInformation:  "Refunds will be processed in 5-10 business days.",
	}

	ctx := context.Background()
	err := svc.NotifyCustomer(ctx, notice)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, "smtp.example.com:587", sentAddr)
	assert.Equal(t, "noreply@eventhub.com", sentFrom)
	assert.Equal(t, []string{"customer@example.com"}, sentTo)

	msgStr := string(sentMsg)
	assert.Contains(t, msgStr, "To: customer@example.com")
	assert.Contains(t, msgStr, "Subject: Event Cancellation Notice: Annual Tech Expo 2026")
	assert.Contains(t, msgStr, "Hello Jane Customer,")
	assert.Contains(t, msgStr, "Event: Annual Tech Expo 2026")
	assert.Contains(t, msgStr, "Date: 2026-12-15 10:00 UTC")
	assert.Contains(t, msgStr, "Reason for Cancellation: Unforeseen venue structural repairs required.")
	assert.Contains(t, msgStr, "Booking Reference(s): BK-11111, BK-22222")
	assert.Contains(t, msgStr, "Tickets Cancelled: 3")
	assert.Contains(t, msgStr, "Refund Amount: 225.00")
	assert.Contains(t, msgStr, "Refund Status: INITIATED")
	assert.Contains(t, msgStr, "Refund Information:\nRefunds will be processed in 5-10 business days.")
}

func TestSMTPEmailService_NotifyCustomer_DefaultRefundInfo(t *testing.T) {
	svc, _ := newTestSMTPEmailService()

	var sentMsg []byte
	svc.SetSendMailFn(func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		sentMsg = msg
		return nil
	})

	notice := domain.CustomerCancellationNotice{
		CustomerID:         uuid.New(),
		CustomerName:       "",
		CustomerEmail:      "customer@example.com",
		EventID:            uuid.New(),
		EventTitle:         "Free Gathering",
		EventDate:          "2026-11-30 09:00 UTC",
		CancellationReason: "Bad weather",
		BookingReferences:  []string{"BK-SINGLE"},
		TicketsCancelled:   1,
		RefundAmount:       decimal.Zero,
		RefundStatus:       "NO_REFUND",
		RefundInformation:  "", // should trigger default
	}

	err := svc.NotifyCustomer(context.Background(), notice)
	require.NoError(t, err)

	msgStr := string(sentMsg)
	assert.Contains(t, msgStr, "Hello Valued Customer,")
	assert.Contains(t, msgStr, "A refund has been initiated to your original payment method in accordance with standard payment processing procedures.")
	assert.Contains(t, msgStr, "Booking Reference(s): BK-SINGLE")
}

func TestSMTPEmailService_NotifyOrganizer_Success(t *testing.T) {
	svc, _ := newTestSMTPEmailService()

	var sentTo []string
	var sentMsg []byte
	svc.SetSendMailFn(func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		sentTo = to
		sentMsg = msg
		return nil
	})

	now := time.Now().UTC()
	notice := domain.OrganizerCancellationNotice{
		OrganizerID:           uuid.New(),
		OrganizerName:         "Summit Org Team",
		OrganizerEmail:        "organizer@example.com",
		EventID:               uuid.New(),
		EventTitle:            "Summit 2026",
		EventDate:             "2026-12-01 10:00 UTC",
		CancelledBy:           "Organizer Jane",
		ActorRole:             "organizer",
		CancellationReason:    "Keynote speaker illness.",
		CancelledAt:           now,
		TotalBookingsAffected: 15,
		TotalTicketsAffected:  35,
		TotalRefundAmount:     decimal.NewFromFloat(3500.00),
		SettlementImpact:      "Pending settlements voided and net organizer earnings retracted.",
	}

	err := svc.NotifyOrganizer(context.Background(), notice)
	require.NoError(t, err)

	assert.Equal(t, []string{"organizer@example.com"}, sentTo)
	msgStr := string(sentMsg)
	assert.Contains(t, msgStr, "To: organizer@example.com")
	assert.Contains(t, msgStr, "Subject: Event Cancellation Confirmation: Summit 2026")
	assert.Contains(t, msgStr, "Hello Summit Org Team,")
	assert.Contains(t, msgStr, "Cancelled By: Organizer Jane (Role: organizer)")
	assert.Contains(t, msgStr, "Cancellation Reason: Keynote speaker illness.")
	assert.Contains(t, msgStr, "Total Bookings Affected: 15")
	assert.Contains(t, msgStr, "Total Tickets Cancelled: 35")
	assert.Contains(t, msgStr, "Total Refund Amount Issued: 3500.00")
	assert.Contains(t, msgStr, "Settlement Impact: Pending settlements voided and net organizer earnings retracted.")
}

func TestSMTPEmailService_TransientRetrySuccess(t *testing.T) {
	svc, _ := newTestSMTPEmailService()

	var attempts int32
	svc.SetSendMailFn(func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			// First attempt fails with transient error
			return &dummyNetError{msg: "connection reset by peer", timeout: false}
		}
		// Second attempt succeeds
		return nil
	})

	notice := domain.CustomerCancellationNotice{
		CustomerEmail: "customer@example.com",
		EventTitle:    "Concert",
	}

	err := svc.NotifyCustomer(context.Background(), notice)
	require.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
}

func TestSMTPEmailService_PermanentFailure_NoRetry(t *testing.T) {
	svc, _ := newTestSMTPEmailService()

	var attempts int32
	permanentErr := errors.New("550 5.1.1 User unknown")

	svc.SetSendMailFn(func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		atomic.AddInt32(&attempts, 1)
		return permanentErr
	})

	notice := domain.CustomerCancellationNotice{
		CustomerEmail: "invalid@example.com",
		EventTitle:    "Concert",
	}

	err := svc.NotifyCustomer(context.Background(), notice)
	require.Error(t, err)
	assert.True(t, errors.Is(err, permanentErr))
	// Exactly 1 attempt should have been made, no retry on 550
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

func TestSMTPEmailService_TransientFailure_MaxRetriesExhausted(t *testing.T) {
	svc, _ := newTestSMTPEmailService()

	var attempts int32
	transientErr := &dummyNetError{msg: "i/o timeout", timeout: true}

	svc.SetSendMailFn(func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		atomic.AddInt32(&attempts, 1)
		return transientErr
	})

	notice := domain.CustomerCancellationNotice{
		CustomerEmail: "customer@example.com",
		EventTitle:    "Concert",
	}

	err := svc.NotifyCustomer(context.Background(), notice)
	require.Error(t, err)
	assert.True(t, errors.Is(err, transientErr))
	// Exactly 3 attempts should be made
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestSMTPEmailService_ContextCancellation_AbortsRetry(t *testing.T) {
	svc, _ := newTestSMTPEmailService()

	ctx, cancel := context.WithCancel(context.Background())

	var attempts int32
	svc.SetSendMailFn(func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			// Cancel context immediately during first attempt
			cancel()
			return &dummyNetError{msg: "connection refused", timeout: false}
		}
		return nil
	})

	notice := domain.CustomerCancellationNotice{
		CustomerEmail: "customer@example.com",
		EventTitle:    "Concert",
	}

	err := svc.NotifyCustomer(ctx, notice)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

func TestSMTPEmailService_Validation(t *testing.T) {
	svc, _ := newTestSMTPEmailService()

	// Missing customer email
	err := svc.NotifyCustomer(context.Background(), domain.CustomerCancellationNotice{
		CustomerEmail: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer email cannot be empty")

	// Missing organizer email
	err = svc.NotifyOrganizer(context.Background(), domain.OrganizerCancellationNotice{
		OrganizerEmail: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "organizer email cannot be empty")
}
