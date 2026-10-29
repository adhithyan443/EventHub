package payment_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateWebhookSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestRazorpayProvider_VerifyWebhookSignature(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	secret := "test_webhook_secret_12345"
	provider, err := payment.NewRazorpayProvider("rzp_test_key", "rzp_test_secret", secret, logger)
	require.NoError(t, err)

	body := []byte(`{"entity":"event","event":"payment.captured"}`)
	validSignature := generateWebhookSignature(body, secret)

	tests := []struct {
		name      string
		body      []byte
		signature string
		expected  bool
	}{
		{
			name:      "valid signature",
			body:      body,
			signature: validSignature,
			expected:  true,
		},
		{
			name:      "tampered body",
			body:      []byte(`{"entity":"event","event":"payment.failed"}`),
			signature: validSignature,
			expected:  false,
		},
		{
			name:      "signature from wrong secret",
			body:      body,
			signature: generateWebhookSignature(body, "wrong_secret"),
			expected:  false,
		},
		{
			name:      "empty body",
			body:      nil,
			signature: validSignature,
			expected:  false,
		},
		{
			name:      "empty signature",
			body:      body,
			signature: "",
			expected:  false,
		},
		{
			name:      "malformed hex signature",
			body:      body,
			signature: "not_a_valid_hex_string!!!",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.VerifyWebhookSignature(tt.body, tt.signature)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRazorpayProvider_FetchRefunds_Validation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	provider, err := payment.NewRazorpayProvider("rzp_test_key", "rzp_test_secret", "secret", logger)
	require.NoError(t, err)

	// Empty gateway payment ID
	_, err = provider.FetchRefunds(t.Context(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gateway payment ID is required")

	// Whitespace gateway payment ID
	_, err = provider.FetchRefunds(t.Context(), "   ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gateway payment ID is required")
}
