package pricingtax

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PricingTaxUsecase struct {
	txManager domain.TransactionManager
	logger    *slog.Logger
}

func NewPricingTaxUsecase(
	txManager domain.TransactionManager,
	logger *slog.Logger,
) *PricingTaxUsecase {
	return &PricingTaxUsecase{
		txManager: txManager,
		logger:    logger,
	}
}

type PublishRequest struct {
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
	AdminUserID              uuid.UUID       `json:"-"`
}

type ScheduleRequest struct {
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
	AdminUserID              uuid.UUID       `json:"-"`
}

func (u *PricingTaxUsecase) GetActiveConfiguration(ctx context.Context, asOf ...time.Time) (*domain.PricingTaxConfiguration, error) {
	evalTime := time.Now().UTC()
	if len(asOf) > 0 && !asOf[0].IsZero() {
		evalTime = asOf[0].UTC()
	}
	var config *domain.PricingTaxConfiguration
	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		var err error
		config, err = tx.PricingTaxConfigurationRepository().FindActiveAt(ctx, evalTime)
		return err
	})
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (u *PricingTaxUsecase) GetTimeline(ctx context.Context, asOf ...time.Time) ([]domain.PricingTaxConfiguration, error) {
	evalTime := time.Now().UTC()
	if len(asOf) > 0 && !asOf[0].IsZero() {
		evalTime = asOf[0].UTC()
	}
	var timeline []domain.PricingTaxConfiguration
	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		var err error
		timeline, err = tx.PricingTaxConfigurationRepository().FindTimeline(ctx, evalTime)
		return err
	})
	if err != nil {
		return nil, err
	}
	return timeline, nil
}

func (u *PricingTaxUsecase) GetHistory(ctx context.Context, page, limit int, asOf ...time.Time) ([]domain.PricingTaxConfiguration, int64, error) {
	evalTime := time.Now().UTC()
	if len(asOf) > 0 && !asOf[0].IsZero() {
		evalTime = asOf[0].UTC()
	}
	var history []domain.PricingTaxConfiguration
	var total int64
	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		var err error
		history, total, err = tx.PricingTaxConfigurationRepository().FindHistory(ctx, page, limit, evalTime)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	return history, total, nil
}

func (u *PricingTaxUsecase) PublishImmediate(ctx context.Context, req PublishRequest) (*domain.PricingTaxConfiguration, error) {
	if err := validateRates(req.PlatformFeePercent, req.TaxPercent); err != nil {
		return nil, err
	}
	if err := validateTaxGate(req.TaxEnabled, req.TaxRulesValidated, req.TaxBasis, req.TaxLegalSignoffReference); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ChangeReason) == "" {
		return nil, domain.ErrInvalidPricingConfiguration
	}

	var createdConfig *domain.PricingTaxConfiguration

	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		repo := tx.PricingTaxConfigurationRepository()

		// 1. Transaction-level serialization
		if err := repo.AcquireAdvisoryLock(ctx); err != nil {
			return err
		}

		// 2. Concurrency check
		latest, err := repo.FindLatest(ctx)
		if err != nil {
			return err
		}
		if latest.Version != req.ExpectedBaseVersion {
			return domain.ErrConcurrentConfigurationModification
		}

		now := time.Now().UTC()
		timeline, err := repo.FindTimeline(ctx, now)
		if err != nil {
			return err
		}

		// Find currently active configuration
		var activeIdx = -1
		for i, c := range timeline {
			if !c.EffectiveFrom.After(now) && (c.EffectiveTo == nil || c.EffectiveTo.After(now)) {
				activeIdx = i
				break
			}
		}
		if activeIdx == -1 {
			return domain.ErrPricingConfigurationUnavailable
		}

		activeConfig := timeline[activeIdx]

		if req.ReplaceFutureSchedules {
			// Cancel all scheduled future configurations
			for i := activeIdx + 1; i < len(timeline); i++ {
				fut := timeline[i]
				fut.Status = domain.PricingTaxStatusCancelled
				fut.CancelledAt = &now
				fut.CancelledBy = &req.AdminUserID
				fut.ChangeReason = "Superseded by immediate configuration update: " + req.ChangeReason
				fut.UpdatedAt = now
				if err := repo.Update(ctx, &fut); err != nil {
					return err
				}
			}

			// Cap active configuration at now
			activeConfig.EffectiveTo = &now
			activeConfig.Status = domain.PricingTaxStatusSuperseded
			activeConfig.UpdatedAt = now
			if err := repo.Update(ctx, &activeConfig); err != nil {
				return err
			}

			// Insert new open-ended configuration
			createdConfig = &domain.PricingTaxConfiguration{
				ID:                       uuid.New(),
				Version:                  latest.Version + 1,
				Status:                   domain.PricingTaxStatusActive,
				PlatformFeeEnabled:       req.PlatformFeeEnabled,
				PlatformFeePercent:       req.PlatformFeePercent,
				TaxEnabled:               req.TaxEnabled,
				TaxPercent:               req.TaxPercent,
				TaxBasis:                 req.TaxBasis,
				TaxRulesValidated:        req.TaxRulesValidated,
				TaxLegalSignoffReference: req.TaxLegalSignoffReference,
				FreeTicketExempt:         req.FreeTicketExempt,
				EffectiveFrom:            now,
				EffectiveTo:              nil,
				CreatedBy:                &req.AdminUserID,
				ChangeReason:             req.ChangeReason,
				CreatedAt:                now,
				UpdatedAt:                now,
			}
			return repo.Create(ctx, createdConfig)
		}

		// Default: Preserve future schedules
		var nextScheduledFrom *time.Time
		if activeIdx+1 < len(timeline) {
			nextScheduledFrom = &timeline[activeIdx+1].EffectiveFrom
		}

		// Cap active configuration at now
		activeConfig.EffectiveTo = &now
		activeConfig.Status = domain.PricingTaxStatusSuperseded
		activeConfig.UpdatedAt = now
		if err := repo.Update(ctx, &activeConfig); err != nil {
			return err
		}

		// Insert new configuration (bounded by next schedule if one exists, otherwise open-ended)
		createdConfig = &domain.PricingTaxConfiguration{
			ID:                       uuid.New(),
			Version:                  latest.Version + 1,
			Status:                   domain.PricingTaxStatusActive,
			PlatformFeeEnabled:       req.PlatformFeeEnabled,
			PlatformFeePercent:       req.PlatformFeePercent,
			TaxEnabled:               req.TaxEnabled,
			TaxPercent:               req.TaxPercent,
			TaxBasis:                 req.TaxBasis,
			TaxRulesValidated:        req.TaxRulesValidated,
			TaxLegalSignoffReference: req.TaxLegalSignoffReference,
			FreeTicketExempt:         req.FreeTicketExempt,
			EffectiveFrom:            now,
			EffectiveTo:              nextScheduledFrom,
			CreatedBy:                &req.AdminUserID,
			ChangeReason:             req.ChangeReason,
			CreatedAt:                now,
			UpdatedAt:                now,
		}

		return repo.Create(ctx, createdConfig)
	})

	if err != nil {
		u.logger.ErrorContext(ctx, "failed to publish immediate pricing configuration",
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "published immediate pricing configuration",
		slog.Int("version", createdConfig.Version),
		slog.Time("effective_from", createdConfig.EffectiveFrom),
	)

	return createdConfig, nil
}

func (u *PricingTaxUsecase) ScheduleFuture(ctx context.Context, req ScheduleRequest) (*domain.PricingTaxConfiguration, error) {
	now := time.Now().UTC()
	if !req.EffectiveFrom.After(now) {
		return nil, domain.ErrInvalidEffectiveDate
	}
	if err := validateRates(req.PlatformFeePercent, req.TaxPercent); err != nil {
		return nil, err
	}
	if err := validateTaxGate(req.TaxEnabled, req.TaxRulesValidated, req.TaxBasis, req.TaxLegalSignoffReference); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ChangeReason) == "" {
		return nil, domain.ErrInvalidPricingConfiguration
	}

	var createdConfig *domain.PricingTaxConfiguration

	err := u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		repo := tx.PricingTaxConfigurationRepository()

		if err := repo.AcquireAdvisoryLock(ctx); err != nil {
			return err
		}

		latest, err := repo.FindLatest(ctx)
		if err != nil {
			return err
		}
		if latest.Version != req.ExpectedBaseVersion {
			return domain.ErrConcurrentConfigurationModification
		}

		timeline, err := repo.FindTimeline(ctx, now)
		if err != nil {
			return err
		}

		// Find configuration spanning req.EffectiveFrom
		var spanningIdx = -1
		for i, c := range timeline {
			if !c.EffectiveFrom.After(req.EffectiveFrom) && (c.EffectiveTo == nil || c.EffectiveTo.After(req.EffectiveFrom)) {
				spanningIdx = i
				break
			}
		}
		if spanningIdx == -1 {
			return domain.ErrPricingTimelineDiscontinuous
		}

		spanning := timeline[spanningIdx]
		// Cannot schedule at the exact boundary of an existing configuration
		if spanning.EffectiveFrom.Equal(req.EffectiveFrom) {
			return domain.ErrInvalidEffectiveDate
		}

		existingEnd := spanning.EffectiveTo

		// Cap spanning predecessor
		spanning.EffectiveTo = &req.EffectiveFrom
		spanning.UpdatedAt = now
		if err := repo.Update(ctx, &spanning); err != nil {
			return err
		}

		// Insert new scheduled configuration
		createdConfig = &domain.PricingTaxConfiguration{
			ID:                       uuid.New(),
			Version:                  latest.Version + 1,
			Status:                   domain.PricingTaxStatusScheduled,
			PlatformFeeEnabled:       req.PlatformFeeEnabled,
			PlatformFeePercent:       req.PlatformFeePercent,
			TaxEnabled:               req.TaxEnabled,
			TaxPercent:               req.TaxPercent,
			TaxBasis:                 req.TaxBasis,
			TaxRulesValidated:        req.TaxRulesValidated,
			TaxLegalSignoffReference: req.TaxLegalSignoffReference,
			FreeTicketExempt:         req.FreeTicketExempt,
			EffectiveFrom:            req.EffectiveFrom,
			EffectiveTo:              existingEnd,
			CreatedBy:                &req.AdminUserID,
			ChangeReason:             req.ChangeReason,
			CreatedAt:                now,
			UpdatedAt:                now,
		}

		return repo.Create(ctx, createdConfig)
	})

	if err != nil {
		u.logger.ErrorContext(ctx, "failed to schedule pricing configuration",
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "scheduled future pricing configuration",
		slog.Int("version", createdConfig.Version),
		slog.Time("effective_from", createdConfig.EffectiveFrom),
	)

	return createdConfig, nil
}

func (u *PricingTaxUsecase) CancelScheduled(
	ctx context.Context,
	configID uuid.UUID,
	adminID uuid.UUID,
	reason string,
	expectedBaseVersion int,
) error {
	if strings.TrimSpace(reason) == "" {
		return domain.ErrInvalidPricingConfiguration
	}

	return u.txManager.WithinTransaction(func(tx domain.TransactionRepositories) error {
		repo := tx.PricingTaxConfigurationRepository()

		if err := repo.AcquireAdvisoryLock(ctx); err != nil {
			return err
		}

		latest, err := repo.FindLatest(ctx)
		if err != nil {
			return err
		}
		if latest.Version != expectedBaseVersion {
			return domain.ErrConcurrentConfigurationModification
		}

		now := time.Now().UTC()
		target, err := repo.FindByID(ctx, configID)
		if err != nil {
			return err
		}

		if target.ResolveStatus(now) != domain.PricingTaxStatusScheduled {
			return domain.ErrCannotCancelActiveConfiguration
		}

		timeline, err := repo.FindTimeline(ctx, now)
		if err != nil {
			return err
		}

		// Find predecessor whose effective_to matches target.EffectiveFrom
		var predIdx = -1
		for i, c := range timeline {
			if c.EffectiveTo != nil && c.EffectiveTo.Equal(target.EffectiveFrom) {
				predIdx = i
				break
			}
		}
		if predIdx == -1 {
			return domain.ErrPricingTimelineDiscontinuous
		}

		pred := timeline[predIdx]

		// Restore predecessor's effective_to to target's effective_to (seamlessly connects tail or bounded schedule)
		pred.EffectiveTo = target.EffectiveTo
		pred.UpdatedAt = now
		if err := repo.Update(ctx, &pred); err != nil {
			return err
		}

		// Cancel target
		target.Status = domain.PricingTaxStatusCancelled
		target.CancelledAt = &now
		target.CancelledBy = &adminID
		target.ChangeReason = "Cancelled: " + reason
		target.UpdatedAt = now

		return repo.Update(ctx, target)
	})
}

func validateRates(platformFeePercent, taxPercent decimal.Decimal) error {
	hundred := decimal.NewFromInt(100)
	if platformFeePercent.IsNegative() || platformFeePercent.GreaterThan(hundred) {
		return domain.ErrInvalidPricingConfiguration
	}
	if taxPercent.IsNegative() || taxPercent.GreaterThan(hundred) {
		return domain.ErrInvalidPricingConfiguration
	}
	return nil
}

func validateTaxGate(taxEnabled, taxRulesValidated bool, taxBasis string, legalRef *string) error {
	if !taxEnabled {
		return nil
	}
	if !taxRulesValidated || taxBasis == domain.TaxBasisUnvalidated || legalRef == nil || strings.TrimSpace(*legalRef) == "" {
		return domain.ErrTaxBasisUnvalidated
	}
	switch taxBasis {
	case domain.TaxBasisPlatformFeeOnly, domain.TaxBasisSubtotalOnly, domain.TaxBasisTotalBeforeTax:
		return nil
	default:
		return domain.ErrTaxBasisUnvalidated
	}
}
