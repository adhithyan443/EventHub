package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func generateOTP() (string, error) {
	const max = 1000000

	bytes := make([]byte, 4)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	number := uint32(bytes[0])<<24 |
		uint32(bytes[1])<<16 |
		uint32(bytes[2])<<8 |
		uint32(bytes[3])

	otp := number % max

	return fmt.Sprintf("%06d", otp), nil
}

func hashOTP(otp string) string {
	hash := sha256.Sum256([]byte(otp))

	return hex.EncodeToString(hash[:])
}
