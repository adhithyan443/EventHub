package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizerBankAccountModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrganizerID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	BankName               string `gorm:"type:varchar(255);not null"`
	AccountHolderName      string `gorm:"type:varchar(255);not null"`
	AccountNumberEncrypted string `gorm:"type:text;not null"`
	IFSCCode               string `gorm:"type:varchar(20);not null"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}


func (OrganizerBankAccountModel) TableName() string {
	return "organizer_bank_accounts"
}