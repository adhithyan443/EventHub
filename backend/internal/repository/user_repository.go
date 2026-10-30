package repository

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewUserRepository(db *gorm.DB, logger *slog.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	user := modelToUser(&model)

	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var model models.UserModel

	if err := r.db.Where("email = ?", email).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	user := modelToUser(&model)

	return &user, nil
}

func (r *UserRepository) Update(user *domain.User) error {
	model := userToModel(user)

	if err := r.db.Save(&model).Error; err != nil {
		r.logger.Error(
			"user_update_failed",
			"user_id", user.ID,
			"error", err,
		)
		return err
	}

	*user = modelToUser(&model)

	r.logger.Info(
		"user_updated",
		"user_id", user.ID,
	)

	return nil
}

func (r *UserRepository) FindByPhone(phone string) (*domain.User, error) {
	var model models.UserModel

	if err := r.db.
		Where("phone = ?", phone).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	user := modelToUser(&model)

	return &user, nil
}

func (r *UserRepository) List(filter domain.UserListFilter) (*domain.UserList, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := r.db.Model(&models.UserModel{})

	if search := strings.TrimSpace(filter.Search); search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"(LOWER(full_name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(phone) LIKE ?)",
			searchPattern,
			searchPattern,
			searchPattern,
		)
	}

	if role := strings.TrimSpace(filter.Role); role != "" {
		query = query.Where("role = ?", strings.ToUpper(role))
	}

	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("status = ?", strings.ToUpper(status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error(
			"user_list_count_failed",
			"error", err,
			"filter", filter,
		)
		return nil, err
	}

	var userModels []models.UserModel
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&userModels).Error; err != nil {
		r.logger.Error(
			"user_list_find_failed",
			"error", err,
			"filter", filter,
		)
		return nil, err
	}

	users := make([]*domain.User, len(userModels))
	for i := range userModels {
		user := modelToUser(&userModels[i])
		users[i] = &user
	}

	// Platform-wide accurate aggregate counts by role
	var roleCounts domain.UserRoleCounts
	type roleCountRow struct {
		Role  string
		Count int64
	}
	var countRows []roleCountRow
	if err := r.db.Model(&models.UserModel{}).Select("role, count(*) as count").Group("role").Scan(&countRows).Error; err != nil {
		r.logger.Warn("user_role_counts_failed", "error", err)
	} else {
		for _, row := range countRows {
			roleCounts.Total += row.Count
			switch strings.ToUpper(row.Role) {
			case "CUSTOMER":
				roleCounts.Customers = row.Count
			case "ORGANIZER":
				roleCounts.Organizers = row.Count
			case "ADMIN":
				roleCounts.Admins = row.Count
			}
		}
	}

	r.logger.Info(
		"users_listed",
		"page", page,
		"limit", limit,
		"total", total,
	)

	return &domain.UserList{
		Users:      users,
		Total:      total,
		Page:       page,
		Limit:      limit,
		RoleCounts: roleCounts,
	}, nil
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
		Status:       model.Status,
		ProfileImage: model.ProfileImage,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
