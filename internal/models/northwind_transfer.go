package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// NorthWind transfer status constants
const (
	NWTransferStatusPending    = "PENDING"
	NWTransferStatusProcessing = "PROCESSING"
	NWTransferStatusCompleted  = "COMPLETED"
	NWTransferStatusFailed     = "FAILED"
	NWTransferStatusCancelled  = "CANCELLED"
	NWTransferStatusReversed   = "REVERSED"
)

// NorthwindTransfer represents an external transfer tracked via NorthWind
type NorthwindTransfer struct {
	ID                         uuid.UUID        `gorm:"type:uuid;primary_key" json:"id"`
	UserID                     *uuid.UUID       `gorm:"type:uuid;index:idx_nw_transfers_user_id" json:"user_id,omitempty"`
	NorthwindTransferID        uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_nw_transfers_nw_id" json:"northwind_transfer_id"`
	Direction                  string           `gorm:"type:text;not null" json:"direction"`
	TransferType               string           `gorm:"type:text;not null" json:"transfer_type"`
	Amount                     decimal.Decimal  `gorm:"type:numeric(15,2);not null" json:"amount"`
	Currency                   string           `gorm:"type:text;not null;default:'USD'" json:"currency"`
	Description                *string          `gorm:"type:text" json:"description,omitempty"`
	ReferenceNumber            string           `gorm:"type:text;not null" json:"reference_number"`
	ScheduledDate              *time.Time       `json:"scheduled_date,omitempty"`
	SourceAccountNumber        string           `gorm:"type:text;not null" json:"source_account_number"`
	SourceRoutingNumber        *string          `gorm:"type:text" json:"source_routing_number,omitempty"`
	SourceAccountHolderName    *string          `gorm:"type:text" json:"source_account_holder_name,omitempty"`
	DestinationAccountNumber   string           `gorm:"type:text;not null" json:"destination_account_number"`
	DestinationRoutingNumber   *string          `gorm:"type:text" json:"destination_routing_number,omitempty"`
	DestinationAccountHolderName *string        `gorm:"type:text" json:"destination_account_holder_name,omitempty"`
	Status                     string           `gorm:"type:text;not null;default:'PENDING';index:idx_nw_transfers_status" json:"status"`
	ErrorCode                  *string          `gorm:"type:text" json:"error_code,omitempty"`
	ErrorMessage               *string          `gorm:"type:text" json:"error_message,omitempty"`
	InitiatedDate              *time.Time       `json:"initiated_date,omitempty"`
	ProcessingDate             *time.Time       `json:"processing_date,omitempty"`
	ExpectedCompletionDate     *time.Time       `json:"expected_completion_date,omitempty"`
	CompletedDate              *time.Time       `json:"completed_date,omitempty"`
	Fee                        *decimal.Decimal `gorm:"type:numeric(15,4)" json:"fee,omitempty"`
	ExchangeRate               *decimal.Decimal `gorm:"type:numeric(15,6)" json:"exchange_rate,omitempty"`
	CreatedAt                  time.Time        `gorm:"not null;index:idx_nw_transfers_created_at" json:"created_at"`
	UpdatedAt                  time.Time        `gorm:"not null" json:"updated_at"`
}

// TableName returns the table name for NorthwindTransfer
func (n *NorthwindTransfer) TableName() string {
	return "northwind_transfers"
}

// BeforeCreate hook for NorthwindTransfer
func (n *NorthwindTransfer) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	now := time.Now()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = now
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = now
	}
	if n.Status == "" {
		n.Status = NWTransferStatusPending
	}
	return nil
}

// BeforeUpdate hook for NorthwindTransfer
func (n *NorthwindTransfer) BeforeUpdate(tx *gorm.DB) error {
	n.UpdatedAt = time.Now()
	return nil
}

// IsTerminal returns true if the transfer is in a terminal state
func (n *NorthwindTransfer) IsTerminal() bool {
	return n.Status == NWTransferStatusCompleted ||
		n.Status == NWTransferStatusFailed ||
		n.Status == NWTransferStatusCancelled ||
		n.Status == NWTransferStatusReversed
}
