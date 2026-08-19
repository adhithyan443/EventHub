package repository

import (
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *domain.User) error {
	model := userToModel(user)

	if err := r.db.Create(&model).Error; err != nil {
		return err
	}

	*user = modelToUser(&model)

	return nil
}

func (r *UserRepository) FindByID(id uuid.UUID) (*domain.User, error) {
	var model models.UserModel

	if err := r.db.First(&model, "id = ?", id).Error; err != nil {
		return nil, err
	}

	user := modelToUser(&model)

	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var model models.UserModel

	if err := r.db.Where("email = ?", email).First(&model).Error; err != nil {
		return nil, err
	}

	user := modelToUser(&model)

	return &user, nil
}

func (r *UserRepository) Update(user *domain.User) error {
	model := userToModel(user)

	if err := r.db.Save(&model).Error; err != nil {
		return err
	}

	*user = modelToUser(&model)

	return nil
}

func userToModel(user *domain.User) models.UserModel {
	return models.UserModel{
		ID:           user.ID,
		FullName:     user.FullName,
		Email:        user.Email,
		Phone:        user.Phone,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
		Status:       user.Status,
		ProfileImage: user.ProfileImage,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func modelToUser(model *models.UserModel) domain.User {
	return domain.User{
		ID:           model.ID,
		FullName:     model.FullName,
		Email:        model.Email,
		Phone:        model.Phone,
		PasswordHash: model.PasswordHash,
		Role:         model.Role,
		Status:        model.Status,
		ProfileImage: model.ProfileImage,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}