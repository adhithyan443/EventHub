package repository

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"gorm.io/gorm"
)

type TransactionManager struct {
	db     *gorm.DB
	logger *slog.Logger
}

type transactionRepositories struct {
	txDB                     *gorm.DB
	userRepo                 *UserRepository
	pendingRegistrationRepo  *PendingRegistrationRepository
	emailChangeRequestRepo   *EmailChangeRequestRepository
	passwordResetTokenRepo   domain.PasswordResetTokenRepository
	refreshTokenRepo         domain.RefreshTokenRepository
	organizerApplicationRepo domain.OrganizerApplicationRepository

	organizerRepo            domain.OrganizerRepository
	organizerProfileRepo     domain.OrganizerProfileRepository
	organizerAddressRepo     domain.OrganizerAddressRepository
	organizerBankAccountRepo domain.OrganizerBankAccountRepository

	categoryRepo domain.CategoryRepository
	venueRepo    domain.VenueRepository

	eventRepo             domain.EventRepository
	eventScheduleRepo     domain.EventScheduleRepository
	eventSettingRepo      domain.EventSettingRepository
	eventCancellationRepo domain.EventCancellationRepository
	eventContactRepo      domain.EventContactRepository

	seatLayoutRepository  domain.SeatLayoutRepository
	seatColumnRepository  domain.SeatColumnRepository
	seatSectionRepository domain.SeatSectionRepository
	seatRowRepository     domain.SeatRowRepository
	seatRepository        domain.SeatRepository

	ticketTypeRepo domain.TicketTypeRepository
	ticketRepo     domain.TicketRepository

	seatReservationRepo      domain.SeatReservationRepository
	checkoutRepo             domain.CheckoutRepository
	bookingRepo              domain.BookingRepository
	paymentRepo              domain.PaymentRepository
	refundRepo               domain.RefundRepository
	razorpayWebhookEventRepo domain.RazorpayWebhookEventRepository
	pricingTaxRepo           domain.PricingTaxConfigurationRepository
	invoiceRepo              domain.InvoiceRepository

	financeAccountRepo domain.FinanceAccountRepository
	financeJournalRepo domain.FinanceJournalRepository
	settlementRepo     domain.SettlementRepository
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
			txDB:                    txDB,
			userRepo:                NewUserRepository(txDB, m.logger),
			pendingRegistrationRepo: NewPendingRegistrationRepository(txDB),
			emailChangeRequestRepo:  NewEmailChangeRequestRepository(txDB),
			passwordResetTokenRepo:  NewPasswordResetTokenRepository(txDB),
			refreshTokenRepo:        NewRefreshTokenRepository(txDB),
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

			categoryRepo: NewCategoryRepository(
				txDB,
				m.logger,
			),

			venueRepo: NewVenueRepository(
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

			eventContactRepo: NewEventContactRepository(
				txDB,
				m.logger,
			),

			seatLayoutRepository:  NewSeatLayoutRepository(txDB, m.logger),
			seatColumnRepository:  NewSeatColumnRepository(txDB, m.logger),
			seatSectionRepository: NewSeatSectionRepository(txDB, m.logger),
			seatRowRepository:     NewSeatRowRepository(txDB, m.logger),
			seatRepository:        NewSeatRepository(txDB, m.logger),

			ticketTypeRepo: NewTicketTypeRepository(txDB, m.logger),
			ticketRepo:     NewTicketRepository(txDB, m.logger),

			seatReservationRepo:      NewSeatReservationRepository(txDB, m.logger),
			checkoutRepo:             NewCheckoutRepository(txDB, m.logger),
			bookingRepo:              NewBookingRepository(txDB, m.logger),
			paymentRepo:              NewPaymentRepository(txDB, m.logger),
			refundRepo:               NewRefundRepository(txDB, m.logger),
			razorpayWebhookEventRepo: NewRazorpayWebhookEventRepository(txDB, m.logger),
			pricingTaxRepo:           NewPricingTaxRepository(txDB, m.logger),
			invoiceRepo:              NewInvoiceRepository(txDB, m.logger),

			financeAccountRepo: NewFinanceAccountRepository(txDB, m.logger),
			financeJournalRepo: NewFinanceJournalRepository(txDB, m.logger),
			settlementRepo:     NewSettlementRepository(txDB, m.logger),
		}

		return fn(txRepos)
	})
}

func (r *transactionRepositories) UserRepository() domain.UserRepository {
	return r.userRepo
}

func (r *transactionRepositories) PendingRegistrationRepository() domain.PendingRegistrationRepository {
	return r.pendingRegistrationRepo
}

func (r *transactionRepositories) EmailChangeRequestRepository() domain.EmailChangeRequestRepository {
	return r.emailChangeRequestRepo
}

func (r *transactionRepositories) AcquireEmailAdvisoryLock(email string) error {
	normalized := strings.ToLower(strings.TrimSpace(email))
	return r.txDB.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", normalized).Error
}

func (r *transactionRepositories) PasswordResetTokenRepository() domain.PasswordResetTokenRepository {
	return r.passwordResetTokenRepo
}

func (r *transactionRepositories) RefreshTokenRepository() domain.RefreshTokenRepository {
	return r.refreshTokenRepo
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

func (r *transactionRepositories) CategoryRepository() domain.CategoryRepository {
	return r.categoryRepo
}

func (r *transactionRepositories) VenueRepository() domain.VenueRepository {
	return r.venueRepo
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

func (r *transactionRepositories) SeatLayoutRepository() domain.SeatLayoutRepository {
	return r.seatLayoutRepository
}

func (r *transactionRepositories) SeatColumnRepository() domain.SeatColumnRepository {
	return r.seatColumnRepository
}

func (r *transactionRepositories) SeatSectionRepository() domain.SeatSectionRepository {
	return r.seatSectionRepository
}

func (r *transactionRepositories) SeatRowRepository() domain.SeatRowRepository {
	return r.seatRowRepository
}

func (r *transactionRepositories) SeatRepository() domain.SeatRepository {
	return r.seatRepository
}

func (r *transactionRepositories) EventContactRepository() domain.EventContactRepository {
	return r.eventContactRepo
}

func (r *transactionRepositories) TicketTypeRepository() domain.TicketTypeRepository {
	return r.ticketTypeRepo
}

func (r *transactionRepositories) SeatReservationRepository() domain.SeatReservationRepository {
	return r.seatReservationRepo
}

func (r *transactionRepositories) CheckoutRepository() domain.CheckoutRepository {
	return r.checkoutRepo
}

func (r *transactionRepositories) BookingRepository() domain.BookingRepository {
	return r.bookingRepo
}

func (r *transactionRepositories) PaymentRepository() domain.PaymentRepository {
	return r.paymentRepo
}

func (r *transactionRepositories) RefundRepository() domain.RefundRepository {
	return r.refundRepo
}

func (r *transactionRepositories) RazorpayWebhookEventRepository() domain.RazorpayWebhookEventRepository {
	return r.razorpayWebhookEventRepo
}

func (r *transactionRepositories) TicketRepository() domain.TicketRepository {
	return r.ticketRepo
}

func (r *transactionRepositories) PricingTaxConfigurationRepository() domain.PricingTaxConfigurationRepository {
	return r.pricingTaxRepo
}

func (r *transactionRepositories) InvoiceRepository() domain.InvoiceRepository {
	return r.invoiceRepo
}

func (r *transactionRepositories) FinanceAccountRepository() domain.FinanceAccountRepository {
	return r.financeAccountRepo
}

func (r *transactionRepositories) FinanceJournalRepository() domain.FinanceJournalRepository {
	return r.financeJournalRepo
}

func (r *transactionRepositories) SettlementRepository() domain.SettlementRepository {
	return r.settlementRepo
}

var trustedSavepoints = map[domain.SavepointName]struct{}{
	domain.SavepointTaxRefundInventory: {},
}

func (r *transactionRepositories) ExecuteWithSavepoint(
	name domain.SavepointName,
	fn func() error,
) (domain.SavepointResult, error) {
	if fn == nil {
		return domain.SavepointResult{}, domain.ErrNilSavepointCallback
	}

	if _, ok := trustedSavepoints[name]; !ok {
		return domain.SavepointResult{}, domain.ErrInvalidSavepointName
	}

	// 1. Establish savepoint
	if err := r.txDB.Exec(fmt.Sprintf("SAVEPOINT %s", name)).Error; err != nil {
		return domain.SavepointResult{}, fmt.Errorf("%w: %w", domain.ErrSavepointCreateFailed, err)
	}

	// 2. Execute callback
	fnErr := fn()
	if fnErr == nil {
		// 3. Success path: release savepoint
		if err := r.txDB.Exec(fmt.Sprintf("RELEASE SAVEPOINT %s", name)).Error; err != nil {
			return domain.SavepointResult{}, fmt.Errorf("%w: %w", domain.ErrSavepointReleaseFail, err)
		}
		return domain.SavepointResult{
			Status:      domain.SavepointStatusCommitted,
			CallbackErr: nil,
		}, nil
	}

	// 4. Failure path: rollback to savepoint
	if rbErr := r.txDB.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", name)).Error; rbErr != nil {
		return domain.SavepointResult{}, fmt.Errorf("%w: %w (callback error: %v)", domain.ErrSavepointRollbackFail, rbErr, fnErr)
	}

	// 5. Post-rollback explicit release of savepoint
	if relErr := r.txDB.Exec(fmt.Sprintf("RELEASE SAVEPOINT %s", name)).Error; relErr != nil {
		return domain.SavepointResult{}, fmt.Errorf("%w: %w (callback error: %v)", domain.ErrSavepointReleaseFail, relErr, fnErr)
	}

	// 6. Return recoverable rolled-back status with the original callback error
	return domain.SavepointResult{
		Status:      domain.SavepointStatusRolledBack,
		CallbackErr: fnErr,
	}, nil
}
