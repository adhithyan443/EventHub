package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/pricingtax"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AdminPricingTaxHandler struct {
	usecase *pricingtax.PricingTaxUsecase
	logger  *slog.Logger
}

func NewAdminPricingTaxHandler(
	usecase *pricingtax.PricingTaxUsecase,
	logger *slog.Logger,
) *AdminPricingTaxHandler {
	return &AdminPricingTaxHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *AdminPricingTaxHandler) GetCurrentAndTimeline(c *gin.Context) {
	ctx := c.Request.Context()

	active, err := h.usecase.GetActiveConfiguration(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrPricingConfigurationUnavailable) {
			c.Error(appErrors.NewInternalError("pricing configuration is unavailable"))
			return
		}
		c.Error(err)
		return
	}

	timeline, err := h.usecase.GetTimeline(ctx)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active":   active,
			"timeline": timeline,
		},
	})
}

func (h *AdminPricingTaxHandler) GetHistory(c *gin.Context) {
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	history, total, err := h.usecase.GetHistory(ctx, page, limit)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"history": history,
			"total":   total,
			"page":    page,
			"limit":   limit,
		},
	})
}

type publishImmediateBody struct {
	ExpectedBaseVersion      int             `json:"expected_base_version"`
	PlatformFeeEnabled       bool            `json:"platform_fee_enabled"`
	PlatformFeePercent       decimal.Decimal `json:"platform_fee_percent"`
	TaxEnabled               bool            `json:"tax_enabled"`
	TaxPercent               decimal.Decimal `json:"tax_percent"`
	TaxBasis                 string          `json:"tax_basis"`
	TaxRulesValidated        bool            `json:"tax_rules_validated"`
	TaxLegalSignoffReference *string         `json:"tax_legal_signoff_reference"`
	FreeTicketExempt         bool            `json:"free_ticket_exempt"`
	ReplaceFutureSchedules   bool            `json:"replace_future_schedules"`
	ChangeReason             string          `json:"change_reason"`
}

func (h *AdminPricingTaxHandler) PublishImmediate(c *gin.Context) {
	userID, ok := getAdminUserID(c)
	if !ok {
		c.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	var body publishImmediateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	ctx := c.Request.Context()

	created, err := h.usecase.PublishImmediate(ctx, pricingtax.PublishRequest{
		ExpectedBaseVersion:      body.ExpectedBaseVersion,
		PlatformFeeEnabled:       body.PlatformFeeEnabled,
		PlatformFeePercent:       body.PlatformFeePercent,
		TaxEnabled:               body.TaxEnabled,
		TaxPercent:               body.TaxPercent,
		TaxBasis:                 body.TaxBasis,
		TaxRulesValidated:        body.TaxRulesValidated,
		TaxLegalSignoffReference: body.TaxLegalSignoffReference,
		FreeTicketExempt:         body.FreeTicketExempt,
		ReplaceFutureSchedules:   body.ReplaceFutureSchedules,
		ChangeReason:             body.ChangeReason,
		AdminUserID:              userID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"configuration": created,
		},
	})
}

type scheduleFutureBody struct {
	ExpectedBaseVersion      int             `json:"expected_base_version"`
	PlatformFeeEnabled       bool            `json:"platform_fee_enabled"`
	PlatformFeePercent       decimal.Decimal `json:"platform_fee_percent"`
	TaxEnabled               bool            `json:"tax_enabled"`
	TaxPercent               decimal.Decimal `json:"tax_percent"`
	TaxBasis                 string          `json:"tax_basis"`
	TaxRulesValidated        bool            `json:"tax_rules_validated"`
	TaxLegalSignoffReference *string         `json:"tax_legal_signoff_reference"`
	FreeTicketExempt         bool            `json:"free_ticket_exempt"`
	EffectiveFrom            time.Time       `json:"effective_from"`
	ChangeReason             string          `json:"change_reason"`
}

func (h *AdminPricingTaxHandler) ScheduleFuture(c *gin.Context) {
	userID, ok := getAdminUserID(c)
	if !ok {
		c.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	var body scheduleFutureBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	ctx := c.Request.Context()

	created, err := h.usecase.ScheduleFuture(ctx, pricingtax.ScheduleRequest{
		ExpectedBaseVersion:      body.ExpectedBaseVersion,
		PlatformFeeEnabled:       body.PlatformFeeEnabled,
		PlatformFeePercent:       body.PlatformFeePercent,
		TaxEnabled:               body.TaxEnabled,
		TaxPercent:               body.TaxPercent,
		TaxBasis:                 body.TaxBasis,
		TaxRulesValidated:        body.TaxRulesValidated,
		TaxLegalSignoffReference: body.TaxLegalSignoffReference,
		FreeTicketExempt:         body.FreeTicketExempt,
		EffectiveFrom:            body.EffectiveFrom,
		ChangeReason:             body.ChangeReason,
		AdminUserID:              userID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"configuration": created,
		},
	})
}

type cancelScheduledBody struct {
	ExpectedBaseVersion int    `json:"expected_base_version"`
	Reason              string `json:"reason"`
}

func (h *AdminPricingTaxHandler) CancelScheduled(c *gin.Context) {
	userID, ok := getAdminUserID(c)
	if !ok {
		c.Error(appErrors.NewUnauthorizedError("unauthorized"))
		return
	}

	configID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(appErrors.NewValidationError("invalid configuration id"))
		return
	}

	var body cancelScheduledBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(appErrors.NewValidationError("invalid request body"))
		return
	}

	ctx := c.Request.Context()

	if err := h.usecase.CancelScheduled(ctx, configID, userID, body.Reason, body.ExpectedBaseVersion); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "scheduled configuration cancelled successfully",
	})
}

func getAdminUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok && id != uuid.Nil
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrConcurrentConfigurationModification):
		c.Error(appErrors.NewConflictError("The configuration was modified concurrently by another administrator. Please reload and try again."))
	case errors.Is(err, domain.ErrInvalidPricingConfiguration):
		c.Error(appErrors.NewValidationError("Invalid configuration parameters: verify rates, positive numbers, and mandatory change reason."))
	case errors.Is(err, domain.ErrTaxBasisUnvalidated):
		c.Error(appErrors.NewValidationError("Tax cannot be enabled until the tax basis is selected, validated, and confirmed with a legal sign-off reference."))
	case errors.Is(err, domain.ErrInvalidEffectiveDate):
		c.Error(appErrors.NewValidationError("Effective date for scheduled configurations must be in the future."))
	case errors.Is(err, domain.ErrCannotCancelActiveConfiguration):
		c.Error(appErrors.NewValidationError("Cannot cancel a configuration that is active or in the past."))
	case errors.Is(err, domain.ErrPricingTimelineDiscontinuous):
		c.Error(appErrors.NewInternalError("Pricing timeline continuity error."))
	default:
		c.Error(err)
	}
}
