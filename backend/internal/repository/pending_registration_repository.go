package repository

import (
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PendingRegistrationRepository struct {
	db *gorm.DB
}

func NewPendingRegistrationRepository(db *gorm.DB) *PendingRegistrationRepository {
	return &PendingRegistrationRepository{
		db: db,
	}
}

func (r *PendingRegistrationRepository) Create(registration *domain.PendingRegistration) error {

	model := pendingRegistrationToModel(registration)

	return r.db.Create(&model).Error
}

func (r *PendingRegistrationRepository) FindByEmail(email string) (*domain.PendingRegistration, error) {

	var model models.PendingRegistrationModel

	if err := r.db.Where("email = ?", email).First(&model).Error; err != nil {
		return nil, err
	}

	registration := modelToPendingRegistration(&model)
	return &registration, nil

}

func (r *PendingRegistrationRepository) Update(registration *domain.PendingRegistration) error {

	model := pendingRegistrationToModel(registration)

	return r.db.
		Model(&models.PendingRegistrationModel{}).
		Where("id = ?", registration.ID).
		Updates(&model).Error
}

func (r *PendingRegistrationRepository) Delete(id uuid.UUID) error {
	return r.db.
		Delete(&models.PendingRegistrationModel{}, "id = ?", id).
		Error
}

func pendingRegistrationToModel(registration *domain.PendingRegistration) models.PendingRegistrationModel {
	return models.PendingRegistrationModel{
		ID:           registration.ID,
		FullName:     registration.FullName,
		Email:        registration.Email,
		Phone:        registration.Phone,
		PasswordHash: registration.PasswordHash,
		OTPHash:      registration.OTPHash,
		OTPExpiresAt: registration.OTPExpiresAt,
		OTPAttempts:  registration.OTPAttempts,
		CreatedAt:    registration.CreatedAt,
		UpdatedAt:    registration.UpdatedAt,
	}
}

func modelToPendingRegistration(model *models.PendingRegistrationModel) domain.PendingRegistration {
	return domain.PendingRegistration{
		ID:           model.ID,
		FullName:     model.FullName,
		Email:        model.Email,
		Phone:        model.Phone,
		PasswordHash: model.PasswordHash,
		OTPHash:      model.OTPHash,
		OTPExpiresAt: model.OTPExpiresAt,
		OTPAttempts:  model.OTPAttempts,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
