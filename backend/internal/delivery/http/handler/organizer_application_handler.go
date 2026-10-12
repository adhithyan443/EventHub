package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

func (h *OrganizerApplicationHandler) SubmitApplication(ctx *gin.Context) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	if !strings.HasPrefix(ctx.GetHeader("Content-Type"), "multipart/form-data") {
		ctx.Error(appErrors.NewValidationError("multipart form data is required"))
		return
	}

	var req organizer.SubmitApplicationInput
	if err := ctx.ShouldBind(&req); err != nil {
		var validationErrs validator.ValidationErrors
		var failedFields []string
		if errors.As(err, &validationErrs) {
			failedFields = make([]string, 0, len(validationErrs))
			for _, fe := range validationErrs {
				failedFields = append(failedFields, fe.Field())
			}
		}

		h.logger.Warn(
			"organizer_application_validation_failed",
			"user_id", userID,
			"failed_fields", failedFields,
			"error", err,
		)
		ctx.Error(appErrors.NewValidationError("invalid application data"))
		return
	}

	// Verification document handling.
	// For new applications, the verification document is mandatory (enforced by usecase).
	// For resubmissions of existing rejected applications, providing a replacement document is optional;
	// if omitted, the usecase retains the previously uploaded document.
	documentHeader, err := ctx.FormFile("verification_document")
	if err == nil {
		if strings.TrimSpace(documentHeader.Filename) != "" && documentHeader.Size > 0 {
			documentFile, documentType, openErr := openValidatedUpload(
				documentHeader,
				10*1024*1024,
				map[string]bool{
					"application/pdf": true,
					"image/jpeg":      true,
					"image/png":       true,
				},
			)
			if openErr != nil {
				ctx.Error(appErrors.NewValidationError(openErr.Error()))
				return
			}
			defer documentFile.Close()

			req.VerificationDocumentReader = documentFile
			req.VerificationDocumentType = documentType
			req.VerificationDocumentSize = documentHeader.Size
		} else if strings.TrimSpace(documentHeader.Filename) != "" && documentHeader.Size <= 0 {
			ctx.Error(appErrors.NewValidationError("uploaded file is empty"))
			return
		}
	} else if err != http.ErrMissingFile {
		ctx.Error(appErrors.NewValidationError("invalid verification document upload"))
		return
	}

	// Organization logo is optional.
	logoHeader, err := ctx.FormFile("business_logo")
	if err == nil {
		// Only validate and upload if a file was actually selected.
		// When Postman or a browser form sends an empty file part, Filename is empty and Size is 0.
		if strings.TrimSpace(logoHeader.Filename) != "" && logoHeader.Size > 0 {
			logoFile, logoType, openErr := openValidatedUpload(
				logoHeader,
				5*1024*1024,
				map[string]bool{
					"image/jpeg": true,
					"image/png":  true,
					"image/webp": true,
				},
			)
			if openErr != nil {
				ctx.Error(appErrors.NewValidationError(openErr.Error()))
				return
			}
			defer logoFile.Close()

			req.LogoReader = logoFile
			req.LogoContentType = logoType
			req.LogoSize = logoHeader.Size
		} else if strings.TrimSpace(logoHeader.Filename) != "" && logoHeader.Size <= 0 {
			// A file was explicitly named, but its content is 0 bytes.
			ctx.Error(appErrors.NewValidationError("uploaded file is empty"))
			return
		}
	} else if err != http.ErrMissingFile {
		ctx.Error(appErrors.NewValidationError("invalid organization logo upload"))
		return
	}

	if err := h.applicationUsecase.SubmitApplication(ctx.Request.Context(), userID, req); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "organizer application submitted successfully",
	})
}

func getAuthenticatedUserID(ctx *gin.Context) (uuid.UUID, error) {
	userIDValue, exists := ctx.Get("user_id")
	if !exists {
		return uuid.Nil, appErrors.NewUnauthorizedError("unauthorized")
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return uuid.Nil, appErrors.NewUnauthorizedError(
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
			ctx.Error(appErrors.NewValidationError("invalid page"))
			return
		}

		page = parsedPage
	}

	if value := ctx.Query("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil {
			ctx.Error(appErrors.NewValidationError("invalid limit"))
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

func (h *OrganizerApplicationHandler) GetApplication(ctx *gin.Context) {
	// Get the application ID from the URL parameter.
	idParam := ctx.Param("id")

	// Convert the string ID into a UUID.
	applicationID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid application id"))
		return
	}

	// Retrieve the application through the admin usecase.
	application, err := h.adminApplicationUsecase.GetApplication(ctx.Request.Context(), applicationID)
	if err != nil {
		ctx.Error(err)
		return
	}

	// Convert the domain entity into the safe admin response DTO.
	response := organizer.ToAdminApplicationResponse(application)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *OrganizerApplicationHandler) ApproveApplication(ctx *gin.Context) {

	idParam := ctx.Param("id")

	applicationID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(appErrors.NewValidationError("invalid application id"))
		return
	}

	if err := h.adminApplicationUsecase.ApproveApplication(applicationID); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "organizer application approved successfully",
	})
}

func (h *OrganizerApplicationHandler) GetMyApplication(ctx *gin.Context) {
	userIDValue, exists := ctx.Get("user_id")
	if !exists {
		ctx.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		ctx.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	application, err := h.applicationUsecase.GetApplication(ctx.Request.Context(), userID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    application,
	})
}

func (h *OrganizerApplicationHandler) RejectApplication(
	ctx *gin.Context,
) {
	// Read the application UUID from the URL.
	idParam := ctx.Param("id")

	applicationID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.Error(
			appErrors.NewValidationError("invalid application id"),
		)
		return
	}

	// Define the rejection request body.
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	// Validate the request body.
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(
			"admin_organizer_application_rejection_validation_failed",
			"application_id", applicationID,
			"error", err,
		)

		ctx.Error(
			appErrors.NewValidationError(
				"rejection reason is required",
			),
		)
		return
	}

	// Reject the application through the admin usecase.
	if err := h.adminApplicationUsecase.RejectApplication(
		applicationID,
		req.Reason,
	); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "organizer application rejected successfully",
	})
}

func openValidatedUpload(
	header *multipart.FileHeader,
	maxSize int64,
	allowedTypes map[string]bool,
) (multipart.File, string, error) {
	if header == nil {
		return nil, "", fmt.Errorf("uploaded file is required")
	}

	if header.Size <= 0 {
		return nil, "", fmt.Errorf("uploaded file is empty")
	}

	if header.Size > maxSize {
		return nil, "", fmt.Errorf("uploaded file exceeds the size limit")
	}

	file, err := header.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to read uploaded file")
	}

	var buffer [512]byte
	n, readErr := file.Read(buffer[:])
	if readErr != nil && readErr != io.EOF {
		file.Close()
		return nil, "", fmt.Errorf("failed to inspect uploaded file")
	}

	contentType := http.DetectContentType(buffer[:n])
	if !allowedTypes[contentType] {
		file.Close()
		return nil, "", fmt.Errorf("unsupported uploaded file type")
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, "", fmt.Errorf("failed to reset uploaded file")
	}

	return file, contentType, nil
}
