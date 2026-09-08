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
			userRepo:                NewUserRepository(txDB, m.logger),
			pendingRegistrationRepo: NewPendingRegistrationRepository(txDB),
			passwordResetTokenRepo:  NewPasswordResetTokenRepository(txDB),
			organizerApplicationRepo: NewOrganizerApplicationRepository(
				txDB,
				m.logger,
			),

			organizerRepo: NewOrganizerRepository(
				txDB,
				m.logger,
			),

			organizerProfileRepo: NewOrganizerProfileRepository(
				txDB,
				m.logger,
			),

			organizerAddressRepo: NewOrganizerAddressRepository(
				txDB,
				m.logger,
			),

			organizerBankAccountRepo: NewOrganizerBankAccountRepository(
				txDB,
				m.logger,
			),

			eventRepo: NewEventRepository(
				txDB,
				m.logger,
			),

			eventScheduleRepo: NewEventScheduleRepository(
				txDB,
				m.logger,
			),

			eventSettingRepo: NewEventSettingRepository(
				txDB,
				m.logger,
			),

			eventCancellationRepo: NewEventCancellationRepository(
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

	organizerRepo            domain.OrganizerRepository
	organizerProfileRepo     domain.OrganizerProfileRepository
	organizerAddressRepo     domain.OrganizerAddressRepository
	organizerBankAccountRepo domain.OrganizerBankAccountRepository

	eventRepo             domain.EventRepository
	eventScheduleRepo     domain.EventScheduleRepository
	eventSettingRepo      domain.EventSettingRepository
	eventCancellationRepo domain.EventCancellationRepository
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

func (r *transactionRepositories) OrganizerRepository() domain.OrganizerRepository {
	return r.organizerRepo
}

func (r *transactionRepositories) OrganizerProfileRepository() domain.OrganizerProfileRepository {
	return r.organizerProfileRepo
}

func (r *transactionRepositories) OrganizerAddressRepository() domain.OrganizerAddressRepository {
	return r.organizerAddressRepo
}

func (r *transactionRepositories) OrganizerBankAccountRepository() domain.OrganizerBankAccountRepository {
	return r.organizerBankAccountRepo
}

func (r *transactionRepositories) EventRepository() domain.EventRepository {
	return r.eventRepo
}

func (r *transactionRepositories) EventScheduleRepository() domain.EventScheduleRepository {
	return r.eventScheduleRepo
}

func (r *transactionRepositories) EventSettingRepository() domain.EventSettingRepository {
	return r.eventSettingRepo
}

func (r *transactionRepositories) EventCancellationRepository() domain.EventCancellationRepository {
	return r.eventCancellationRepo
}
