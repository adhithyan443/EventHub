package repository

import (
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"gorm.io/gorm"
)

type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{
		db: db,
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
		}

		return fn(txRepos)
	})
}

type transactionRepositories struct {
	userRepo                *UserRepository
	pendingRegistrationRepo *PendingRegistrationRepository
	passwordResetTokenRepo  domain.PasswordResetTokenRepository
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
