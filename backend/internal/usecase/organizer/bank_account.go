package organizer

import "strings"

func getAccountNumberLast4(accountNumber string) string {
	accountNumber = strings.TrimSpace(accountNumber)

	if len(accountNumber) < 4 {
		return accountNumber
	}

	return accountNumber[len(accountNumber)-4:]
}
