package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NorthwindExternalAccount represents a registered external bank account validated via NorthWind
type NorthwindExternalAccount struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	UserID            *uuid.UUID `gorm:"type:uuid;index:idx_nw_ext_accounts_user_id" json:"user_id,omitempty"`
	AccountHolderName string     `gorm:"type:text;not null" json:"account_holder_name"`
	AccountNumber     string     `gorm:"type:text;not null" json:"account_number"`
	RoutingNumber     string     `gorm:"type:text;not null" json:"routing_number"`
	InstitutionName   *string    `gorm:"type:text" json:"institution_name,omitempty"`
	Validated         bool       `gorm:"not null;default:false" json:"validated"`
	ValidationTime    *time.Time `json:"validation_time,omitempty"`
	CreatedAt         time.Time  `gorm:"not null" json:"created_at"`
}

// TableName returns the table name for NorthwindExternalAccount
func (n *NorthwindExternalAccount) TableName() string {
	return "northwind_external_accounts"
}

// BeforeCreate hook for NorthwindExternalAccount
func (n *NorthwindExternalAccount) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	return nil
}
