package ticket

import (
	"errors"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

var ErrQRGenerationFailed = errors.New("QR code generation failed")

// GenerateQRImage creates a PNG QR code containing a signed ticket payload.
func (s *QRService) GenerateQRImage(ticketID uuid.UUID) ([]byte, error) {
	payload, err := s.GeneratePayload(ticketID)
	if err != nil {
		return nil, err
	}

	png, err := qrcode.Encode(
		payload,
		qrcode.Medium,
		256,
	)
	if err != nil {
		return nil, ErrQRGenerationFailed
	}

	return png, nil
}