package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizerAddressRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewOrganizerAddressRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *OrganizerAddressRepository {
	return &OrganizerAddressRepository{
		db:     db,
		logger: logger,
	}
}

func (r *OrganizerAddressRepository) Create(
	address *domain.OrganizerAddress,
) error {
	model := organizerAddressToModel(address)

	if err := r.db.Create(&model).Error; err != nil {
		r.logger.Error(
			"organizer_address_create_failed",
			"organizer_id", address.OrganizerID,
			"error", err,
		)
		return err
	}

	*address = *modelToOrganizerAddress(&model)

	r.logger.Info(
		"organizer_address_created",
		"organizer_address_id", address.ID,
		"organizer_id", address.OrganizerID,
	)

	return nil
}

func (r *OrganizerAddressRepository) FindByOrganizerID(
	organizerID uuid.UUID,
) (*domain.OrganizerAddress, error) {
	var model models.OrganizerAddressModel

	err := r.db.
		Where("organizer_id = ?", organizerID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrganizerAddressNotFound
		}

		r.logger.Error(
			"organizer_address_find_failed",
			"organizer_id", organizerID,
			"error", err,
		)

		return nil, err
	}

	return modelToOrganizerAddress(&model), nil
}

func organizerAddressToModel(
	address *domain.OrganizerAddress,
) models.OrganizerAddressModel {
	return models.OrganizerAddressModel{
		ID:          address.ID,
		OrganizerID: address.OrganizerID,
		AddressLine: address.AddressLine,
		City:        address.City,
		State:       address.State,
		Country:     address.Country,
		PostalCode:  address.PostalCode,
		CreatedAt:   address.CreatedAt,
		UpdatedAt:   address.UpdatedAt,
	}
}

func modelToOrganizerAddress(
	model *models.OrganizerAddressModel,
) *domain.OrganizerAddress {
	return &domain.OrganizerAddress{
		ID:          model.ID,
		OrganizerID: model.OrganizerID,
		AddressLine: model.AddressLine,
		City:        model.City,
		State:       model.State,
		Country:     model.Country,
		PostalCode:  model.PostalCode,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}
