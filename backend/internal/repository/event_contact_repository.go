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

type EventContactRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewEventContactRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *EventContactRepository {
	return &EventContactRepository{
		db:     db,
		logger: logger,
	}
}

func (r *EventContactRepository) Create(
	contact *domain.EventContact,
) error {
	contactModel := toEventContactModel(contact)

	if err := r.db.Create(contactModel).Error; err != nil {
		r.logger.Error(
			"event_contact_create_failed",
			"event_id", contact.EventID,
			"error", err,
		)

		return err
	}

	*contact = *toEventContactDomain(contactModel)

	r.logger.Info(
		"event_contact_created",
		"event_id", contact.EventID,
		"contact_id", contact.ID,
	)

	return nil
}

func (r *EventContactRepository) FindByEventID(
	eventID uuid.UUID,
) (*domain.EventContact, error) {
	var contactModel models.EventContactModel

	if err := r.db.
		Where("event_id = ?", eventID).
		First(&contactModel).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEventNotFound
		}

		r.logger.Error(
			"event_contact_find_by_event_id_failed",
			"event_id", eventID,
			"error", err,
		)

		return nil, err
	}

	return toEventContactDomain(&contactModel), nil
}

func (r *EventContactRepository) Update(
	contact *domain.EventContact,
) error {
	if contact == nil {
		return errors.New("event contact cannot be nil")
	}

	contactModel := toEventContactModel(contact)

	result := r.db.
		Model(&models.EventContactModel{}).
		Where("event_id = ?", contact.EventID).
		Updates(map[string]interface{}{
			"name":       contactModel.Name,
			"phone":      contactModel.Phone,
			"email":      contactModel.Email,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		r.logger.Error(
			"event_contact_update_failed",
			"event_id", contact.EventID,
			"error", result.Error,
		)

		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn(
			"event_contact_update_not_found",
			"event_id", contact.EventID,
		)

		return domain.ErrEventNotFound
	}

	r.logger.Info(
		"event_contact_updated",
		"event_id", contact.EventID,
		"contact_id", contact.ID,
	)

	return nil
}

func (r *EventContactRepository) DeleteByEventID(
	eventID uuid.UUID,
) error {
	result := r.db.
		Where("event_id = ?", eventID).
		Delete(&models.EventContactModel{})

	if result.Error != nil {
		r.logger.Error(
			"event_contact_delete_by_event_id_failed",
			"event_id", eventID,
			"error", result.Error,
		)

		return result.Error
	}

	r.logger.Info(
		"event_contact_deleted_by_event_id",
		"event_id", eventID,
		"rows_affected", result.RowsAffected,
	)

	return nil
}

func toEventContactModel(
	contact *domain.EventContact,
) *models.EventContactModel {
	return &models.EventContactModel{
		ID:        contact.ID,
		EventID:   contact.EventID,
		Name:      contact.Name,
		Phone:     contact.Phone,
		Email:     contact.Email,
		CreatedAt: contact.CreatedAt,
		UpdatedAt: contact.UpdatedAt,
	}
}

func toEventContactDomain(
	contactModel *models.EventContactModel,
) *domain.EventContact {
	return &domain.EventContact{
		ID:        contactModel.ID,
		EventID:   contactModel.EventID,
		Name:      contactModel.Name,
		Phone:     contactModel.Phone,
		Email:     contactModel.Email,
		CreatedAt: contactModel.CreatedAt,
		UpdatedAt: contactModel.UpdatedAt,
	}
}
