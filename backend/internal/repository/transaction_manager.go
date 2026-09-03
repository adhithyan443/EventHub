package repository

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"gorm.io/gorm"
)

type TransactionManager struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewTransactionManager(
	db *gorm.DB,
	logger *slog.Logger,
) *TransactionManager {
	return &TransactionManager{
		db:     db,
		logger: logger,
	}
}

func (m *TransactionManager) WithinTransaction(
	fn func(tx domain.TransactionRepositories) error,
) error {
	return m.db.Transaction(func(txDB *gorm.DB) error {
		txRepos := &transactionRepositories{
			userRepo:                NewUserRepository(txDB),
			pendingRegistrationRepo: NewPendingRegistrationRepository(txDB),
			passwordResetTokenRepo:  NewPasswordResetTokenRepository(txDB),
			organizerApplicationRepo: NewOrganizerApplicationRepository(
				txDB,
				m.logger,
			),
		}

		return fn(txRepos)
	})
}

type transactionRepositories struct {
	userRepo                 *UserRepository
	pendingRegistrationRepo  *PendingRegistrationRepository
	passwordResetTokenRepo   domain.PasswordResetTokenRepository
	organizerApplicationRepo domain.OrganizerApplicationRepository
}

func (r *transactionRepositories) UserRepository() domain.UserRepository {
	return r.userRepo
}

func (r *transactionRepositories) PendingRegistrationRepository() domain.PendingRegistrationRepository {
	return r.pendingRegistrationRepo
}

func (r *transactionRepositories) PasswordResetTokenRepository() domain.PasswordResetTokenRepository {
	return r.passwordResetTokenRepo
}

func (r *transactionRepositories) OrganizerApplicationRepository() domain.OrganizerApplicationRepository {
	return r.organizerApplicationRepo
}