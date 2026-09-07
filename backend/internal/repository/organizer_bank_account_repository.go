package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizerBankAccountRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewOrganizerBankAccountRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *OrganizerBankAccountRepository {
	return &OrganizerBankAccountRepository{
		db:     db,
		logger: logger,
	}
}

func (r *OrganizerBankAccountRepository) Create(
	account *domain.OrganizerBankAccount,
) error {
	model := organizerBankAccountToModel(account)

	if err := r.db.Create(&model).Error; err != nil {
		r.logger.Error(
			"organizer_bank_account_create_failed",
			"organizer_id", account.OrganizerID,
			"error", err,
		)
		return err
	}

	*account = *modelToOrganizerBankAccount(&model)

	r.logger.Info(
		"organizer_bank_account_created",
		"organizer_bank_account_id", account.ID,
		"organizer_id", account.OrganizerID,
	)

	return nil
}

func (r *OrganizerBankAccountRepository) FindByOrganizerID(
	organizerID uuid.UUID,
) (*domain.OrganizerBankAccount, error) {
	var model models.OrganizerBankAccountModel

	err := r.db.
		Where("organizer_id = ?", organizerID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrganizerBankAccountNotFound
		}

		r.logger.Error(
			"organizer_bank_account_find_failed",
			"organizer_id", organizerID,
			"error", err,
		)

		return nil, err
	}

	return modelToOrganizerBankAccount(&model), nil
}

func organizerBankAccountToModel(
	account *domain.OrganizerBankAccount,
) models.OrganizerBankAccountModel {
	return models.OrganizerBankAccountModel{
		ID:                     account.ID,
		OrganizerID:            account.OrganizerID,
		BankName:               account.BankName,
		AccountHolderName:      account.AccountHolderName,
		AccountNumberEncrypted: account.AccountNumberEncrypted,
		IFSCCode:               account.IFSCCode,
		CreatedAt:              account.CreatedAt,
		UpdatedAt:              account.UpdatedAt,
	}
}

func modelToOrganizerBankAccount(
	model *models.OrganizerBankAccountModel,
) *domain.OrganizerBankAccount {
	return &domain.OrganizerBankAccount{
		ID:                     model.ID,
		OrganizerID:            model.OrganizerID,
		BankName:               model.BankName,
		AccountHolderName:      model.AccountHolderName,
		AccountNumberEncrypted: model.AccountNumberEncrypted,
		IFSCCode:               model.IFSCCode,
		CreatedAt:              model.CreatedAt,
		UpdatedAt:              model.UpdatedAt,
	}
}
