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

func validatePassword(password string) bool {
	if len(password) < 8 || len(password) > 72 {
		return false
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[^A-Za-z0-9]`).MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSpecial
}