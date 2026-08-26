package repository

import (
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db: db,
	}
}

func (r *RefreshTokenRepository) Create(token *domain.RefreshToken) error {
	model := refreshTokenToModel(token)

	return r.db.Create(&model).Error
}

func (r *RefreshTokenRepository) FindByTokenHash(hash string) (*domain.RefreshToken, error) {
	var model models.RefreshTokenModel

	if err := r.db.
		Where("token_hash = ?", hash).
		First(&model).Error; err != nil {
		return nil, err
	}

	token := modelToRefreshToken(&model)

	return &token, nil
}

func (r *RefreshTokenRepository) Revoke(id uuid.UUID) error {
	return r.db.
		Model(&models.RefreshTokenModel{}).
		Where("id = ?", id).
		Update("revoked_at", gorm.Expr("NOW()")).
		Error
}

func refreshTokenToModel(token *domain.RefreshToken) models.RefreshTokenModel {
	return models.RefreshTokenModel{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		RevokedAt: token.RevokedAt,
		CreatedAt: token.CreatedAt,
	}
}

func modelToRefreshToken(model *models.RefreshTokenModel) domain.RefreshToken {
	return domain.RefreshToken{
		ID:        model.ID,
		UserID:    model.UserID,
		TokenHash: model.TokenHash,
		ExpiresAt: model.ExpiresAt,
		RevokedAt: model.RevokedAt,
		CreatedAt: model.CreatedAt,
	}
}
