package profile

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/google/uuid"
)

const (
	maxOTPAttempts      = 5
	otpExpiryDuration   = 10 * time.Minute
	resendCooldownTime  = 60 * time.Second
	maxAvatarFileSize   = 5 * 1024 * 1024 // 5 MB
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)
)

type ProfileResponse struct {
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

type UpdateProfileInput struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type EmailChangeRequestResponse struct {
	NewEmail              string `json:"new_email"`
	ResendCooldownSeconds int    `json:"resend_cooldown_seconds"`
	ExpiresInSeconds      int    `json:"expires_in_seconds"`
}

type ResendOTPResponse struct {
	NewEmail              string `json:"new_email"`
	ResendCooldownSeconds int    `json:"resend_cooldown_seconds"`
}

type StorageService interface {
	Upload(ctx context.Context, key string, file io.Reader, contentType string) error
	Delete(ctx context.Context, key string) error
	GetPresignedURL(ctx context.Context, key string) (string, error)
}

type ProfileUsecase struct {
	userRepo                domain.UserRepository
	emailChangeRepo         domain.EmailChangeRequestRepository
	pendingRegistrationRepo domain.PendingRegistrationRepository
	txManager               domain.TransactionManager
	emailService            domain.EmailService
	storageService          StorageService
	otpSecret               string
	logger                  *slog.Logger
}

func NewProfileUsecase(
	userRepo domain.UserRepository,
	emailChangeRepo domain.EmailChangeRequestRepository,
	pendingRegistrationRepo domain.PendingRegistrationRepository,
	txManager domain.TransactionManager,
	emailService domain.EmailService,
	storageService StorageService,
	otpSecret string,
	logger *slog.Logger,
) *ProfileUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &ProfileUsecase{
		userRepo:                userRepo,
		emailChangeRepo:         emailChangeRepo,
		pendingRegistrationRepo: pendingRegistrationRepo,
		txManager:               txManager,
		emailService:            emailService,
		storageService:          storageService,
		otpSecret:               otpSecret,
		logger:                  logger,
	}
}

func (u *ProfileUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error) {
	user, err := u.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, appErrors.NewNotFoundError("user not found")
		}
		u.logger.Error("get_profile_lookup_failed", "user_id", userID, "error", err)
		return nil, err
	}

	avatarURL := u.resolveAvatarURL(ctx, user.ProfileImage)

	return &ProfileResponse{
		ID:           user.ID,
		FullName:     user.FullName,
		Email:        user.Email,
		Phone:        user.Phone,
		Role:         user.Role,
		Status:       user.Status,
		ProfileImage: avatarURL,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}, nil
}

func (u *ProfileUsecase) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*ProfileResponse, error) {
	user, err := u.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, appErrors.NewNotFoundError("user not found")
		}
		u.logger.Error("update_profile_lookup_failed", "user_id", userID, "error", err)
		return nil, err
	}

	fullName := strings.TrimSpace(input.FullName)
	if fullName == "" {
		return nil, appErrors.NewValidationError("full name is required")
	}
	if len(fullName) < 2 || len(fullName) > 100 {
		return nil, appErrors.NewValidationError("full name must be between 2 and 100 characters")
	}
	if !validateNameCharacters(fullName) {
		return nil, appErrors.NewValidationError("full name contains invalid characters")
	}

	phone := strings.TrimSpace(input.Phone)
	if phone != "" {
		if !phoneRegex.MatchString(phone) {
			return nil, appErrors.NewValidationError("phone number must be a valid format with 7 to 15 digits (e.g. +919876543210)")
		}

		existingByPhone, err := u.userRepo.FindByPhone(phone)
		if err == nil && existingByPhone != nil && existingByPhone.ID != userID {
			u.logger.Warn("update_profile_phone_conflict", "user_id", userID, "phone", phone)
			return nil, appErrors.NewConflictError("phone number is already in use by another account")
		}
		if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
			u.logger.Error("update_profile_phone_lookup_failed", "user_id", userID, "error", err)
			return nil, err
		}
	}

	user.FullName = fullName
	user.Phone = phone
	user.UpdatedAt = time.Now()

	if err := u.userRepo.Update(user); err != nil {
		u.logger.Error("update_profile_save_failed", "user_id", userID, "error", err)
		return nil, err
	}

	u.logger.Info("profile_updated_successfully", "user_id", userID)

	avatarURL := u.resolveAvatarURL(ctx, user.ProfileImage)

	return &ProfileResponse{
		ID:           user.ID,
		FullName:     user.FullName,
		Email:        user.Email,
		Phone:        user.Phone,
		Role:         user.Role,
		Status:       user.Status,
		ProfileImage: avatarURL,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}, nil
}

func (u *ProfileUsecase) RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail string) (*EmailChangeRequestResponse, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(newEmail))
	if normalizedEmail == "" {
		return nil, appErrors.NewValidationError("new email address is required")
	}
	if len(normalizedEmail) > 254 || !emailRegex.MatchString(normalizedEmail) {
		return nil, appErrors.NewValidationError("invalid email address format")
	}

	currentUser, err := u.userRepo.FindByID(userID)
	if err != nil {
		return nil, appErrors.NewNotFoundError("user not found")
	}

	if normalizedEmail == strings.ToLower(strings.TrimSpace(currentUser.Email)) {
		return nil, appErrors.NewValidationError("new email cannot be the same as your current email")
	}

	otp, err := generateOTP()
	if err != nil {
		u.logger.Error("email_change_otp_generation_failed", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to generate verification code")
	}

	otpHash := u.hashOTP(otp)
	now := time.Now()
	reqID := uuid.New()

	changeReq := &domain.EmailChangeRequest{
		ID:                reqID,
		UserID:            userID,
		NewEmail:          normalizedEmail,
		OTPHash:           otpHash,
		OTPExpiresAt:      now.Add(otpExpiryDuration),
		OTPAttempts:       0,
		ResendAvailableAt: now.Add(resendCooldownTime),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	err = u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		if err := tx.AcquireEmailAdvisoryLock(normalizedEmail); err != nil {
			return err
		}

		if existingUserReq, err := tx.EmailChangeRequestRepository().FindByUserID(userID); err == nil && existingUserReq != nil {
			if time.Now().Before(existingUserReq.ResendAvailableAt) {
				waitSecs := int(time.Until(existingUserReq.ResendAvailableAt).Seconds())
				if waitSecs < 1 {
					waitSecs = 1
				}
				return &appErrors.AppError{
					Code:       "TOO_MANY_REQUESTS",
					Message:    fmt.Sprintf("please wait %d seconds before requesting another code", waitSecs),
					HTTPStatus: http.StatusTooManyRequests,
				}
			}
		} else if err != nil && !errors.Is(err, domain.ErrEmailChangeRequestNotFound) {
			return err
		}

		if existingUser, err := tx.UserRepository().FindByEmail(normalizedEmail); err == nil && existingUser != nil {
			return appErrors.NewConflictError("email is already registered")
		} else if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
			return err
		}

		if pendingReg, err := tx.PendingRegistrationRepository().FindByEmail(normalizedEmail); err == nil && pendingReg != nil {
			if time.Now().Before(pendingReg.OTPExpiresAt) {
				return appErrors.NewConflictError("email is currently pending registration")
			}
		} else if err != nil && !errors.Is(err, domain.ErrPendingRegistrationNotFound) {
			return err
		}

		if existingReq, err := tx.EmailChangeRequestRepository().FindByNewEmail(normalizedEmail); err == nil && existingReq != nil {
			if existingReq.UserID != userID && time.Now().Before(existingReq.OTPExpiresAt) {
				return appErrors.NewConflictError("email is reserved by another pending request")
			}
		} else if err != nil && !errors.Is(err, domain.ErrEmailChangeRequestNotFound) {
			return err
		}

		return tx.EmailChangeRequestRepository().Upsert(changeReq)
	})

	if err != nil {
		u.logger.Warn("email_change_request_conflict", "user_id", userID, "error", err)
		return nil, err
	}

	if err := u.emailService.SendEmailChangeOTP(normalizedEmail, otp); err != nil {
		u.logger.Error("email_change_smtp_send_failed", "user_id", userID, "request_id", reqID, "error", err)
		_ = u.emailChangeRepo.DeleteByIDAndUserID(reqID, userID)
		return nil, appErrors.NewInternalError("failed to send verification email. Please check your address and try again")
	}

	u.logger.Info("email_change_otp_sent", "user_id", userID, "request_id", reqID)

	return &EmailChangeRequestResponse{
		NewEmail:              normalizedEmail,
		ResendCooldownSeconds: int(resendCooldownTime.Seconds()),
		ExpiresInSeconds:      int(otpExpiryDuration.Seconds()),
	}, nil
}

func (u *ProfileUsecase) ResendEmailChangeOTP(ctx context.Context, userID uuid.UUID, newEmail string) (*ResendOTPResponse, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(newEmail))
	if normalizedEmail == "" {
		return nil, appErrors.NewValidationError("email address is required")
	}

	req, err := u.emailChangeRepo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrEmailChangeRequestNotFound) {
			return nil, appErrors.NewNotFoundError("no pending email change request found")
		}
		return nil, err
	}

	if req.NewEmail != normalizedEmail {
		return nil, appErrors.NewValidationError("email address does not match pending change request")
	}

	if req.OTPAttempts >= maxOTPAttempts {
		return nil, appErrors.NewValidationError("maximum verification attempts exceeded. Please start a new email change request")
	}

	now := time.Now()
	if now.Before(req.ResendAvailableAt) {
		remaining := int(time.Until(req.ResendAvailableAt).Seconds()) + 1
		return nil, &appErrors.AppError{
			Code:       "TOO_MANY_REQUESTS",
			Message:    fmt.Sprintf("please wait %d seconds before requesting another code", remaining),
			HTTPStatus: http.StatusTooManyRequests,
		}
	}

	targetCooldown := now.Add(resendCooldownTime)
	if err := u.emailChangeRepo.UpdateResendCooldown(req.ID, userID, targetCooldown); err != nil {
		u.logger.Error("email_change_resend_cooldown_update_failed", "request_id", req.ID, "error", err)
		return nil, err
	}

	otp, err := generateOTP()
	if err != nil {
		_ = u.emailChangeRepo.RollbackResendCooldown(req.ID, userID, targetCooldown)
		return nil, fmt.Errorf("failed to generate verification code")
	}

	newHash := u.hashOTP(otp)

	if err := u.emailService.SendEmailChangeOTP(normalizedEmail, otp); err != nil {
		u.logger.Error("email_change_smtp_resend_failed", "user_id", userID, "request_id", req.ID, "error", err)
		_ = u.emailChangeRepo.RollbackResendCooldown(req.ID, userID, targetCooldown)
		return nil, appErrors.NewInternalError("failed to send verification email. Your previous code remains valid if not expired")
	}

	updated, err := u.emailChangeRepo.UpdateOTPAndCooldown(
		req.ID,
		req.OTPHash,
		newHash,
		now.Add(otpExpiryDuration),
		targetCooldown,
	)
	if err != nil || !updated {
		u.logger.Warn("otp_resend_db_activation_failed", "request_id", req.ID, "error", err)
		return nil, appErrors.NewInternalError("failed to activate new code. Your previous code remains valid if not expired")
	}

	u.logger.Info("email_change_otp_resent", "user_id", userID, "request_id", req.ID)

	return &ResendOTPResponse{
		NewEmail:              normalizedEmail,
		ResendCooldownSeconds: int(resendCooldownTime.Seconds()),
	}, nil
}

func (u *ProfileUsecase) VerifyEmailChange(ctx context.Context, userID uuid.UUID, newEmail string, otp string) (*ProfileResponse, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(newEmail))
	otp = strings.TrimSpace(otp)

	if normalizedEmail == "" || otp == "" {
		return nil, appErrors.NewValidationError("email and verification code are required")
	}

	req, err := u.emailChangeRepo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrEmailChangeRequestNotFound) {
			return nil, appErrors.NewNotFoundError("no pending email change request found")
		}
		return nil, err
	}

	if req.NewEmail != normalizedEmail {
		return nil, appErrors.NewValidationError("email address does not match pending change request")
	}

	if req.OTPAttempts >= maxOTPAttempts {
		return nil, appErrors.NewValidationError("maximum verification attempts exceeded. Please request a new code")
	}

	if time.Now().After(req.OTPExpiresAt) {
		return nil, appErrors.NewValidationError("verification code has expired. Please request a new code")
	}

	submittedHash := u.hashOTP(otp)
	isMatch := subtle.ConstantTimeCompare([]byte(submittedHash), []byte(req.OTPHash)) == 1

	if !isMatch {
		updatedAttempts, _ := u.emailChangeRepo.IncrementAttempts(req.ID)
		remaining := maxOTPAttempts - updatedAttempts
		u.logger.Warn("email_change_otp_mismatch", "user_id", userID, "request_id", req.ID, "attempts", updatedAttempts)

		if remaining <= 0 {
			return nil, appErrors.NewValidationError("invalid verification code. Maximum attempts reached. Please request a new code")
		}
		return nil, appErrors.NewValidationError(fmt.Sprintf("invalid verification code. %d attempts remaining", remaining))
	}

	var updatedUser *domain.User
	err = u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		if err := tx.AcquireEmailAdvisoryLock(normalizedEmail); err != nil {
			return err
		}

		if existing, err := tx.UserRepository().FindByEmail(normalizedEmail); err == nil && existing != nil {
			return appErrors.NewConflictError("email is already registered")
		}

		user, err := tx.UserRepository().FindByID(userID)
		if err != nil {
			return err
		}

		user.Email = normalizedEmail
		user.UpdatedAt = time.Now()

		if err := tx.UserRepository().Update(user); err != nil {
			return err
		}

		if err := tx.EmailChangeRequestRepository().DeleteByID(req.ID); err != nil {
			return err
		}

		updatedUser = user
		return nil
	})

	if err != nil {
		u.logger.Error("email_change_verification_commit_failed", "user_id", userID, "error", err)
		return nil, err
	}

	u.logger.Info("email_changed_successfully", "user_id", userID)

	avatarURL := u.resolveAvatarURL(ctx, updatedUser.ProfileImage)

	return &ProfileResponse{
		ID:           updatedUser.ID,
		FullName:     updatedUser.FullName,
		Email:        updatedUser.Email,
		Phone:        updatedUser.Phone,
		Role:         updatedUser.Role,
		Status:       updatedUser.Status,
		ProfileImage: avatarURL,
		CreatedAt:    updatedUser.CreatedAt,
		UpdatedAt:    updatedUser.UpdatedAt,
	}, nil
}

func (u *ProfileUsecase) UploadAvatar(
	ctx context.Context,
	userID uuid.UUID,
	fileReader io.Reader,
	contentType string,
	fileSize int64,
) (*ProfileResponse, error) {
	if fileSize > maxAvatarFileSize {
		return nil, appErrors.NewValidationError("profile picture must not exceed 5 MB")
	}

	ext := avatarExtension(contentType)
	if ext == "" {
		return nil, appErrors.NewValidationError("invalid image type. Only JPEG, PNG, and WebP are allowed")
	}

	user, err := u.userRepo.FindByID(userID)
	if err != nil {
		return nil, appErrors.NewNotFoundError("user not found")
	}

	if u.storageService == nil {
		return nil, appErrors.NewInternalError("storage service is not available")
	}

	newKey := fmt.Sprintf("profile-images/%s/%s%s", userID.String(), uuid.New().String(), ext)

	if err := u.storageService.Upload(ctx, newKey, fileReader, contentType); err != nil {
		u.logger.Error("avatar_s3_upload_failed", "user_id", userID, "key", newKey, "error", err)
		return nil, appErrors.NewInternalError("failed to upload image to storage")
	}

	oldKey := user.ProfileImage
	user.ProfileImage = newKey
	user.UpdatedAt = time.Now()

	if err := u.userRepo.Update(user); err != nil {
		u.logger.Error("avatar_db_update_failed", "user_id", userID, "error", err)
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_ = u.storageService.Delete(cleanupCtx, newKey)
		return nil, appErrors.NewInternalError("failed to save profile image reference")
	}

	if oldKey != "" && !strings.HasPrefix(oldKey, "http://") && !strings.HasPrefix(oldKey, "https://") && strings.HasPrefix(oldKey, "profile-images/") {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if delErr := u.storageService.Delete(cleanupCtx, oldKey); delErr != nil {
			u.logger.Warn("s3_old_profile_image_cleanup_failed", "user_id", userID, "old_key", oldKey, "error", delErr)
		}
	}

	u.logger.Info("avatar_uploaded_successfully", "user_id", userID, "key", newKey)

	avatarURL := u.resolveAvatarURL(ctx, user.ProfileImage)

	return &ProfileResponse{
		ID:           user.ID,
		FullName:     user.FullName,
		Email:        user.Email,
		Phone:        user.Phone,
		Role:         user.Role,
		Status:       user.Status,
		ProfileImage: avatarURL,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}, nil
}

func (u *ProfileUsecase) RemoveAvatar(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error) {
	user, err := u.userRepo.FindByID(userID)
	if err != nil {
		return nil, appErrors.NewNotFoundError("user not found")
	}

	oldKey := user.ProfileImage
	if oldKey == "" {
		return &ProfileResponse{
			ID:           user.ID,
			FullName:     user.FullName,
			Email:        user.Email,
			Phone:        user.Phone,
			Role:         user.Role,
			Status:       user.Status,
			ProfileImage: "",
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
		}, nil
	}

	user.ProfileImage = ""
	user.UpdatedAt = time.Now()

	if err := u.userRepo.Update(user); err != nil {
		u.logger.Error("avatar_remove_db_update_failed", "user_id", userID, "error", err)
		return nil, err
	}

	if u.storageService != nil && oldKey != "" && !strings.HasPrefix(oldKey, "http://") && !strings.HasPrefix(oldKey, "https://") && strings.HasPrefix(oldKey, "profile-images/") {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if delErr := u.storageService.Delete(cleanupCtx, oldKey); delErr != nil {
			u.logger.Warn("s3_removed_profile_image_cleanup_failed", "user_id", userID, "old_key", oldKey, "error", delErr)
		}
	}

	u.logger.Info("avatar_removed_successfully", "user_id", userID)

	return &ProfileResponse{
		ID:           user.ID,
		FullName:     user.FullName,
		Email:        user.Email,
		Phone:        user.Phone,
		Role:         user.Role,
		Status:       user.Status,
		ProfileImage: "",
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}, nil
}

func (u *ProfileUsecase) resolveAvatarURL(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		return key
	}
	if u.storageService != nil {
		presigned, err := u.storageService.GetPresignedURL(ctx, key)
		if err == nil && presigned != "" {
			return presigned
		}
	}
	return key
}

func (u *ProfileUsecase) hashOTP(otp string) string {
	h := hmac.New(sha256.New, []byte(u.otpSecret))
	h.Write([]byte(otp))
	return hex.EncodeToString(h.Sum(nil))
}

func generateOTP() (string, error) {
	const max = 1000000
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	number := uint32(bytes[0])<<24 |
		uint32(bytes[1])<<16 |
		uint32(bytes[2])<<8 |
		uint32(bytes[3])
	otp := number % max
	return fmt.Sprintf("%06d", otp), nil
}

func avatarExtension(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

func validateNameCharacters(name string) bool {
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			r == ' ' || r == '-' || r == '\'' || r == '.' ||
			(r >= 0x00C0 && r <= 0x024F) { // Latin-1 Supplement & Extended
			continue
		}
		return false
	}
	return true
}
