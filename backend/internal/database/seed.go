package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrFinanceAccountConfigMismatch = errors.New("finance account configuration mismatch")

// ExpectedFinanceAccount defines the required configuration for a seeded financial ledger account.
type ExpectedFinanceAccount struct {
	Code          string
	Name          string
	Type          finance.AccountType
	NormalBalance finance.NormalBalance
	Currency      string
	IsActive      bool
}

// StandardFinanceAccounts defines the approved standard chart of accounts for EventHub.
var StandardFinanceAccounts = []ExpectedFinanceAccount{
	{
		Code:          finance.AccountCodeRazorpayClearing,
		Name:          "Razorpay Clearing",
		Type:          finance.AccountTypeAsset,
		NormalBalance: finance.NormalBalanceDebit,
		Currency:      finance.CurrencyINR,
		IsActive:      true,
	},
	{
		Code:          finance.AccountCodeOperatingBank,
		Name:          "Operating Bank",
		Type:          finance.AccountTypeAsset,
		NormalBalance: finance.NormalBalanceDebit,
		Currency:      finance.CurrencyINR,
		IsActive:      true,
	},
	{
		Code:          finance.AccountCodeOrganizerPayable,
		Name:          "Organizer Payable",
		Type:          finance.AccountTypeLiability,
		NormalBalance: finance.NormalBalanceCredit,
		Currency:      finance.CurrencyINR,
		IsActive:      true,
	},
	{
		Code:          finance.AccountCodeCustomerRefundPayable,
		Name:          "Customer Refund Payable",
		Type:          finance.AccountTypeLiability,
		NormalBalance: finance.NormalBalanceCredit,
		Currency:      finance.CurrencyINR,
		IsActive:      true,
	},
	{
		Code:          finance.AccountCodeTaxPayable,
		Name:          "Tax Payable",
		Type:          finance.AccountTypeLiability,
		NormalBalance: finance.NormalBalanceCredit,
		Currency:      finance.CurrencyINR,
		IsActive:      true,
	},
	{
		Code:          finance.AccountCodeRetainedEarnings,
		Name:          "Retained Earnings",
		Type:          finance.AccountTypeEquity,
		NormalBalance: finance.NormalBalanceCredit,
		Currency:      finance.CurrencyINR,
		IsActive:      true,
	},
	{
		Code:          finance.AccountCodePlatformFeeRevenue,
		Name:          "Platform Fee Revenue",
		Type:          finance.AccountTypeRevenue,
		NormalBalance: finance.NormalBalanceCredit,
		Currency:      finance.CurrencyINR,
		IsActive:      true,
	},
	{
		Code:          finance.AccountCodePaymentGatewayExpense,
		Name:          "Payment Gateway Expense",
		Type:          finance.AccountTypeExpense,
		NormalBalance: finance.NormalBalanceDebit,
		Currency:      finance.CurrencyINR,
		IsActive:      true,
	},
}

// ValidateFinanceAccountConfig checks that an existing financial account strictly matches the expected configuration.
func ValidateFinanceAccountConfig(account *finance.Account, expected ExpectedFinanceAccount) error {
	if account == nil {
		return errors.New("finance account cannot be nil")
	}
	if account.Code() != expected.Code {
		return fmt.Errorf("%w: code mismatch: expected %q, got %q", ErrFinanceAccountConfigMismatch, expected.Code, account.Code())
	}
	if account.Type() != expected.Type {
		return fmt.Errorf("%w for account %s: type mismatch: expected %s, got %s", ErrFinanceAccountConfigMismatch, expected.Code, expected.Type, account.Type())
	}
	if account.NormalBalance() != expected.NormalBalance {
		return fmt.Errorf("%w for account %s: normal balance mismatch: expected %s, got %s", ErrFinanceAccountConfigMismatch, expected.Code, expected.NormalBalance, account.NormalBalance())
	}
	if account.Currency() != expected.Currency {
		return fmt.Errorf("%w for account %s: currency mismatch: expected %s, got %s", ErrFinanceAccountConfigMismatch, expected.Code, expected.Currency, account.Currency())
	}
	if account.IsActive() != expected.IsActive {
		return fmt.Errorf("%w for account %s: active status mismatch: expected %v, got %v", ErrFinanceAccountConfigMismatch, expected.Code, expected.IsActive, account.IsActive())
	}
	if account.Name() != expected.Name {
		return fmt.Errorf("%w for account %s: name mismatch: expected %q, got %q", ErrFinanceAccountConfigMismatch, expected.Code, expected.Name, account.Name())
	}
	return nil
}

// SeedFinanceAccountsWithRepo seeds the standard chart of accounts using the provided repository.
func SeedFinanceAccountsWithRepo(ctx context.Context, repo domain.FinanceAccountRepository, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	for _, expected := range StandardFinanceAccounts {
		existing, err := repo.FindByCode(ctx, expected.Code)
		if err == nil {
			if valErr := ValidateFinanceAccountConfig(existing, expected); valErr != nil {
				logger.ErrorContext(ctx,
					"finance_account_config_mismatch",
					slog.String("code", expected.Code),
					slog.Any("error", valErr),
				)
				return valErr
			}
			logger.InfoContext(ctx,
				"finance_account_already_exists",
				slog.String("code", expected.Code),
				slog.String("name", expected.Name),
			)
			continue
		}

		if !errors.Is(err, domain.ErrFinanceAccountNotFound) {
			logger.ErrorContext(ctx,
				"finance_account_lookup_failed",
				slog.String("code", expected.Code),
				slog.Any("error", err),
			)
			return err
		}

		newAcc, accErr := finance.NewAccount(
			uuid.New(),
			expected.Code,
			expected.Name,
			expected.Type,
			expected.NormalBalance,
			expected.Currency,
			expected.IsActive,
		)
		if accErr != nil {
			logger.ErrorContext(ctx,
				"finance_account_instantiation_failed",
				slog.String("code", expected.Code),
				slog.Any("error", accErr),
			)
			return accErr
		}

		createErr := repo.Create(ctx, newAcc)
		if createErr == nil {
			logger.InfoContext(ctx,
				"finance_account_seeded",
				slog.String("code", expected.Code),
				slog.String("name", expected.Name),
			)
			continue
		}

		if errors.Is(createErr, domain.ErrFinanceAccountAlreadyExists) {
			logger.WarnContext(ctx,
				"finance_account_concurrent_insert_detected",
				slog.String("code", expected.Code),
			)
			concurrentAcc, findErr := repo.FindByCode(ctx, expected.Code)
			if findErr != nil {
				logger.ErrorContext(ctx,
					"finance_account_concurrent_lookup_failed",
					slog.String("code", expected.Code),
					slog.Any("error", findErr),
				)
				return findErr
			}
			if valErr := ValidateFinanceAccountConfig(concurrentAcc, expected); valErr != nil {
				logger.ErrorContext(ctx,
					"finance_account_concurrent_config_mismatch",
					slog.String("code", expected.Code),
					slog.Any("error", valErr),
				)
				return valErr
			}
			logger.InfoContext(ctx,
				"finance_account_concurrent_validated",
				slog.String("code", expected.Code),
			)
			continue
		}

		logger.ErrorContext(ctx,
			"finance_account_seed_failed",
			slog.String("code", expected.Code),
			slog.Any("error", createErr),
		)
		return createErr
	}

	return nil
}

// SeedFinanceAccounts seeds the approved standard chart of accounts into the database.
func SeedFinanceAccounts(db *gorm.DB, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	repo := repository.NewFinanceAccountRepository(db, logger)
	return SeedFinanceAccountsWithRepo(context.Background(), repo, logger)
}

func SeedCategories(db *gorm.DB, logger *slog.Logger) error {

	categories := []models.CategoryModel{
		{
			Name:        "Comedy",
			Description: "Comedy events and shows",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Music",
			Description: "Music events, concerts, and live performances",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Sports",
			Description: "Sports events, matches, and tournaments",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Arts & Theatre",
			Description: "Theatre, drama, and performing arts",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Festivals",
			Description: "Cultural festivals and public celebrations",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Education",
			Description: "Workshops, seminars, and educational events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Technology",
			Description: "Tech conferences, developer meetups, and technology events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Business & Networking",
			Description: "Business conferences, networking, and professional events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Food & Drink",
			Description: "Food festivals, culinary experiences, and tastings",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Health & Wellness",
			Description: "Fitness sessions, wellness programs, and health events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Travel & Outdoor",
			Description: "Outdoor activities, adventure events, and travel experiences",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Family & Kids",
			Description: "Family-friendly activities and children's events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Movies & Film",
			Description: "Movie screenings, film festivals, and cinema events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Gaming & Esports",
			Description: "Gaming tournaments, esports, and gaming conventions",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Fashion & Beauty",
			Description: "Fashion shows, beauty events, and styling workshops",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Religion & Spirituality",
			Description: "Spiritual gatherings and religious events",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Charity & Causes",
			Description: "Fundraisers, charity events, and community causes",
			Status:      domain.CategoryStatusActive,
		},
		{
			Name:        "Networking & Meetups",
			Description: "Community gatherings and interest-based meetups",
			Status:      domain.CategoryStatusActive,
		},
	}

	for _, category := range categories {
		var existing models.CategoryModel

		err := db.
			Where("name = ?", category.Name).
			First(&existing).Error

		if err == nil {
			continue
		}

		if err != gorm.ErrRecordNotFound {
			logger.Error(
				"category_seed_lookup_failed",
				"name", category.Name,
				"error", err,
			)
			return err
		}

		category.ID = uuid.New()

		if err := db.Create(&category).Error; err != nil {
			logger.Error(
				"category_seed_failed",
				"name", category.Name,
				"error", err,
			)
			return err
		}

		logger.Info(
			"category_seeded",
			"name", category.Name,
		)
	}

	return nil
}
