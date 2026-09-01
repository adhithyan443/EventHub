package organizer

type SubmitApplicationInput struct {
	BusinessName        string `json:"business_name" binding:"required,min=2,max=255"`
	BusinessType        string `json:"business_type" binding:"required,min=2,max=100"`
	BusinessDescription string `json:"business_description" binding:"required,min=10,max=2000"`
	Website             string `json:"website" binding:"omitempty,url,max=500"`

	ContactPhone string `json:"contact_phone" binding:"required,min=10,max=30"`

	BankName          string `json:"bank_name" binding:"required,min=2,max=255"`
	AccountHolderName string `json:"account_holder_name" binding:"required,min=2,max=255"`
	AccountNumber     string `json:"account_number" binding:"required,min=8,max=34"`
	IFSCCode          string `json:"ifsc_code" binding:"required"`

	GSTNumber string `json:"gst_number" binding:"omitempty"`
	PANNumber string `json:"pan_number" binding:"required"`

	LogoURL                 string `json:"logo_url" binding:"omitempty,url,max=1000"`
	VerificationDocumentURL string `json:"verification_document_url" binding:"required,url,max=1000"`
}
