package handler

import (
	"crypto/rand"
	"encoding/base64"
)

func generateOAuthStatae() (string, error) {
	state := make([]byte, 32)

	if _, err := rand.Read(state); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(state), nil
}
