package repository

import (
	"errors"
	"log/slog"
	"time"

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

func (r *EventSettingRepository) Update(
	setting *domain.EventSetting,
) error {
	if setting == nil {
		return errors.New("event setting cannot be nil")
	}

	settingModel := toEventSettingModel(setting)

	result := r.db.
		Model(&models.EventSettingModel{}).
		Where("event_id = ?", setting.EventID).
		Updates(map[string]interface{}{
			"seat_layout_type":       settingModel.SeatLayoutType,
			"booking_limit_per_user": settingModel.BookingLimitPerUser,
			"sales_start_date":       settingModel.SalesStartDate,
			"sales_end_date":         settingModel.SalesEndDate,
			"updated_at":             time.Now(),
		})

	if result.Error != nil {
		r.logger.Error(
			"event_setting_update_failed",
			"event_id", setting.EventID,
			"error", result.Error,
		)

		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn(
			"event_setting_update_not_found",
			"event_id", setting.EventID,
		)

		return domain.ErrEventNotFound
	}

	r.logger.Info(
		"event_setting_updated",
		"event_id", setting.EventID,
		"setting_id", setting.ID,
	)

	return nil
}

func toEventSettingModel(
	setting *domain.EventSetting,
) *models.EventSettingModel {
	return &models.EventSettingModel{
		ID:                  setting.ID,
		EventID:             setting.EventID,
		SeatLayoutType:      setting.SeatLayoutType,
		BookingLimitPerUser: setting.BookingLimitPerUser,
		SalesStartDate:      setting.SalesStartDate,
		SalesEndDate:        setting.SalesEndDate,
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
		SalesStartDate:      settingModel.SalesStartDate,
		SalesEndDate:        settingModel.SalesEndDate,
		CreatedAt:           settingModel.CreatedAt,
		UpdatedAt:           settingModel.UpdatedAt,
	}
}
