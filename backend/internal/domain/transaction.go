package domain

type TransactionManager interface {
	WithinTransaction(fn func(tx TransactionRepositories) error) error
}

type TransactionRepositories interface {
	UserRepository() UserRepository
	PendingRegistrationRepository() PendingRegistrationRepository
}
