package handler

import (
	"log/slog"
	"net/http"

	"github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrganizerProfileHandler struct {
	usecase *organizer.ProfileUsecase
	logger  *slog.Logger
}

func NewOrganizerProfileHandler(
	usecase *organizer.ProfileUsecase,
	logger *slog.Logger,
) *OrganizerProfileHandler {
	return &OrganizerProfileHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *OrganizerProfileHandler) GetProfile(
	ctx *gin.Context,
) {
	userIDValue, exists := ctx.Get("user_id")
	if !exists {
		ctx.Error(
			errors.NewUnauthorizedError("unauthorized"),
		)
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		ctx.Error(
			errors.NewUnauthorizedError("invalid user identity"),
		)
		return
	}

	profile, err := h.usecase.GetProfile(userID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    profile,
	})
}
