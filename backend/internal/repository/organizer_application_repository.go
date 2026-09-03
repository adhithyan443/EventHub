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

	model := organizerApplicationToModel(application)

	if err := r.db.Create(&model).Error; err != nil {
		r.logger.Error(
			"organizer_application_create_failed",
			"user_id", application.UserID,
			"error", err,
		)

		return err
	}

	*application = *modelToOrganizerApplication(&model)

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

	return modelToOrganizerApplication(&model), nil
}

func (r *OrganizerApplicationRepository) Update(
	application *domain.OrganizerApplication,
) error {

	model := organizerApplicationToModel(application)

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

	// Reload the updated record.
	var updatedModel models.OrganizerApplicationModel

	if err := r.db.
		Where("id = ?", application.ID).
		First(&updatedModel).Error; err != nil {
		r.logger.Error(
			"organizer_application_reload_failed",
			"application_id", application.ID,
			"error", err,
		)

		return err
	}

	*application = *modelToOrganizerApplication(&updatedModel)

	r.logger.Info(
		"organizer_application_updated",
		"application_id", application.ID,
		"user_id", application.UserID,
	)

	return nil
}

func organizerApplicationToModel(
	application *domain.OrganizerApplication,
) models.OrganizerApplicationModel {

	return models.OrganizerApplicationModel{
		ID:                      application.ID,
		UserID:                  application.UserID,
		BusinessName:            application.BusinessName,
		BusinessType:            application.BusinessType,
		Description:             application.Description,
		Phone:                   application.Phone,
		Website:                 application.Website,
		BankName:                application.BankName,
		AccountHolderName:       application.AccountHolderName,
		AccountNumberEncrypted:  application.AccountNumberEncrypted,
		IFSCCode:                application.IFSCCode,
		GSTNumber:               application.GSTNumber,
		PANNumber:               application.PANNumber,
		LogoURL:                 application.LogoURL,
		VerificationDocumentURL: application.VerificationDocumentURL,
		Status:                  string(application.Status),
		RejectionReason:         application.RejectionReason,
		CreatedAt:               application.CreatedAt,
		UpdatedAt:               application.UpdatedAt,
	}
}

func modelToOrganizerApplication(
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
		BankName:                model.BankName,
		AccountHolderName:       model.AccountHolderName,
		AccountNumberEncrypted:  model.AccountNumberEncrypted,
		IFSCCode:                model.IFSCCode,
		GSTNumber:               model.GSTNumber,
		PANNumber:               model.PANNumber,
		LogoURL:                 model.LogoURL,
		VerificationDocumentURL: model.VerificationDocumentURL,
		Status:                  domain.ApplicationStatus(model.Status),
		RejectionReason:         model.RejectionReason,
		CreatedAt:               model.CreatedAt,
		UpdatedAt:               model.UpdatedAt,
	}
}

func (r *OrganizerApplicationRepository) List(
	page int,
	limit int,
	status domain.ApplicationStatus,
) (*domain.OrganizerApplicationList, error) {

	var modelsList []models.OrganizerApplicationModel
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(
		&models.OrganizerApplicationModel{},
	)

	// Apply status filter only when a status was provided.
	if status != "" {
		query = query.Where(
			"status = ?",
			string(status),
		)
	}

	// Get total number of matching applications.
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error(
			"organizer_application_count_failed",
			"status", status,
			"error", err,
		)

		return nil, err
	}

	// Fetch newest applications first.
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&modelsList).Error; err != nil {

		r.logger.Error(
			"organizer_application_list_failed",
			"page", page,
			"limit", limit,
			"status", status,
			"error", err,
		)

		return nil, err
	}

	applications := make(
		[]*domain.OrganizerApplication,
		0,
		len(modelsList),
	)

	for i := range modelsList {
		applications = append(
			applications,
			modelToOrganizerApplication(&modelsList[i]),
		)
	}

	r.logger.Info(
		"organizer_applications_listed",
		"page", page,
		"limit", limit,
		"status", status,
		"count", len(applications),
		"total", total,
	)

	return &domain.OrganizerApplicationList{
		Applications: applications,
		Total:        total,
		Page:         page,
		Limit:        limit,
	}, nil
}



func (r *OrganizerApplicationRepository) FindByID(
	id uuid.UUID,
) (*domain.OrganizerApplication, error) {

	var model models.OrganizerApplicationModel

	// Query the database using the application primary key.
	if err := r.db.First(&model, "id = ?", id).Error; err != nil {
		r.logger.Error(
			"organizer_application_find_by_id_failed",
			"application_id", id,
			"error", err,
		)

		return nil, err
	}

	// Convert the persistence model into the domain entity.
	application := &domain.OrganizerApplication{
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
		Status:                  domain.ApplicationStatus(model.Status),
		RejectionReason:         model.RejectionReason,
		CreatedAt:               model.CreatedAt,
		UpdatedAt:               model.UpdatedAt,
	}

	r.logger.Info(
		"organizer_application_found_by_id",
		"application_id", id,
	)

	return application, nil
}