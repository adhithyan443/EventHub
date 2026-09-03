package organizer

type SubmitApplicationInput struct {
	BusinessName        string `json:"business_name" binding:"required"`
	BusinessType        string `json:"business_type" binding:"required"`
	BusinessDescription string `json:"business_description" binding:"required"`
	Website             string `json:"website"`
	ContactPhone        string `json:"contact_phone" binding:"required"`

	AddressLine string `json:"address_line" binding:"required"`
	City        string `json:"city" binding:"required"`
	State       string `json:"state" binding:"required"`
	Country     string `json:"country" binding:"required"`
	PostalCode  string `json:"postal_code" binding:"required"`

	BankName          string `json:"bank_name" binding:"required"`
	AccountHolderName string `json:"account_holder_name" binding:"required"`
	AccountNumber     string `json:"account_number" binding:"required"`
	IFSCCode          string `json:"ifsc_code" binding:"required"`

	GSTNumber string `json:"gst_number"`
	PANNumber string `json:"pan_number" binding:"required"`

	LogoURL                 string `json:"logo_url"`
	VerificationDocumentURL string `json:"verification_document_url" binding:"required"`
}