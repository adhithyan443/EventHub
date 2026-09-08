package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VenueRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewVenueRepository(db *gorm.DB, logger *slog.Logger) *VenueRepository {
	return &VenueRepository{
		db:     db,
		logger: logger,
	}
}

func (r *VenueRepository) Create(venue *domain.Venue) error {
	venueModel := toVenueModel(venue)

	if err := r.db.Create(venueModel).Error; err != nil {
		r.logger.Error(
			"venue_create_failed",
			"google_place_id", venue.GooglePlaceID,
			"error", err,
		)
		return err
	}

	*venue = *toVenueDomain(venueModel)

	r.logger.Info(
		"venue_created",
		"venue_id", venue.ID,
		"google_place_id", venue.GooglePlaceID,
	)

	return nil
}

func (r *VenueRepository) FindByID(id uuid.UUID) (*domain.Venue, error) {
	var venueModel models.VenueModel

	err := r.db.First(&venueModel, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrVenueNotFound
		}

		r.logger.Error(
			"venue_find_by_id_failed",
			"venue_id", id,
			"error", err,
		)

		return nil, err
	}

	return toVenueDomain(&venueModel), nil
}

func (r *VenueRepository) FindByGooglePlaceID(placeID string) (*domain.Venue, error) {
	var venueModel models.VenueModel

	err := r.db.
		Where("google_place_id = ?", placeID).
		First(&venueModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrVenueNotFound
		}

		r.logger.Error(
			"venue_find_by_google_place_id_failed",
			"google_place_id", placeID,
			"error", err,
		)

		return nil, err
	}

	return toVenueDomain(&venueModel), nil
}

func toVenueModel(venue *domain.Venue) *models.VenueModel {
	return &models.VenueModel{
		ID:            venue.ID,
		GooglePlaceID: venue.GooglePlaceID,
		Name:          venue.Name,
		Address:       venue.Address,
		City:          venue.City,
		State:         venue.State,
		Country:       venue.Country,
		PostalCode:    venue.PostalCode,
		Latitude:      venue.Latitude,
		Longitude:     venue.Longitude,
		CreatedAt:     venue.CreatedAt,
		UpdatedAt:     venue.UpdatedAt,
	}
}

func toVenueDomain(model *models.VenueModel) *domain.Venue {
	return &domain.Venue{
		ID:            model.ID,
		GooglePlaceID: model.GooglePlaceID,
		Name:          model.Name,
		Address:       model.Address,
		City:          model.City,
		State:         model.State,
		Country:       model.Country,
		PostalCode:    model.PostalCode,
		Latitude:      model.Latitude,
		Longitude:     model.Longitude,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}
