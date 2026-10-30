package repository

import (
	"errors"
	"log/slog"
	"strings"
	"time"

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

type adminOrganizerListRow struct {
	ID                  uuid.UUID `gorm:"column:id"`
	UserID              uuid.UUID `gorm:"column:user_id"`
	Status              string    `gorm:"column:status"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	UserName            string    `gorm:"column:user_name"`
	UserEmail           string    `gorm:"column:user_email"`
	UserPhone           string    `gorm:"column:user_phone"`
	BusinessName        string    `gorm:"column:business_name"`
	BusinessType        string    `gorm:"column:business_type"`
	BusinessDescription string    `gorm:"column:business_description"`
	BusinessPhone       string    `gorm:"column:business_phone"`
	BusinessEmail       string    `gorm:"column:business_email"`
	Website             string    `gorm:"column:website"`
	LogoURL             string    `gorm:"column:logo_url"`
	GSTNumber           string    `gorm:"column:gst_number"`
	PANNumber           string    `gorm:"column:pan_number"`
}

func (r *OrganizerRepository) List(filter domain.AdminOrganizerListFilter) (*domain.AdminOrganizerList, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := r.db.Table("organizers AS o").
		Joins("INNER JOIN users AS u ON u.id = o.user_id").
		Joins("LEFT JOIN organizer_profiles AS op ON op.organizer_id = o.id")

	if search := strings.TrimSpace(filter.Search); search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"(LOWER(op.business_name) LIKE ? OR LOWER(u.full_name) LIKE ? OR LOWER(u.email) LIKE ?)",
			searchPattern,
			searchPattern,
			searchPattern,
		)
	}

	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("o.status = ?", strings.ToUpper(status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error(
			"admin_organizer_count_failed",
			"error", err,
			"filter", filter,
		)
		return nil, err
	}

	var rows []adminOrganizerListRow
	if err := query.
		Select(`
			o.id,
			o.user_id,
			o.status,
			o.created_at,
			u.full_name AS user_name,
			u.email AS user_email,
			COALESCE(u.phone, '') AS user_phone,
			COALESCE(op.business_name, '') AS business_name,
			COALESCE(op.business_type, '') AS business_type,
			COALESCE(op.description, '') AS business_description,
			COALESCE(op.phone, '') AS business_phone,
			COALESCE(op.email, '') AS business_email,
			COALESCE(op.website, '') AS website,
			COALESCE(op.logo_url, '') AS logo_url,
			COALESCE(op.gst_number, '') AS gst_number,
			COALESCE(op.pan_number, '') AS pan_number
		`).
		Order("o.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		r.logger.Error(
			"admin_organizer_list_failed",
			"error", err,
			"filter", filter,
		)
		return nil, err
	}

	items := make([]*domain.AdminOrganizerItem, len(rows))
	for i, row := range rows {
		items[i] = &domain.AdminOrganizerItem{
			ID:                  row.ID,
			UserID:              row.UserID,
			UserName:            row.UserName,
			UserEmail:           row.UserEmail,
			UserPhone:           row.UserPhone,
			BusinessName:        row.BusinessName,
			BusinessType:        row.BusinessType,
			BusinessDescription: row.BusinessDescription,
			BusinessPhone:       row.BusinessPhone,
			BusinessEmail:       row.BusinessEmail,
			Website:             row.Website,
			LogoURL:             row.LogoURL,
			GSTNumber:           row.GSTNumber,
			PANNumber:           row.PANNumber,
			Status:              row.Status,
			CreatedAt:           row.CreatedAt,
		}
	}

	r.logger.Info(
		"admin_organizers_listed",
		"page", page,
		"limit", limit,
		"total", total,
	)

	return &domain.AdminOrganizerList{
		Organizers: items,
		Total:      total,
		Page:       page,
		Limit:      limit,
	}, nil
}
