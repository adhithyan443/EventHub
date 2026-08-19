package handler

import (
	"net/http"

	"github.com/adhithyan443/EventHub/backend/internal/usecase/auth"
	"github.com/gin-gonic/gin"
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

	user, err := h.authUsecase.Register(auth.RegisterInput{
		FullName: req.FullName,
		Email:    req.Email,
		Password: req.Password,
		Phone:    req.Phone,
	})

	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
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
