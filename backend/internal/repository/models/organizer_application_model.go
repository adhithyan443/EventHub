package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizerApplicationModel struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	BusinessName string `gorm:"type:varchar(255);not null"`
	BusinessType string `gorm:"type:varchar(100);not null"`
	Description  string `gorm:"type:text;not null"`
	Phone        string `gorm:"type:varchar(30);not null"`
	Website      string `gorm:"type:varchar(500)"`

	GSTNumber string `gorm:"type:varchar(50)"`
	PANNumber string `gorm:"type:varchar(20);not null"`

	BankName               string `gorm:"type:varchar(255);not null"`
	AccountHolderName      string `gorm:"type:varchar(255);not null"`
	AccountNumberEncrypted string `gorm:"type:text;not null"`
	IFSCCode               string `gorm:"type:varchar(20);not null"`

	LogoURL                 string `gorm:"type:varchar(1000)"`
	VerificationDocumentURL string `gorm:"type:varchar(1000);not null"`

	Status          string `gorm:"type:varchar(30);not null;index"`
	RejectionReason string `gorm:"type:text"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (OrganizerApplicationModel) TableName() string {
	return "organizer_applications"
}
