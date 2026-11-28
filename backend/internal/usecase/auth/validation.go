package auth

import (
	"regexp"
	"strings"
)

func validateFullName(name string) bool {
	name = strings.TrimSpace(name)

	if len(name) < 2 || len(name) > 50 {
		return false
	}

	pattern := `^[A-Za-z]+(?: [A-Za-z]+)*$`

	return regexp.MustCompile(pattern).MatchString(name)
}

func validateEmail(email string) bool {
	email = strings.TrimSpace(email)

	if len(email) > 254 {
		return false
	}

	pattern := `^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`

	return regexp.MustCompile(pattern).MatchString(email)
}

func validatePhone(phone string) bool {
	phone = strings.TrimSpace(phone)

	// Indian 10-digit mobile number.
	return regexp.MustCompile(`^[6-9][0-9]{9}$`).MatchString(phone)
}

var (
	hasUpperRegex   = regexp.MustCompile(`[A-Z]`)
	hasLowerRegex   = regexp.MustCompile(`[a-z]`)
	hasNumberRegex  = regexp.MustCompile(`[0-9]`)
	hasSpecialRegex = regexp.MustCompile(`[^A-Za-z0-9]`)
)

func validatePassword(password string) bool {
	return ValidatePassword(password)
}

func ValidatePassword(password string) bool {
	if len(password) < 8 || len(password) > 72 {
		return false
	}

	return hasUpperRegex.MatchString(password) &&
		hasLowerRegex.MatchString(password) &&
		hasNumberRegex.MatchString(password) &&
		hasSpecialRegex.MatchString(password)
}
