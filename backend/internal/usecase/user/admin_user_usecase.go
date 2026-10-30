package user

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
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

type AdminUserDetailsResponse struct {
	ID           uuid.UUID `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	ProfileImage string    `json:"profile_image"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AdminUserStatusResponse struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
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
	userRepo         domain.UserRepository
	refreshTokenRepo domain.RefreshTokenRepository
	txManager        domain.TransactionManager
	logger           *slog.Logger
}

func NewAdminUserUsecase(
	userRepo domain.UserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	txManager domain.TransactionManager,
	logger *slog.Logger,
) *AdminUserUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminUserUsecase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		txManager:        txManager,
		logger:           logger,
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
		case "ACTIVE", "BLOCKED":
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

func (u *AdminUserUsecase) GetUser(id uuid.UUID) (*AdminUserDetailsResponse, error) {
	user, err := u.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, appErrors.NewNotFoundError("user not found")
		}
		u.logger.Error("admin_get_user_failed", "user_id", id, "error", err)
		return nil, appErrors.NewInternalError("failed to retrieve user details")
	}

	return &AdminUserDetailsResponse{
		ID:           user.ID,
		FullName:     user.FullName,
		Email:        user.Email,
		Phone:        user.Phone,
		Role:         user.Role,
		Status:       user.Status,
		ProfileImage: user.ProfileImage,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}, nil
}

func (u *AdminUserUsecase) BlockUser(adminID uuid.UUID, targetID uuid.UUID, reason string) (*AdminUserStatusResponse, error) {
	if adminID == targetID {
		return nil, appErrors.NewValidationError("cannot block own admin account")
	}

	user, err := u.userRepo.FindByID(targetID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, appErrors.NewNotFoundError("user not found")
		}
		u.logger.Error("admin_block_find_user_failed", "target_id", targetID, "error", err)
		return nil, appErrors.NewInternalError("failed to retrieve user")
	}

	if strings.ToUpper(user.Role) == "ADMIN" {
		return nil, appErrors.NewValidationError("cannot block administrator account")
	}

	if strings.ToUpper(user.Status) == "BLOCKED" {
		return nil, appErrors.NewConflictError("user is already blocked")
	}

	user.Status = "BLOCKED"

	if u.txManager != nil {
		err = u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
			if err := tx.UserRepository().Update(user); err != nil {
				return err
			}

			if err := tx.RefreshTokenRepository().RevokeAllByUserID(targetID); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			u.logger.Error("admin_block_transaction_failed", "target_id", targetID, "error", err)
			return nil, appErrors.NewInternalError("failed to update user status")
		}
	} else {
		if err := u.userRepo.Update(user); err != nil {
			u.logger.Error("admin_block_update_failed", "target_id", targetID, "error", err)
			return nil, appErrors.NewInternalError("failed to update user status")
		}

		if u.refreshTokenRepo != nil {
			if err := u.refreshTokenRepo.RevokeAllByUserID(targetID); err != nil {
				u.logger.Warn("admin_block_revoke_tokens_failed", "target_id", targetID, "error", err)
			}
		}
	}

	u.logger.Warn("admin_blocked_user",
		"admin_id", adminID,
		"target_user_id", targetID,
		"reason", strings.TrimSpace(reason),
	)

	return &AdminUserStatusResponse{
		ID:     user.ID,
		Status: user.Status,
	}, nil
}

func (u *AdminUserUsecase) UnblockUser(adminID uuid.UUID, targetID uuid.UUID) (*AdminUserStatusResponse, error) {
	user, err := u.userRepo.FindByID(targetID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, appErrors.NewNotFoundError("user not found")
		}
		u.logger.Error("admin_unblock_find_user_failed", "target_id", targetID, "error", err)
		return nil, appErrors.NewInternalError("failed to retrieve user")
	}

	if strings.ToUpper(user.Status) == "ACTIVE" {
		return nil, appErrors.NewConflictError("user is already active")
	}

	user.Status = "ACTIVE"

	if u.txManager != nil {
		err = u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
			return tx.UserRepository().Update(user)
		})
		if err != nil {
			u.logger.Error("admin_unblock_transaction_failed", "target_id", targetID, "error", err)
			return nil, appErrors.NewInternalError("failed to update user status")
		}
	} else {
		if err := u.userRepo.Update(user); err != nil {
			u.logger.Error("admin_unblock_update_failed", "target_id", targetID, "error", err)
			return nil, appErrors.NewInternalError("failed to update user status")
		}
	}

	u.logger.Info("admin_unblocked_user",
		"admin_id", adminID,
		"target_user_id", targetID,
	)

	return &AdminUserStatusResponse{
		ID:     user.ID,
		Status: user.Status,
	}, nil
}
