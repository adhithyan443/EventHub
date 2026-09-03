package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrganizerApplicationHandler struct {
	applicationUsecase      *organizer.ApplicationUsecase
	adminApplicationUsecase *organizer.AdminApplicationUsecase
	logger                  *slog.Logger
}

func NewOrganizerApplicationHandler(
	applicationUsecase *organizer.ApplicationUsecase,
	adminApplicationUsecase *organizer.AdminApplicationUsecase,
	logger *slog.Logger,

) *OrganizerApplicationHandler {
	return &OrganizerApplicationHandler{
		applicationUsecase:      applicationUsecase,
		adminApplicationUsecase: adminApplicationUsecase,
		logger:                  logger,
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

func (h *OrganizerApplicationHandler) ListApplications(ctx *gin.Context) {
	page := 1
	limit := 10

	if value := ctx.Query("page"); value != "" {
		parsedPage, err := strconv.Atoi(value)
		if err != nil {
			ctx.Error(errors.NewValidationError("invalid page"))
			return
		}

		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil {
			ctx.Error(errors.NewValidationError("invalid limit"))
			return
		}

		limit = parsedLimit
	}

	status := ctx.Query("status")

	result, err := h.adminApplicationUsecase.ListApplications(
		page,
		limit,
		status,
	)
	if err != nil {
		ctx.Error(err)
		return
	}

	applications := make(
		[]organizer.AdminApplicationResponse,
		0,
		len(result.Applications),
	)

	for _, application := range result.Applications {
		applications = append(
			applications,
			organizer.ToAdminApplicationResponse(application),
		)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"applications": applications,
			"page":         result.Page,
			"limit":        result.Limit,
			"total":        result.Total,
		},
	})
}
