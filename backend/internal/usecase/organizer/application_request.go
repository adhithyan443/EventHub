package organizer

import "io"

type SubmitApplicationInput struct {
	BusinessName        string `form:"business_name" json:"business_name" binding:"required"`
	BusinessType        string `form:"business_type" json:"business_type" binding:"required"`
	BusinessDescription string `form:"business_description" json:"business_description" binding:"required"`
	Website             string `form:"website" json:"website"`
	ContactPhone        string `form:"contact_phone" json:"contact_phone" binding:"required"`

	AddressLine string `form:"address_line" json:"address_line" binding:"required"`
	City        string `form:"city" json:"city" binding:"required"`
	State       string `form:"state" json:"state" binding:"required"`
	Country     string `form:"country" json:"country" binding:"required"`
	PostalCode  string `form:"postal_code" json:"postal_code" binding:"required"`

	BankName          string `form:"bank_name" json:"bank_name" binding:"required"`
	AccountHolderName string `form:"account_holder_name" json:"account_holder_name" binding:"required"`
	AccountNumber     string `form:"account_number" json:"account_number" binding:"required"`
	IFSCCode          string `form:"ifsc_code" json:"ifsc_code" binding:"required"`

	GSTNumber string `form:"gst_number" json:"gst_number"`
	PANNumber string `form:"pan_number" json:"pan_number" binding:"required"`

	// These are populated internally after the files are validated and uploaded.
	LogoURL                 string `form:"-" json:"-"`
	VerificationDocumentURL string `form:"-" json:"-"`

	// Uploaded files are populated by the HTTP handler.
	LogoReader                 io.Reader `form:"-" json:"-"`
	LogoContentType            string    `form:"-" json:"-"`
	LogoSize                   int64     `form:"-" json:"-"`
	VerificationDocumentReader io.Reader `form:"-" json:"-"`
	VerificationDocumentType   string    `form:"-" json:"-"`
	VerificationDocumentSize   int64     `form:"-" json:"-"`
}
