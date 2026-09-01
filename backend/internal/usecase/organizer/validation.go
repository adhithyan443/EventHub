package organizer

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	panRegex = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)

	ifscRegex = regexp.MustCompile(`^[A-Z]{4}0[A-Z0-9]{6}$`)

	gstRegex = regexp.MustCompile(
		`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$`,
	)
)

func validateApplicationInput(input SubmitApplicationInput) error {
	input.BusinessName = strings.TrimSpace(input.BusinessName)
	input.BusinessDescription = strings.TrimSpace(input.BusinessDescription)
	input.ContactPhone = strings.TrimSpace(input.ContactPhone)

	input.BankName = strings.TrimSpace(input.BankName)
	input.AccountHolderName = strings.TrimSpace(input.AccountHolderName)

	input.PANNumber = strings.ToUpper(strings.TrimSpace(input.PANNumber))
	input.IFSCCode = strings.ToUpper(strings.TrimSpace(input.IFSCCode))
	input.BusinessType = strings.TrimSpace(input.BusinessType)

	if input.BusinessType == "" {
		return fmt.Errorf("business type cannot be empty")
	}

	if len(input.BusinessType) < 2 {
		return fmt.Errorf("business type must contain at least 2 characters")
	}

	if input.BusinessName == "" {
		return fmt.Errorf("business name cannot be empty")
	}

	if len(input.BusinessName) < 2 {
		return fmt.Errorf("business name must contain at least 2 characters")
	}

	if len(input.BusinessDescription) < 10 {
		return fmt.Errorf("business description must contain at least 10 characters")
	}

	if !panRegex.MatchString(input.PANNumber) {
		return fmt.Errorf("invalid PAN number")
	}

	if !ifscRegex.MatchString(input.IFSCCode) {
		return fmt.Errorf("invalid IFSC code")
	}

	if input.GSTNumber != "" {
		input.GSTNumber = strings.ToUpper(strings.TrimSpace(input.GSTNumber))

		if !gstRegex.MatchString(input.GSTNumber) {
			return fmt.Errorf("invalid GST number")
		}
	}

	if len(input.AccountNumber) < 8 {
		return fmt.Errorf("invalid account number")
	}

	return nil
}
