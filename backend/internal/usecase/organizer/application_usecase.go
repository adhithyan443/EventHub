package organizer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/service/encryption"
	"github.com/adhithyan443/EventHub/backend/internal/service/storage"
	"github.com/google/uuid"
)

type ApplicationUsecase struct {
	repository        domain.OrganizerApplicationRepository
	encryptionService *encryption.Service
	storageService    *storage.S3Service
	logger            *slog.Logger
}

func NewApplicationUsecase(
	repository domain.OrganizerApplicationRepository,
	encryptionService *encryption.Service,
	storageService *storage.S3Service,
	logger *slog.Logger,
) *ApplicationUsecase {

	if logger == nil {
		logger = slog.Default()
	}

	return &ApplicationUsecase{
		repository:        repository,
		encryptionService: encryptionService,
		storageService:    storageService,
		logger:            logger,
	}
}

func (u *ApplicationUsecase) uploadApplicationFile(
	ctx context.Context,
	submissionID uuid.UUID,
	file io.Reader,
	contentType string,
	fileType string,
) (string, error) {
	if file == nil {
		return "", fmt.Errorf("%s file is required", fileType)
	}

	if u.storageService == nil {
		return "", fmt.Errorf("storage service is not configured")
	}

	var extension string

	switch contentType {
	case "application/pdf":
		extension = ".pdf"
	case "image/jpeg":
		extension = ".jpg"
	case "image/png":
		extension = ".png"
	case "image/webp":
		extension = ".webp"
	default:
		return "", fmt.Errorf("unsupported %s file type", fileType)
	}

	if fileType == "verification-document" && contentType == "image/webp" {
		return "", fmt.Errorf("unsupported verification document type")
	}

	key := filepath.ToSlash(filepath.Join(
		"organizer-applications",
		submissionID.String(),
		fileType+extension,
	))

	if err := u.storageService.Upload(ctx, key, file, contentType); err != nil {
		u.logger.Error(
			"organizer_application_file_upload_failed",
			"file_type", fileType,
			"error", err,
		)
		return "", fmt.Errorf("failed to upload %s", fileType)
	}

	u.logger.Info(
		"organizer_application_file_uploaded",
		"file_type", fileType,
		"object_key", key,
	)

	return key, nil
}

func (u *ApplicationUsecase) cleanupUploadedFiles(
	ctx context.Context,
	keys []string,
) {
	if u.storageService == nil {
		return
	}

	cleanupCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		10*time.Second,
	)
	defer cancel()

	for _, key := range keys {
		if key == "" {
			continue
		}

		if err := u.storageService.Delete(cleanupCtx, key); err != nil {
			u.logger.Error(
				"organizer_application_file_cleanup_failed",
				"object_key", key,
				"error", err,
			)
		}
	}
}

func (u *ApplicationUsecase) SubmitApplication(
	ctx context.Context,
	userID uuid.UUID,
	input SubmitApplicationInput,
) error {
	if err := validateApplicationInput(input); err != nil {
		u.logger.Warn(
			"organizer_application_validation_failed",
			"user_id", userID,
			"error", err,
		)
		return err
	}

	encryptedAccountNumber, err := u.encryptionService.Encrypt(
		input.AccountNumber,
	)
	if err != nil {
		u.logger.Error(
			"organizer_application_account_encryption_failed",
			"user_id", userID,
			"error", err,
		)

		return fmt.Errorf("failed to secure account information")
	}

	accountNumberLast4 := getAccountNumberLast4(input.AccountNumber)

	existing, err := u.repository.FindByUserID(userID)

	if err != nil && !errors.Is(err, domain.ErrOrganizerApplicationNotFound) {
		u.logger.Error(
			"organizer_application_lookup_failed",
			"user_id", userID,
			"error", err,
		)

		return fmt.Errorf("failed to check existing application: %w", err)
	}

	/*
		No existing application.

		Create the first application with PENDING status.
	*/
	if errors.Is(err, domain.ErrOrganizerApplicationNotFound) {
		if input.VerificationDocumentReader == nil {
			return appErrors.NewValidationError("verification document is required")
		}

		application := u.buildApplication(
			userID,
			input,
			encryptedAccountNumber,
			accountNumberLast4,
		)

		uploadedKeys := make([]string, 0, 2)

		documentKey, err := u.uploadApplicationFile(
			ctx,
			application.ID,
			input.VerificationDocumentReader,
			input.VerificationDocumentType,
			"verification-document",
		)
		if err != nil {
			return err
		}
		uploadedKeys = append(uploadedKeys, documentKey)

		application.VerificationDocumentURL = documentKey

		if input.LogoReader != nil {
			logoKey, err := u.uploadApplicationFile(
				ctx,
				application.ID,
				input.LogoReader,
				input.LogoContentType,
				"logo",
			)
			if err != nil {
				u.cleanupUploadedFiles(ctx, uploadedKeys)
				return err
			}

			uploadedKeys = append(uploadedKeys, logoKey)
			application.LogoURL = logoKey
		}

		if err := u.repository.Create(application); err != nil {
			u.cleanupUploadedFiles(ctx, uploadedKeys)

			u.logger.Error(
				"organizer_application_create_failed",
				"user_id", userID,
				"error", err,
			)
			return fmt.Errorf("failed to create organizer application: %w", err)
		}

		u.logger.Info(
			"organizer_application_submitted",
			"user_id", userID,
			"application_id", application.ID,
		)

		return nil
	}

	/*
		An application already exists.

		Only REJECTED applications can be resubmitted.
	*/
	switch existing.Status {
	case domain.ApplicationPending:
		u.logger.Warn(
			"organizer_application_already_pending",
			"user_id", userID,
			"application_id", existing.ID,
		)

		return fmt.Errorf("organizer application is already under review")

	case domain.ApplicationApproved:
		u.logger.Warn(
			"organizer_application_already_approved",
			"user_id", userID,
			"application_id", existing.ID,
		)

		return fmt.Errorf("organizer application is already approved")

	case domain.ApplicationRejected:

		uploadedKeys := make([]string, 0, 2)

		if input.VerificationDocumentReader != nil {
			documentKey, err := u.uploadApplicationFile(
				ctx,
				existing.ID,
				input.VerificationDocumentReader,
				input.VerificationDocumentType,
				"verification-document",
			)
			if err != nil {
				return err
			}
			uploadedKeys = append(uploadedKeys, documentKey)
			existing.VerificationDocumentURL = documentKey
		} else if strings.TrimSpace(existing.VerificationDocumentURL) == "" {
			return appErrors.NewValidationError("verification document is required")
		}

		if input.LogoReader != nil {
			logoKey, err := u.uploadApplicationFile(
				ctx,
				existing.ID,
				input.LogoReader,
				input.LogoContentType,
				"logo",
			)
			if err != nil {
				u.cleanupUploadedFiles(ctx, uploadedKeys)
				return err
			}
			uploadedKeys = append(uploadedKeys, logoKey)
			existing.LogoURL = logoKey
		}

		encryptedAccountNumber, err := u.encryptionService.Encrypt(
			input.AccountNumber,
		)
		if err != nil {
			u.cleanupUploadedFiles(ctx, uploadedKeys)
			u.logger.Error(
				"organizer_application_account_encryption_failed",
				"user_id", userID,
				"application_id", existing.ID,
				"error", err,
			)

			return fmt.Errorf("failed to secure account information")
		}

		accountNumberLast4 := getAccountNumberLast4(input.AccountNumber)
		updateApplication(existing, input, encryptedAccountNumber, accountNumberLast4)

		existing.Status = domain.ApplicationPending
		existing.RejectionReason = ""

		if err := u.repository.Update(existing); err != nil {
			u.cleanupUploadedFiles(ctx, uploadedKeys)
			u.logger.Error(
				"organizer_application_resubmit_failed",
				"user_id", userID,
				"application_id", existing.ID,
				"error", err,
			)

			return fmt.Errorf("failed to resubmit organizer application: %w", err)
		}

		u.logger.Info(
			"organizer_application_resubmitted",
			"user_id", userID,
			"application_id", existing.ID,
		)

		return nil

	default:
		u.logger.Error(
			"organizer_application_invalid_status",
			"user_id", userID,
			"application_id", existing.ID,
			"status", existing.Status,
		)

		return fmt.Errorf("invalid organizer application status")
	}
}

func (u *ApplicationUsecase) buildApplication(
	userID uuid.UUID,
	input SubmitApplicationInput,
	encryptedAccountNumber string,
	accountNumberLast4 string,
) *domain.OrganizerApplication {
	return &domain.OrganizerApplication{
		ID:     uuid.New(),
		UserID: userID,

		BusinessName: strings.TrimSpace(input.BusinessName),
		BusinessType: strings.TrimSpace(input.BusinessType),
		Description:  strings.TrimSpace(input.BusinessDescription),
		Phone:        strings.TrimSpace(input.ContactPhone),
		Website:      strings.TrimSpace(input.Website),
		AddressLine: strings.TrimSpace(
			input.AddressLine,
		),

		City: strings.TrimSpace(
			input.City,
		),

		State: strings.TrimSpace(
			input.State,
		),

		Country: strings.TrimSpace(
			input.Country,
		),

		PostalCode: strings.TrimSpace(
			input.PostalCode,
		),

		GSTNumber: strings.ToUpper(
			strings.TrimSpace(input.GSTNumber),
		),
		PANNumber: strings.ToUpper(
			strings.TrimSpace(input.PANNumber),
		),

		BankName: strings.TrimSpace(input.BankName),
		AccountHolderName: strings.TrimSpace(
			input.AccountHolderName,
		),

		AccountNumberEncrypted: encryptedAccountNumber,
		AccountNumberLast4:     accountNumberLast4,

		IFSCCode: strings.ToUpper(
			strings.TrimSpace(input.IFSCCode),
		),

		LogoURL: strings.TrimSpace(
			input.LogoURL,
		),

		VerificationDocumentURL: strings.TrimSpace(
			input.VerificationDocumentURL,
		),

		Status: domain.ApplicationPending,
	}
}

func updateApplication(
	application *domain.OrganizerApplication,
	input SubmitApplicationInput,
	encryptedAccountNumber string,
	accountNumberLast4 string,
) {
	application.BusinessName = strings.TrimSpace(
		input.BusinessName,
	)

	application.BusinessType = strings.TrimSpace(
		input.BusinessType,
	)

	application.Description = strings.TrimSpace(
		input.BusinessDescription,
	)

	application.Phone = strings.TrimSpace(
		input.ContactPhone,
	)

	application.Website = strings.TrimSpace(
		input.Website,
	)

	application.GSTNumber = strings.ToUpper(
		strings.TrimSpace(input.GSTNumber),
	)

	application.PANNumber = strings.ToUpper(
		strings.TrimSpace(input.PANNumber),
	)

	application.BankName = strings.TrimSpace(
		input.BankName,
	)

	application.AccountHolderName = strings.TrimSpace(
		input.AccountHolderName,
	)

	// TODO: Encrypt before storing.
	application.AccountNumberEncrypted = encryptedAccountNumber
	application.AccountNumberLast4 = accountNumberLast4

	application.IFSCCode = strings.ToUpper(
		strings.TrimSpace(input.IFSCCode),
	)

	if input.LogoURL != "" {
		application.LogoURL = strings.TrimSpace(
			input.LogoURL,
		)
	}

	if input.VerificationDocumentURL != "" {
		application.VerificationDocumentURL = strings.TrimSpace(
			input.VerificationDocumentURL,
		)
	}

	application.AddressLine = strings.TrimSpace(
		input.AddressLine,
	)

	application.City = strings.TrimSpace(
		input.City,
	)

	application.State = strings.TrimSpace(
		input.State,
	)

	application.Country = strings.TrimSpace(
		input.Country,
	)

	application.PostalCode = strings.TrimSpace(
		input.PostalCode,
	)
}

type OrganizerApplicationResponse struct {
	ID                      uuid.UUID `json:"id"`
	BusinessName            string    `json:"business_name"`
	BusinessType            string    `json:"business_type"`
	BusinessDescription     string    `json:"business_description"`
	Website                 string    `json:"website"`
	ContactPhone            string    `json:"contact_phone"`
	AddressLine             string    `json:"address_line"`
	City                    string    `json:"city"`
	State                   string    `json:"state"`
	Country                 string    `json:"country"`
	PostalCode              string    `json:"postal_code"`
	BankName                string    `json:"bank_name"`
	AccountHolderName       string    `json:"account_holder_name"`
	IFSCCode                string    `json:"ifsc_code"`
	GSTNumber               string    `json:"gst_number"`
	PANNumber               string    `json:"pan_number"`
	LogoURL                 string    `json:"logo_url"`
	VerificationDocumentURL string    `json:"verification_document_url"`
	Status                  string    `json:"status"`
	RejectionReason         string    `json:"rejection_reason,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (u *ApplicationUsecase) GetApplication(
	ctx context.Context,
	userID uuid.UUID,
) (*OrganizerApplicationResponse, error) {

	application, err := u.repository.FindByUserID(userID)
	if err != nil {

		if errors.Is(err, domain.ErrOrganizerApplicationNotFound) {
			return nil, appErrors.NewNotFoundError(
				"organizer application not found",
			)
		}

		u.logger.Error(
			"organizer_application_fetch_failed",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)

		return nil, fmt.Errorf("failed to fetch organizer application")
	}

	logoURL := application.LogoURL
	if logoURL != "" && u.storageService != nil {
		presigned, err := u.storageService.GetPresignedURL(ctx, logoURL)
		if err != nil {
			u.logger.Error(
				"organizer_application_logo_url_failed",
				"user_id", userID,
				"error", err,
			)
			return nil, fmt.Errorf("failed to generate organizer logo URL")
		}
		logoURL = presigned
	}

	documentURL := application.VerificationDocumentURL
	if documentURL != "" && u.storageService != nil {
		presigned, err := u.storageService.GetPresignedURL(ctx, documentURL)
		if err != nil {
			u.logger.Error(
				"organizer_application_document_url_failed",
				"user_id", userID,
				"error", err,
			)
			return nil, fmt.Errorf("failed to generate verification document URL")
		}
		documentURL = presigned
	}

	return &OrganizerApplicationResponse{
		ID:                      application.ID,
		BusinessName:            application.BusinessName,
		BusinessType:            application.BusinessType,
		BusinessDescription:     application.Description,
		Website:                 application.Website,
		ContactPhone:            application.Phone,
		AddressLine:             application.AddressLine,
		City:                    application.City,
		State:                   application.State,
		Country:                 application.Country,
		PostalCode:              application.PostalCode,
		BankName:                application.BankName,
		AccountHolderName:       application.AccountHolderName,
		IFSCCode:                application.IFSCCode,
		GSTNumber:               application.GSTNumber,
		PANNumber:               application.PANNumber,
		LogoURL:                 logoURL,
		VerificationDocumentURL: documentURL,
		Status:                  string(application.Status),
		RejectionReason:         application.RejectionReason,
		CreatedAt:               application.CreatedAt,
		UpdatedAt:               application.UpdatedAt,
	}, nil
}

func (u *AdminApplicationUsecase) RejectApplication(
	ctx context.Context,
	id uuid.UUID,
	reason string,
	adminID uuid.UUID,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	reason = strings.TrimSpace(reason)

	if reason == "" {
		u.logger.Warn(
			"admin_organizer_application_rejection_failed",
			"application_id", id,
			"admin_id", adminID,
			"reason", "empty_rejection_reason",
		)
		return appErrors.NewValidationError(
			"rejection reason is required",
		)
	}

	application, err := u.repository.FindByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerApplicationNotFound) {
			u.logger.Warn(
				"admin_organizer_application_rejection_failed",
				"application_id", id,
				"admin_id", adminID,
				"reason", "application_not_found",
			)
			return appErrors.NewNotFoundError(
				"organizer application not found",
			)
		}

		u.logger.Error(
			"admin_organizer_application_rejection_find_failed",
			"application_id", id,
			"admin_id", adminID,
			"error", err,
		)

		return fmt.Errorf(
			"failed to retrieve organizer application",
		)
	}

	if application.Status != domain.ApplicationPending {
		u.logger.Warn(
			"admin_organizer_application_rejection_failed",
			"application_id", id,
			"admin_id", adminID,
			"reason", "application_not_pending",
			"status", application.Status,
		)
		return appErrors.NewConflictError(
			"only pending applications can be rejected",
		)
	}

	// Update the application instead of deleting it.
	application.Status = domain.ApplicationRejected
	application.RejectionReason = reason

	if err := u.repository.Update(application); err != nil {
		u.logger.Error(
			"admin_organizer_application_rejection_update_failed",
			"application_id", id,
			"admin_id", adminID,
			"error", err,
		)

		return fmt.Errorf(
			"failed to reject organizer application",
		)
	}

	u.logger.Info(
		"admin_organizer_application_rejected",
		"application_id", id,
		"user_id", application.UserID,
		"admin_id", adminID,
	)

	return nil
}
