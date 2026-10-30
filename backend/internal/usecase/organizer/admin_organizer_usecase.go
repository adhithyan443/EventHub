package organizer

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type AdminOrganizerResponse struct {
	ID                  uuid.UUID `json:"id"`
	UserID              uuid.UUID `json:"user_id"`
	UserName            string    `json:"user_name"`
	UserEmail           string    `json:"user_email"`
	UserPhone           string    `json:"user_phone"`
	BusinessName        string    `json:"business_name"`
	BusinessType        string    `json:"business_type"`
	BusinessDescription string    `json:"business_description"`
	BusinessPhone       string    `json:"business_phone"`
	BusinessEmail       string    `json:"business_email"`
	Website             string    `json:"website"`
	LogoURL             string    `json:"logo_url"`
	GSTNumber           string    `json:"gst_number"`
	PANNumber           string    `json:"pan_number"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
}

type AdminOrganizerListResponse struct {
	Organizers []AdminOrganizerResponse `json:"organizers"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
}

type AdminOrganizerUsecase struct {
	organizerRepo domain.OrganizerRepository
	logger        *slog.Logger
}

func NewAdminOrganizerUsecase(
	organizerRepo domain.OrganizerRepository,
	logger *slog.Logger,
) *AdminOrganizerUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminOrganizerUsecase{
		organizerRepo: organizerRepo,
		logger:        logger,
	}
}

func (u *AdminOrganizerUsecase) ListOrganizers(
	page int,
	limit int,
	search string,
	status string,
) (*AdminOrganizerListResponse, error) {
	if page < 1 {
		return nil, fmt.Errorf("page must be greater than zero")
	}

	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	cleanStatus := strings.ToUpper(strings.TrimSpace(status))
	if cleanStatus != "" {
		switch cleanStatus {
		case "ACTIVE":
			// valid
		default:
			return nil, fmt.Errorf("invalid status filter: %s", status)
		}
	}

	filter := domain.AdminOrganizerListFilter{
		Page:   page,
		Limit:  limit,
		Search: strings.TrimSpace(search),
		Status: cleanStatus,
	}

	list, err := u.organizerRepo.List(filter)
	if err != nil {
		u.logger.Error(
			"admin_organizer_list_failed",
			"error", err,
			"page", page,
			"limit", limit,
			"status", cleanStatus,
		)
		return nil, fmt.Errorf("failed to retrieve organizers")
	}

	organizers := make([]AdminOrganizerResponse, len(list.Organizers))
	for i, o := range list.Organizers {
		organizers[i] = AdminOrganizerResponse{
			ID:                  o.ID,
			UserID:              o.UserID,
			UserName:            o.UserName,
			UserEmail:           o.UserEmail,
			UserPhone:           o.UserPhone,
			BusinessName:        o.BusinessName,
			BusinessType:        o.BusinessType,
			BusinessDescription: o.BusinessDescription,
			BusinessPhone:       o.BusinessPhone,
			BusinessEmail:       o.BusinessEmail,
			Website:             o.Website,
			LogoURL:             o.LogoURL,
			GSTNumber:           o.GSTNumber,
			PANNumber:           o.PANNumber,
			Status:              o.Status,
			CreatedAt:           o.CreatedAt,
		}
	}

	u.logger.Info(
		"admin_organizers_retrieved",
		"page", page,
		"limit", limit,
		"total", list.Total,
	)

	return &AdminOrganizerListResponse{
		Organizers: organizers,
		Total:      list.Total,
		Page:       list.Page,
		Limit:      list.Limit,
	}, nil
}
