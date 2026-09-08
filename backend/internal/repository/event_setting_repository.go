package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventSettingRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewEventSettingRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *EventSettingRepository {
	return &EventSettingRepository{
		db:     db,
		logger: logger,
	}
}

func (r *EventSettingRepository) Create(
	setting *domain.EventSetting,
) error {
	settingModel := toEventSettingModel(setting)

	if err := r.db.Create(settingModel).Error; err != nil {
		r.logger.Error(
			"event_setting_create_failed",
			"event_id", setting.EventID,
			"error", err,
		)

		return err
	}

	*setting = *toEventSettingDomain(settingModel)

	r.logger.Info(
		"event_setting_created",
		"event_id", setting.EventID,
		"setting_id", setting.ID,
	)

	return nil
}

func (r *EventSettingRepository) FindByEventID(
	eventID uuid.UUID,
) (*domain.EventSetting, error) {
	var settingModel models.EventSettingModel

	if err := r.db.
		Where("event_id = ?", eventID).
		First(&settingModel).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEventNotFound
		}

		r.logger.Error(
			"event_setting_find_by_event_id_failed",
			"event_id", eventID,
			"error", err,
		)

		return nil, err
	}

	return toEventSettingDomain(&settingModel), nil
}

func toEventSettingModel(
	setting *domain.EventSetting,
) *models.EventSettingModel {
	return &models.EventSettingModel{
		ID:                  setting.ID,
		EventID:             setting.EventID,
		SeatLayoutType:      setting.SeatLayoutType,
		BookingLimitPerUser: setting.BookingLimitPerUser,
		CreatedAt:           setting.CreatedAt,
		UpdatedAt:           setting.UpdatedAt,
	}
}

func toEventSettingDomain(
	settingModel *models.EventSettingModel,
) *domain.EventSetting {
	return &domain.EventSetting{
		ID:                  settingModel.ID,
		EventID:             settingModel.EventID,
		SeatLayoutType:      settingModel.SeatLayoutType,
		BookingLimitPerUser: settingModel.BookingLimitPerUser,
		CreatedAt:           settingModel.CreatedAt,
		UpdatedAt:           settingModel.UpdatedAt,
	}
}
