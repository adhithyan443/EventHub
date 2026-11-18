package repository

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type financeAccountRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewFinanceAccountRepository creates a new PostgreSQL-backed FinanceAccountRepository.
func NewFinanceAccountRepository(db *gorm.DB, logger *slog.Logger) domain.FinanceAccountRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &financeAccountRepository{
		db:     db,
		logger: logger,
	}
}

func (r *financeAccountRepository) Create(ctx context.Context, account *finance.Account) error {
	if account == nil {
		return errors.New("account cannot be nil")
	}

	if err := account.Validate(); err != nil {
		return err
	}

	model := toFinanceAccountModel(account)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			r.logger.WarnContext(ctx, "finance account creation conflict",
				slog.String("account_id", account.ID().String()),
				slog.String("code", account.Code()),
				slog.String("constraint", pgErr.ConstraintName),
			)
			return domain.ErrFinanceAccountAlreadyExists
		}

		r.logger.ErrorContext(ctx, "failed to create finance account",
			slog.String("account_id", account.ID().String()),
			slog.String("code", account.Code()),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

func (r *financeAccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.Account, error) {
	if id == uuid.Nil {
		return nil, finance.ErrInvalidAccountID
	}

	var model models.FinanceAccountModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFinanceAccountNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find finance account by ID",
			slog.String("account_id", id.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	return toFinanceAccountDomain(&model)
}

func (r *financeAccountRepository) FindByCode(ctx context.Context, code string) (*finance.Account, error) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, finance.ErrInvalidAccountCode
	}

	var model models.FinanceAccountModel
	err := r.db.WithContext(ctx).Where("code = ?", cleanCode).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFinanceAccountNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find finance account by code",
			slog.String("code", cleanCode),
			slog.Any("error", err),
		)
		return nil, err
	}

	return toFinanceAccountDomain(&model)
}

func toFinanceAccountModel(account *finance.Account) *models.FinanceAccountModel {
	return &models.FinanceAccountModel{
		ID:            account.ID(),
		Code:          account.Code(),
		Name:          account.Name(),
		Type:          string(account.Type()),
		NormalBalance: string(account.NormalBalance()),
		Currency:      account.Currency(),
		IsActive:      account.IsActive(),
		CreatedAt:     time.Now().UTC(),
	}
}

func toFinanceAccountDomain(model *models.FinanceAccountModel) (*finance.Account, error) {
	return finance.NewAccount(
		model.ID,
		model.Code,
		model.Name,
		finance.AccountType(model.Type),
		finance.NormalBalance(model.NormalBalance),
		model.Currency,
		model.IsActive,
	)
}
