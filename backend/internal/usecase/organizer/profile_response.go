package organizer

import (
	"fmt"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

type OrganizerProfileResponse struct {
	OrganizerID string `json:"organizer_id"`
	Status      string `json:"status"`

	BusinessName string `json:"business_name"`
	BusinessType string `json:"business_type"`
	Description  string `json:"description"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Website      string `json:"website"`
	LogoURL      string `json:"logo_url"`

	Address     *OrganizerAddressResponse     `json:"address,omitempty"`
	BankAccount *OrganizerBankAccountResponse `json:"bank_account,omitempty"`
}

type OrganizerAddressResponse struct {
	AddressLine string `json:"address_line"`
	City        string `json:"city"`
	State       string `json:"state"`
	Country     string `json:"country"`
	PostalCode  string `json:"postal_code"`
}

type OrganizerBankAccountResponse struct {
	BankName            string `json:"bank_name"`
	AccountHolderName   string `json:"account_holder_name"`
	AccountNumberMasked string `json:"account_number_masked"`
	IFSCCode            string `json:"ifsc_code"`
}

func toOrganizerProfileResponse(
	organizer *domain.Organizer,
	profile *domain.OrganizerProfile,
	address *domain.OrganizerAddress,
	bankAccount *domain.OrganizerBankAccount,
) OrganizerProfileResponse {
	response := OrganizerProfileResponse{
		OrganizerID:  organizer.ID.String(),
		Status:       organizer.Status,
		BusinessName: profile.BusinessName,
		BusinessType: profile.BusinessType,
		Description:  profile.Description,
		Phone:        profile.Phone,
		Email:        profile.Email,
		Website:      profile.Website,
		LogoURL:      profile.LogoURL,
	}

	if address != nil {
		response.Address = &OrganizerAddressResponse{
			AddressLine: address.AddressLine,
			City:        address.City,
			State:       address.State,
			Country:     address.Country,
			PostalCode:  address.PostalCode,
		}
	}

	if bankAccount != nil {
		response.BankAccount = &OrganizerBankAccountResponse{
			BankName:            bankAccount.BankName,
			AccountHolderName:   bankAccount.AccountHolderName,
			AccountNumberMasked: maskAccountNumber(bankAccount.AccountNumberLast4),
			IFSCCode:            bankAccount.IFSCCode,
		}
	}

	return response
}

func maskAccountNumber(last4 string) string {
	if len(last4) != 4 {
		return ""
	}

	return fmt.Sprintf("********%s", last4)
}
