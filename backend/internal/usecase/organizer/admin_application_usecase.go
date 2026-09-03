package organizer

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type AdminApplicationUsecase struct {
	repository domain.OrganizerApplicationRepository
	logger     *slog.Logger
}

func NewAdminApplicationUsecase(
	repository domain.OrganizerApplicationRepository,
	logger *slog.Logger,
) *AdminApplicationUsecase {
	return &AdminApplicationUsecase{
		repository: repository,
		logger:     logger,
	}
}

func (u *AdminApplicationUsecase) ListApplications(
	page int,
	limit int,
	status string,
) (*domain.OrganizerApplicationList, error) {

	if page < 1 {
		return nil, fmt.Errorf("page must be greater than zero")
	}

	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	applicationStatus := domain.ApplicationStatus(
		strings.ToUpper(strings.TrimSpace(status)),
	)

	// Validate status when supplied.
	if applicationStatus != "" {
		switch applicationStatus {
		case domain.ApplicationPending,
			domain.ApplicationApproved,
			domain.ApplicationRejected:

		default:
			return nil, fmt.Errorf("invalid application status")
		}
	}

	applications, err := u.repository.List(
		page,
		limit,
		applicationStatus,
	)
	if err != nil {
		u.logger.Error(
			"admin_organizer_application_list_failed",
			"page", page,
			"limit", limit,
			"status", applicationStatus,
			"error", err,
		)

		return nil, fmt.Errorf(
			"failed to retrieve organizer applications",
		)
	}

	u.logger.Info(
		"admin_organizer_applications_listed",
		"page", page,
		"limit", limit,
		"status", applicationStatus,
		"total", applications.Total,
	)

	return applications, nil
}

// Keep this helper available for future admin operations.
func applicationID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid application id")
	}

	return id, nil
}

// GetApplication retrieves a single organizer application for admin review.
func (u *AdminApplicationUsecase) GetApplication(
	id uuid.UUID,
) (*domain.OrganizerApplication, error) {

	application, err := u.repository.FindByID(id)
	if err != nil {
		u.logger.Error(
			"admin_organizer_application_get_failed",
			"application_id", id,
			"error", err,
		)

		return nil, fmt.Errorf(
			"failed to retrieve organizer application",
		)
	}

	u.logger.Info(
		"admin_organizer_application_retrieved",
		"application_id", id,
	)

	return application, nil
}
