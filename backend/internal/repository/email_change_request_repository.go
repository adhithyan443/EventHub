package repository

import (
	"errors"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EmailChangeRequestRepository struct {
	db *gorm.DB
}

func NewEmailChangeRequestRepository(db *gorm.DB) *EmailChangeRequestRepository {
	return &EmailChangeRequestRepository{db: db}
}

func (r *EmailChangeRequestRepository) Create(req *domain.EmailChangeRequest) error {
	model := emailChangeRequestToModel(req)
	if err := r.db.Create(&model).Error; err != nil {
		return err
	}
	*req = modelToEmailChangeRequest(&model)
	return nil
}

func (r *EmailChangeRequestRepository) Upsert(req *domain.EmailChangeRequest) error {
	model := emailChangeRequestToModel(req)
	err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"new_email",
			"otp_hash",
			"otp_expires_at",
			"otp_attempts",
			"resend_available_at",
			"updated_at",
		}),
	}).Create(&model).Error
	if err != nil {
		return err
	}
	*req = modelToEmailChangeRequest(&model)
	return nil
}

func (r *EmailChangeRequestRepository) FindByID(id uuid.UUID) (*domain.EmailChangeRequest, error) {
	var model models.EmailChangeRequestModel
	if err := r.db.First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEmailChangeRequestNotFound
		}
		return nil, err
	}
	req := modelToEmailChangeRequest(&model)
	return &req, nil
}

func (r *EmailChangeRequestRepository) FindByUserID(userID uuid.UUID) (*domain.EmailChangeRequest, error) {
	var model models.EmailChangeRequestModel
	if err := r.db.First(&model, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEmailChangeRequestNotFound
		}
		return nil, err
	}
	req := modelToEmailChangeRequest(&model)
	return &req, nil
}

func (r *EmailChangeRequestRepository) FindByNewEmail(email string) (*domain.EmailChangeRequest, error) {
	var model models.EmailChangeRequestModel
	if err := r.db.First(&model, "new_email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEmailChangeRequestNotFound
		}
		return nil, err
	}
	req := modelToEmailChangeRequest(&model)
	return &req, nil
}

func (r *EmailChangeRequestRepository) Update(req *domain.EmailChangeRequest) error {
	model := emailChangeRequestToModel(req)
	if err := r.db.Save(&model).Error; err != nil {
		return err
	}
	*req = modelToEmailChangeRequest(&model)
	return nil
}

func (r *EmailChangeRequestRepository) IncrementAttempts(id uuid.UUID) (int, error) {
	var updatedAttempts int
	row := r.db.Raw(
		`UPDATE email_change_requests 
		 SET otp_attempts = otp_attempts + 1, updated_at = ? 
		 WHERE id = ? AND otp_attempts < 5 
		 RETURNING otp_attempts`,
		time.Now(), id,
	).Row()

	if err := row.Scan(&updatedAttempts); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 5, nil
		}
		// If query returned no rows (because otp_attempts was already >= 5)
		return 5, nil
	}

	return updatedAttempts, nil
}

func (r *EmailChangeRequestRepository) UpdateResendCooldown(id uuid.UUID, userID uuid.UUID, newCooldown time.Time) error {
	res := r.db.Model(&models.EmailChangeRequestModel{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"resend_available_at": newCooldown,
			"updated_at":          time.Now(),
		})
	return res.Error
}

func (r *EmailChangeRequestRepository) RollbackResendCooldown(id uuid.UUID, userID uuid.UUID, targetCooldown time.Time) error {
	res := r.db.Model(&models.EmailChangeRequestModel{}).
		Where("id = ? AND user_id = ? AND resend_available_at = ?", id, userID, targetCooldown).
		Updates(map[string]interface{}{
			"resend_available_at": time.Now(),
			"updated_at":          time.Now(),
		})
	return res.Error
}

func (r *EmailChangeRequestRepository) UpdateOTPAndCooldown(
	id uuid.UUID,
	oldHash string,
	newHash string,
	newExpiry time.Time,
	newCooldown time.Time,
) (bool, error) {
	res := r.db.Model(&models.EmailChangeRequestModel{}).
		Where("id = ? AND otp_hash = ?", id, oldHash).
		Updates(map[string]interface{}{
			"otp_hash":            newHash,
			"otp_expires_at":      newExpiry,
			"otp_attempts":        0,
			"resend_available_at": newCooldown,
			"updated_at":          time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *EmailChangeRequestRepository) DeleteByIDAndUserID(id uuid.UUID, userID uuid.UUID) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.EmailChangeRequestModel{}).Error
}

func (r *EmailChangeRequestRepository) DeleteByUserID(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).
		Delete(&models.EmailChangeRequestModel{}).Error
}

func (r *EmailChangeRequestRepository) DeleteByID(id uuid.UUID) error {
	return r.db.Where("id = ?", id).
		Delete(&models.EmailChangeRequestModel{}).Error
}

func emailChangeRequestToModel(req *domain.EmailChangeRequest) models.EmailChangeRequestModel {
	return models.EmailChangeRequestModel{
		ID:                req.ID,
		UserID:            req.UserID,
		NewEmail:          req.NewEmail,
		OTPHash:           req.OTPHash,
		OTPExpiresAt:      req.OTPExpiresAt,
		OTPAttempts:       int64(req.OTPAttempts),
		ResendAvailableAt: req.ResendAvailableAt,
		CreatedAt:         req.CreatedAt,
		UpdatedAt:         req.UpdatedAt,
	}
}

func modelToEmailChangeRequest(model *models.EmailChangeRequestModel) domain.EmailChangeRequest {
	return domain.EmailChangeRequest{
		ID:                model.ID,
		UserID:            model.UserID,
		NewEmail:          model.NewEmail,
		OTPHash:           model.OTPHash,
		OTPExpiresAt:      model.OTPExpiresAt,
		OTPAttempts:       int(model.OTPAttempts),
		ResendAvailableAt: model.ResendAvailableAt,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}
