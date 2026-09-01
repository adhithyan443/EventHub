package handler

import (
	"log/slog"
	"net/http"

	"github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrganizerApplicationHandler struct {
	applicationUsecase *organizer.ApplicationUsecase
	logger             *slog.Logger
}

func NewOrganizerApplicationHandler(
	applicationUsecase *organizer.ApplicationUsecase,
	logger *slog.Logger,

) *OrganizerApplicationHandler {
	return &OrganizerApplicationHandler{
		applicationUsecase: applicationUsecase,
		logger:             logger,
	}
}

func (h *OrganizerApplicationHandler) SubmitApplication(
	ctx *gin.Context,
) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	var req organizer.SubmitApplicationInput

	// if err := ctx.ShouldBindJSON(&req); err != nil {
	// 	ctx.JSON(http.StatusBadRequest, gin.H{
	// 		"code":    "VALIDATION_ERROR",
	// 		"message": "invalid request data",
	// 	})
	// 	return
	// }

	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(
			"organizer_application_validation_failed",
			"error", err,
		)

		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request data",
		})
		return
	}

	if err := h.applicationUsecase.SubmitApplication(userID, req); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "organizer application submitted successfully",
	})
}

func getAuthenticatedUserID(ctx *gin.Context) (uuid.UUID, error) {
	userIDValue, exists := ctx.Get("user_id")
	if !exists {
		return uuid.Nil, errors.NewUnauthorizedError("unauthorized")
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.NewUnauthorizedError(
			"invalid user identity",
		)
	}

	return userID, nil
}
