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
			Description: "Music events, concerts, and live performances",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Sports",
			Description: "Sports events, matches, and tournaments",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Arts & Theatre",
			Description: "Theatre, drama, and performing arts",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Festivals",
			Description: "Cultural festivals and public celebrations",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Education",
			Description: "Workshops, seminars, and educational events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Technology",
			Description: "Tech conferences, developer meetups, and technology events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Business & Networking",
			Description: "Business conferences, networking, and professional events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Food & Drink",
			Description: "Food festivals, culinary experiences, and tastings",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Health & Wellness",
			Description: "Fitness sessions, wellness programs, and health events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Travel & Outdoor",
			Description: "Outdoor activities, adventure events, and travel experiences",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Family & Kids",
			Description: "Family-friendly activities and children's events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Movies & Film",
			Description: "Movie screenings, film festivals, and cinema events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Gaming & Esports",
			Description: "Gaming tournaments, esports, and gaming conventions",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Fashion & Beauty",
			Description: "Fashion shows, beauty events, and styling workshops",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Religion & Spirituality",
			Description: "Spiritual gatherings and religious events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Charity & Causes",
			Description: "Fundraisers, charity events, and community causes",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Networking & Meetups",
			Description: "Community gatherings and interest-based meetups",
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
