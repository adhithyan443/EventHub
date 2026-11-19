package domain

import "errors"

type SavepointName string

const (
	SavepointTaxRefundInventory SavepointName = "sp_tax_refund_inventory"
)

type SavepointStatus string

const (
	SavepointStatusCommitted  SavepointStatus = "COMMITTED"
	SavepointStatusRolledBack SavepointStatus = "ROLLED_BACK"
)

type SavepointResult struct {
	Status      SavepointStatus
	CallbackErr error
}

var (
	ErrNilSavepointCallback  = errors.New("savepoint callback function cannot be nil")
	ErrInvalidSavepointName  = errors.New("invalid or untrusted savepoint name")
	ErrSavepointCreateFailed = errors.New("failed to establish database savepoint")
	ErrSavepointRollbackFail = errors.New("failed to rollback to savepoint")
	ErrSavepointReleaseFail  = errors.New("failed to release savepoint")
)

type TransactionManager interface {
	WithinTransaction(fn func(tx TransactionRepositories) error) error
}

type TransactionRepositories interface {
	ExecuteWithSavepoint(name SavepointName, fn func() error) (SavepointResult, error)

	UserRepository() UserRepository
	PendingRegistrationRepository() PendingRegistrationRepository
	EmailChangeRequestRepository() EmailChangeRequestRepository
	AcquireEmailAdvisoryLock(email string) error
	PasswordResetTokenRepository() PasswordResetTokenRepository
	RefreshTokenRepository() RefreshTokenRepository

	OrganizerApplicationRepository() OrganizerApplicationRepository
	OrganizerRepository() OrganizerRepository
	OrganizerProfileRepository() OrganizerProfileRepository
	OrganizerAddressRepository() OrganizerAddressRepository
	OrganizerBankAccountRepository() OrganizerBankAccountRepository

	CategoryRepository() CategoryRepository
	VenueRepository() VenueRepository

	EventRepository() EventRepository
	EventScheduleRepository() EventScheduleRepository
	EventSettingRepository() EventSettingRepository
	EventCancellationRepository() EventCancellationRepository
	EventContactRepository() EventContactRepository

	SeatLayoutRepository() SeatLayoutRepository
	SeatColumnRepository() SeatColumnRepository
	SeatSectionRepository() SeatSectionRepository
	SeatRowRepository() SeatRowRepository
	SeatRepository() SeatRepository

	TicketTypeRepository() TicketTypeRepository
	TicketRepository() TicketRepository

	SeatReservationRepository() SeatReservationRepository
	CheckoutRepository() CheckoutRepository
	BookingRepository() BookingRepository
	PaymentRepository() PaymentRepository
	RefundRepository() RefundRepository
	RazorpayWebhookEventRepository() RazorpayWebhookEventRepository
	PricingTaxConfigurationRepository() PricingTaxConfigurationRepository
	InvoiceRepository() InvoiceRepository

	FinanceAccountRepository() FinanceAccountRepository
	FinanceJournalRepository() FinanceJournalRepository
	SettlementRepository() SettlementRepository
}
