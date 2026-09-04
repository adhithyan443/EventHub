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
		ID:           application.ID,
		UserID:       application.UserID,
		BusinessName: application.BusinessName,
		BusinessType: application.BusinessType,
		Description:  application.Description,
		Phone:        application.Phone,
		Website:      application.Website,

		// Address fields copied from the domain to the database model.
		AddressLine: application.AddressLine,
		City:        application.City,
		State:       application.State,
		Country:     application.Country,
		PostalCode:  application.PostalCode,

		BankName:               application.BankName,
		AccountHolderName:      application.AccountHolderName,
		AccountNumberEncrypted: application.AccountNumberEncrypted,
		IFSCCode:               application.IFSCCode,

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
		ID:           model.ID,
		UserID:       model.UserID,
		BusinessName: model.BusinessName,
		BusinessType: model.BusinessType,
		Description:  model.Description,
		Phone:        model.Phone,
		Website:      model.Website,

		// Address fields copied from the database model to the domain.
		AddressLine: model.AddressLine,
		City:        model.City,
		State:       model.State,
		Country:     model.Country,
		PostalCode:  model.PostalCode,

		BankName:               model.BankName,
		AccountHolderName:      model.AccountHolderName,
		AccountNumberEncrypted: model.AccountNumberEncrypted,
		IFSCCode:               model.IFSCCode,

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

// organizerApplicationListRow represents an organizer application
// together with the applicant's name from the users table.
type organizerApplicationListRow struct {
	models.OrganizerApplicationModel

	ApplicantName string `gorm:"column:applicant_name"`
}

func organizerApplicationRowToDomain(
	row *organizerApplicationListRow,
) *domain.OrganizerApplication {
	application := modelToOrganizerApplication(
		&row.OrganizerApplicationModel,
	)

	application.ApplicantName = row.ApplicantName

	return application
}

func (r *OrganizerApplicationRepository) List(
	page int,
	limit int,
	status domain.ApplicationStatus,
) (*domain.OrganizerApplicationList, error) {
	var rows []organizerApplicationListRow
	var total int64

	offset := (page - 1) * limit

	query := r.db.
		Model(&models.OrganizerApplicationModel{}).
		Joins(
			"JOIN users ON users.id = organizer_applications.user_id",
		)

	// Apply status filter only when a status was provided.
	if status != "" {
		query = query.Where(
			"organizer_applications.status = ?",
			string(status),
		)
	}

	// Count matching applications.
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error(
			"organizer_application_count_failed",
			"status", status,
			"error", err,
		)

		return nil, err
	}

	// Fetch applications together with users.full_name.
	if err := query.
		Select(
			"organizer_applications.*, users.full_name AS applicant_name",
		).
		Order("organizer_applications.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {

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
		len(rows),
	)

	for i := range rows {
		applications = append(
			applications,
			organizerApplicationRowToDomain(&rows[i]),
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
	var row organizerApplicationListRow

	result := r.db.
		Model(&models.OrganizerApplicationModel{}).
		Joins(
			"JOIN users ON users.id = organizer_applications.user_id",
		).
		Select(
			"organizer_applications.*, users.full_name AS applicant_name",
		).
		Where(
			"organizer_applications.id = ?",
			id,
		).
		Scan(&row)

	if result.Error != nil {
		r.logger.Error(
			"organizer_application_find_by_id_failed",
			"application_id", id,
			"error", result.Error,
		)

		return nil, result.Error
	}

	if row.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	return organizerApplicationRowToDomain(&row), nil
}

func (r *OrganizerApplicationRepository) Delete(
	id uuid.UUID,
) error {
	err := r.db.Delete(
		&models.OrganizerApplicationModel{},
		"id = ?",
		id,
	).Error

	if err != nil {
		r.logger.Error(
			"organizer_application_delete_failed",
			"application_id", id,
			"error", err,
		)

		return err
	}

	r.logger.Info(
		"organizer_application_deleted",
		"application_id", id,
	)

	return nil
}
