package organizer

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/service/encryption"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApplicationUsecase struct {
	repository        domain.OrganizerApplicationRepository
	encryptionService *encryption.Service
	logger            *slog.Logger
}

func NewApplicationUsecase(
	repository domain.OrganizerApplicationRepository,
	encryptionService *encryption.Service,
	logger *slog.Logger,
) *ApplicationUsecase {
	return &ApplicationUsecase{
		repository:        repository,
		encryptionService: encryptionService,
		logger:            logger,
	}
}

func (u *ApplicationUsecase) SubmitApplication(
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

	existing, err := u.repository.FindByUserID(userID)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
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
	if errors.Is(err, gorm.ErrRecordNotFound) {

		application := u.buildApplication(
			userID,
			input,
			encryptedAccountNumber,
		)

		if err := u.repository.Create(application); err != nil {
			u.logger.Error(
				"organizer_application_create_failed",
				"user_id", userID,
				"error", err,
			)

			return fmt.Errorf("failed to create organizer application: %w", err)
		}

		u.logger.Info(
			"organizer_application_created",
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

		encryptedAccountNumber, err := u.encryptionService.Encrypt(
			input.AccountNumber,
		)
		if err != nil {
			u.logger.Error(
				"organizer_application_account_encryption_failed",
				"user_id", userID,
				"application_id", existing.ID,
				"error", err,
			)

			return fmt.Errorf("failed to secure account information")
		}

		updateApplication(existing, input, encryptedAccountNumber)

		existing.Status = domain.ApplicationPending
		existing.RejectionReason = ""

		if err := u.repository.Update(existing); err != nil {
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

	application.IFSCCode = strings.ToUpper(
		strings.TrimSpace(input.IFSCCode),
	)

	application.LogoURL = strings.TrimSpace(
		input.LogoURL,
	)

	application.VerificationDocumentURL = strings.TrimSpace(
		input.VerificationDocumentURL,
	)

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
