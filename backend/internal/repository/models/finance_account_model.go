package models

import (
	"time"

	"github.com/google/uuid"
)

// FinanceAccountModel represents an immutable financial account in the chart of accounts.
// Database table: finance_accounts
type FinanceAccountModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code          string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name          string    `gorm:"type:varchar(255);not null"`
	Type          string    `gorm:"type:varchar(50);not null"`
	NormalBalance string    `gorm:"type:varchar(10);not null"`
	Currency      string    `gorm:"type:varchar(3);not null;default:'INR'"`
	IsActive      bool      `gorm:"not null;default:true"`
	CreatedAt     time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (FinanceAccountModel) TableName() string {
	return "finance_accounts"
}
