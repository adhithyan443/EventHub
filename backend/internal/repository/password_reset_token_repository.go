package repository

import (
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PasswordResetTokenRepository struct {
	db *gorm.DB
}

func NewPasswordResetTokenRepository(db *gorm.DB) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{db: db}
}

func (r *PasswordResetTokenRepository) Create(token *domain.PasswordResetToken) error {
	model := passwordResetTokenToModel(token)

	if err := r.db.Create(&model).Error; err != nil {
		return err
	}

	*token = modelToPasswordResetToken(&model)

	return nil
}

func (r *PasswordResetTokenRepository) FindByTokenHash(hash string) (*domain.PasswordResetToken, error) {
	var model models.PasswordResetTokenModel

	if err := r.db.
		Where("token_hash = ?", hash).
		First(&model).Error; err != nil {
		return nil, err
	}

	token := modelToPasswordResetToken(&model)

	return &token, nil
}

func (r *PasswordResetTokenRepository) MarkUsed(id uuid.UUID) error {
	return r.db.
		Model(&models.PasswordResetTokenModel{}).
		Where("id = ?", id).
		Update("used_at", gorm.Expr("NOW()")).
		Error
}

func passwordResetTokenToModel(
	token *domain.PasswordResetToken,
) models.PasswordResetTokenModel {
	return models.PasswordResetTokenModel{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		UsedAt:    token.UsedAt,
		CreatedAt: token.CreatedAt,
	}
}

func modelToPasswordResetToken(
	model *models.PasswordResetTokenModel,
) domain.PasswordResetToken {
	return domain.PasswordResetToken{
		ID:        model.ID,
		UserID:    model.UserID,
		TokenHash: model.TokenHash,
		ExpiresAt: model.ExpiresAt,
		UsedAt:    model.UsedAt,
		CreatedAt: model.CreatedAt,
	}
}
