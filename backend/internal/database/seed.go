package database

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedCategories(db *gorm.DB, logger *slog.Logger) error {
	categories := []models.CategoryModel{
		{
			Name:        "Comedy",
			Description: "Comedy events and shows",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Music",
			Description: "Music events and concerts",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Sports",
			Description: "Sports events and matches",
			Status:      domain.CategoryStatusActive,
		},
	}

	for _, category := range categories {
		var existing models.CategoryModel

		err := db.
			Where("name = ?", category.Name).
			First(&existing).Error

		if err == nil {
			continue
		}

		if err != gorm.ErrRecordNotFound {
			logger.Error(
				"category_seed_lookup_failed",
				"name", category.Name,
				"error", err,
			)
			return err
		}

		category.ID = uuid.New()

		if err := db.Create(&category).Error; err != nil {
			logger.Error(
				"category_seed_failed",
				"name", category.Name,
				"error", err,
			)
			return err
		}

		logger.Info(
			"category_seeded",
			"name", category.Name,
		)
	}

	return nil
}
