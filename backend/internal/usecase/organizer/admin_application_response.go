//DTO
package organizer

import (
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type AdminApplicationResponse struct {
	ID                      uuid.UUID `json:"id"`
	UserID                  uuid.UUID `json:"user_id"`
	BusinessName            string    `json:"business_name"`
	BusinessType            string    `json:"business_type"`
	Description             string    `json:"description"`
	Phone                   string    `json:"phone"`
	Website                 string    `json:"website"`
	GSTNumber               string    `json:"gst_number"`
	PANNumber               string    `json:"pan_number"`
	BankName                string    `json:"bank_name"`
	AccountHolderName       string    `json:"account_holder_name"`
	IFSCCode                string    `json:"ifsc_code"`
	LogoURL                 string    `json:"logo_url"`
	VerificationDocumentURL string    `json:"verification_document_url"`
	Status                  string    `json:"status"`
	RejectionReason         string    `json:"rejection_reason"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func ToAdminApplicationResponse(
	application *domain.OrganizerApplication,
) AdminApplicationResponse {
	return AdminApplicationResponse{
		ID:                      application.ID,
		UserID:                  application.UserID,
		BusinessName:            application.BusinessName,
		BusinessType:            application.BusinessType,
		Description:             application.Description,
		Phone:                   application.Phone,
		Website:                 application.Website,
		GSTNumber:               application.GSTNumber,
		PANNumber:               application.PANNumber,
		BankName:                application.BankName,
		AccountHolderName:       application.AccountHolderName,
		IFSCCode:                application.IFSCCode,
		LogoURL:                 application.LogoURL,
		VerificationDocumentURL: application.VerificationDocumentURL,
		Status:                  string(application.Status),
		RejectionReason:         application.RejectionReason,
		CreatedAt:               application.CreatedAt,
		UpdatedAt:               application.UpdatedAt,
	}
}