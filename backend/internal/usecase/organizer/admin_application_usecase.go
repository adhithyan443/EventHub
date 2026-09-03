package organizer

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminApplicationUsecase struct {
	repository         domain.OrganizerApplicationRepository
	transactionManager domain.TransactionManager
	logger             *slog.Logger
}

func NewAdminApplicationUsecase(
	repository domain.OrganizerApplicationRepository,
	transactionManager domain.TransactionManager,
	logger *slog.Logger,
) *AdminApplicationUsecase {
	return &AdminApplicationUsecase{
		repository:         repository,
		transactionManager: transactionManager,
		logger:             logger,
	}
}

func (u *AdminApplicationUsecase) ListApplications(
	page int,
	limit int,
	status string,
) (*domain.OrganizerApplicationList, error) {

	if page < 1 {
		return nil, fmt.Errorf("page must be greater than zero")
	}

	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	applicationStatus := domain.ApplicationStatus(
		strings.ToUpper(strings.TrimSpace(status)),
	)

	// Validate status when supplied.
	if applicationStatus != "" {
		switch applicationStatus {
		case domain.ApplicationPending,
			domain.ApplicationApproved,
			domain.ApplicationRejected:

		default:
			return nil, fmt.Errorf("invalid application status")
		}
	}

	applications, err := u.repository.List(
		page,
		limit,
		applicationStatus,
	)
	if err != nil {
		u.logger.Error(
			"admin_organizer_application_list_failed",
			"page", page,
			"limit", limit,
			"status", applicationStatus,
			"error", err,
		)

		return nil, fmt.Errorf(
			"failed to retrieve organizer applications",
		)
	}

	u.logger.Info(
		"admin_organizer_applications_listed",
		"page", page,
		"limit", limit,
		"status", applicationStatus,
		"total", applications.Total,
	)

	return applications, nil
}

// Keep this helper available for future admin operations.
// func applicationID(value string) (uuid.UUID, error) {
// 	id, err := uuid.Parse(value)
// 	if err != nil {
// 		return uuid.Nil, fmt.Errorf("invalid application id")
// 	}

// 	return id, nil
// }

// GetApplication retrieves a single organizer application for admin review.
func (u *AdminApplicationUsecase) GetApplication(
	id uuid.UUID,
) (*domain.OrganizerApplication, error) {

	application, err := u.repository.FindByID(id)
	if err != nil {
		u.logger.Error(
			"admin_organizer_application_get_failed",
			"application_id", id,
			"error", err,
		)

		return nil, fmt.Errorf(
			"failed to retrieve organizer application",
		)
	}

	u.logger.Info(
		"admin_organizer_application_retrieved",
		"application_id", id,
	)

	return application, nil
}

func (u *AdminApplicationUsecase) ApproveApplication(id uuid.UUID) error {
	return u.transactionManager.WithinTransaction(func(tx domain.TransactionRepositories) error {

		application, err := tx.OrganizerApplicationRepository().FindByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return appErrors.NewNotFoundError("organizer application not found")
			}

			u.logger.Error(
				"admin_organizer_application_approval_find_failed",
				"application_id", id,
				"error", err,
			)
			return fmt.Errorf("failed to retrieve organizer application")
		}

		// Only pending applications can be approved.
		if application.Status != domain.ApplicationPending {
			return appErrors.NewConflictError(
				"only pending applications can be approved",
			)
		}

		// Fetch the applicant
		user, err := tx.UserRepository().FindByID(application.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return appErrors.NewNotFoundError("application user not found")
			}

			u.logger.Error(
				"admin_organizer_application_user_find_failed",
				"application_id", id,
				"user_id", application.UserID,
				"error", err,
			)
			return fmt.Errorf("failed to retrieve application user")
		}

		// Create the organizer account as ACTIVE immediately after approval.
		organizer := &domain.Organizer{
			ID:     uuid.New(),
			UserID: application.UserID,
			Status: "ACTIVE",
		}

		if err := tx.OrganizerRepository().Create(organizer); err != nil {
			u.logger.Error(
				"admin_organizer_creation_failed",
				"application_id", id,
				"user_id", application.UserID,
				"error", err,
			)
			return fmt.Errorf("failed to create organizer")
		}

		// Move business information from the application to organizer
		profile := &domain.OrganizerProfile{
			ID:           uuid.New(),
			OrganizerID:  organizer.ID,
			BusinessName: application.BusinessName,
			BusinessType: application.BusinessType,
			Description:  application.Description,
			Phone:        application.Phone,
			Email:        user.Email,
			Website:      application.Website,
			LogoURL:      application.LogoURL,
			GSTNumber:    application.GSTNumber,
			PANNumber:    application.PANNumber,
		}

		if err := tx.OrganizerProfileRepository().Create(profile); err != nil {
			u.logger.Error(
				"admin_organizer_profile_creation_failed",
				"application_id", id,
				"organizer_id", organizer.ID,
				"error", err,
			)
			return fmt.Errorf("failed to create organizer profile")
		}

		// Move the application address into the permanent organizer address.
		address := &domain.OrganizerAddress{
			ID:          uuid.New(),
			OrganizerID: organizer.ID,
			AddressLine: application.AddressLine,
			City:        application.City,
			State:       application.State,
			Country:     application.Country,
			PostalCode:  application.PostalCode,
		}

		if err := tx.OrganizerAddressRepository().Create(address); err != nil {
			u.logger.Error(
				"admin_organizer_address_creation_failed",
				"application_id", id,
				"organizer_id", organizer.ID,
				"error", err,
			)
			return fmt.Errorf("failed to create organizer address")
		}

		// Move the encrypted bank account information into the permanent
		// organizer bank account record.
		bankAccount := &domain.OrganizerBankAccount{
			ID:                     uuid.New(),
			OrganizerID:            organizer.ID,
			BankName:               application.BankName,
			AccountHolderName:      application.AccountHolderName,
			AccountNumberEncrypted: application.AccountNumberEncrypted,
			IFSCCode:               application.IFSCCode,
		}

		if err := tx.OrganizerBankAccountRepository().Create(bankAccount); err != nil {
			u.logger.Error(
				"admin_organizer_bank_account_creation_failed",
				"application_id", id,
				"organizer_id", organizer.ID,
				"error", err,
			)
			return fmt.Errorf("failed to create organizer bank account")
		}

		// Promote the applicant to the ORGANIZER role.
		user.Role = "ORGANIZER"

		if err := tx.UserRepository().Update(user); err != nil {
			u.logger.Error(
				"admin_organizer_application_user_role_update_failed",
				"application_id", id,
				"user_id", application.UserID,
				"error", err,
			)
			return fmt.Errorf("failed to update user role")
		}

		// The application is no longer needed because its data has been
		// transferred to the permanent organizer records.
		if err := tx.OrganizerApplicationRepository().Delete(id); err != nil {
			u.logger.Error(
				"admin_organizer_application_delete_failed",
				"application_id", id,
				"organizer_id", organizer.ID,
				"error", err,
			)
			return fmt.Errorf("failed to remove organizer application")
		}

		u.logger.Info(
			"admin_organizer_application_approved",
			"application_id", id,
			"user_id", application.UserID,
			"organizer_id", organizer.ID,
		)

		return nil
	})
}
