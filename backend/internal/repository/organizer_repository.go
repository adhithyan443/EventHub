package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizerRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewOrganizerRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *OrganizerRepository {
	return &OrganizerRepository{
		db:     db,
		logger: logger,
	}
}

func (r *OrganizerRepository) Create(
	organizer *domain.Organizer,
) error {
	model := organizerToModel(organizer)

	if err := r.db.Create(&model).Error; err != nil {
		r.logger.Error(
			"organizer_create_failed",
			"user_id", organizer.UserID,
			"error", err,
		)

		return err
	}

	*organizer = *modelToOrganizer(&model)

	r.logger.Info(
		"organizer_created",
		"organizer_id", organizer.ID,
		"user_id", organizer.UserID,
	)

	return nil
}

func (r *OrganizerRepository) FindByID(
	id uuid.UUID,
) (*domain.Organizer, error) {
	var model models.OrganizerModel

	err := r.db.
		Where("id = ?", id).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrganizerNotFound
		}

		r.logger.Error(
			"organizer_find_by_id_failed",
			"organizer_id", id,
			"error", err,
		)

		return nil, err
	}

	return modelToOrganizer(&model), nil
}

func (r *OrganizerRepository) FindByUserID(
	userID uuid.UUID,
) (*domain.Organizer, error) {
	var model models.OrganizerModel

	err := r.db.
		Where("user_id = ?", userID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrganizerNotFound
		}

		r.logger.Error(
			"organizer_find_by_user_id_failed",
			"user_id", userID,
			"error", err,
		)

		return nil, err
	}

	return modelToOrganizer(&model), nil
}

func organizerToModel(
	organizer *domain.Organizer,
) models.OrganizerModel {
	return models.OrganizerModel{
		ID:        organizer.ID,
		UserID:    organizer.UserID,
		Status:    organizer.Status,
		CreatedAt: organizer.CreatedAt,
		UpdatedAt: organizer.UpdatedAt,
	}
}

func modelToOrganizer(
	model *models.OrganizerModel,
) *domain.Organizer {
	return &domain.Organizer{
		ID:        model.ID,
		UserID:    model.UserID,
		Status:    model.Status,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
