package handler

import (
	"crypto/subtle"
	"net/http"

	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authUsecase *auth.AuthUsecase
}

func NewAuthHandler(authUsecase *auth.AuthUsecase) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
	}
}

type registerRequest struct {
	FullName string `json:"fullName" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Phone    string `json:"phone" binding:"required"`
}

func (h *AuthHandler) Register(ctx *gin.Context) {
	var req registerRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request data",
		})
		return
	}

	err := h.authUsecase.Register(auth.RegisterInput{
		FullName: req.FullName,
		Email:    req.Email,
		Password: req.Password,
		Phone:    req.Phone,
	})

	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "verification OTP sent",
		"email":   req.Email,
	},
	)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(ctx *gin.Context) {

	var req loginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request data",
		})
		return
	}

	user, accessToken, refreshToken, err := h.authUsecase.Login(auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "login successful",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"fullName": user.FullName,
			"email":    user.Email,
			"phone":    user.Phone,
			"role":     user.Role,
			"status":   user.Status,
		},
	})
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func (h *AuthHandler) RefreshToken(ctx *gin.Context) {
	var req refreshTokenRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request data",
		})
		return
	}

	accessToken, refreshToken, err := h.authUsecase.RefreshToken(
		auth.RefreshTokenInput{
			RefreshToken: req.RefreshToken,
		},
	)

	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func (h *AuthHandler) Logout(ctx *gin.Context) {
	var req logoutRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request data",
		})
		return
	}

	userIDvalue, exists := ctx.Get("user_id")
	if !exists {
		ctx.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	userID, ok := userIDvalue.(uuid.UUID)
	if !ok {
		ctx.Error(appErrors.NewUnauthorizedError("invalid user identity"))
		return
	}

	if err := h.authUsecase.Logout(
		userID,
		auth.LogoutInput{
			RefreshToken: req.RefreshToken,
		},
	); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "logout successful",
	})
}

type verifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

func (h *AuthHandler) VerifyOTP(ctx *gin.Context) {
	var req verifyOTPRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request data",
		})
		return
	}

	user, err := h.authUsecase.VerifyOTP(auth.VerifyOTPInput{
		Email: req.Email,
		OTP:   req.OTP,
	})

	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "registration verified successfully",
		"user": gin.H{
			"id":       user.ID,
			"fullName": user.FullName,
			"email":    user.Email,
			"phone":    user.Phone,
			"role":     user.Role,
			"status":   user.Status,
		},
	})
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) ForgotPassword(ctx *gin.Context) {

	var req forgotPasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request data",
		})
		return
	}

	err := h.authUsecase.ForgotPassword(auth.ForgotPasswordInput{
		Email: req.Email,
	})

	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "If the mail is registered, a password reset link has been sent",
	})
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

func (h *AuthHandler) ResetPassword(ctx *gin.Context) {
	var req resetPasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request data",
		})
		return
	}

	err := h.authUsecase.ResetPassword(auth.ResetPasswordInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})

	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "password has been reset successfully",
	})
}

func (h *AuthHandler) GoogleLogin(ctx *gin.Context) {
	state, err := generateOAuthStatae()

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "failed to start google authentication",
		})
		return
	}

	ctx.SetCookie(
		"oauth_state",
		state,
		300,
		"/",
		"",
		false,
		true,
	)

	authURL := h.authUsecase.GetGoogleAuthURL(state)

	ctx.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *AuthHandler) GoogleCallback(ctx *gin.Context) {

	expectedState, err := ctx.Cookie("oauth_state")
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "invalid google authentication state",
		})
		return
	}

	returnedState := ctx.Query("state")

	if returnedState == "" || subtle.ConstantTimeCompare(
		[]byte(returnedState),
		[]byte(expectedState),
	) != 1 {

		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "invalid google authentication state",
		})

		return
	}

	ctx.SetCookie( //Deleting the cookie after verify
		"oauth_state",
		"",
		-1,
		"/",
		"",
		false,
		true,
	)

	code := ctx.Query("code")

	user, accessToken, refreshToken, err :=
		h.authUsecase.HandleGoogleCallback(code)

	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "google login successful",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"fullName": user.FullName,
			"email":    user.Email,
			"phone":    user.Phone,
			"role":     user.Role,
			"status":   user.Status,
		},
	})
}
