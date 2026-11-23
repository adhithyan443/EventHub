package email

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

var _ domain.EmailService = (*SMTPEmailService)(nil)
var _ domain.EventCancellationNotifier = (*SMTPEmailService)(nil)

type SMTPEmailService struct {
	host        string
	port        string
	username    string
	password    string
	from        string
	frontendURL string
	logger      *slog.Logger
	sendMailFn  func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

func (s *SMTPEmailService) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

func (s *SMTPEmailService) SetSendMailFn(fn func(addr string, a smtp.Auth, from string, to []string, msg []byte) error) {
	s.sendMailFn = fn
}

func (s *SMTPEmailService) getLogger() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.Default()
}

func NewSMTPEmailService(
	host string,
	port string,
	username string,
	password string,
	from string,
	frontendURL string,
) domain.EmailService {
	return &SMTPEmailService{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		from:        from,
		frontendURL: frontendURL,
	}
}

func (s *SMTPEmailService) SendOTP(to string, otp string) error {
	auth := smtp.PlainAuth(
		"",
		s.username,
		s.password,
		s.host,
	)

	subject := "EventHub - Verify Your Email Address"

	body := fmt.Sprintf(`Hello,

Thank you for registering with EventHub!

To complete your registration and verify your email address, please use the following verification code:

%s

This verification code is valid for 10 minutes.

For your security, please do not share this code with anyone. EventHub will never ask you for your verification code.

If you did not create an EventHub account, you can safely ignore this email.

Best regards,
The EventHub Team
`, otp)

	message := []byte(
		fmt.Sprintf(
			"From: EventHub <%s>\r\n"+
				"To: %s\r\n"+
				"Subject: %s\r\n"+
				"MIME-Version: 1.0\r\n"+
				"Content-Type: text/plain; charset=UTF-8\r\n"+
				"\r\n"+
				"%s",
			s.from,
			to,
			subject,
			body,
		),
	)

	address := s.host + ":" + s.port

	if err := smtp.SendMail(
		address,
		auth,
		s.from,
		[]string{to},
		message,
	); err != nil {
		return err
	}

	return nil
}

func (s *SMTPEmailService) SendEmailChangeOTP(to string, otp string) error {
	auth := smtp.PlainAuth(
		"",
		s.username,
		s.password,
		s.host,
	)

	subject := "EventHub - Verify Your New Email Address"

	body := fmt.Sprintf(`Hello,

We received a request to change your EventHub account email to this address.

To complete your email verification, please use the following verification code:

%s

This verification code is valid for 10 minutes.

For your security, please do not share this code with anyone. EventHub will never ask you for your verification code.

If you did not request this email change, please ignore this email or contact support immediately.

Best regards,
The EventHub Team
`, otp)

	message := []byte(
		fmt.Sprintf(
			"From: EventHub <%s>\r\n"+
				"To: %s\r\n"+
				"Subject: %s\r\n"+
				"MIME-Version: 1.0\r\n"+
				"Content-Type: text/plain; charset=UTF-8\r\n"+
				"\r\n"+
				"%s",
			s.from,
			to,
			subject,
			body,
		),
	)

	address := s.host + ":" + s.port

	if err := smtp.SendMail(
		address,
		auth,
		s.from,
		[]string{to},
		message,
	); err != nil {
		return err
	}

	return nil
}

func (s *SMTPEmailService) SendPasswordResetEmail(
	to string,
	token string,
) error {
	auth := smtp.PlainAuth(
		"",
		s.username,
		s.password,
		s.host,
	)

	resetURL := fmt.Sprintf(
		"%s/reset-password?token=%s",
		strings.TrimRight(s.frontendURL, "/"),
		token,
	)

	subject := "EventHub - Reset Your Password"

	body := fmt.Sprintf(`Hello,

We received a request to reset the password for your EventHub account.

To create a new password, click the link below:

%s

This password reset link is valid for 15 minutes and can only be used once.

If you did not request a password reset, you can safely ignore this email. Your account will remain secure.

For your security, never share this reset link with anyone.

Best regards,
The EventHub Team
`, resetURL)

	message := []byte(
		fmt.Sprintf(
			"From: EventHub <%s>\r\n"+
				"To: %s\r\n"+
				"Subject: %s\r\n"+
				"MIME-Version: 1.0\r\n"+
				"Content-Type: text/plain; charset=UTF-8\r\n"+
				"\r\n"+
				"%s",
			s.from,
			to,
			subject,
			body,
		),
	)

	address := s.host + ":" + s.port

	if err := smtp.SendMail(
		address,
		auth,
		s.from,
		[]string{to},
		message,
	); err != nil {
		return err
	}

	return nil
}

type deliveryMeta struct {
	recipientType string
	eventID       uuid.UUID
	actorID       uuid.UUID
	actorRole     string
	bookingRef    string
}

// NotifyCustomer sends a structured event cancellation notice to an attendee with affected bookings.
func (s *SMTPEmailService) NotifyCustomer(ctx context.Context, notice domain.CustomerCancellationNotice) error {
	to := strings.TrimSpace(notice.CustomerEmail)
	if to == "" {
		return errors.New("customer email cannot be empty")
	}

	subject := fmt.Sprintf("Event Cancellation Notice: %s", notice.EventTitle)

	var refStr string
	if len(notice.BookingReferences) == 1 {
		refStr = notice.BookingReferences[0]
	} else if len(notice.BookingReferences) > 1 {
		refStr = strings.Join(notice.BookingReferences, ", ")
	} else {
		refStr = "N/A"
	}

	refundInfo := strings.TrimSpace(notice.RefundInformation)
	if refundInfo == "" {
		refundInfo = "A refund has been initiated to your original payment method in accordance with standard payment processing procedures."
	}

	customerName := strings.TrimSpace(notice.CustomerName)
	if customerName == "" {
		customerName = "Valued Customer"
	}

	body := fmt.Sprintf(`Hello %s,

We regret to inform you that the following event has been cancelled:

Event: %s
Date: %s
Reason for Cancellation: %s

Your Booking Details:
Booking Reference(s): %s
Tickets Cancelled: %d
Refund Amount: %s
Refund Status: %s

Refund Information:
%s

If you have any questions or require assistance, please contact EventHub support.

Best regards,
The EventHub Team
`,
		customerName,
		notice.EventTitle,
		notice.EventDate,
		notice.CancellationReason,
		refStr,
		notice.TicketsCancelled,
		notice.RefundAmount.StringFixed(2),
		notice.RefundStatus,
		refundInfo,
	)

	meta := deliveryMeta{
		recipientType: "customer",
		eventID:       notice.EventID,
		actorRole:     "customer",
		bookingRef:    refStr,
	}

	return s.sendWithRetry(ctx, to, subject, body, meta)
}

// NotifyOrganizer sends an event cancellation summary to the organizer.
func (s *SMTPEmailService) NotifyOrganizer(ctx context.Context, notice domain.OrganizerCancellationNotice) error {
	to := strings.TrimSpace(notice.OrganizerEmail)
	if to == "" {
		return errors.New("organizer email cannot be empty")
	}

	subject := fmt.Sprintf("Event Cancellation Confirmation: %s", notice.EventTitle)

	organizerName := strings.TrimSpace(notice.OrganizerName)
	if organizerName == "" {
		organizerName = "Event Organizer"
	}

	actorRole := notice.ActorRole
	if actorRole == "" {
		actorRole = "organizer"
	}

	cancelledBy := notice.CancelledBy
	if cancelledBy == "" {
		cancelledBy = "Event Organizer"
	}

	settlementImpact := strings.TrimSpace(notice.SettlementImpact)
	if settlementImpact == "" {
		settlementImpact = "Pending settlements for this event have been voided and net earnings have been revoked."
	}

	cancelledAtStr := notice.CancelledAt.Format(time.RFC1123)
	if notice.CancelledAt.IsZero() {
		cancelledAtStr = time.Now().Format(time.RFC1123)
	}

	body := fmt.Sprintf(`Hello %s,

This email confirms that the following event has been cancelled on EventHub:

Event: %s
Scheduled Date: %s
Cancelled By: %s (Role: %s)
Cancellation Reason: %s
Timestamp: %s

Cancellation Impact Summary:
Total Bookings Affected: %d
Total Tickets Cancelled: %d
Total Refund Amount Issued: %s
Settlement Impact: %s

If you did not initiate this cancellation or have urgent questions, please contact platform administration immediately.

Best regards,
The EventHub Team
`,
		organizerName,
		notice.EventTitle,
		notice.EventDate,
		cancelledBy,
		actorRole,
		notice.CancellationReason,
		cancelledAtStr,
		notice.TotalBookingsAffected,
		notice.TotalTicketsAffected,
		notice.TotalRefundAmount.StringFixed(2),
		settlementImpact,
	)

	meta := deliveryMeta{
		recipientType: "organizer",
		eventID:       notice.EventID,
		actorRole:     actorRole,
	}

	return s.sendWithRetry(ctx, to, subject, body, meta)
}

func (s *SMTPEmailService) sendRaw(to, subject, body string) error {
	auth := smtp.PlainAuth(
		"",
		s.username,
		s.password,
		s.host,
	)

	message := []byte(
		fmt.Sprintf(
			"From: EventHub <%s>\r\n"+
				"To: %s\r\n"+
				"Subject: %s\r\n"+
				"MIME-Version: 1.0\r\n"+
				"Content-Type: text/plain; charset=UTF-8\r\n"+
				"\r\n"+
				"%s",
			s.from,
			to,
			subject,
			body,
		),
	)

	address := s.host + ":" + s.port

	if s.sendMailFn != nil {
		return s.sendMailFn(address, auth, s.from, []string{to}, message)
	}

	return smtp.SendMail(
		address,
		auth,
		s.from,
		[]string{to},
		message,
	)
}

func (s *SMTPEmailService) sendWithRetry(ctx context.Context, to, subject, body string, meta deliveryMeta) error {
	var lastErr error
	maxAttempts := 3
	backoff := 100 * time.Millisecond

	s.getLogger().InfoContext(ctx, "cancellation_notification_delivery_started",
		slog.String("recipient_type", meta.recipientType),
		slog.String("event_id", meta.eventID.String()),
		slog.String("actor_role", meta.actorRole),
		slog.String("booking_ref", meta.bookingRef),
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			s.getLogger().WarnContext(ctx, "cancellation_notification_aborted_context_done",
				slog.String("recipient_type", meta.recipientType),
				slog.String("event_id", meta.eventID.String()),
				slog.Any("error", ctx.Err()),
			)
			return ctx.Err()
		default:
		}

		err := s.sendRaw(to, subject, body)
		if err == nil {
			s.getLogger().InfoContext(ctx, "cancellation_notification_delivery_success",
				slog.String("recipient_type", meta.recipientType),
				slog.String("event_id", meta.eventID.String()),
				slog.String("actor_role", meta.actorRole),
				slog.Int("attempt", attempt),
			)
			return nil
		}

		lastErr = err
		if !isTransientSMTPError(err) {
			s.getLogger().ErrorContext(ctx, "cancellation_notification_permanent_failure",
				slog.String("recipient_type", meta.recipientType),
				slog.String("event_id", meta.eventID.String()),
				slog.String("actor_role", meta.actorRole),
				slog.Int("attempt", attempt),
				slog.Any("error", err),
			)
			return err
		}

		if attempt < maxAttempts {
			s.getLogger().WarnContext(ctx, "cancellation_notification_transient_failure_retrying",
				slog.String("recipient_type", meta.recipientType),
				slog.String("event_id", meta.eventID.String()),
				slog.String("actor_role", meta.actorRole),
				slog.Int("attempt", attempt),
				slog.Any("error", err),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}
	}

	s.getLogger().ErrorContext(ctx, "cancellation_notification_exhausted_retries",
		slog.String("recipient_type", meta.recipientType),
		slog.String("event_id", meta.eventID.String()),
		slog.String("actor_role", meta.actorRole),
		slog.Int("attempts", maxAttempts),
		slog.Any("error", lastErr),
	)
	return lastErr
}

func isTransientSMTPError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "reset by peer") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "eof") {
		return true
	}
	if strings.Contains(errStr, "421") ||
		strings.Contains(errStr, "450") ||
		strings.Contains(errStr, "451") ||
		strings.Contains(errStr, "452") {
		return true
	}
	return false
}
