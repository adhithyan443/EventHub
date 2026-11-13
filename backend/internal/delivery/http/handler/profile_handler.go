package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/profile"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProfileHandler struct {
	usecase *profile.ProfileUsecase
	logger  *slog.Logger
}

func NewProfileHandler(
	usecase *profile.ProfileUsecase,
	logger *slog.Logger,
) *ProfileHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ProfileHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *ProfileHandler) GetProfile(ctx *gin.Context) {
	userID, ok := h.getUserID(ctx)
	if !ok {
		return
	}

	res, err := h.usecase.GetProfile(ctx.Request.Context(), userID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res,
	})
}

type updateProfileRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone"`
}

func (h *ProfileHandler) UpdateProfile(ctx *gin.Context) {
	userID, ok := h.getUserID(ctx)
	if !ok {
		return
	}

	var req updateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "full_name is required",
		})
		return
	}

	res, err := h.usecase.UpdateProfile(ctx.Request.Context(), userID, profile.UpdateProfileInput{
		FullName: req.FullName,
		Phone:    req.Phone,
	})
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Profile updated successfully",
		"data":    res,
	})
}

type requestEmailChangeRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"`
}

func (h *ProfileHandler) RequestEmailChange(ctx *gin.Context) {
	userID, ok := h.getUserID(ctx)
	if !ok {
		return
	}

	var req requestEmailChangeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "valid new_email is required",
		})
		return
	}

	res, err := h.usecase.RequestEmailChange(ctx.Request.Context(), userID, req.NewEmail)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Verification code sent to your new email address",
		"data":    res,
	})
}

type resendEmailChangeRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"`
}

func (h *ProfileHandler) ResendEmailChangeOTP(ctx *gin.Context) {
	userID, ok := h.getUserID(ctx)
	if !ok {
		return
	}

	var req resendEmailChangeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "valid new_email is required",
		})
		return
	}

	res, err := h.usecase.ResendEmailChangeOTP(ctx.Request.Context(), userID, req.NewEmail)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Verification code resent successfully",
		"data":    res,
	})
}

type verifyEmailChangeRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"`
	OTP      string `json:"otp" binding:"required"`
}

func (h *ProfileHandler) VerifyEmailChange(ctx *gin.Context) {
	userID, ok := h.getUserID(ctx)
	if !ok {
		return
	}

	var req verifyEmailChangeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "valid new_email and otp are required",
		})
		return
	}

	res, err := h.usecase.VerifyEmailChange(ctx.Request.Context(), userID, req.NewEmail, req.OTP)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Email address updated successfully",
		"data":    res,
	})
}

func (h *ProfileHandler) UploadAvatar(ctx *gin.Context) {
	userID, ok := h.getUserID(ctx)
	if !ok {
		return
	}

	fileHeader, err := ctx.FormFile("avatar")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "avatar image file is required in 'avatar' field",
		})
		return
	}

	if fileHeader.Size > 5*1024*1024 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "profile picture must not exceed 5 MB",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error("avatar_file_open_failed", "user_id", userID, "error", err)
		ctx.Error(appErrors.NewInternalError("failed to read uploaded file"))
		return
	}
	defer file.Close()

	contentType, err := detectAvatarContentType(file)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "invalid avatar image file",
		})
		return
	}

	res, err := h.usecase.UploadAvatar(ctx.Request.Context(), userID, file, contentType, fileHeader.Size)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Profile picture updated successfully",
		"data":    res,
	})
}

func (h *ProfileHandler) RemoveAvatar(ctx *gin.Context) {
	userID, ok := h.getUserID(ctx)
	if !ok {
		return
	}

	res, err := h.usecase.RemoveAvatar(ctx.Request.Context(), userID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Profile picture removed successfully",
		"data":    res,
	})
}

func (h *ProfileHandler) getUserID(ctx *gin.Context) (uuid.UUID, bool) {
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return uuid.Nil, false
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		ctx.Error(appErrors.NewUnauthorizedError("invalid user identity"))
		return uuid.Nil, false
	}

	return userID, true
}

func detectAvatarContentType(file io.ReadSeeker) (string, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if n == 0 {
		return "", errors.New("empty file")
	}

	contentType := http.DetectContentType(buffer[:n])

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return contentType, nil
}
