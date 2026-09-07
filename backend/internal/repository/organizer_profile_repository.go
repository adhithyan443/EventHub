package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizerProfileRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewOrganizerProfileRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *OrganizerProfileRepository {
	return &OrganizerProfileRepository{
		db:     db,
		logger: logger,
	}
}

func (r *OrganizerProfileRepository) Create(
	profile *domain.OrganizerProfile,
) error {
	model := organizerProfileToModel(profile)

	if err := r.db.Create(&model).Error; err != nil {
		r.logger.Error(
			"organizer_profile_create_failed",
			"organizer_id", profile.OrganizerID,
			"error", err,
		)
		return err
	}

	*profile = *modelToOrganizerProfile(&model)

	r.logger.Info(
		"organizer_profile_created",
		"organizer_profile_id", profile.ID,
		"organizer_id", profile.OrganizerID,
	)

	return nil
}

func (r *OrganizerProfileRepository) FindByOrganizerID(
	organizerID uuid.UUID,
) (*domain.OrganizerProfile, error) {
	var model models.OrganizerProfileModel

	err := r.db.
		Where("organizer_id = ?", organizerID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrganizerProfileNotFound
		}

		r.logger.Error(
			"organizer_profile_find_failed",
			"organizer_id", organizerID,
			"error", err,
		)

		return nil, err
	}

	return modelToOrganizerProfile(&model), nil
}

func organizerProfileToModel(
	profile *domain.OrganizerProfile,
) models.OrganizerProfileModel {
	return models.OrganizerProfileModel{
		ID:           profile.ID,
		OrganizerID:  profile.OrganizerID,
		BusinessName: profile.BusinessName,
		BusinessType: profile.BusinessType,
		Description:  profile.Description,
		Phone:        profile.Phone,
		Email:        profile.Email,
		Website:      profile.Website,
		LogoURL:      profile.LogoURL,
		GSTNumber:    profile.GSTNumber,
		PANNumber:    profile.PANNumber,
		CreatedAt:    profile.CreatedAt,
		UpdatedAt:    profile.UpdatedAt,
	}
}

func modelToOrganizerProfile(
	model *models.OrganizerProfileModel,
) *domain.OrganizerProfile {
	return &domain.OrganizerProfile{
		ID:           model.ID,
		OrganizerID:  model.OrganizerID,
		BusinessName: model.BusinessName,
		BusinessType: model.BusinessType,
		Description:  model.Description,
		Phone:        model.Phone,
		Email:        model.Email,
		Website:      model.Website,
		LogoURL:      model.LogoURL,
		GSTNumber:    model.GSTNumber,
		PANNumber:    model.PANNumber,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
