package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"

	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthUsecase struct {
	userRepo                domain.UserRepository
	pendingRegistrationRepo domain.PendingRegistrationRepository
	refreshTokenRepo        domain.RefreshTokenRepository
	passwordResetTokenRepo  domain.PasswordResetTokenRepository
	jwtService              *token.JWTService
	txManager               domain.TransactionManager
	emailService            domain.EmailService
	googleOAuthService      domain.GoogleOAuthService
	logger                  *slog.Logger
}

func NewAuthUsecase(
	userRepo domain.UserRepository,
	pendingRegistrationRepo domain.PendingRegistrationRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	passwordResetTokenRepo domain.PasswordResetTokenRepository,
	jwtService *token.JWTService,
	txManager domain.TransactionManager,
	emailService domain.EmailService,
	googleOAuthService domain.GoogleOAuthService,
	logger *slog.Logger,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:                userRepo,
		refreshTokenRepo:        refreshTokenRepo,
		pendingRegistrationRepo: pendingRegistrationRepo,
		passwordResetTokenRepo:  passwordResetTokenRepo,
		jwtService:              jwtService,
		txManager:               txManager,
		emailService:            emailService,
		googleOAuthService:      googleOAuthService,
		logger:                  logger,
	}
}

type RegisterInput struct {
	FullName string
	Email    string
	Password string
	Phone    string
}

type LoginInput struct {
	Email    string
	Password string
}

func generateRefreshToken() (string, error) {
	tokenBytes := make([]byte, 32)

	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (u *AuthUsecase) Register(input RegisterInput) error {

	input.FullName = strings.TrimSpace(input.FullName)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)

	if input.FullName == "" {
		return appErrors.NewValidationError("full name is required")
	}

	if input.Email == "" {
		return appErrors.NewValidationError("email is required")
	}

	if input.Password == "" {
		return appErrors.NewValidationError("password is required")
	}

	if !validatePassword(input.Password) {
		return appErrors.NewValidationError(
			"password must be at least 8 characters and contain uppercase, lowercase, number, and special character",
		)
	}

	if input.Phone == "" {
		return appErrors.NewValidationError("phone is required")
	}

	existingUser, err := u.userRepo.FindByEmail(input.Email)

	if err == nil && existingUser != nil {
		return appErrors.NewConflictError("email already exists")
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	pendingRegistration, err := u.pendingRegistrationRepo.FindByEmail(input.Email)

	if err == nil && pendingRegistration != nil {

		otp, err := generateOTP()
		if err != nil {
			return err
		}

		otpHash := hashOTP(otp)

		// u.logger.Info("otp_generated_debug", "email", input.Email, "otp", otp)

		now := time.Now()

		pendingRegistration.FullName = input.FullName
		pendingRegistration.Phone = input.Phone
		pendingRegistration.PasswordHash = string(passwordHash)
		pendingRegistration.OTPHash = otpHash
		pendingRegistration.OTPExpiresAt = now.Add(10 * time.Minute)
		pendingRegistration.OTPAttempts = 0
		pendingRegistration.UpdatedAt = now

		if err := u.pendingRegistrationRepo.Update(pendingRegistration); err != nil {
			return err
		}

		if err := u.emailService.SendOTP(input.Email, otp); err != nil {
			u.logger.Error(
				"otp_email_send_failed",
				"email", input.Email,
				"error", err,
			)
			return err
		}

		u.logger.Info(
			"otp_email_sent",
			"email", input.Email,
		)

		u.logger.Info(
			"registration_verification_renewed",
			"registration_id", pendingRegistration.ID,
		)

		return nil
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	otp, err := generateOTP()
	if err != nil {
		return err
	}

	otpHash := hashOTP(otp)
	// u.logger.Info("otp_generated_debug", "email", input.Email, "otp", otp)

	now := time.Now()

	pendingRegistration = &domain.PendingRegistration{
		ID:           uuid.New(),
		FullName:     input.FullName,
		Email:        input.Email,
		Phone:        input.Phone,
		PasswordHash: string(passwordHash),
		OTPHash:      otpHash,
		OTPExpiresAt: now.Add(10 * time.Minute),
		OTPAttempts:  0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := u.pendingRegistrationRepo.Create(pendingRegistration); err != nil {
		return err
	}

	if err := u.emailService.SendOTP(input.Email, otp); err != nil {
		u.logger.Error(
			"otp_email_send_failed",
			"email", input.Email,
			"error", err,
		)
		return err
	}

	u.logger.Info(
		"otp_email_sent",
		"email", input.Email,
	)

	u.logger.Info(
		"registration_verification_created",
		"registration_id", pendingRegistration.ID,
	)

	return nil
}

func (u *AuthUsecase) Login(input LoginInput) (*domain.User, string, string, error) {

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Email == "" || input.Password == "" {
		return nil, "", "", appErrors.NewValidationError("email and password are required")
	}

	user, err := u.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, "", "", appErrors.NewUnauthorizedError("invalid email or password")
	}

	if user.Status != "ACTIVE" {
		return nil, "", "", appErrors.NewUnauthorizedError("user account is not active")
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(input.Password),
	); err != nil {
		return nil, "", "", appErrors.NewUnauthorizedError("invalid email or password")
	}

	accessToken, rawRefreshToken, err := u.generateAuthTokens(user)
	if err != nil {
		return nil, "", "", err
	}

	u.logger.Info(
		"user_login_success",
		"user_id", user.ID,
	)

	return user, accessToken, rawRefreshToken, nil

}

type RefreshTokenInput struct {
	RefreshToken string
}

func (u *AuthUsecase) RefreshToken(input RefreshTokenInput) (string, string, error) {
	input.RefreshToken = strings.TrimSpace(input.RefreshToken)

	if input.RefreshToken == "" {
		return "", "", appErrors.NewValidationError("refresh token is required")
	}

	tokenHash := hashRefreshToken(input.RefreshToken)

	storedToken, err := u.refreshTokenRepo.FindByTokenHash(tokenHash)
	if err != nil {
		return "", "", appErrors.NewUnauthorizedError("invalid refresh token")
	}

	if storedToken.RevokedAt != nil {
		return "", "", appErrors.NewUnauthorizedError("invalid refresh token")
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return "", "", appErrors.NewUnauthorizedError("refresh token expired")
	}

	user, err := u.userRepo.FindByID(storedToken.UserID)
	if err != nil {
		return "", "", appErrors.NewUnauthorizedError("invalid refresh token")
	}

	if user.Status != "ACTIVE" {
		return "", "", appErrors.NewUnauthorizedError("user account is not active")
	}

	accessToken, err := u.jwtService.GenerateAccessToken(
		user.ID,
		user.Role,
	)

	if err != nil {
		return "", "", err
	}

	if err := u.refreshTokenRepo.Revoke(storedToken.ID); err != nil {
		return "", "", err
	}
	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	now := time.Now()

	newToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashRefreshToken(newRefreshToken),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	if err := u.refreshTokenRepo.Create(newToken); err != nil {
		return "", "", err
	}

	u.logger.Info(
		"refresh_token_rotated",
		"user_id", user.ID,
	)

	return accessToken, newRefreshToken, nil
}

type LogoutInput struct {
	RefreshToken string
}

func (u *AuthUsecase) Logout(userID uuid.UUID, input LogoutInput) error {
	input.RefreshToken = strings.TrimSpace(input.RefreshToken)

	if input.RefreshToken == "" {
		return appErrors.NewValidationError("refresh token is required")
	}

	tokenHash := hashRefreshToken(input.RefreshToken)

	storedToken, err := u.refreshTokenRepo.FindByTokenHash(tokenHash)
	if err != nil {
		return appErrors.NewUnauthorizedError("invalid refresh token")
	}

	if storedToken.UserID != userID {
		return appErrors.NewUnauthorizedError("invalid refresh token")
	}

	if storedToken.RevokedAt != nil { // Prevent logging out an already revoked session.
		return appErrors.NewUnauthorizedError("invalid refresh token")
	}

	if err := u.refreshTokenRepo.Revoke(storedToken.ID); err != nil {
		return err
	}

	u.logger.Info(
		"user_logout_success",
		"user_id", userID,
		"refresh_token_id", storedToken.ID,
	)

	return nil
}

const maxOTPAttempts = 5

type VerifyOTPInput struct {
	Email string
	OTP   string
}

func (u *AuthUsecase) VerifyOTP(input VerifyOTPInput) (*domain.User, error) {

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.OTP = strings.TrimSpace(input.OTP)

	if input.Email == "" {
		return nil, appErrors.NewValidationError("email is required")
	}

	if input.OTP == "" {
		return nil, appErrors.NewValidationError("otp is required")
	}

	pendingRegistration, err := u.pendingRegistrationRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, appErrors.NewValidationError("invalid or expired verification code")
	}

	if pendingRegistration.OTPAttempts >= maxOTPAttempts {
		return nil, appErrors.NewValidationError("invalid or expired verification code")
	}

	if time.Now().After(pendingRegistration.OTPExpiresAt) {
		return nil, appErrors.NewValidationError("invalid or expired verification code")
	}

	submittedOTPHash := hashOTP(input.OTP)

	if submittedOTPHash != pendingRegistration.OTPHash {

		pendingRegistration.OTPAttempts += 1
		pendingRegistration.UpdatedAt = time.Now()

		if err := u.pendingRegistrationRepo.Update(pendingRegistration); err != nil {
			return nil, err
		}

		u.logger.Warn(
			"otp_verification_failed",
			"registration_id", pendingRegistration.ID,
			"attempts", pendingRegistration.OTPAttempts,
		)
		return nil, appErrors.NewValidationError("invalid or expired verification code")

	}

	newUser := &domain.User{
		ID:           uuid.New(),
		FullName:     pendingRegistration.FullName,
		Email:        pendingRegistration.Email,
		Phone:        pendingRegistration.Phone,
		PasswordHash: pendingRegistration.PasswordHash,
		Role:         "CUSTOMER",
		Status:       "ACTIVE",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {

		if err := tx.UserRepository().Create(newUser); err != nil {
			return err
		}

		if err := tx.PendingRegistrationRepository().Delete(pendingRegistration.ID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		u.logger.Error(
			"user_registration_verification_failed",
			"registration_id", pendingRegistration.ID,
			"error", err,
		)

		return nil, err
	}

	u.logger.Info(
		"user_registration_verified",
		"user_id", newUser.ID,
	)

	return newUser, nil
}

type ForgotPasswordInput struct {
	Email string
}

func (u *AuthUsecase) ForgotPassword(input ForgotPasswordInput) error {

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Email == "" {
		return appErrors.NewValidationError("email is required")
	}

	user, err := u.userRepo.FindByEmail(input.Email)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil //This is called account enumeration protection.
		}

		return err
	}

	resetToken, err := generatePasswordResetToken()
	if err != nil {
		return err
	}

	tokenHash := hashPasswordResetToken(resetToken)

	now := time.Now()

	passwordResetToken := &domain.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(15 * time.Minute),
		CreatedAt: now,
	}

	if err := u.passwordResetTokenRepo.Create(passwordResetToken); err != nil {
		return err
	}

	if err := u.emailService.SendPasswordResetEmail(user.Email, resetToken); err != nil {
		u.logger.Error(
			"password_reset_email_send_failed",
			"user_id", user.ID,
			"error", err,
		)
		return err
	}

	u.logger.Info(
		"password_reset_requested",
		"user_id", user.ID,
	)

	return nil
}

type ResetPasswordInput struct {
	Token       string
	NewPassword string
}

func (u *AuthUsecase) ResetPassword(input ResetPasswordInput) error {
	input.Token = strings.TrimSpace(input.Token)

	if input.Token == "" {
		return appErrors.NewValidationError("reset token is required")
	}

	if input.NewPassword == "" {
		return appErrors.NewValidationError("new password is required")
	}

	if !validatePassword(input.NewPassword) {
		return appErrors.NewValidationError(
			"password must be at least 8 characters and contain uppercase, lowercase, number, and special character",
		)
	}

	tokenHash := hashPasswordResetToken(input.Token)

	resetToken, err := u.passwordResetTokenRepo.FindByTokenHash(tokenHash)
	if err != nil {
		return appErrors.NewValidationError("invalid or expired reset token")
	}

	if resetToken.UsedAt != nil {
		return appErrors.NewValidationError("invalid or expired reset token")
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return appErrors.NewValidationError("invalid or expired reset token")
	}

	user, err := u.userRepo.FindByID(resetToken.UserID)
	if err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.NewPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	user.PasswordHash = string(passwordHash)
	user.UpdatedAt = time.Now()

	err = u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		if err := tx.UserRepository().Update(user); err != nil {
			return err
		}

		if err := tx.PasswordResetTokenRepository().MarkUsed(resetToken.ID); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		u.logger.Error(
			"password_reset_failed",
			"user_id", user.ID,
			"error", err,
		)

		return err
	}

	u.logger.Info(
		"password_reset_completed",
		"user_id", user.ID,
	)

	return nil
}

func (u *AuthUsecase) GetGoogleAuthURL(state string) string {
	return u.googleOAuthService.GetAuthURL(state)
}

func (u *AuthUsecase) HandleGoogleCallback(
	code string,
) (*domain.User, string, string, error) {

	code = strings.TrimSpace(code)

	if code == "" {
		return nil, "", "", appErrors.NewValidationError(
			"google authorization code is required",
		)
	}

	googleUser, err := u.googleOAuthService.GetUser(code)
	if err != nil {
		u.logger.Error(
			"google_oauth_failed",
			"error", err,
		)

		return nil, "", "", appErrors.NewUnauthorizedError(
			"google authentication failed",
		)
	}

	email := strings.ToLower(
		strings.TrimSpace(googleUser.Email),
	)

	if email == "" {
		return nil, "", "", appErrors.NewValidationError(
			"google account email is required",
		)
	}

	user, err := u.userRepo.FindByEmail(email)

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", "", err
		}

		now := time.Now()

		user = &domain.User{
			ID:           uuid.New(),
			FullName:     strings.TrimSpace(googleUser.Name),
			Email:        email,
			ProfileImage: googleUser.AvatarURL,
			Role:         "CUSTOMER",
			Status:       "ACTIVE",
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := u.userRepo.Create(user); err != nil {
			return nil, "", "", err
		}

		u.logger.Info(
			"google_oauth_user_created",
			"user_id", user.ID,
		)
	}

	if user.Status != "ACTIVE" {
		return nil, "", "", appErrors.NewUnauthorizedError(
			"user account is not active",
		)
	}

	accessToken, rawRefreshToken, err := u.generateAuthTokens(user)
	if err != nil {
		return nil, "", "", err
	}

	u.logger.Info(
		"google_oauth_login_success",
		"user_id", user.ID,
	)

	return user, accessToken, rawRefreshToken, nil
}
func (u *AuthUsecase) generateAuthTokens(
	user *domain.User,
) (string, string, error) {
	accessToken, err := u.jwtService.GenerateAccessToken(
		user.ID,
		user.Role,
	)
	if err != nil {
		return "", "", err
	}

	rawRefreshToken, err := generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	now := time.Now()

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashRefreshToken(rawRefreshToken),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	if err := u.refreshTokenRepo.Create(refreshToken); err != nil {
		return "", "", err
	}

	return accessToken, rawRefreshToken, nil
}


func(u *AuthUsecase) GetCurrentUser(userID uuid.UUID)(*domain.User,error){
	user,err := u.userRepo.FindByID(userID)
	if err != nil {
		return nil,appErrors.NewUnauthorizedError("user not found")
	}

	if user.Status != "ACTIVE"{
		return nil,appErrors.NewUnauthorizedError("user account is not active")
	}
	return user,nil
}