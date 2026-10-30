package user

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

type AdminUserResponse struct {
	ID           uuid.UUID `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	ProfileImage string    `json:"profile_image"`
	CreatedAt    time.Time `json:"created_at"`
}

type AdminUserRoleCounts struct {
	Total      int64 `json:"total"`
	Customers  int64 `json:"customers"`
	Organizers int64 `json:"organizers"`
	Admins     int64 `json:"admins"`
}

type AdminUserListResponse struct {
	Users      []AdminUserResponse `json:"users"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	RoleCounts AdminUserRoleCounts `json:"role_counts"`
}

type AdminUserUsecase struct {
	userRepo domain.UserRepository
	logger   *slog.Logger
}

func NewAdminUserUsecase(
	userRepo domain.UserRepository,
	logger *slog.Logger,
) *AdminUserUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminUserUsecase{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (u *AdminUserUsecase) ListUsers(
	page int,
	limit int,
	search string,
	role string,
	status string,
) (*AdminUserListResponse, error) {
	if page < 1 {
		return nil, fmt.Errorf("page must be greater than zero")
	}

	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	cleanRole := strings.ToUpper(strings.TrimSpace(role))
	if cleanRole != "" {
		switch cleanRole {
		case "CUSTOMER", "ORGANIZER", "ADMIN":
			// valid
		default:
			return nil, fmt.Errorf("invalid role filter: %s", role)
		}
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

	filter := domain.UserListFilter{
		Page:   page,
		Limit:  limit,
		Search: strings.TrimSpace(search),
		Role:   cleanRole,
		Status: cleanStatus,
	}

	list, err := u.userRepo.List(filter)
	if err != nil {
		u.logger.Error(
			"admin_user_list_failed",
			"error", err,
			"page", page,
			"limit", limit,
			"role", cleanRole,
		)
		return nil, fmt.Errorf("failed to retrieve users")
	}

	users := make([]AdminUserResponse, len(list.Users))
	for i, u := range list.Users {
		users[i] = AdminUserResponse{
			ID:           u.ID,
			FullName:     u.FullName,
			Email:        u.Email,
			Phone:        u.Phone,
			Role:         u.Role,
			Status:       u.Status,
			ProfileImage: u.ProfileImage,
			CreatedAt:    u.CreatedAt,
		}
	}

	roleCounts := AdminUserRoleCounts{
		Total:      list.RoleCounts.Total,
		Customers:  list.RoleCounts.Customers,
		Organizers: list.RoleCounts.Organizers,
		Admins:     list.RoleCounts.Admins,
	}

	u.logger.Info(
		"admin_users_retrieved",
		"page", page,
		"limit", limit,
		"total", list.Total,
	)

	return &AdminUserListResponse{
		Users:      users,
		Total:      list.Total,
		Page:       list.Page,
		Limit:      list.Limit,
		RoleCounts: roleCounts,
	}, nil
}
