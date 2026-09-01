package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizerApplicationRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewOrganizerApplicationRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *OrganizerApplicationRepository {
	return &OrganizerApplicationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *OrganizerApplicationRepository) Create(
	application *domain.OrganizerApplication,
) error {
	model := &models.OrganizerApplicationModel{
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
		AccountNumberEncrypted:  application.AccountNumberEncrypted,
		IFSCCode:                application.IFSCCode,
		LogoURL:                 application.LogoURL,
		VerificationDocumentURL: application.VerificationDocumentURL,
		Status:                  application.Status,
		RejectionReason:         application.RejectionReason,
		CreatedAt:               application.CreatedAt,
		UpdatedAt:               application.UpdatedAt,
	}

	if err := r.db.Create(model).Error; err != nil {
		r.logger.Error(
			"organizer_application_create_failed",
			"user_id", application.UserID,
			"error", err,
		)

		return err
	}

	r.logger.Info(
		"organizer_application_created",
		"application_id", application.ID,
		"user_id", application.UserID,
	)

	return nil
}

func (r *OrganizerApplicationRepository) FindByUserID(
	userID uuid.UUID,
) (*domain.OrganizerApplication, error) {
	var model models.OrganizerApplicationModel

	err := r.db.
		Where("user_id = ?", userID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}

		r.logger.Error(
			"organizer_application_find_failed",
			"user_id", userID,
			"error", err,
		)

		return nil, err
	}

	return toOrganizerApplicationDomain(&model), nil
}

func (r *OrganizerApplicationRepository) Update(
	application *domain.OrganizerApplication,
) error {
	model := &models.OrganizerApplicationModel{
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
		AccountNumberEncrypted:  application.AccountNumberEncrypted,
		IFSCCode:                application.IFSCCode,
		LogoURL:                 application.LogoURL,
		VerificationDocumentURL: application.VerificationDocumentURL,
		Status:                  application.Status,
		RejectionReason:         application.RejectionReason,
		CreatedAt:               application.CreatedAt,
		UpdatedAt:               application.UpdatedAt,
	}

	result := r.db.
		Model(&models.OrganizerApplicationModel{}).
		Where("id = ?", application.ID).
		Updates(model)

	if result.Error != nil {
		r.logger.Error(
			"organizer_application_update_failed",
			"application_id", application.ID,
			"user_id", application.UserID,
			"error", result.Error,
		)

		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn(
			"organizer_application_not_found_for_update",
			"application_id", application.ID,
			"user_id", application.UserID,
		)

		return gorm.ErrRecordNotFound
	}

	r.logger.Info(
		"organizer_application_updated",
		"application_id", application.ID,
		"user_id", application.UserID,
	)

	return nil
}

func toOrganizerApplicationDomain(
	model *models.OrganizerApplicationModel,
) *domain.OrganizerApplication {
	return &domain.OrganizerApplication{
		ID:                      model.ID,
		UserID:                  model.UserID,
		BusinessName:            model.BusinessName,
		BusinessType:            model.BusinessType,
		Description:             model.Description,
		Phone:                   model.Phone,
		Website:                 model.Website,
		GSTNumber:               model.GSTNumber,
		PANNumber:               model.PANNumber,
		BankName:                model.BankName,
		AccountHolderName:       model.AccountHolderName,
		AccountNumberEncrypted:  model.AccountNumberEncrypted,
		IFSCCode:                model.IFSCCode,
		LogoURL:                 model.LogoURL,
		VerificationDocumentURL: model.VerificationDocumentURL,
		Status:                  model.Status,
		RejectionReason:         model.RejectionReason,
		CreatedAt:               model.CreatedAt,
		UpdatedAt:               model.UpdatedAt,
	}
}
