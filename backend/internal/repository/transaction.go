package repository

import "gorm.io/gorm"

type Transaction struct {
	db *gorm.DB
}

func NewTransaction(db *gorm.DB) *Transaction {
	return &Transaction{
		db: db,
	}
}

func (t *Transaction) UserRepository() *UserRepository {
	return NewUserRepository(t.db)
}

func (t *Transaction) PendingRegistrationRepository() *PendingRegistrationRepository {
	return NewPendingRegistrationRepository(t.db)
}

func (t *Transaction) Run(fn func(tx *Transaction) error) error {
	return t.db.Transaction(func(txDB *gorm.DB) error {
		tx := NewTransaction(txDB)

		if err := fn(tx); err != nil {
			return err
		}

		return nil
	})
}