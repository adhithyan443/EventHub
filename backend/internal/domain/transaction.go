package domain

type TransactionManager interface {
	WithinTransaction(fn func(tx TransactionRepositories) error) error
}

type TransactionRepositories interface {
	UserRepository() UserRepository
	PendingRegistrationRepository() PendingRegistrationRepository
	PasswordResetTokenRepository() PasswordResetTokenRepository

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
}
