package repository

import (
	"errors"
	"log/slog"

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
