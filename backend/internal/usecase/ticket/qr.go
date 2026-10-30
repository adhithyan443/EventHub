package ticket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidQRSecret  = errors.New("QR signing secret is invalid")
	ErrInvalidQRPayload = errors.New("QR payload is invalid")
)

type QRService struct {
	secret []byte
}

func NewQRService(secret string) (*QRService, error) {
	if len(strings.TrimSpace(secret)) < 32 {
		return nil, ErrInvalidQRSecret
	}

	return &QRService{
		secret: []byte(secret),
	}, nil
}

// GeneratePayload produces a signed payload containing the ticket ID.
func (s *QRService) GeneratePayload(ticketID uuid.UUID) (string, error) {
	if ticketID == uuid.Nil {
		return "", ErrInvalidQRPayload
	}

	id := ticketID.String()
	signature := s.sign(id)

	return id + "." + signature, nil
}

// VerifyPayload validates the ticket ID and its HMAC signature.
func (s *QRService) VerifyPayload(payload string) (uuid.UUID, error) {
	id, signature, ok := strings.Cut(payload, ".")
	if !ok || id == "" || signature == "" {
		return uuid.Nil, ErrInvalidQRPayload
	}

	ticketID, err := uuid.Parse(id)
	if err != nil || ticketID == uuid.Nil {
		return uuid.Nil, ErrInvalidQRPayload
	}

	expected := s.sign(ticketID.String())

	if !hmac.Equal(
		[]byte(signature),
		[]byte(expected),
	) {
		return uuid.Nil, ErrInvalidQRPayload
	}

	return ticketID, nil
}

const bookingQRPrefix = "EVH:BK:"

// GenerateBookingPayload produces a signed payload for a booking ID.
func (s *QRService) GenerateBookingPayload(bookingID uuid.UUID) (string, error) {
	if bookingID == uuid.Nil {
		return "", ErrInvalidQRPayload
	}

	raw := bookingQRPrefix + bookingID.String()
	signature := s.sign(raw)

	return raw + "." + signature, nil
}

// VerifyBookingPayload validates a booking QR payload and its HMAC signature, returning the booking ID.
func (s *QRService) VerifyBookingPayload(payload string) (uuid.UUID, error) {
	raw, signature, ok := strings.Cut(payload, ".")
	if !ok || raw == "" || signature == "" {
		return uuid.Nil, ErrInvalidQRPayload
	}

	if !strings.HasPrefix(raw, bookingQRPrefix) {
		return uuid.Nil, ErrInvalidQRPayload
	}

	idStr := strings.TrimPrefix(raw, bookingQRPrefix)
	bookingID, err := uuid.Parse(idStr)
	if err != nil || bookingID == uuid.Nil {
		return uuid.Nil, ErrInvalidQRPayload
	}

	expected := s.sign(raw)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return uuid.Nil, ErrInvalidQRPayload
	}

	return bookingID, nil
}

func (s *QRService) sign(raw string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(raw))

	return hex.EncodeToString(mac.Sum(nil))
}
